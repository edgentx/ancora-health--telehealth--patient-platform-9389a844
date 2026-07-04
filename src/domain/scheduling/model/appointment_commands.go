package model

// BookAppointmentCmd requests that a held slot be confirmed into a booked
// appointment. It carries the hold token identifying the slot lock being
// confirmed, the patient the appointment is for, and the reason for the visit.
//
// Booking is the act that turns a held slot into a confirmed appointment. The
// command must satisfy the scheduling invariants before it can succeed: a slot
// may be booked by at most one appointment at a time (no double-booking), a
// held slot must be confirmed before its hold lock expires or it is released,
// reschedule and cancel are only permitted within the configured policy window,
// and an appointment cannot be booked outside the provider's published
// availability. HoldToken, PatientID, and Reason are all mandatory.
type BookAppointmentCmd struct {
	// HoldToken identifies the hold lock on the slot being confirmed into a
	// booked appointment.
	HoldToken string
	// PatientID identifies the patient the appointment is booked for.
	PatientID string
	// Reason is the reason for the visit captured at booking time.
	Reason string
}
