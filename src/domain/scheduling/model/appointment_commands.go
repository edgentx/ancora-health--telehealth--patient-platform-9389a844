package model

// HoldSlotCmd requests that a provider's time slot be reserved for a patient by
// acquiring a distributed hold lock over it.
//
// Holding is the reservation act that brings an appointment into being: it
// places a short-lived lock on the slot so no other appointment can claim it
// while the patient completes booking. The scheduling invariants gate the hold —
// the slot must fall within the provider's published availability, at most one
// appointment may hold a given slot at a time (no double-booking), a slot whose
// prior hold lock has expired without confirmation must have been released first,
// and reschedule/cancel activity is only permitted within the configured policy
// window. ProviderId, TimeSlot and PatientId identify the provider, the slot and
// the patient the hold is placed for; all three are mandatory.
type HoldSlotCmd struct {
	// ProviderId identifies the provider whose slot is being held.
	ProviderId string
	// TimeSlot identifies the specific slot on the provider's schedule being held.
	TimeSlot string
	// PatientId identifies the patient the slot is being reserved for.
	PatientId string
}

// RescheduleAppointmentCmd requests that an existing appointment be moved to a
// new time slot within policy.
//
// Rescheduling re-points a held appointment at a different slot: the same
// scheduling invariants that gate the original hold gate the move — the new slot
// must fall within the provider's published availability, at most one appointment
// may occupy a given slot at a time (no double-booking), a slot whose prior hold
// lock has expired without confirmation must have been released first, and
// reschedule activity is only permitted within the configured policy window.
// AppointmentId identifies the appointment being moved and NewTimeSlot the slot
// it is moving to; both are mandatory.
type RescheduleAppointmentCmd struct {
	// AppointmentId identifies the appointment being rescheduled.
	AppointmentId string
	// NewTimeSlot identifies the slot on the provider's schedule the appointment
	// is being moved to.
	NewTimeSlot string
}

// RegisterWalkInCmd requests that an unscheduled patient who presents at the
// front desk be registered as an appointment with a provider at a clinic.
//
// A walk-in is the front-desk registration of a patient who arrives without a
// prior booking: the patient is assigned to a provider at the clinic on the
// spot. Because registration still books the patient onto the provider, the same
// scheduling invariants that gate a hold gate the walk-in — the slot must fall
// within the provider's published availability, at most one appointment may
// occupy a given slot at a time (no double-booking), a slot whose prior hold lock
// has expired without confirmation must have been released first, and the action
// is only permitted within the configured policy window. PatientId, ClinicId and
// ProviderId identify the patient, the clinic they present at and the provider
// they are registered with; all three are mandatory.
type RegisterWalkInCmd struct {
	// PatientId identifies the walk-in patient being registered.
	PatientId string
	// ClinicId identifies the clinic the patient presents at.
	ClinicId string
	// ProviderId identifies the provider the walk-in is registered with.
	ProviderId string
}
