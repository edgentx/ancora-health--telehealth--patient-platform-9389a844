package model

import (
	"errors"
	"testing"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"
)

func validSubmitIntakeFormCmd() SubmitIntakeFormCmd {
	return SubmitIntakeFormCmd{
		PatientId:    "patient-1",
		History:      "no chronic conditions",
		Demographics: "dob 1990-01-01",
	}
}

func validSeedChartFromIntakeCmd() SeedChartFromIntakeCmd {
	return SeedChartFromIntakeCmd{
		IntakeId:  "intake-1",
		PatientId: "patient-1",
	}
}

// assertIntakeRejected checks that a command execution produced the expected
// sentinel error, emitted no events, buffered nothing and left the version
// untouched.
func assertIntakeRejected(t *testing.T, agg *IntakeFormAggregate, events []shared.DomainEvent, err error, wantErr error) {
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

func TestIntakeFormExecuteSubmitEmitsSubmittedEvent(t *testing.T) {
	agg := &IntakeFormAggregate{ID: "intake-form-1"}
	cmd := validSubmitIntakeFormCmd()

	events, err := agg.Execute(cmd)
	if err != nil {
		t.Fatalf("Execute(SubmitIntakeFormCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	evt, ok := events[0].(IntakeFormSubmittedEvent)
	if !ok {
		t.Fatalf("expected IntakeFormSubmittedEvent, got %T", events[0])
	}
	if evt.Type() != IntakeFormSubmittedEventType || evt.Type() != "intake.form.submitted" {
		t.Fatalf("unexpected event type %q", evt.Type())
	}
	if evt.AggregateID() != "intake-form-1" {
		t.Fatalf("expected aggregate id intake-form-1, got %q", evt.AggregateID())
	}
	if evt.IntakeFormID != "intake-form-1" {
		t.Fatalf("expected intake form id intake-form-1, got %q", evt.IntakeFormID)
	}
	if evt.PatientID != cmd.PatientId || evt.History != cmd.History || evt.Demographics != cmd.Demographics {
		t.Fatalf("event fields not copied from command: %+v", evt)
	}

	if agg.Status != IntakeFormStatusSubmitted {
		t.Fatalf("expected status %q, got %q", IntakeFormStatusSubmitted, agg.Status)
	}
	if agg.ScopedPatientID != cmd.PatientId || agg.History != cmd.History || agg.Demographics != cmd.Demographics {
		t.Fatalf("aggregate not scoped to submission: %+v", agg)
	}
	if agg.Version != 1 {
		t.Fatalf("expected version 1, got %d", agg.Version)
	}
	if got := agg.Events(); len(got) != 1 {
		t.Fatalf("expected 1 buffered event, got %d", len(got))
	}
}

func TestIntakeFormExecuteSubmitRejectsMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		cmd     SubmitIntakeFormCmd
		wantErr error
	}{
		{
			name:    "missing patient",
			cmd:     SubmitIntakeFormCmd{History: "h", Demographics: "d"},
			wantErr: ErrMissingIntakePatient,
		},
		{
			name:    "blank patient",
			cmd:     SubmitIntakeFormCmd{PatientId: "   ", History: "h", Demographics: "d"},
			wantErr: ErrMissingIntakePatient,
		},
		{
			name:    "missing history",
			cmd:     SubmitIntakeFormCmd{PatientId: "patient-1", Demographics: "d"},
			wantErr: ErrMissingIntakeHistory,
		},
		{
			name:    "missing demographics",
			cmd:     SubmitIntakeFormCmd{PatientId: "patient-1", History: "h"},
			wantErr: ErrMissingIntakeDemographics,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := &IntakeFormAggregate{ID: "intake-form-1"}
			events, err := agg.Execute(tt.cmd)
			assertIntakeRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestIntakeFormExecuteSubmitRejectsInvariantViolations(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*IntakeFormAggregate)
		wantErr error
	}{
		{
			name:    "incomplete form",
			mutate:  func(a *IntakeFormAggregate) { a.Incomplete = true },
			wantErr: ErrIntakeFormIncomplete,
		},
		{
			name:    "already submitted immutable",
			mutate:  func(a *IntakeFormAggregate) { a.Status = IntakeFormStatusSubmitted },
			wantErr: ErrIntakeFormSubmittedImmutable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := &IntakeFormAggregate{ID: "intake-form-1"}
			tt.mutate(agg)
			events, err := agg.Execute(validSubmitIntakeFormCmd())
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
			if len(events) != 0 {
				t.Fatalf("expected no events, got %d", len(events))
			}
			if got := agg.Events(); len(got) != 0 {
				t.Fatalf("expected no buffered events, got %d", len(got))
			}
			if agg.Version != 0 {
				t.Fatalf("expected version 0, got %d", agg.Version)
			}
		})
	}
}

func TestIntakeFormExecuteSeedChartEmitsChartSeededEvent(t *testing.T) {
	agg := &IntakeFormAggregate{ID: "intake-form-1", Status: IntakeFormStatusSubmitted}
	cmd := validSeedChartFromIntakeCmd()

	events, err := agg.Execute(cmd)
	if err != nil {
		t.Fatalf("Execute(SeedChartFromIntakeCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	evt, ok := events[0].(IntakeChartSeededEvent)
	if !ok {
		t.Fatalf("expected IntakeChartSeededEvent, got %T", events[0])
	}
	if evt.Type() != IntakeChartSeededEventType || evt.Type() != "intake.chart.seeded" {
		t.Fatalf("unexpected event type %q", evt.Type())
	}
	if evt.AggregateID() != "intake-form-1" {
		t.Fatalf("expected aggregate id intake-form-1, got %q", evt.AggregateID())
	}
	if evt.IntakeFormID != "intake-form-1" || evt.IntakeID != cmd.IntakeId || evt.PatientID != cmd.PatientId {
		t.Fatalf("event fields not copied from command: %+v", evt)
	}

	if agg.Status != IntakeFormStatusSeeded {
		t.Fatalf("expected status %q, got %q", IntakeFormStatusSeeded, agg.Status)
	}
	if agg.ScopedPatientID != cmd.PatientId {
		t.Fatalf("expected scoped patient %q, got %q", cmd.PatientId, agg.ScopedPatientID)
	}
	if agg.Version != 1 {
		t.Fatalf("expected version 1, got %d", agg.Version)
	}
	if got := agg.Events(); len(got) != 1 {
		t.Fatalf("expected 1 buffered event, got %d", len(got))
	}
}

func TestIntakeFormExecuteSeedChartRejectsMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		cmd     SeedChartFromIntakeCmd
		wantErr error
	}{
		{
			name:    "missing intake id",
			cmd:     SeedChartFromIntakeCmd{PatientId: "patient-1"},
			wantErr: ErrMissingIntakeID,
		},
		{
			name:    "blank intake id",
			cmd:     SeedChartFromIntakeCmd{IntakeId: "  ", PatientId: "patient-1"},
			wantErr: ErrMissingIntakeID,
		},
		{
			name:    "missing patient",
			cmd:     SeedChartFromIntakeCmd{IntakeId: "intake-1"},
			wantErr: ErrMissingIntakePatient,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := &IntakeFormAggregate{ID: "intake-form-1"}
			events, err := agg.Execute(tt.cmd)
			assertIntakeRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestIntakeFormExecuteSeedChartRejectsInvariantViolations(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*IntakeFormAggregate)
		wantErr error
	}{
		{
			name:    "incomplete form",
			mutate:  func(a *IntakeFormAggregate) { a.Incomplete = true },
			wantErr: ErrIntakeFormIncomplete,
		},
		{
			name:    "already seeded immutable",
			mutate:  func(a *IntakeFormAggregate) { a.Status = IntakeFormStatusSeeded },
			wantErr: ErrIntakeFormSubmittedImmutable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := &IntakeFormAggregate{ID: "intake-form-1"}
			tt.mutate(agg)
			events, err := agg.Execute(validSeedChartFromIntakeCmd())
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
			if len(events) != 0 {
				t.Fatalf("expected no events, got %d", len(events))
			}
			if got := agg.Events(); len(got) != 0 {
				t.Fatalf("expected no buffered events, got %d", len(got))
			}
			if agg.Version != 0 {
				t.Fatalf("expected version 0, got %d", agg.Version)
			}
		})
	}
}

func TestIntakeFormExecuteUnknownCommand(t *testing.T) {
	agg := &IntakeFormAggregate{ID: "intake-form-1"}

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
