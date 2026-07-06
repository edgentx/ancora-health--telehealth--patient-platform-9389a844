package model

import (
	"errors"
	"testing"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"
)

func validComposePrescriptionCmd() ComposePrescriptionCmd {
	return ComposePrescriptionCmd{
		PatientId:  "patient-1",
		ProviderId: "provider-1",
		Medication: "amoxicillin",
		Dosage:     "500mg twice daily",
	}
}

func validTransmitPrescriptionCmd() TransmitPrescriptionCmd {
	return TransmitPrescriptionCmd{
		PrescriptionId: "rx-1",
		PharmacyId:     "pharmacy-1",
	}
}

func validRunSafetyCheckCmd() RunSafetyCheckCmd {
	return RunSafetyCheckCmd{
		PrescriptionId: "rx-1",
	}
}

// prescriptionInvariantCases enumerates the shared invariant violations that
// gate every prescription command.
func prescriptionInvariantCases() []struct {
	name    string
	mutate  func(*PrescriptionAggregate)
	wantErr error
} {
	return []struct {
		name    string
		mutate  func(*PrescriptionAggregate)
		wantErr error
	}{
		{
			name:    "provider unauthorized",
			mutate:  func(a *PrescriptionAggregate) { a.ProviderUnauthorized = true },
			wantErr: ErrProviderNotAuthorized,
		},
		{
			name:    "safety check failed and unacknowledged",
			mutate:  func(a *PrescriptionAggregate) { a.SafetyCheckFailed = true },
			wantErr: ErrSafetyCheckUnacknowledged,
		},
		{
			name:    "transmitted immutable",
			mutate:  func(a *PrescriptionAggregate) { a.Status = PrescriptionStatusTransmitted },
			wantErr: ErrTransmittedImmutable,
		},
	}
}

// assertPrescriptionRejected checks that a command execution produced the
// expected sentinel error, emitted no events, buffered nothing and left the
// version untouched.
func assertPrescriptionRejected(t *testing.T, agg *PrescriptionAggregate, events []shared.DomainEvent, err error, wantErr error) {
	t.Helper()
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
	if len(events) != 0 {
		t.Fatalf("expected no events on rejection, got %d", len(events))
	}
	if got := agg.Events(); len(got) != 0 {
		t.Fatalf("expected no buffered events on rejection, got %d", len(got))
	}
	if agg.Version != 0 {
		t.Fatalf("expected version to remain 0 on rejection, got %d", agg.Version)
	}
}

func TestPrescriptionExecuteComposeEmitsComposedEvent(t *testing.T) {
	agg := &PrescriptionAggregate{ID: "rx-1"}
	cmd := validComposePrescriptionCmd()

	events, err := agg.Execute(cmd)
	if err != nil {
		t.Fatalf("Execute(ComposePrescriptionCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	evt, ok := events[0].(PrescriptionComposedEvent)
	if !ok {
		t.Fatalf("expected PrescriptionComposedEvent, got %T", events[0])
	}
	if evt.Type() != PrescriptionComposedEventType || evt.Type() != "prescription.composed" {
		t.Fatalf("unexpected event type %q", evt.Type())
	}
	if evt.AggregateID() != "rx-1" {
		t.Fatalf("expected aggregate id rx-1, got %q", evt.AggregateID())
	}
	if evt.PrescriptionID != "rx-1" || evt.PatientID != cmd.PatientId || evt.ProviderID != cmd.ProviderId {
		t.Fatalf("event fields not copied from command: %+v", evt)
	}
	if evt.Medication != cmd.Medication || evt.Dosage != cmd.Dosage {
		t.Fatalf("event order fields not copied: %+v", evt)
	}

	if agg.Status != PrescriptionStatusComposed {
		t.Fatalf("expected status %q, got %q", PrescriptionStatusComposed, agg.Status)
	}
	if agg.ScopedPatientID != cmd.PatientId || agg.ScopedProviderID != cmd.ProviderId {
		t.Fatalf("aggregate not scoped to prescription: %+v", agg)
	}
	if agg.Medication != cmd.Medication || agg.Dosage != cmd.Dosage {
		t.Fatalf("aggregate order not set: %+v", agg)
	}
	if agg.Version != 1 {
		t.Fatalf("expected version 1, got %d", agg.Version)
	}
	if got := agg.Events(); len(got) != 1 {
		t.Fatalf("expected 1 buffered event, got %d", len(got))
	}
}

func TestPrescriptionExecuteComposeRejectsMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		cmd     ComposePrescriptionCmd
		wantErr error
	}{
		{
			name:    "missing patient",
			cmd:     ComposePrescriptionCmd{ProviderId: "provider-1", Medication: "m", Dosage: "d"},
			wantErr: ErrMissingPatient,
		},
		{
			name:    "missing provider",
			cmd:     ComposePrescriptionCmd{PatientId: "patient-1", Medication: "m", Dosage: "d"},
			wantErr: ErrMissingProvider,
		},
		{
			name:    "missing medication",
			cmd:     ComposePrescriptionCmd{PatientId: "patient-1", ProviderId: "provider-1", Dosage: "d"},
			wantErr: ErrMissingMedication,
		},
		{
			name:    "missing dosage",
			cmd:     ComposePrescriptionCmd{PatientId: "patient-1", ProviderId: "provider-1", Medication: "m"},
			wantErr: ErrMissingDosage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := &PrescriptionAggregate{ID: "rx-1"}
			events, err := agg.Execute(tt.cmd)
			assertPrescriptionRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestPrescriptionExecuteComposeRejectsInvariantViolations(t *testing.T) {
	for _, tt := range prescriptionInvariantCases() {
		t.Run(tt.name, func(t *testing.T) {
			agg := &PrescriptionAggregate{ID: "rx-1"}
			tt.mutate(agg)
			events, err := agg.Execute(validComposePrescriptionCmd())
			assertPrescriptionRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestPrescriptionExecuteComposeAllowsAcknowledgedSafetyOverride(t *testing.T) {
	agg := &PrescriptionAggregate{
		ID:                         "rx-1",
		SafetyCheckFailed:          true,
		SafetyOverrideAcknowledged: true,
	}

	events, err := agg.Execute(validComposePrescriptionCmd())
	if err != nil {
		t.Fatalf("Execute(ComposePrescriptionCmd) with acknowledged override returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if agg.Status != PrescriptionStatusComposed {
		t.Fatalf("expected status composed, got %q", agg.Status)
	}
}

func TestPrescriptionExecuteTransmitEmitsTransmittedEvent(t *testing.T) {
	agg := &PrescriptionAggregate{ID: "rx-1", Status: PrescriptionStatusComposed}
	cmd := validTransmitPrescriptionCmd()

	events, err := agg.Execute(cmd)
	if err != nil {
		t.Fatalf("Execute(TransmitPrescriptionCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	evt, ok := events[0].(PrescriptionTransmittedEvent)
	if !ok {
		t.Fatalf("expected PrescriptionTransmittedEvent, got %T", events[0])
	}
	if evt.Type() != PrescriptionTransmittedEventType || evt.Type() != "prescription.transmitted" {
		t.Fatalf("unexpected event type %q", evt.Type())
	}
	if evt.AggregateID() != "rx-1" {
		t.Fatalf("expected aggregate id rx-1, got %q", evt.AggregateID())
	}
	if evt.PrescriptionID != "rx-1" || evt.PharmacyID != cmd.PharmacyId {
		t.Fatalf("event fields not copied from command: %+v", evt)
	}

	if agg.Status != PrescriptionStatusTransmitted {
		t.Fatalf("expected status %q, got %q", PrescriptionStatusTransmitted, agg.Status)
	}
	if agg.Version != 1 {
		t.Fatalf("expected version 1, got %d", agg.Version)
	}
	if got := agg.Events(); len(got) != 1 {
		t.Fatalf("expected 1 buffered event, got %d", len(got))
	}
}

func TestPrescriptionExecuteTransmitRejectsMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		cmd     TransmitPrescriptionCmd
		wantErr error
	}{
		{
			name:    "missing prescription",
			cmd:     TransmitPrescriptionCmd{PharmacyId: "pharmacy-1"},
			wantErr: ErrMissingPrescription,
		},
		{
			name:    "missing pharmacy",
			cmd:     TransmitPrescriptionCmd{PrescriptionId: "rx-1"},
			wantErr: ErrMissingPharmacy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := &PrescriptionAggregate{ID: "rx-1", Status: PrescriptionStatusComposed}
			events, err := agg.Execute(tt.cmd)
			assertPrescriptionRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestPrescriptionExecuteTransmitRejectsInvariantViolations(t *testing.T) {
	for _, tt := range prescriptionInvariantCases() {
		t.Run(tt.name, func(t *testing.T) {
			agg := &PrescriptionAggregate{ID: "rx-1"}
			tt.mutate(agg)
			events, err := agg.Execute(validTransmitPrescriptionCmd())
			assertPrescriptionRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestPrescriptionExecuteRunSafetyCheckEmitsSafetyCheckedEvent(t *testing.T) {
	agg := &PrescriptionAggregate{ID: "rx-1", Status: PrescriptionStatusComposed}
	cmd := validRunSafetyCheckCmd()

	events, err := agg.Execute(cmd)
	if err != nil {
		t.Fatalf("Execute(RunSafetyCheckCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	evt, ok := events[0].(PrescriptionSafetyCheckedEvent)
	if !ok {
		t.Fatalf("expected PrescriptionSafetyCheckedEvent, got %T", events[0])
	}
	if evt.Type() != PrescriptionSafetyCheckedEventType || evt.Type() != "prescription.safety.checked" {
		t.Fatalf("unexpected event type %q", evt.Type())
	}
	if evt.AggregateID() != "rx-1" {
		t.Fatalf("expected aggregate id rx-1, got %q", evt.AggregateID())
	}
	if evt.PrescriptionID != "rx-1" {
		t.Fatalf("event prescription id not set: %+v", evt)
	}

	if !agg.SafetyChecked {
		t.Fatalf("expected SafetyChecked true after run")
	}
	if agg.Version != 1 {
		t.Fatalf("expected version 1, got %d", agg.Version)
	}
	if got := agg.Events(); len(got) != 1 {
		t.Fatalf("expected 1 buffered event, got %d", len(got))
	}
}

func TestPrescriptionExecuteRunSafetyCheckRejectsMissingFields(t *testing.T) {
	agg := &PrescriptionAggregate{ID: "rx-1"}
	events, err := agg.Execute(RunSafetyCheckCmd{})
	assertPrescriptionRejected(t, agg, events, err, ErrMissingPrescription)
}

func TestPrescriptionExecuteRunSafetyCheckRejectsInvariantViolations(t *testing.T) {
	for _, tt := range prescriptionInvariantCases() {
		t.Run(tt.name, func(t *testing.T) {
			agg := &PrescriptionAggregate{ID: "rx-1"}
			tt.mutate(agg)
			events, err := agg.Execute(validRunSafetyCheckCmd())
			assertPrescriptionRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestPrescriptionExecuteUnknownCommand(t *testing.T) {
	agg := &PrescriptionAggregate{ID: "rx-1"}

	events, err := agg.Execute(struct{ Unrecognized string }{Unrecognized: "x"})
	if !errors.Is(err, shared.ErrUnknownCommand) {
		t.Fatalf("expected ErrUnknownCommand, got %v", err)
	}
	if events != nil {
		t.Fatalf("expected nil events, got %v", events)
	}
	if agg.Version != 0 {
		t.Fatalf("expected version to remain 0, got %d", agg.Version)
	}
	if got := agg.Events(); len(got) != 0 {
		t.Fatalf("expected no buffered events, got %d", len(got))
	}
}
