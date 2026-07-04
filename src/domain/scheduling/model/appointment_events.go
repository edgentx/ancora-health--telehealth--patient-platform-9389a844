package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// AppointmentSlotHeldEventType is the stable wire name emitted when a provider's
// time slot is reserved for a patient by acquiring a hold lock.
const AppointmentSlotHeldEventType = "appointment.slot.held"

// AppointmentSlotHeldEvent is emitted when a HoldSlotCmd succeeds. It records the
// provider whose slot was held, the slot itself, and the patient the hold was
// placed for. Its emission marks the appointment as holding a reserved slot that
// must be confirmed before its hold lock expires or it is released.
type AppointmentSlotHeldEvent struct {
	// AppointmentID is the identity of the AppointmentAggregate that produced the
	// event.
	AppointmentID string
	// ProviderID is the provider whose slot was held.
	ProviderID string
	// TimeSlot is the slot on the provider's schedule that was reserved.
	TimeSlot string
	// PatientID is the patient the slot was reserved for.
	PatientID string
}

// Type identifies the event kind.
func (e AppointmentSlotHeldEvent) Type() string { return AppointmentSlotHeldEventType }

// AggregateID ties the event back to the appointment that produced it.
func (e AppointmentSlotHeldEvent) AggregateID() string { return e.AppointmentID }

// Compile-time assertion that AppointmentSlotHeldEvent satisfies the DomainEvent
// contract.
var _ shared.DomainEvent = AppointmentSlotHeldEvent{}
