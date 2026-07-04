package model

import (
	"errors"
	"testing"
)

func TestAppointmentExecuteBookAppointmentEmitsBookedEvent(t *testing.T) {
	agg := &AppointmentAggregate{
		ID:     "appointment-123",
		Status: AppointmentStatusHeld,
	}

	events, err := agg.Execute(BookAppointmentCmd{
		HoldToken: "hold-token-123",
		PatientId: "patient-123",
		Reason:    "annual exam",
	})
	if err != nil {
		t.Fatalf("Execute(BookAppointmentCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	event, ok := events[0].(AppointmentBookedEvent)
	if !ok {
		t.Fatalf("expected AppointmentBookedEvent, got %T", events[0])
	}
	if event.Type() != AppointmentBookedEventType {
		t.Fatalf("expected event type %q, got %q", AppointmentBookedEventType, event.Type())
	}
	if event.Type() != "appointment.booked" {
		t.Fatalf("expected wire event type appointment.booked, got %q", event.Type())
	}
	if event.AggregateID() != "appointment-123" {
		t.Fatalf("expected aggregate id appointment-123, got %q", event.AggregateID())
	}
	if event.HoldToken != "hold-token-123" {
		t.Fatalf("expected hold token to be copied onto event, got %q", event.HoldToken)
	}
	if event.PatientID != "patient-123" {
		t.Fatalf("expected patient id to be copied onto event, got %q", event.PatientID)
	}
	if event.Reason != "annual exam" {
		t.Fatalf("expected reason to be copied onto event, got %q", event.Reason)
	}
	if agg.Status != AppointmentStatusBooked {
		t.Fatalf("expected aggregate status %q, got %q", AppointmentStatusBooked, agg.Status)
	}
	if agg.ScopedPatientID != "patient-123" {
		t.Fatalf("expected aggregate patient patient-123, got %q", agg.ScopedPatientID)
	}
	if agg.Version != 1 {
		t.Fatalf("expected aggregate version 1, got %d", agg.Version)
	}
	if got := agg.Events(); len(got) != 1 {
		t.Fatalf("expected aggregate to buffer 1 event, got %d", len(got))
	}
}

func TestAppointmentExecuteBookAppointmentRejectsInvariantViolations(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*AppointmentAggregate)
		wantErr error
	}{
		{
			name: "slot double booked",
			mutate: func(agg *AppointmentAggregate) {
				agg.SlotAlreadyBooked = true
			},
			wantErr: ErrSlotDoubleBooked,
		},
		{
			name: "hold lock expired",
			mutate: func(agg *AppointmentAggregate) {
				agg.HoldLockExpired = true
			},
			wantErr: ErrHoldLockExpired,
		},
		{
			name: "outside policy window",
			mutate: func(agg *AppointmentAggregate) {
				agg.OutsidePolicyWindow = true
			},
			wantErr: ErrOutsidePolicyWindow,
		},
		{
			name: "outside provider availability",
			mutate: func(agg *AppointmentAggregate) {
				agg.SlotOutsideAvailability = true
			},
			wantErr: ErrSlotOutsideAvailability,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := &AppointmentAggregate{
				ID:     "appointment-123",
				Status: AppointmentStatusHeld,
			}
			tt.mutate(agg)

			events, err := agg.Execute(BookAppointmentCmd{
				HoldToken: "hold-token-123",
				PatientId: "patient-123",
				Reason:    "annual exam",
			})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
			if len(events) != 0 {
				t.Fatalf("expected no events on rejection, got %d", len(events))
			}
			if got := agg.Events(); len(got) != 0 {
				t.Fatalf("expected no buffered events on rejection, got %d", len(got))
			}
		})
	}
}

func TestAppointmentExecuteCancelAppointmentEmitsCancelledEvent(t *testing.T) {
	agg := &AppointmentAggregate{
		ID:           "appointment-123",
		Status:       AppointmentStatusHeld,
		HeldTimeSlot: "2026-07-06T09:00:00Z",
	}

	events, err := agg.Execute(CancelAppointmentCmd{
		AppointmentId: "appointment-123",
		Reason:        "patient requested",
	})
	if err != nil {
		t.Fatalf("Execute(CancelAppointmentCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	event, ok := events[0].(AppointmentCancelledEvent)
	if !ok {
		t.Fatalf("expected AppointmentCancelledEvent, got %T", events[0])
	}
	if event.Type() != AppointmentCancelledEventType {
		t.Fatalf("expected event type %q, got %q", AppointmentCancelledEventType, event.Type())
	}
	if event.Type() != "appointment.cancelled" {
		t.Fatalf("expected wire event type appointment.cancelled, got %q", event.Type())
	}
	if event.AggregateID() != "appointment-123" {
		t.Fatalf("expected aggregate id appointment-123, got %q", event.AggregateID())
	}
	if event.Reason != "patient requested" {
		t.Fatalf("expected reason to be copied onto event, got %q", event.Reason)
	}
	if agg.Status != AppointmentStatusCancelled {
		t.Fatalf("expected aggregate status %q, got %q", AppointmentStatusCancelled, agg.Status)
	}
	if agg.HeldTimeSlot != "" {
		t.Fatalf("expected held slot to be released on cancel, got %q", agg.HeldTimeSlot)
	}
	if agg.Version != 1 {
		t.Fatalf("expected aggregate version 1, got %d", agg.Version)
	}
	if got := agg.Events(); len(got) != 1 {
		t.Fatalf("expected aggregate to buffer 1 event, got %d", len(got))
	}
}

func TestAppointmentExecuteCancelAppointmentRejectsInvariantViolations(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*AppointmentAggregate)
		wantErr error
	}{
		{
			name: "slot double booked",
			mutate: func(agg *AppointmentAggregate) {
				agg.SlotAlreadyBooked = true
			},
			wantErr: ErrSlotDoubleBooked,
		},
		{
			name: "hold lock expired",
			mutate: func(agg *AppointmentAggregate) {
				agg.HoldLockExpired = true
			},
			wantErr: ErrHoldLockExpired,
		},
		{
			name: "outside policy window",
			mutate: func(agg *AppointmentAggregate) {
				agg.OutsidePolicyWindow = true
			},
			wantErr: ErrOutsidePolicyWindow,
		},
		{
			name: "outside provider availability",
			mutate: func(agg *AppointmentAggregate) {
				agg.SlotOutsideAvailability = true
			},
			wantErr: ErrSlotOutsideAvailability,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := &AppointmentAggregate{
				ID:     "appointment-123",
				Status: AppointmentStatusHeld,
			}
			tt.mutate(agg)

			events, err := agg.Execute(CancelAppointmentCmd{
				AppointmentId: "appointment-123",
				Reason:        "patient requested",
			})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
			if len(events) != 0 {
				t.Fatalf("expected no events on rejection, got %d", len(events))
			}
			if got := agg.Events(); len(got) != 0 {
				t.Fatalf("expected no buffered events on rejection, got %d", len(got))
			}
		})
	}
}
