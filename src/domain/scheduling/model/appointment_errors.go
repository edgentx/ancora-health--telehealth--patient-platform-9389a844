package model

import "errors"

var (
	// ErrMissingHoldToken is returned when BookAppointmentCmd omits the hold token
	// identifying the slot lock being confirmed.
	ErrMissingHoldToken = errors.New("appointment: hold token is required")

	// ErrMissingPatientID is returned when BookAppointmentCmd omits the patient the
	// appointment is booked for.
	ErrMissingPatientID = errors.New("appointment: patient id is required")

	// ErrMissingReason is returned when BookAppointmentCmd omits the reason for the
	// visit.
	ErrMissingReason = errors.New("appointment: reason is required")

	// ErrSlotAlreadyBooked is returned when the slot is already claimed by another
	// appointment. Invariant: a slot may be booked by at most one appointment at a
	// time (no double-booking).
	ErrSlotAlreadyBooked = errors.New("appointment: a slot may be booked by at most one appointment at a time (no double-booking)")

	// ErrHoldExpired is returned when the slot's hold lock has expired before the
	// booking was confirmed. Invariant: a held slot must be confirmed before its
	// hold lock expires or it is released.
	ErrHoldExpired = errors.New("appointment: a held slot must be confirmed before its hold lock expires or it is released")

	// ErrOutsidePolicyWindow is returned when the change falls outside the
	// configured policy window. Invariant: reschedule and cancel are only permitted
	// within the configured policy window.
	ErrOutsidePolicyWindow = errors.New("appointment: reschedule and cancel are only permitted within the configured policy window")

	// ErrOutsideProviderAvailability is returned when the requested time is not
	// within the provider's published availability. Invariant: an appointment
	// cannot be booked outside the provider's published availability.
	ErrOutsideProviderAvailability = errors.New("appointment: an appointment cannot be booked outside the provider's published availability")
)
