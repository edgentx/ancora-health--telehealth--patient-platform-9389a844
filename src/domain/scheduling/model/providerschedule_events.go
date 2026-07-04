package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// ProviderAvailabilityPublishedEventType is the stable wire name emitted when a
// provider publishes bookable availability windows onto their schedule.
const ProviderAvailabilityPublishedEventType = "provider.availability.published"

// ProviderAvailabilityPublishedEvent is emitted when a PublishAvailabilityCmd
// succeeds. It records the provider whose schedule was published and the
// availability windows that were offered. Its emission marks those windows as
// bookable so that patients may hold slots within them.
type ProviderAvailabilityPublishedEvent struct {
	// ProviderScheduleID is the identity of the ProviderScheduleAggregate that
	// produced the event.
	ProviderScheduleID string
	// ProviderID is the provider whose availability was published.
	ProviderID string
	// Windows are the bookable availability windows that were offered on the
	// provider's schedule.
	Windows []string
}

// Type identifies the event kind.
func (e ProviderAvailabilityPublishedEvent) Type() string {
	return ProviderAvailabilityPublishedEventType
}

// AggregateID ties the event back to the provider schedule that produced it.
func (e ProviderAvailabilityPublishedEvent) AggregateID() string { return e.ProviderScheduleID }

// Compile-time assertion that ProviderAvailabilityPublishedEvent satisfies the
// DomainEvent contract.
var _ shared.DomainEvent = ProviderAvailabilityPublishedEvent{}

// ProviderTimeBlockedEventType is the stable wire name emitted when an interval
// on a provider's schedule is marked unavailable.
const ProviderTimeBlockedEventType = "provider.time.blocked"

// ProviderTimeBlockedEvent is emitted when a BlockTimeCmd succeeds. It records
// the provider whose schedule was blocked, the interval that was marked
// unavailable, and the reason it was blocked. Its emission removes that interval
// from the calendar so that no bookable slot may be offered within it.
type ProviderTimeBlockedEvent struct {
	// ProviderScheduleID is the identity of the ProviderScheduleAggregate that
	// produced the event.
	ProviderScheduleID string
	// ProviderID is the provider whose time was blocked.
	ProviderID string
	// Interval is the span on the provider's schedule that was marked unavailable.
	Interval string
	// Reason records why the interval was blocked.
	Reason string
}

// Type identifies the event kind.
func (e ProviderTimeBlockedEvent) Type() string {
	return ProviderTimeBlockedEventType
}

// AggregateID ties the event back to the provider schedule that produced it.
func (e ProviderTimeBlockedEvent) AggregateID() string { return e.ProviderScheduleID }

// Compile-time assertion that ProviderTimeBlockedEvent satisfies the DomainEvent
// contract.
var _ shared.DomainEvent = ProviderTimeBlockedEvent{}
