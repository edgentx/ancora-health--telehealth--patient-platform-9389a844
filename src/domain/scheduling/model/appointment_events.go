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

// AppointmentRescheduledEventType is the stable wire name emitted when an
// appointment is moved to a new time slot within policy.
const AppointmentRescheduledEventType = "appointment.rescheduled"

// AppointmentRescheduledEvent is emitted when a RescheduleAppointmentCmd
// succeeds. It records the appointment that was moved, the slot it previously
// held, and the new slot it now holds. Its emission re-points the appointment at
// the new slot while leaving it in the held state that must still be confirmed
// before its hold lock expires or it is released.
type AppointmentRescheduledEvent struct {
	// AppointmentID is the identity of the AppointmentAggregate that produced the
	// event.
	AppointmentID string
	// PreviousTimeSlot is the slot the appointment held before the reschedule.
	PreviousTimeSlot string
	// NewTimeSlot is the slot the appointment was moved to.
	NewTimeSlot string
}

// Type identifies the event kind.
func (e AppointmentRescheduledEvent) Type() string { return AppointmentRescheduledEventType }

// AggregateID ties the event back to the appointment that produced it.
func (e AppointmentRescheduledEvent) AggregateID() string { return e.AppointmentID }

// Compile-time assertion that AppointmentRescheduledEvent satisfies the
// DomainEvent contract.
var _ shared.DomainEvent = AppointmentRescheduledEvent{}

// AppointmentWalkInRegisteredEventType is the stable wire name emitted when an
// unscheduled walk-in patient is registered with a provider at a clinic.
const AppointmentWalkInRegisteredEventType = "appointment.walkin.registered"

// AppointmentWalkInRegisteredEvent is emitted when a RegisterWalkInCmd succeeds.
// It records the patient that was registered, the clinic they presented at, and
// the provider they were registered with. Its emission brings the walk-in into
// the held state that must still be confirmed before its hold lock expires or it
// is released.
type AppointmentWalkInRegisteredEvent struct {
	// AppointmentID is the identity of the AppointmentAggregate that produced the
	// event.
	AppointmentID string
	// PatientID is the walk-in patient that was registered.
	PatientID string
	// ClinicID is the clinic the patient presented at.
	ClinicID string
	// ProviderID is the provider the walk-in was registered with.
	ProviderID string
}

// Type identifies the event kind.
func (e AppointmentWalkInRegisteredEvent) Type() string {
	return AppointmentWalkInRegisteredEventType
}

// AggregateID ties the event back to the appointment that produced it.
func (e AppointmentWalkInRegisteredEvent) AggregateID() string { return e.AppointmentID }

// Compile-time assertion that AppointmentWalkInRegisteredEvent satisfies the
// DomainEvent contract.
var _ shared.DomainEvent = AppointmentWalkInRegisteredEvent{}
