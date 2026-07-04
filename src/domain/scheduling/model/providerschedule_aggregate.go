package model

import (
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"
)

// ProviderScheduleAggregate is the scheduling ProviderSchedule aggregate. It
// embeds shared.AggregateRoot for version tracking and an uncommitted-event
// buffer, and carries its own string identity in ID.
//
// Beyond identity it tracks the state that command invariants read: whether a
// requested block would overlap the provider's existing availability windows,
// whether the interval being blocked is still offered as a bookable slot, and
// whether the provider's published availability falls within the clinic's
// operating hours.
//
// The invariant flags follow the repository convention that a freshly
// constructed aggregate is valid: their zero value is the compliant state, and
// a non-zero value marks a violation the guards reject.
type ProviderScheduleAggregate struct {
	shared.AggregateRoot
	ID string

	// AvailabilityWindowsOverlap reports that blocking the interval would leave
	// overlapping availability windows for the provider. Invariant: availability
	// windows for the same provider must not overlap.
	AvailabilityWindowsOverlap bool

	// BlockedIntervalBookable reports that the interval being blocked is still
	// offered as a bookable slot. Invariant: a blocked interval cannot be offered
	// as a bookable slot.
	BlockedIntervalBookable bool

	// AvailabilityOutsideOperatingHours reports that the provider's published
	// availability falls outside the clinic's operating hours. Invariant:
	// published availability must fall within the provider's clinic operating
	// hours.
	AvailabilityOutsideOperatingHours bool
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. Unrecognized command types return shared.ErrUnknownCommand.
func (a *ProviderScheduleAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch c := cmd.(type) {
	case BlockTimeCmd:
		return a.blockTime(c)
	default:
		return nil, shared.ErrUnknownCommand
	}
}

// blockTime handles BlockTimeCmd: it validates the command input, enforces the
// scheduling invariants, then emits a ProviderTimeBlockedEvent and buffers it on
// the aggregate. Blocking carves an off-limits window out of the provider's
// schedule so no bookable slot may be offered over it.
//
// The guards enforce, in order:
//
//   - Completeness: provider, interval and reason must all be present.
//   - Non-overlap: blocking must not leave overlapping availability windows for
//     the provider.
//   - Blocked-not-bookable: the interval being blocked must not still be offered
//     as a bookable slot.
//   - Operating hours: the provider's published availability must fall within the
//     clinic's operating hours.
func (a *ProviderScheduleAggregate) blockTime(cmd BlockTimeCmd) ([]shared.DomainEvent, error) {
	if cmd.ProviderId == "" {
		return nil, ErrMissingProviderScheduleProvider
	}
	if cmd.Interval == "" {
		return nil, ErrMissingBlockInterval
	}
	if cmd.Reason == "" {
		return nil, ErrMissingBlockReason
	}

	// Invariant: availability windows for the same provider must not overlap.
	if a.AvailabilityWindowsOverlap {
		return nil, ErrAvailabilityWindowsOverlap
	}

	// Invariant: a blocked interval cannot be offered as a bookable slot.
	if a.BlockedIntervalBookable {
		return nil, ErrBlockedIntervalBookable
	}

	// Invariant: published availability must fall within the provider's clinic
	// operating hours.
	if a.AvailabilityOutsideOperatingHours {
		return nil, ErrAvailabilityOutsideOperatingHours
	}

	evt := ProviderTimeBlockedEvent{
		ProviderScheduleID: a.ID,
		ProviderID:         cmd.ProviderId,
		Interval:           cmd.Interval,
		Reason:             cmd.Reason,
	}

	a.applyTimeBlocked(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// applyTimeBlocked mutates aggregate state from a ProviderTimeBlockedEvent. It
// is the single place state changes for the block, so the same function serves
// both command handling and future event replay when rehydrating the aggregate
// from the store. Once an interval is blocked it is no longer offered as a
// bookable slot.
func (a *ProviderScheduleAggregate) applyTimeBlocked(evt ProviderTimeBlockedEvent) {
	a.BlockedIntervalBookable = false
}
