package model

import (
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"
)

// ProviderScheduleAggregate is the scheduling ProviderSchedule aggregate. It
// embeds shared.AggregateRoot for version tracking and an uncommitted-event
// buffer, and carries its own string identity in ID.
//
// Beyond identity it tracks the state that command invariants read: the provider
// it is scoped to, the windows it has published, and the flags describing whether
// the windows being published overlap one another, whether any of them offers an
// interval the provider has blocked, and whether any of them falls outside the
// provider's clinic operating hours.
//
// The invariant flags follow the repository convention that a freshly
// constructed aggregate is valid: their zero value is the compliant state, and a
// non-zero value marks a violation the guards reject.
type ProviderScheduleAggregate struct {
	shared.AggregateRoot
	ID string

	// ScopedProviderID and PublishedWindows are the provider and windows the
	// schedule is bound to. They are empty until availability is published, at
	// which point the schedule is scoped to the publishing provider and its
	// offered windows.
	ScopedProviderID string
	PublishedWindows []string

	// BlockedIntervals are the intervals that have been marked unavailable on the
	// schedule. They are empty until time is blocked, at which point each blocked
	// interval is recorded so it can never be offered as a bookable slot.
	BlockedIntervals []string

	// WindowsOverlap reports that the windows being published overlap one another.
	// Invariant: availability windows for the same provider must not overlap.
	WindowsOverlap bool

	// WindowOffersBlockedInterval reports that a published window offers an
	// interval the provider has blocked. Invariant: a blocked interval cannot be
	// offered as a bookable slot.
	WindowOffersBlockedInterval bool

	// WindowOutsideOperatingHours reports that a published window falls outside the
	// provider's clinic operating hours. Invariant: published availability must
	// fall within the provider's clinic operating hours.
	WindowOutsideOperatingHours bool
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. Unrecognized command types return shared.ErrUnknownCommand.
func (a *ProviderScheduleAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch c := cmd.(type) {
	case PublishAvailabilityCmd:
		return a.publishAvailability(c)
	case BlockTimeCmd:
		return a.blockTime(c)
	default:
		return nil, shared.ErrUnknownCommand
	}
}

// publishAvailability handles PublishAvailabilityCmd: it validates the command
// input, enforces the scheduling invariants, then emits a
// ProviderAvailabilityPublishedEvent and buffers it on the aggregate. Publishing
// availability opens the provider's calendar for booking by declaring the
// windows patients may hold slots within.
//
// The guards enforce, in order:
//
//   - Completeness: the provider id and at least one availability window must be
//     present.
//   - Non-overlap: availability windows for the same provider must not overlap.
//   - Blocked intervals: a blocked interval cannot be offered as a bookable slot.
//   - Operating hours: published availability must fall within the provider's
//     clinic operating hours.
func (a *ProviderScheduleAggregate) publishAvailability(cmd PublishAvailabilityCmd) ([]shared.DomainEvent, error) {
	if cmd.ProviderId == "" {
		return nil, ErrMissingScheduleProvider
	}
	if len(cmd.Windows) == 0 {
		return nil, ErrMissingWindows
	}

	// Invariant: availability windows for the same provider must not overlap.
	if a.WindowsOverlap {
		return nil, ErrOverlappingWindows
	}

	// Invariant: a blocked interval cannot be offered as a bookable slot.
	if a.WindowOffersBlockedInterval {
		return nil, ErrBlockedIntervalOffered
	}

	// Invariant: published availability must fall within the provider's clinic
	// operating hours.
	if a.WindowOutsideOperatingHours {
		return nil, ErrOutsideOperatingHours
	}

	evt := ProviderAvailabilityPublishedEvent{
		ProviderScheduleID: a.ID,
		ProviderID:         cmd.ProviderId,
		Windows:            cmd.Windows,
	}

	a.apply(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// blockTime handles BlockTimeCmd: it validates the command input, enforces the
// scheduling invariants, then emits a ProviderTimeBlockedEvent and buffers it on
// the aggregate. Blocking time carves an interval out of the provider's calendar
// so that no bookable slot may be offered within it.
//
// The guards enforce, in order:
//
//   - Completeness: the provider id, the interval and the reason must all be
//     present.
//   - Non-overlap: availability windows for the same provider must not overlap.
//   - Blocked intervals: a blocked interval cannot be offered as a bookable slot.
//   - Operating hours: published availability must fall within the provider's
//     clinic operating hours.
func (a *ProviderScheduleAggregate) blockTime(cmd BlockTimeCmd) ([]shared.DomainEvent, error) {
	if cmd.ProviderId == "" {
		return nil, ErrMissingScheduleProvider
	}
	if cmd.Interval == "" {
		return nil, ErrMissingInterval
	}
	if cmd.Reason == "" {
		return nil, ErrMissingBlockReason
	}

	// Invariant: availability windows for the same provider must not overlap.
	if a.WindowsOverlap {
		return nil, ErrOverlappingWindows
	}

	// Invariant: a blocked interval cannot be offered as a bookable slot.
	if a.WindowOffersBlockedInterval {
		return nil, ErrBlockedIntervalOffered
	}

	// Invariant: published availability must fall within the provider's clinic
	// operating hours.
	if a.WindowOutsideOperatingHours {
		return nil, ErrOutsideOperatingHours
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

// apply mutates aggregate state from a domain event. It is the single place
// state changes, so the same function serves both command handling and future
// event replay when rehydrating the aggregate from the store.
func (a *ProviderScheduleAggregate) apply(evt ProviderAvailabilityPublishedEvent) {
	a.ScopedProviderID = evt.ProviderID
	a.PublishedWindows = evt.Windows
}

// applyTimeBlocked mutates aggregate state from a ProviderTimeBlockedEvent. It
// scopes the schedule to the blocking provider and records the blocked interval,
// mirroring apply as the single mutation point for the time-blocked event during
// both command handling and replay.
func (a *ProviderScheduleAggregate) applyTimeBlocked(evt ProviderTimeBlockedEvent) {
	a.ScopedProviderID = evt.ProviderID
	a.BlockedIntervals = append(a.BlockedIntervals, evt.Interval)
}
