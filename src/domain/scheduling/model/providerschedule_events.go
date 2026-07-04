package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// ProviderTimeBlockedEventType is the stable wire name emitted when an interval
// on a provider's schedule is marked unavailable.
const ProviderTimeBlockedEventType = "provider.time.blocked"

// ProviderTimeBlockedEvent is emitted when a BlockTimeCmd succeeds. It records
// the provider whose schedule was blocked, the interval marked unavailable, and
// the reason the block was placed. Its emission removes the interval from the
// provider's bookable availability so no slot may be offered over it.
type ProviderTimeBlockedEvent struct {
	// ProviderScheduleID is the identity of the ProviderScheduleAggregate that
	// produced the event.
	ProviderScheduleID string
	// ProviderID is the provider whose schedule the interval was blocked on.
	ProviderID string
	// Interval is the span of time on the schedule that was marked unavailable.
	Interval string
	// Reason records why the interval was blocked.
	Reason string
}

// Type identifies the event kind.
func (e ProviderTimeBlockedEvent) Type() string { return ProviderTimeBlockedEventType }

// AggregateID ties the event back to the provider schedule that produced it.
func (e ProviderTimeBlockedEvent) AggregateID() string { return e.ProviderScheduleID }

// Compile-time assertion that ProviderTimeBlockedEvent satisfies the
// DomainEvent contract.
var _ shared.DomainEvent = ProviderTimeBlockedEvent{}
