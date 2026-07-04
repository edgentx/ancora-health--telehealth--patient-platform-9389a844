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
