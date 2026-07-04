package model

import "errors"

var (
	// ErrMissingProvider is returned when HoldSlotCmd omits the provider id.
	ErrMissingProvider = errors.New("appointment: provider id is required")

	// ErrMissingTimeSlot is returned when HoldSlotCmd omits the time slot.
	ErrMissingTimeSlot = errors.New("appointment: time slot is required")

	// ErrMissingPatient is returned when HoldSlotCmd omits the patient id.
	ErrMissingPatient = errors.New("appointment: patient id is required")

	// ErrMissingAppointment is returned when RescheduleAppointmentCmd omits the
	// appointment id.
	ErrMissingAppointment = errors.New("appointment: appointment id is required")

	// ErrMissingNewTimeSlot is returned when RescheduleAppointmentCmd omits the
	// new time slot.
	ErrMissingNewTimeSlot = errors.New("appointment: new time slot is required")

	// ErrMissingReason is returned when CancelAppointmentCmd omits the
	// cancellation reason.
	ErrMissingReason = errors.New("appointment: cancellation reason is required")

	// ErrSlotOutsideAvailability is returned when the requested slot falls outside
	// the provider's published availability. Invariant: an appointment cannot be
	// booked outside the provider's published availability.
	ErrSlotOutsideAvailability = errors.New("appointment: an appointment cannot be booked outside the provider's published availability")

	// ErrSlotDoubleBooked is returned when the requested slot is already held or
	// booked by another appointment. Invariant: a slot may be booked by at most
	// one appointment at a time (no double-booking).
	ErrSlotDoubleBooked = errors.New("appointment: a slot may be booked by at most one appointment at a time")

	// ErrHoldLockExpired is returned when a prior hold lock on the slot expired
	// without being confirmed and the slot was not released. Invariant: a held
	// slot must be confirmed before its hold lock expires or it is released.
	ErrHoldLockExpired = errors.New("appointment: a held slot must be confirmed before its hold lock expires or it is released")

	// ErrOutsidePolicyWindow is returned when the hold would occur outside the
	// configured reschedule/cancel policy window. Invariant: reschedule and cancel
	// are only permitted within the configured policy window.
	ErrOutsidePolicyWindow = errors.New("appointment: reschedule and cancel are only permitted within the configured policy window")
)
