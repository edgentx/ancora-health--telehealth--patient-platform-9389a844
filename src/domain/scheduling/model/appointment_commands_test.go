package model

import (
	"errors"
	"testing"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"
)

// invariantCase is a shared invariant-violation scenario: each mutation flips one
// invariant flag and expects the matching sentinel error.
type invariantCase struct {
	name    string
	mutate  func(*AppointmentAggregate)
	wantErr error
}

func appointmentInvariantCases() []invariantCase {
	return []invariantCase{
		{
			name:    "slot outside availability",
			mutate:  func(a *AppointmentAggregate) { a.SlotOutsideAvailability = true },
			wantErr: ErrSlotOutsideAvailability,
		},
		{
			name:    "slot double booked",
			mutate:  func(a *AppointmentAggregate) { a.SlotAlreadyBooked = true },
			wantErr: ErrSlotDoubleBooked,
		},
		{
			name:    "hold lock expired",
			mutate:  func(a *AppointmentAggregate) { a.HoldLockExpired = true },
			wantErr: ErrHoldLockExpired,
		},
		{
			name:    "outside policy window",
			mutate:  func(a *AppointmentAggregate) { a.OutsidePolicyWindow = true },
			wantErr: ErrOutsidePolicyWindow,
		},
	}
}

// assertRejected checks that a command execution produced the expected sentinel
// error, emitted no events, buffered nothing and left the version untouched.
func assertRejected(t *testing.T, agg *AppointmentAggregate, events []shared.DomainEvent, err error, wantErr error) {
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

func TestAppointmentExecuteHoldSlotEmitsSlotHeldEvent(t *testing.T) {
	agg := &AppointmentAggregate{ID: "appointment-1"}

	events, err := agg.Execute(HoldSlotCmd{
		ProviderId: "provider-1",
		TimeSlot:   "2026-07-06T09:00:00Z",
		PatientId:  "patient-1",
	})
	if err != nil {
		t.Fatalf("Execute(HoldSlotCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	event, ok := events[0].(AppointmentSlotHeldEvent)
	if !ok {
		t.Fatalf("expected AppointmentSlotHeldEvent, got %T", events[0])
	}
	if event.Type() != AppointmentSlotHeldEventType || event.Type() != "appointment.slot.held" {
		t.Fatalf("unexpected event type %q", event.Type())
	}
	if event.AggregateID() != "appointment-1" {
		t.Fatalf("expected aggregate id appointment-1, got %q", event.AggregateID())
	}
	if event.ProviderID != "provider-1" || event.TimeSlot != "2026-07-06T09:00:00Z" || event.PatientID != "patient-1" {
		t.Fatalf("event fields not copied from command: %+v", event)
	}
	if agg.Status != AppointmentStatusHeld {
		t.Fatalf("expected status %q, got %q", AppointmentStatusHeld, agg.Status)
	}
	if agg.ScopedProviderID != "provider-1" || agg.ScopedPatientID != "patient-1" || agg.HeldTimeSlot != "2026-07-06T09:00:00Z" {
		t.Fatalf("aggregate not scoped to held slot: %+v", agg)
	}
	if agg.Version != 1 {
		t.Fatalf("expected version 1, got %d", agg.Version)
	}
	if got := agg.Events(); len(got) != 1 {
		t.Fatalf("expected 1 buffered event, got %d", len(got))
	}
}

func TestAppointmentExecuteHoldSlotRejectsMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		cmd     HoldSlotCmd
		wantErr error
	}{
		{
			name:    "missing provider",
			cmd:     HoldSlotCmd{TimeSlot: "slot", PatientId: "patient-1"},
			wantErr: ErrMissingProvider,
		},
		{
			name:    "missing time slot",
			cmd:     HoldSlotCmd{ProviderId: "provider-1", PatientId: "patient-1"},
			wantErr: ErrMissingTimeSlot,
		},
		{
			name:    "missing patient",
			cmd:     HoldSlotCmd{ProviderId: "provider-1", TimeSlot: "slot"},
			wantErr: ErrMissingPatient,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := &AppointmentAggregate{ID: "appointment-1"}
			events, err := agg.Execute(tt.cmd)
			assertRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestAppointmentExecuteHoldSlotRejectsInvariantViolations(t *testing.T) {
	for _, tt := range appointmentInvariantCases() {
		t.Run(tt.name, func(t *testing.T) {
			agg := &AppointmentAggregate{ID: "appointment-1"}
			tt.mutate(agg)
			events, err := agg.Execute(HoldSlotCmd{
				ProviderId: "provider-1",
				TimeSlot:   "slot",
				PatientId:  "patient-1",
			})
			assertRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestAppointmentExecuteRescheduleEmitsRescheduledEvent(t *testing.T) {
	agg := &AppointmentAggregate{
		ID:           "appointment-1",
		Status:       AppointmentStatusHeld,
		HeldTimeSlot: "2026-07-06T09:00:00Z",
	}

	events, err := agg.Execute(RescheduleAppointmentCmd{
		AppointmentId: "appointment-1",
		NewTimeSlot:   "2026-07-07T10:00:00Z",
	})
	if err != nil {
		t.Fatalf("Execute(RescheduleAppointmentCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	event, ok := events[0].(AppointmentRescheduledEvent)
	if !ok {
		t.Fatalf("expected AppointmentRescheduledEvent, got %T", events[0])
	}
	if event.Type() != AppointmentRescheduledEventType || event.Type() != "appointment.rescheduled" {
		t.Fatalf("unexpected event type %q", event.Type())
	}
	if event.AggregateID() != "appointment-1" {
		t.Fatalf("expected aggregate id appointment-1, got %q", event.AggregateID())
	}
	if event.PreviousTimeSlot != "2026-07-06T09:00:00Z" {
		t.Fatalf("expected previous slot to carry the held slot, got %q", event.PreviousTimeSlot)
	}
	if event.NewTimeSlot != "2026-07-07T10:00:00Z" {
		t.Fatalf("expected new slot from command, got %q", event.NewTimeSlot)
	}
	if agg.Status != AppointmentStatusHeld {
		t.Fatalf("expected status to remain held, got %q", agg.Status)
	}
	if agg.HeldTimeSlot != "2026-07-07T10:00:00Z" {
		t.Fatalf("expected held slot to move to new slot, got %q", agg.HeldTimeSlot)
	}
	if agg.Version != 1 {
		t.Fatalf("expected version 1, got %d", agg.Version)
	}
	if got := agg.Events(); len(got) != 1 {
		t.Fatalf("expected 1 buffered event, got %d", len(got))
	}
}

func TestAppointmentExecuteRescheduleRejectsMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		cmd     RescheduleAppointmentCmd
		wantErr error
	}{
		{
			name:    "missing appointment",
			cmd:     RescheduleAppointmentCmd{NewTimeSlot: "slot"},
			wantErr: ErrMissingAppointment,
		},
		{
			name:    "missing new time slot",
			cmd:     RescheduleAppointmentCmd{AppointmentId: "appointment-1"},
			wantErr: ErrMissingNewTimeSlot,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := &AppointmentAggregate{ID: "appointment-1"}
			events, err := agg.Execute(tt.cmd)
			assertRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestAppointmentExecuteRescheduleRejectsInvariantViolations(t *testing.T) {
	for _, tt := range appointmentInvariantCases() {
		t.Run(tt.name, func(t *testing.T) {
			agg := &AppointmentAggregate{ID: "appointment-1"}
			tt.mutate(agg)
			events, err := agg.Execute(RescheduleAppointmentCmd{
				AppointmentId: "appointment-1",
				NewTimeSlot:   "slot",
			})
			assertRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestAppointmentExecuteRegisterWalkInEmitsWalkInRegisteredEvent(t *testing.T) {
	agg := &AppointmentAggregate{ID: "appointment-1"}

	events, err := agg.Execute(RegisterWalkInCmd{
		PatientId:  "patient-1",
		ClinicId:   "clinic-1",
		ProviderId: "provider-1",
	})
	if err != nil {
		t.Fatalf("Execute(RegisterWalkInCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	event, ok := events[0].(AppointmentWalkInRegisteredEvent)
	if !ok {
		t.Fatalf("expected AppointmentWalkInRegisteredEvent, got %T", events[0])
	}
	if event.Type() != AppointmentWalkInRegisteredEventType || event.Type() != "appointment.walkin.registered" {
		t.Fatalf("unexpected event type %q", event.Type())
	}
	if event.AggregateID() != "appointment-1" {
		t.Fatalf("expected aggregate id appointment-1, got %q", event.AggregateID())
	}
	if event.PatientID != "patient-1" || event.ClinicID != "clinic-1" || event.ProviderID != "provider-1" {
		t.Fatalf("event fields not copied from command: %+v", event)
	}
	if agg.Status != AppointmentStatusHeld {
		t.Fatalf("expected status %q, got %q", AppointmentStatusHeld, agg.Status)
	}
	if agg.ScopedProviderID != "provider-1" || agg.ScopedPatientID != "patient-1" {
		t.Fatalf("aggregate not scoped to walk-in: %+v", agg)
	}
	if agg.Version != 1 {
		t.Fatalf("expected version 1, got %d", agg.Version)
	}
	if got := agg.Events(); len(got) != 1 {
		t.Fatalf("expected 1 buffered event, got %d", len(got))
	}
}

func TestAppointmentExecuteRegisterWalkInRejectsMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		cmd     RegisterWalkInCmd
		wantErr error
	}{
		{
			name:    "missing patient",
			cmd:     RegisterWalkInCmd{ClinicId: "clinic-1", ProviderId: "provider-1"},
			wantErr: ErrMissingPatient,
		},
		{
			name:    "missing clinic",
			cmd:     RegisterWalkInCmd{PatientId: "patient-1", ProviderId: "provider-1"},
			wantErr: ErrMissingClinic,
		},
		{
			name:    "missing provider",
			cmd:     RegisterWalkInCmd{PatientId: "patient-1", ClinicId: "clinic-1"},
			wantErr: ErrMissingProvider,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := &AppointmentAggregate{ID: "appointment-1"}
			events, err := agg.Execute(tt.cmd)
			assertRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestAppointmentExecuteRegisterWalkInRejectsInvariantViolations(t *testing.T) {
	for _, tt := range appointmentInvariantCases() {
		t.Run(tt.name, func(t *testing.T) {
			agg := &AppointmentAggregate{ID: "appointment-1"}
			tt.mutate(agg)
			events, err := agg.Execute(RegisterWalkInCmd{
				PatientId:  "patient-1",
				ClinicId:   "clinic-1",
				ProviderId: "provider-1",
			})
			assertRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestAppointmentExecuteBookAppointmentRejectsMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		cmd     BookAppointmentCmd
		wantErr error
	}{
		{
			name:    "missing hold token",
			cmd:     BookAppointmentCmd{PatientId: "patient-1", Reason: "annual exam"},
			wantErr: ErrMissingHoldToken,
		},
		{
			name:    "missing patient",
			cmd:     BookAppointmentCmd{HoldToken: "hold-1", Reason: "annual exam"},
			wantErr: ErrMissingPatient,
		},
		{
			name:    "missing reason",
			cmd:     BookAppointmentCmd{HoldToken: "hold-1", PatientId: "patient-1"},
			wantErr: ErrMissingReason,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := &AppointmentAggregate{ID: "appointment-1", Status: AppointmentStatusHeld}
			events, err := agg.Execute(tt.cmd)
			assertRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestAppointmentExecuteCancelAppointmentRejectsMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		cmd     CancelAppointmentCmd
		wantErr error
	}{
		{
			name:    "missing appointment",
			cmd:     CancelAppointmentCmd{Reason: "patient requested"},
			wantErr: ErrMissingAppointment,
		},
		{
			name:    "missing reason",
			cmd:     CancelAppointmentCmd{AppointmentId: "appointment-1"},
			wantErr: ErrMissingCancelReason,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := &AppointmentAggregate{ID: "appointment-1", Status: AppointmentStatusHeld}
			events, err := agg.Execute(tt.cmd)
			assertRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestAppointmentExecuteUnknownCommand(t *testing.T) {
	agg := &AppointmentAggregate{ID: "appointment-1"}

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

func TestAppointmentAggregateRootHelpers(t *testing.T) {
	agg := &AppointmentAggregate{ID: "appointment-1"}

	if _, err := agg.Execute(HoldSlotCmd{
		ProviderId: "provider-1",
		TimeSlot:   "slot",
		PatientId:  "patient-1",
	}); err != nil {
		t.Fatalf("Execute(HoldSlotCmd) returned error: %v", err)
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
