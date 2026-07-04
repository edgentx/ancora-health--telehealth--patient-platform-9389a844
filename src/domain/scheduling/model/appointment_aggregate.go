// Package model holds the Appointment aggregate for the scheduling bounded
// context.
package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// AppointmentStatus is the lifecycle state of an appointment. The zero value is
// a held slot that has not yet been confirmed, which is what BookAppointmentCmd
// acts on.
type AppointmentStatus string

const (
	// AppointmentStatusHeld is a slot that has been held but not yet confirmed
	// into a booking. It is the zero value, so a freshly constructed aggregate is
	// held.
	AppointmentStatusHeld AppointmentStatus = ""
	// AppointmentStatusBooked is a slot that has been confirmed into a booked
	// appointment.
	AppointmentStatusBooked AppointmentStatus = "booked"
)

// AppointmentAggregate is the scheduling Appointment aggregate. It embeds
// shared.AggregateRoot for version tracking and an uncommitted-event buffer,
// and carries its own string identity.
//
// Beyond identity it tracks the state that command invariants read: its
// lifecycle status, the patient and reason recorded when the slot is booked,
// and the flags describing whether confirming the booking would violate one of
// the scheduling invariants.
//
// The invariant flags follow the repository convention that a freshly
// constructed aggregate is valid: their zero value is the compliant state, and
// a non-zero value marks a violation the guards reject.
type AppointmentAggregate struct {
	shared.AggregateRoot
	ID string

	// Status is the appointment's lifecycle state.
	Status AppointmentStatus

	// PatientID is the patient the confirmed booking is for. It is empty until the
	// appointment is booked.
	PatientID string

	// Reason is the reason for the visit recorded on the confirmed booking. It is
	// empty until the appointment is booked.
	Reason string

	// SlotAlreadyBooked reports that the slot is already claimed by another
	// appointment. Invariant: a slot may be booked by at most one appointment at a
	// time (no double-booking).
	SlotAlreadyBooked bool

	// HoldExpired reports that the slot's hold lock has expired. Invariant: a held
	// slot must be confirmed before its hold lock expires or it is released.
	HoldExpired bool

	// OutsidePolicyWindow reports that the change falls outside the configured
	// policy window. Invariant: reschedule and cancel are only permitted within
	// the configured policy window.
	OutsidePolicyWindow bool

	// OutsideProviderAvailability reports that the requested time is not within the
	// provider's published availability. Invariant: an appointment cannot be booked
	// outside the provider's published availability.
	OutsideProviderAvailability bool
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. Unrecognized command types return shared.ErrUnknownCommand.
func (a *AppointmentAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch c := cmd.(type) {
	case BookAppointmentCmd:
		return a.bookAppointment(c)
	default:
		return nil, shared.ErrUnknownCommand
	}
}

// bookAppointment handles BookAppointmentCmd: it validates the command input,
// enforces the scheduling invariants, then emits an AppointmentBookedEvent and
// buffers it on the aggregate.
//
// The guards enforce, in order:
//
//   - Completeness: the hold token, patient, and reason must all be present.
//   - No double-booking: a slot may be booked by at most one appointment at a
//     time.
//   - Hold lock: a held slot must be confirmed before its hold lock expires or
//     it is released.
//   - Policy window: reschedule and cancel are only permitted within the
//     configured policy window.
//   - Published availability: an appointment cannot be booked outside the
//     provider's published availability.
func (a *AppointmentAggregate) bookAppointment(cmd BookAppointmentCmd) ([]shared.DomainEvent, error) {
	if cmd.HoldToken == "" {
		return nil, ErrMissingHoldToken
	}
	if cmd.PatientID == "" {
		return nil, ErrMissingPatientID
	}
	if cmd.Reason == "" {
		return nil, ErrMissingReason
	}

	// Invariant: a slot may be booked by at most one appointment at a time (no
	// double-booking).
	if a.SlotAlreadyBooked {
		return nil, ErrSlotAlreadyBooked
	}

	// Invariant: a held slot must be confirmed before its hold lock expires or it
	// is released.
	if a.HoldExpired {
		return nil, ErrHoldExpired
	}

	// Invariant: reschedule and cancel are only permitted within the configured
	// policy window.
	if a.OutsidePolicyWindow {
		return nil, ErrOutsidePolicyWindow
	}

	// Invariant: an appointment cannot be booked outside the provider's published
	// availability.
	if a.OutsideProviderAvailability {
		return nil, ErrOutsideProviderAvailability
	}

	evt := AppointmentBookedEvent{
		AppointmentID: a.ID,
		HoldToken:     cmd.HoldToken,
		PatientID:     cmd.PatientID,
		Reason:        cmd.Reason,
	}

	a.apply(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// apply mutates aggregate state from a domain event. It is the single place
// state changes, so the same function serves both command handling and future
// event replay when rehydrating the aggregate from the store.
func (a *AppointmentAggregate) apply(evt AppointmentBookedEvent) {
	a.Status = AppointmentStatusBooked
	a.PatientID = evt.PatientID
	a.Reason = evt.Reason
}
