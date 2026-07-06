package model

import (
	"errors"
	"testing"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"
)

func validInsurancePolicyAggregate() *InsurancePolicyAggregate {
	return &InsurancePolicyAggregate{ID: "policy-1"}
}

func validRegisterInsurancePolicyCmd() RegisterInsurancePolicyCmd {
	return RegisterInsurancePolicyCmd{
		PatientId:       "patient-1",
		PayerIdentifier: "payer-1",
		EffectiveDates:  EffectiveDates{Start: "2026-01-01", End: "2026-12-31"},
	}
}

func validVerifyEligibilityCmd() VerifyEligibilityCmd {
	return VerifyEligibilityCmd{
		PolicyId:    "policy-1",
		ServiceDate: "2026-07-06",
	}
}

// insurancePolicyInvariantCases flips one shared invariant flag at a time and
// expects the matching sentinel error. These guards run after field-completeness
// checks in every handler.
func insurancePolicyInvariantCases() []struct {
	name    string
	mutate  func(*InsurancePolicyAggregate)
	wantErr error
} {
	return []struct {
		name    string
		mutate  func(*InsurancePolicyAggregate)
		wantErr error
	}{
		{
			name:    "eligibility not verified",
			mutate:  func(a *InsurancePolicyAggregate) { a.EligibilityNotVerified = true },
			wantErr: ErrEligibilityNotVerified,
		},
		{
			name:    "active primary policy exists",
			mutate:  func(a *InsurancePolicyAggregate) { a.ActivePrimaryPolicyExists = true },
			wantErr: ErrActivePrimaryPolicyExists,
		},
		{
			name:    "policy expired",
			mutate:  func(a *InsurancePolicyAggregate) { a.PolicyExpired = true },
			wantErr: ErrPolicyExpired,
		},
	}
}

func assertInsurancePolicyRejected(t *testing.T, agg *InsurancePolicyAggregate, events []shared.DomainEvent, err error, wantErr error) {
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

func TestRegisterInsurancePolicyEmitsPolicyRegisteredEvent(t *testing.T) {
	agg := validInsurancePolicyAggregate()
	cmd := validRegisterInsurancePolicyCmd()

	events, err := agg.Execute(cmd)
	if err != nil {
		t.Fatalf("Execute(RegisterInsurancePolicyCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	evt, ok := events[0].(PolicyRegisteredEvent)
	if !ok {
		t.Fatalf("event type = %T, want PolicyRegisteredEvent", events[0])
	}
	if evt.Type() != PolicyRegisteredEventType || evt.Type() != "policy.registered" {
		t.Fatalf("unexpected event type %q", evt.Type())
	}
	if evt.AggregateID() != agg.ID {
		t.Fatalf("event aggregate id = %q, want %q", evt.AggregateID(), agg.ID)
	}
	if evt.PolicyID != agg.ID {
		t.Fatalf("event policy id = %q, want %q", evt.PolicyID, agg.ID)
	}
	if evt.PatientID != cmd.PatientId || evt.PayerIdentifier != cmd.PayerIdentifier || evt.EffectiveDates != cmd.EffectiveDates {
		t.Fatalf("event fields not copied from command: %#v", evt)
	}

	if agg.Status != PolicyStatusRegistered {
		t.Fatalf("aggregate status = %q, want %q", agg.Status, PolicyStatusRegistered)
	}
	if agg.PatientID != cmd.PatientId || agg.PayerIdentifier != cmd.PayerIdentifier || agg.EffectiveDates != cmd.EffectiveDates {
		t.Fatalf("aggregate not scoped to registration: %#v", agg)
	}
	if agg.Version != 1 {
		t.Fatalf("aggregate version = %d, want 1", agg.Version)
	}
	if buffered := agg.Events(); len(buffered) != 1 {
		t.Fatalf("aggregate buffered %d events, want 1", len(buffered))
	}
}

func TestRegisterInsurancePolicyRejectsMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		cmd     RegisterInsurancePolicyCmd
		wantErr error
	}{
		{
			name:    "missing patient",
			cmd:     RegisterInsurancePolicyCmd{PayerIdentifier: "payer-1", EffectiveDates: EffectiveDates{Start: "2026-01-01", End: "2026-12-31"}},
			wantErr: ErrMissingPatient,
		},
		{
			name:    "missing payer identifier",
			cmd:     RegisterInsurancePolicyCmd{PatientId: "patient-1", EffectiveDates: EffectiveDates{Start: "2026-01-01", End: "2026-12-31"}},
			wantErr: ErrMissingPayerIdentifier,
		},
		{
			name:    "missing effective start",
			cmd:     RegisterInsurancePolicyCmd{PatientId: "patient-1", PayerIdentifier: "payer-1", EffectiveDates: EffectiveDates{End: "2026-12-31"}},
			wantErr: ErrMissingEffectiveDates,
		},
		{
			name:    "missing effective end",
			cmd:     RegisterInsurancePolicyCmd{PatientId: "patient-1", PayerIdentifier: "payer-1", EffectiveDates: EffectiveDates{Start: "2026-01-01"}},
			wantErr: ErrMissingEffectiveDates,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := validInsurancePolicyAggregate()
			events, err := agg.Execute(tt.cmd)
			assertInsurancePolicyRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestRegisterInsurancePolicyRejectsInvariantViolations(t *testing.T) {
	for _, tt := range insurancePolicyInvariantCases() {
		t.Run(tt.name, func(t *testing.T) {
			agg := validInsurancePolicyAggregate()
			tt.mutate(agg)
			events, err := agg.Execute(validRegisterInsurancePolicyCmd())
			assertInsurancePolicyRejected(t, agg, events, err, tt.wantErr)
			if agg.Status != PolicyStatusNew {
				t.Fatalf("status = %q, want new (unchanged)", agg.Status)
			}
		})
	}
}

func TestVerifyEligibilityEmitsEligibilityVerifiedEvent(t *testing.T) {
	agg := validInsurancePolicyAggregate()
	agg.EligibilityNotVerified = false
	cmd := validVerifyEligibilityCmd()

	events, err := agg.Execute(cmd)
	if err != nil {
		t.Fatalf("Execute(VerifyEligibilityCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	evt, ok := events[0].(EligibilityVerifiedEvent)
	if !ok {
		t.Fatalf("event type = %T, want EligibilityVerifiedEvent", events[0])
	}
	if evt.Type() != EligibilityVerifiedEventType || evt.Type() != "eligibility.verified" {
		t.Fatalf("unexpected event type %q", evt.Type())
	}
	if evt.AggregateID() != agg.ID {
		t.Fatalf("event aggregate id = %q, want %q", evt.AggregateID(), agg.ID)
	}
	if evt.PolicyID != agg.ID {
		t.Fatalf("event policy id = %q, want %q", evt.PolicyID, agg.ID)
	}
	if evt.ServiceDate != cmd.ServiceDate {
		t.Fatalf("event service date = %q, want %q", evt.ServiceDate, cmd.ServiceDate)
	}

	if agg.VerifiedServiceDate != cmd.ServiceDate {
		t.Fatalf("aggregate verified service date = %q, want %q", agg.VerifiedServiceDate, cmd.ServiceDate)
	}
	if agg.EligibilityNotVerified {
		t.Fatalf("expected eligibility-not-verified flag cleared")
	}
	if agg.Version != 1 {
		t.Fatalf("aggregate version = %d, want 1", agg.Version)
	}
	if buffered := agg.Events(); len(buffered) != 1 {
		t.Fatalf("aggregate buffered %d events, want 1", len(buffered))
	}
}

func TestVerifyEligibilityRejectsMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		cmd     VerifyEligibilityCmd
		wantErr error
	}{
		{
			name:    "missing policy id",
			cmd:     VerifyEligibilityCmd{ServiceDate: "2026-07-06"},
			wantErr: ErrMissingPolicyID,
		},
		{
			name:    "missing service date",
			cmd:     VerifyEligibilityCmd{PolicyId: "policy-1"},
			wantErr: ErrMissingServiceDate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := validInsurancePolicyAggregate()
			events, err := agg.Execute(tt.cmd)
			assertInsurancePolicyRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestVerifyEligibilityRejectsInvariantViolations(t *testing.T) {
	for _, tt := range insurancePolicyInvariantCases() {
		t.Run(tt.name, func(t *testing.T) {
			agg := validInsurancePolicyAggregate()
			tt.mutate(agg)
			events, err := agg.Execute(validVerifyEligibilityCmd())
			assertInsurancePolicyRejected(t, agg, events, err, tt.wantErr)
			if agg.VerifiedServiceDate != "" {
				t.Fatalf("verified service date = %q, want empty (unchanged)", agg.VerifiedServiceDate)
			}
		})
	}
}

func TestInsurancePolicyExecuteRejectsUnknownCommand(t *testing.T) {
	agg := validInsurancePolicyAggregate()

	events, err := agg.Execute(struct{ Unrecognized string }{Unrecognized: "x"})
	if !errors.Is(err, shared.ErrUnknownCommand) {
		t.Fatalf("error = %v, want %v", err, shared.ErrUnknownCommand)
	}
	if events != nil {
		t.Fatalf("expected nil events, got %v", events)
	}
	if len(agg.Events()) != 0 {
		t.Fatalf("expected no buffered events, got %d", len(agg.Events()))
	}
	if agg.Version != 0 {
		t.Fatalf("expected version 0, got %d", agg.Version)
	}
}

func TestInsurancePolicyEventAccessors(t *testing.T) {
	registered := PolicyRegisteredEvent{PolicyID: "policy-9"}
	if registered.Type() != PolicyRegisteredEventType || registered.AggregateID() != "policy-9" {
		t.Fatalf("PolicyRegisteredEvent accessors = %q/%q", registered.Type(), registered.AggregateID())
	}

	verified := EligibilityVerifiedEvent{PolicyID: "policy-9"}
	if verified.Type() != EligibilityVerifiedEventType || verified.AggregateID() != "policy-9" {
		t.Fatalf("EligibilityVerifiedEvent accessors = %q/%q", verified.Type(), verified.AggregateID())
	}
}

func TestInsurancePolicyAggregateRootHelpers(t *testing.T) {
	agg := validInsurancePolicyAggregate()

	if _, err := agg.Execute(validRegisterInsurancePolicyCmd()); err != nil {
		t.Fatalf("Execute(RegisterInsurancePolicyCmd) returned error: %v", err)
	}

	if agg.GetVersion() != 1 {
		t.Fatalf("expected GetVersion 1, got %d", agg.GetVersion())
	}
	if len(agg.Events()) != 1 {
		t.Fatalf("expected 1 buffered event, got %d", len(agg.Events()))
	}

	agg.ClearEvents()
	if len(agg.Events()) != 0 {
		t.Fatalf("expected events cleared, got %d", len(agg.Events()))
	}
	if agg.GetVersion() != 1 {
		t.Fatalf("expected version unchanged after ClearEvents, got %d", agg.GetVersion())
	}
}
