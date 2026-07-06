package model

import (
	"errors"
	"testing"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"
)

func validPlaceLabOrderCmd() PlaceLabOrderCmd {
	return PlaceLabOrderCmd{
		PatientId:  "patient-1",
		ProviderId: "provider-1",
		TestCode:   "LOINC-1234",
	}
}

// assertLabOrderRejected checks that a command execution produced the expected
// sentinel error, emitted no events, buffered nothing and left the version at
// the supplied baseline.
func assertLabOrderRejected(t *testing.T, agg *LabOrderAggregate, events []shared.DomainEvent, err, wantErr error, wantVersion int) {
	t.Helper()
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
	if len(events) != 0 {
		t.Fatalf("expected no events on rejection, got %d", len(events))
	}
	if got := agg.Events(); len(got) != 0 {
		t.Fatalf("expected no buffered events on rejection, got %d", len(got))
	}
	if agg.Version != wantVersion {
		t.Fatalf("expected version %d on rejection, got %d", wantVersion, agg.Version)
	}
}

func TestLabOrderExecutePlaceLabOrderEmitsPlacedEvent(t *testing.T) {
	agg := &LabOrderAggregate{ID: "laborder-1"}
	cmd := validPlaceLabOrderCmd()

	events, err := agg.Execute(cmd)
	if err != nil {
		t.Fatalf("Execute(PlaceLabOrderCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	evt, ok := events[0].(LabOrderPlacedEvent)
	if !ok {
		t.Fatalf("event type = %T, want LabOrderPlacedEvent", events[0])
	}
	if evt.Type() != LabOrderPlacedEventType || evt.Type() != "lab.order.placed" {
		t.Fatalf("unexpected event type %q", evt.Type())
	}
	if evt.AggregateID() != "laborder-1" {
		t.Fatalf("event aggregate id = %q, want laborder-1", evt.AggregateID())
	}
	if evt.LabOrderID != "laborder-1" {
		t.Fatalf("event lab order id = %q, want laborder-1", evt.LabOrderID)
	}
	if evt.PatientID != cmd.PatientId || evt.ProviderID != cmd.ProviderId || evt.TestCode != cmd.TestCode {
		t.Fatalf("event fields not copied from command: %+v", evt)
	}

	if agg.Status != LabOrderStatusOrdered {
		t.Fatalf("aggregate status = %q, want %q", agg.Status, LabOrderStatusOrdered)
	}
	if agg.ScopedPatientID != cmd.PatientId || agg.ScopedProviderID != cmd.ProviderId {
		t.Fatalf("aggregate not scoped to participants: %+v", agg)
	}
	if !agg.CareRelationshipActive {
		t.Fatalf("expected care relationship marked active")
	}
	if agg.TestCode != cmd.TestCode {
		t.Fatalf("aggregate test code = %q, want %q", agg.TestCode, cmd.TestCode)
	}
	if agg.Version != 1 {
		t.Fatalf("aggregate version = %d, want 1", agg.Version)
	}
	if buffered := agg.Events(); len(buffered) != 1 {
		t.Fatalf("aggregate buffered %d events, want 1", len(buffered))
	}
}

func TestLabOrderExecutePlaceLabOrderRejectsMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		cmd     PlaceLabOrderCmd
		wantErr error
	}{
		{
			name:    "missing patient",
			cmd:     PlaceLabOrderCmd{ProviderId: "provider-1", TestCode: "LOINC-1234"},
			wantErr: ErrMissingLabPatient,
		},
		{
			name:    "missing provider",
			cmd:     PlaceLabOrderCmd{PatientId: "patient-1", TestCode: "LOINC-1234"},
			wantErr: ErrMissingLabProvider,
		},
		{
			name:    "missing test code",
			cmd:     PlaceLabOrderCmd{PatientId: "patient-1", ProviderId: "provider-1"},
			wantErr: ErrMissingTestCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := &LabOrderAggregate{ID: "laborder-1"}
			events, err := agg.Execute(tt.cmd)
			assertLabOrderRejected(t, agg, events, err, tt.wantErr, 0)
		})
	}
}

func TestLabOrderExecutePlaceLabOrderRejectsInvariantViolations(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*LabOrderAggregate)
		wantErr error
	}{
		{
			name: "provider not scoped",
			mutate: func(a *LabOrderAggregate) {
				a.ScopedProviderID = "other-provider"
				a.CareRelationshipActive = true
			},
			wantErr: ErrProviderNotInCare,
		},
		{
			name: "patient not scoped",
			mutate: func(a *LabOrderAggregate) {
				a.ScopedProviderID = "provider-1"
				a.ScopedPatientID = "other-patient"
				a.CareRelationshipActive = true
			},
			wantErr: ErrProviderNotInCare,
		},
		{
			name: "care relationship inactive",
			mutate: func(a *LabOrderAggregate) {
				a.ScopedProviderID = "provider-1"
				a.ScopedPatientID = "patient-1"
				a.CareRelationshipActive = false
			},
			wantErr: ErrProviderNotInCare,
		},
		{
			name: "order cancelled",
			mutate: func(a *LabOrderAggregate) {
				a.Status = LabOrderStatusCancelled
			},
			wantErr: ErrOrderCancelled,
		},
		{
			name: "resulted cannot revert",
			mutate: func(a *LabOrderAggregate) {
				a.Status = LabOrderStatusResulted
			},
			wantErr: ErrResultedCannotRevert,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := &LabOrderAggregate{ID: "laborder-1"}
			tt.mutate(agg)
			events, err := agg.Execute(validPlaceLabOrderCmd())
			assertLabOrderRejected(t, agg, events, err, tt.wantErr, 0)
		})
	}
}

func TestLabOrderExecuteRejectsUnknownCommand(t *testing.T) {
	agg := &LabOrderAggregate{ID: "laborder-1"}

	events, err := agg.Execute(struct{ Unrecognized string }{Unrecognized: "x"})
	if !errors.Is(err, shared.ErrUnknownCommand) {
		t.Fatalf("error = %v, want %v", err, shared.ErrUnknownCommand)
	}
	if events != nil {
		t.Fatalf("expected nil events, got %v", events)
	}
	if agg.Version != 0 {
		t.Fatalf("expected version 0, got %d", agg.Version)
	}
	if got := agg.Events(); len(got) != 0 {
		t.Fatalf("expected no buffered events, got %d", len(got))
	}
}

func TestLabOrderExecutePlaceLabOrderRescopedMatchingProviderSucceeds(t *testing.T) {
	// An already-scoped order whose provider still holds an active care
	// relationship accepts a command naming the same participants.
	agg := &LabOrderAggregate{
		ID:                     "laborder-1",
		Status:                 LabOrderStatusOrdered,
		ScopedProviderID:       "provider-1",
		ScopedPatientID:        "patient-1",
		CareRelationshipActive: true,
	}

	events, err := agg.Execute(validPlaceLabOrderCmd())
	if err != nil {
		t.Fatalf("Execute(PlaceLabOrderCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if agg.Version != 1 {
		t.Fatalf("aggregate version = %d, want 1", agg.Version)
	}
}
