package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// AppointmentBookedEventType is the stable wire name emitted when a held slot
// is confirmed into a booked appointment.
const AppointmentBookedEventType = "appointment.booked"

// AppointmentBookedEvent is emitted when a BookAppointmentCmd succeeds. It
// records the hold token that was confirmed, the patient the appointment is
// for, and the reason for the visit.
type AppointmentBookedEvent struct {
	// AppointmentID is the identity of the AppointmentAggregate that produced the
	// event.
	AppointmentID string
	// HoldToken is the hold lock that was confirmed into the booking.
	HoldToken string
	// PatientID identifies the patient the appointment is booked for.
	PatientID string
	// Reason is the reason for the visit captured at booking time.
	Reason string
}

// Type identifies the event kind.
func (e AppointmentBookedEvent) Type() string { return AppointmentBookedEventType }

// AggregateID ties the event back to the appointment that produced it.
func (e AppointmentBookedEvent) AggregateID() string { return e.AppointmentID }

// Compile-time assertion that AppointmentBookedEvent satisfies the DomainEvent
// contract.
var _ shared.DomainEvent = AppointmentBookedEvent{}
