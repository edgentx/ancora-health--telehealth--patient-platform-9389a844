package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// ProviderRegisteredEventType is the stable wire name emitted when a provider is
// registered into the clinic directory.
const ProviderRegisteredEventType = "provider.registered"

// ProviderRegisteredEvent is emitted when a RegisterProviderCmd succeeds. It
// records the provider that was registered, the specialties they practice, and
// the clinics they are assigned to.
type ProviderRegisteredEvent struct {
	// DirectoryID is the identity of the ClinicDirectoryAggregate that produced
	// the event.
	DirectoryID string
	// ProviderID is the provider that was registered into the directory.
	ProviderID string
	// Specialties are the specialty codes the provider practices.
	Specialties []string
	// ClinicIDs are the clinics the provider is assigned to.
	ClinicIDs []string
}

// Type identifies the event kind.
func (e ProviderRegisteredEvent) Type() string { return ProviderRegisteredEventType }

// AggregateID ties the event back to the directory that produced it.
func (e ProviderRegisteredEvent) AggregateID() string { return e.DirectoryID }

// Compile-time assertion that ProviderRegisteredEvent satisfies the DomainEvent
// contract.
var _ shared.DomainEvent = ProviderRegisteredEvent{}

// SpecialtyUpdatedEventType is the stable wire name emitted when a specialty
// entry is created or updated in the clinic directory.
const SpecialtyUpdatedEventType = "specialty.updated"

// SpecialtyUpdatedEvent is emitted when a ManageSpecialtyCmd succeeds. It records
// the specialty code that was created or updated and the display name it now
// carries.
type SpecialtyUpdatedEvent struct {
	// DirectoryID is the identity of the ClinicDirectoryAggregate that produced
	// the event.
	DirectoryID string
	// SpecialtyCode is the specialty entry that was created or updated.
	SpecialtyCode string
	// DisplayName is the display name the specialty now carries.
	DisplayName string
}

// Type identifies the event kind.
func (e SpecialtyUpdatedEvent) Type() string { return SpecialtyUpdatedEventType }

// AggregateID ties the event back to the directory that produced it.
func (e SpecialtyUpdatedEvent) AggregateID() string { return e.DirectoryID }

// Compile-time assertion that SpecialtyUpdatedEvent satisfies the DomainEvent
// contract.
var _ shared.DomainEvent = SpecialtyUpdatedEvent{}
