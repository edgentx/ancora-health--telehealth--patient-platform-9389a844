// Package model holds the aggregates for the administration-and-analytics
// bounded context. ClinicDirectoryAggregate maintains the directory of clinics
// and providers; RegisterProviderCmd admits a provider into the directory.
package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// ClinicDirectoryAggregate is the administration-and-analytics aggregate that
// maintains the directory of clinics and the providers registered against them.
// It embeds shared.AggregateRoot for version tracking and an uncommitted-event
// buffer, and carries its own identity in ID.
//
// Beyond identity it tracks the state that command invariants read: the
// providers already registered and, as flags, whether the pending registration
// would leave a provider unbookable, reuse a specialty code, or block a clinic
// deactivation.
//
// The invariant flags follow the repository convention that a freshly
// constructed aggregate is valid: their zero value is the compliant state, and a
// non-zero value marks a violation the guards reject.
type ClinicDirectoryAggregate struct {
	shared.AggregateRoot
	ID string

	// ProviderIDs are the providers already registered in the directory.
	ProviderIDs []string

	// SpecialtyCodes are the specialty codes already present in the directory.
	// ManageSpecialtyCmd appends a newly created code here; updating an existing
	// entry leaves the set unchanged.
	SpecialtyCodes []string

	// ClinicIDs are the clinic identities already configured in the directory.
	// ConfigureClinicCmd appends a newly created identity here; updating an
	// existing entry leaves the set unchanged.
	ClinicIDs []string

	// ProviderNotBookable reports that the provider being registered is not
	// assigned to any clinic. Invariant: a provider must be assigned to at least
	// one clinic to be bookable.
	ProviderNotBookable bool

	// DuplicateSpecialtyCode reports that the provider being registered reuses a
	// specialty code that already exists in the directory. Invariant: a specialty
	// code must be unique within the directory.
	DuplicateSpecialtyCode bool

	// ClinicDeactivationBlocked reports that registering the provider would
	// deactivate a clinic that still has future booked appointments. Invariant: a
	// clinic cannot be deactivated while it has future booked appointments.
	ClinicDeactivationBlocked bool
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. Unrecognized command types return shared.ErrUnknownCommand.
func (a *ClinicDirectoryAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch c := cmd.(type) {
	case RegisterProviderCmd:
		return a.registerProvider(c)
	case ManageSpecialtyCmd:
		return a.manageSpecialty(c)
	case ConfigureClinicCmd:
		return a.configureClinic(c)
	default:
		return nil, shared.ErrUnknownCommand
	}
}

// registerProvider handles RegisterProviderCmd: it validates the command input,
// enforces the directory invariants, then emits a ProviderRegisteredEvent and
// buffers it on the aggregate.
//
// The guards enforce, in order:
//
//   - Completeness: the provider id, at least one specialty, and at least one
//     clinic must all be present.
//   - Bookable: a provider must be assigned to at least one clinic to be
//     bookable.
//   - Unique specialty: a specialty code must be unique within the directory.
//   - Clinic deactivation: a clinic cannot be deactivated while it has future
//     booked appointments.
func (a *ClinicDirectoryAggregate) registerProvider(cmd RegisterProviderCmd) ([]shared.DomainEvent, error) {
	if cmd.ProviderId == "" {
		return nil, ErrMissingProvider
	}
	if len(cmd.Specialties) == 0 {
		return nil, ErrMissingSpecialties
	}
	if len(cmd.ClinicIds) == 0 {
		return nil, ErrMissingClinics
	}

	// Invariant: a provider must be assigned to at least one clinic to be
	// bookable.
	if a.ProviderNotBookable {
		return nil, ErrProviderNotBookable
	}

	// Invariant: a specialty code must be unique within the directory.
	if a.DuplicateSpecialtyCode {
		return nil, ErrDuplicateSpecialtyCode
	}

	// Invariant: a clinic cannot be deactivated while it has future booked
	// appointments.
	if a.ClinicDeactivationBlocked {
		return nil, ErrClinicDeactivationBlocked
	}

	evt := ProviderRegisteredEvent{
		DirectoryID: a.ID,
		ProviderID:  cmd.ProviderId,
		Specialties: cmd.Specialties,
		ClinicIDs:   cmd.ClinicIds,
	}

	a.apply(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// manageSpecialty handles ManageSpecialtyCmd: it validates the command input,
// enforces the directory invariants, then emits a SpecialtyUpdatedEvent and
// buffers it on the aggregate.
//
// The command is an upsert — a code not yet in the directory is created and one
// already present has its display name updated — so a single specialty.updated
// event covers both paths.
//
// The guards enforce, in order:
//
//   - Completeness: the specialty code and display name must both be present.
//   - Bookable: a provider must be assigned to at least one clinic to be
//     bookable.
//   - Unique specialty: a specialty code must be unique within the directory.
//   - Clinic deactivation: a clinic cannot be deactivated while it has future
//     booked appointments.
func (a *ClinicDirectoryAggregate) manageSpecialty(cmd ManageSpecialtyCmd) ([]shared.DomainEvent, error) {
	if cmd.SpecialtyCode == "" {
		return nil, ErrMissingSpecialtyCode
	}
	if cmd.DisplayName == "" {
		return nil, ErrMissingDisplayName
	}

	// Invariant: a provider must be assigned to at least one clinic to be
	// bookable.
	if a.ProviderNotBookable {
		return nil, ErrProviderNotBookable
	}

	// Invariant: a specialty code must be unique within the directory.
	if a.DuplicateSpecialtyCode {
		return nil, ErrDuplicateSpecialtyCode
	}

	// Invariant: a clinic cannot be deactivated while it has future booked
	// appointments.
	if a.ClinicDeactivationBlocked {
		return nil, ErrClinicDeactivationBlocked
	}

	evt := SpecialtyUpdatedEvent{
		DirectoryID:   a.ID,
		SpecialtyCode: cmd.SpecialtyCode,
		DisplayName:   cmd.DisplayName,
	}

	a.applySpecialtyUpdated(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// configureClinic handles ConfigureClinicCmd: it validates the command input,
// enforces the directory invariants, then emits a ClinicConfiguredEvent and
// buffers it on the aggregate.
//
// The command is an upsert — a clinic identity not yet in the directory is
// created and one already present has its operating hours updated — so a single
// clinic.configured event covers both paths.
//
// The guards enforce, in order:
//
//   - Completeness: the clinic identity and operating hours must both be present.
//   - Bookable: a provider must be assigned to at least one clinic to be
//     bookable.
//   - Unique specialty: a specialty code must be unique within the directory.
//   - Clinic deactivation: a clinic cannot be deactivated while it has future
//     booked appointments.
func (a *ClinicDirectoryAggregate) configureClinic(cmd ConfigureClinicCmd) ([]shared.DomainEvent, error) {
	if cmd.ClinicIdentity == "" {
		return nil, ErrMissingClinicIdentity
	}
	if cmd.OperatingHours == "" {
		return nil, ErrMissingOperatingHours
	}

	// Invariant: a provider must be assigned to at least one clinic to be
	// bookable.
	if a.ProviderNotBookable {
		return nil, ErrProviderNotBookable
	}

	// Invariant: a specialty code must be unique within the directory.
	if a.DuplicateSpecialtyCode {
		return nil, ErrDuplicateSpecialtyCode
	}

	// Invariant: a clinic cannot be deactivated while it has future booked
	// appointments.
	if a.ClinicDeactivationBlocked {
		return nil, ErrClinicDeactivationBlocked
	}

	evt := ClinicConfiguredEvent{
		DirectoryID:    a.ID,
		ClinicIdentity: cmd.ClinicIdentity,
		OperatingHours: cmd.OperatingHours,
	}

	a.applyClinicConfigured(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// apply mutates aggregate state from a domain event. It is the single place
// state changes, so the same function serves both command handling and future
// event replay when rehydrating the aggregate from the store.
func (a *ClinicDirectoryAggregate) apply(evt ProviderRegisteredEvent) {
	a.ProviderIDs = append(a.ProviderIDs, evt.ProviderID)
}

// applySpecialtyUpdated mutates aggregate state from a SpecialtyUpdatedEvent.
// Because the command is an upsert, a code already present is left in place
// (the update only changes its display name, which the aggregate does not
// index) and only a newly created code is appended to the registry.
func (a *ClinicDirectoryAggregate) applySpecialtyUpdated(evt SpecialtyUpdatedEvent) {
	for _, code := range a.SpecialtyCodes {
		if code == evt.SpecialtyCode {
			return
		}
	}
	a.SpecialtyCodes = append(a.SpecialtyCodes, evt.SpecialtyCode)
}

// applyClinicConfigured mutates aggregate state from a ClinicConfiguredEvent.
// Because the command is an upsert, a clinic identity already present is left in
// place (the update only changes its operating hours, which the aggregate does
// not index) and only a newly created identity is appended to the registry.
func (a *ClinicDirectoryAggregate) applyClinicConfigured(evt ClinicConfiguredEvent) {
	for _, id := range a.ClinicIDs {
		if id == evt.ClinicIdentity {
			return
		}
	}
	a.ClinicIDs = append(a.ClinicIDs, evt.ClinicIdentity)
}
