// Package model holds the Appointment aggregate for the scheduling bounded
// context.
package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// AppointmentStatus is the lifecycle state of an appointment. The zero value is
// an open (unbooked) appointment, which is what HoldSlotCmd acts on.
type AppointmentStatus string

const (
	// AppointmentStatusOpen is an appointment that has not yet reserved a slot. It
	// is the zero value, so a freshly constructed aggregate is open.
	AppointmentStatusOpen AppointmentStatus = ""
	// AppointmentStatusHeld is an appointment holding a hold lock over a slot that
	// must be confirmed before the lock expires or it is released.
	AppointmentStatusHeld AppointmentStatus = "held"
	// AppointmentStatusCancelled is an appointment that has been cancelled and has
	// released the slot it held.
	AppointmentStatusCancelled AppointmentStatus = "cancelled"
)

// AppointmentAggregate is the scheduling Appointment aggregate. It embeds
// shared.AggregateRoot for version tracking and an uncommitted-event buffer,
// and carries its own string identity.
//
// Beyond identity it tracks the state that command invariants read: its
// lifecycle status, the provider/patient and slot it is scoped to, and the flags
// describing whether the slot lies within the provider's published availability,
// whether the slot is already claimed by another appointment, whether a prior
// hold lock has expired without confirmation, and whether the action falls
// within the configured reschedule/cancel policy window.
//
// The invariant flags follow the repository convention that a freshly
// constructed aggregate is valid: their zero value is the compliant state, and a
// non-zero value marks a violation the guards reject.
type AppointmentAggregate struct {
	shared.AggregateRoot
	ID string

	// Status is the appointment's lifecycle state.
	Status AppointmentStatus

	// ScopedProviderID, ScopedPatientID and HeldTimeSlot are the participants and
	// slot the appointment is bound to. They are empty until a slot is held, at
	// which point the appointment is scoped to the holding provider, patient and
	// slot.
	ScopedProviderID string
	ScopedPatientID  string
	HeldTimeSlot     string

	// SlotOutsideAvailability reports that the requested slot falls outside the
	// provider's published availability. Invariant: an appointment cannot be
	// booked outside the provider's published availability.
	SlotOutsideAvailability bool

	// SlotAlreadyBooked reports that the slot is already held or booked by another
	// appointment. Invariant: a slot may be booked by at most one appointment at a
	// time (no double-booking).
	SlotAlreadyBooked bool

	// HoldLockExpired reports that a prior hold lock over the slot expired without
	// confirmation and the slot was not released. Invariant: a held slot must be
	// confirmed before its hold lock expires or it is released.
	HoldLockExpired bool

	// OutsidePolicyWindow reports that the action falls outside the configured
	// reschedule/cancel policy window. Invariant: reschedule and cancel are only
	// permitted within the configured policy window.
	OutsidePolicyWindow bool
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. Unrecognized command types return shared.ErrUnknownCommand.
func (a *AppointmentAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch c := cmd.(type) {
	case HoldSlotCmd:
		return a.holdSlot(c)
	case RescheduleAppointmentCmd:
		return a.rescheduleAppointment(c)
	case CancelAppointmentCmd:
		return a.cancelAppointment(c)
	default:
		return nil, shared.ErrUnknownCommand
	}
}

// holdSlot handles HoldSlotCmd: it validates the command input, enforces the
// scheduling invariants, then emits an AppointmentSlotHeldEvent and buffers it on
// the aggregate. Acquiring the hold is the distributed-lock reservation that
// stops any other appointment from claiming the slot while booking completes.
//
// The guards enforce, in order:
//
//   - Completeness: provider, time slot and patient must all be present.
//   - Provider availability: the slot must fall within the provider's published
//     availability.
//   - Double-booking: a slot may be held by at most one appointment at a time.
//   - Hold-lock expiry: a slot whose prior hold lock expired without confirmation
//     must have been released before it can be held again.
//   - Policy window: reschedule/cancel activity is only permitted within the
//     configured policy window.
func (a *AppointmentAggregate) holdSlot(cmd HoldSlotCmd) ([]shared.DomainEvent, error) {
	if cmd.ProviderId == "" {
		return nil, ErrMissingProvider
	}
	if cmd.TimeSlot == "" {
		return nil, ErrMissingTimeSlot
	}
	if cmd.PatientId == "" {
		return nil, ErrMissingPatient
	}

	// Invariant: an appointment cannot be booked outside the provider's published
	// availability.
	if a.SlotOutsideAvailability {
		return nil, ErrSlotOutsideAvailability
	}

	// Invariant: a slot may be booked by at most one appointment at a time — the
	// hold lock is what makes the reservation exclusive.
	if a.SlotAlreadyBooked {
		return nil, ErrSlotDoubleBooked
	}

	// Invariant: a held slot must be confirmed before its hold lock expires or it
	// is released; an expired-but-unreleased lock cannot be re-held.
	if a.HoldLockExpired {
		return nil, ErrHoldLockExpired
	}

	// Invariant: reschedule and cancel are only permitted within the configured
	// policy window.
	if a.OutsidePolicyWindow {
		return nil, ErrOutsidePolicyWindow
	}

	evt := AppointmentSlotHeldEvent{
		AppointmentID: a.ID,
		ProviderID:    cmd.ProviderId,
		TimeSlot:      cmd.TimeSlot,
		PatientID:     cmd.PatientId,
	}

	a.apply(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// rescheduleAppointment handles RescheduleAppointmentCmd: it validates the
// command input, enforces the scheduling invariants against the new slot, then
// emits an AppointmentRescheduledEvent and buffers it on the aggregate. Moving an
// appointment re-acquires the slot reservation at the new time, so the same
// guards that gate an initial hold gate the move.
//
// The guards enforce, in order:
//
//   - Completeness: the appointment id and the new time slot must both be present.
//   - Provider availability: the new slot must fall within the provider's
//     published availability.
//   - Double-booking: a slot may be occupied by at most one appointment at a time.
//   - Hold-lock expiry: a slot whose prior hold lock expired without confirmation
//     must have been released before the appointment can move onto it.
//   - Policy window: reschedule activity is only permitted within the configured
//     policy window.
func (a *AppointmentAggregate) rescheduleAppointment(cmd RescheduleAppointmentCmd) ([]shared.DomainEvent, error) {
	if cmd.AppointmentId == "" {
		return nil, ErrMissingAppointment
	}
	if cmd.NewTimeSlot == "" {
		return nil, ErrMissingNewTimeSlot
	}

	// Invariant: an appointment cannot be booked outside the provider's published
	// availability.
	if a.SlotOutsideAvailability {
		return nil, ErrSlotOutsideAvailability
	}

	// Invariant: a slot may be booked by at most one appointment at a time — the
	// destination slot must not already be claimed by another appointment.
	if a.SlotAlreadyBooked {
		return nil, ErrSlotDoubleBooked
	}

	// Invariant: a held slot must be confirmed before its hold lock expires or it
	// is released; an expired-but-unreleased lock blocks the move.
	if a.HoldLockExpired {
		return nil, ErrHoldLockExpired
	}

	// Invariant: reschedule and cancel are only permitted within the configured
	// policy window.
	if a.OutsidePolicyWindow {
		return nil, ErrOutsidePolicyWindow
	}

	evt := AppointmentRescheduledEvent{
		AppointmentID:    a.ID,
		PreviousTimeSlot: a.HeldTimeSlot,
		NewTimeSlot:      cmd.NewTimeSlot,
	}

	a.applyRescheduled(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// cancelAppointment handles CancelAppointmentCmd: it validates the command
// input, enforces the scheduling invariants, then emits an
// AppointmentCancelledEvent and buffers it on the aggregate. Cancelling releases
// the slot the appointment held so other appointments may claim it, and applies
// whatever policy penalties the cancellation incurs.
//
// The guards enforce, in order:
//
//   - Completeness: the appointment id and a cancellation reason must both be
//     present.
//   - Provider availability: the slot must fall within the provider's published
//     availability.
//   - Double-booking: a slot may be occupied by at most one appointment at a time.
//   - Hold-lock expiry: a slot whose prior hold lock expired without confirmation
//     must have been released.
//   - Policy window: cancel activity is only permitted within the configured
//     policy window.
func (a *AppointmentAggregate) cancelAppointment(cmd CancelAppointmentCmd) ([]shared.DomainEvent, error) {
	if cmd.AppointmentId == "" {
		return nil, ErrMissingAppointment
	}
	if cmd.Reason == "" {
		return nil, ErrMissingReason
	}

	// Invariant: an appointment cannot be booked outside the provider's published
	// availability.
	if a.SlotOutsideAvailability {
		return nil, ErrSlotOutsideAvailability
	}

	// Invariant: a slot may be booked by at most one appointment at a time (no
	// double-booking).
	if a.SlotAlreadyBooked {
		return nil, ErrSlotDoubleBooked
	}

	// Invariant: a held slot must be confirmed before its hold lock expires or it
	// is released.
	if a.HoldLockExpired {
		return nil, ErrHoldLockExpired
	}

	// Invariant: reschedule and cancel are only permitted within the configured
	// policy window.
	if a.OutsidePolicyWindow {
		return nil, ErrOutsidePolicyWindow
	}

	evt := AppointmentCancelledEvent{
		AppointmentID:    a.ID,
		ReleasedTimeSlot: a.HeldTimeSlot,
		Reason:           cmd.Reason,
	}

	a.applyCancelled(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// apply mutates aggregate state from a domain event. It is the single place
// state changes, so the same function serves both command handling and future
// event replay when rehydrating the aggregate from the store.
func (a *AppointmentAggregate) apply(evt AppointmentSlotHeldEvent) {
	a.Status = AppointmentStatusHeld
	a.ScopedProviderID = evt.ProviderID
	a.ScopedPatientID = evt.PatientID
	a.HeldTimeSlot = evt.TimeSlot
}

// applyRescheduled mutates aggregate state from an AppointmentRescheduledEvent.
// It re-points the appointment at the new slot while keeping it in the held
// state, mirroring apply as the single mutation point for the reschedule event
// during both command handling and replay.
func (a *AppointmentAggregate) applyRescheduled(evt AppointmentRescheduledEvent) {
	a.Status = AppointmentStatusHeld
	a.HeldTimeSlot = evt.NewTimeSlot
}

// applyCancelled mutates aggregate state from an AppointmentCancelledEvent. It
// moves the appointment into the cancelled state and clears the slot it held,
// releasing it for other appointments. Like apply it is the single mutation
// point for the cancelled event during both command handling and replay.
func (a *AppointmentAggregate) applyCancelled(evt AppointmentCancelledEvent) {
	a.Status = AppointmentStatusCancelled
	a.HeldTimeSlot = ""
}
