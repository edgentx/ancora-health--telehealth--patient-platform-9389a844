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
