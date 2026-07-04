// Package model holds the aggregates for the authorization bounded context.
package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// RelationshipStatus is the lifecycle state of a care relationship. The zero
// value is a relationship that has not yet been established, which is what
// EstablishCareRelationshipCmd acts on.
type RelationshipStatus string

const (
	// RelationshipStatusPending is a relationship that has not yet been
	// established. It is the zero value, so a freshly constructed aggregate is
	// pending.
	RelationshipStatusPending RelationshipStatus = ""
	// RelationshipStatusActive is a relationship whose grant is in force,
	// authorizing the provider to access the patient's PHI.
	RelationshipStatusActive RelationshipStatus = "active"
	// RelationshipStatusRevoked is a relationship whose grant has been ended,
	// withdrawing the provider's authorization to access the patient's PHI. It is
	// the terminal state RevokeCareRelationshipCmd produces.
	RelationshipStatusRevoked RelationshipStatus = "revoked"
)

// CareRelationshipAggregate is the authorization aggregate that tracks the
// relationship granting a provider access to a patient's care. It embeds
// shared.AggregateRoot for version tracking and event buffering, and carries
// its own identity in ID.
//
// Beyond identity it tracks the state that command invariants read: its
// lifecycle status, the provider/patient/clinic that scope an established
// grant, and the flags describing whether establishing the grant would violate
// one of the care-relationship invariants.
//
// The invariant flags follow the repository convention that a freshly
// constructed aggregate is valid: their zero value is the compliant state, and
// a non-zero value marks a violation the guards reject.
type CareRelationshipAggregate struct {
	shared.AggregateRoot
	ID string

	// Status is the relationship's lifecycle state.
	Status RelationshipStatus

	// ProviderID is the care-team member granted access by the in-force grant. It
	// is empty until the relationship is established.
	ProviderID string

	// PatientID is the patient whose PHI the in-force grant authorizes access to.
	// It is empty until the relationship is established.
	PatientID string

	// ClinicID is the clinic the in-force grant is scoped within. It is empty
	// until the relationship is established.
	ClinicID string

	// Inactive reports that the grant would not be active. Invariant: a provider
	// may only access a patient's PHI when an active care relationship exists.
	Inactive bool

	// EpisodeEnded reports that the care episode has ended but the relationship
	// has not been revoked. Invariant: a care relationship must be revoked when
	// the care episode ends.
	EpisodeEnded bool

	// SelfAsserted reports that the relationship is asserted by the accessing
	// party without a governing grant. Invariant: a relationship cannot be
	// self-asserted by the accessing party without a governing grant.
	SelfAsserted bool

	// ScopedRoleAccountID, ScopedRole and ScopedRoleClinicID record the most
	// recent clinic-scoped role assignment. They are empty until a role is
	// assigned via AssignScopedRoleCmd, at which point they capture the account,
	// role and clinic the grant is bounded to.
	ScopedRoleAccountID string
	ScopedRole          string
	ScopedRoleClinicID  string
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. Unrecognized command types return shared.ErrUnknownCommand.
func (a *CareRelationshipAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch c := cmd.(type) {
	case EstablishCareRelationshipCmd:
		return a.establishCareRelationship(c)
	case RevokeCareRelationshipCmd:
		return a.revokeCareRelationship(c)
	case AssignScopedRoleCmd:
		return a.assignScopedRole(c)
	default:
		return nil, shared.ErrUnknownCommand
	}
}

// establishCareRelationship handles EstablishCareRelationshipCmd: it validates
// the command input, enforces the care-relationship invariants, then emits a
// CareRelationshipEstablishedEvent and buffers it on the aggregate.
//
// The guards enforce, in order:
//
//   - Completeness: the provider, patient, and clinic must all be present.
//   - Active relationship: a provider may only access a patient's PHI when an
//     active care relationship exists.
//   - Episode revocation: a care relationship must be revoked when the care
//     episode ends.
//   - Governing grant: a relationship cannot be self-asserted by the accessing
//     party without a governing grant.
func (a *CareRelationshipAggregate) establishCareRelationship(cmd EstablishCareRelationshipCmd) ([]shared.DomainEvent, error) {
	if cmd.ProviderID == "" {
		return nil, ErrMissingProviderID
	}
	if cmd.PatientID == "" {
		return nil, ErrMissingPatientID
	}
	if cmd.ClinicID == "" {
		return nil, ErrMissingClinicID
	}

	// Invariant: a provider may only access a patient's PHI when an active care
	// relationship exists.
	if a.Inactive {
		return nil, ErrNoActiveRelationship
	}

	// Invariant: a care relationship must be revoked when the care episode ends.
	if a.EpisodeEnded {
		return nil, ErrCareEpisodeEnded
	}

	// Invariant: a relationship cannot be self-asserted by the accessing party
	// without a governing grant.
	if a.SelfAsserted {
		return nil, ErrSelfAssertedRelationship
	}

	evt := CareRelationshipEstablishedEvent{
		RelationshipID: a.ID,
		ProviderID:     cmd.ProviderID,
		PatientID:      cmd.PatientID,
		ClinicID:       cmd.ClinicID,
	}

	a.apply(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// revokeCareRelationship handles RevokeCareRelationshipCmd: it validates the
// command input, enforces the care-relationship invariants, then emits a
// CareRelationshipRevokedEvent and buffers it on the aggregate. Revoking takes
// the in-force grant out of force, withdrawing the provider's authorization to
// access the patient's PHI.
//
// The guards enforce, in order:
//
//   - Completeness: the relationship id and the reason must both be present.
//   - Active relationship: a provider may only access a patient's PHI when an
//     active care relationship exists.
//   - Episode revocation: a care relationship must be revoked when the care
//     episode ends.
//   - Governing grant: a relationship cannot be self-asserted by the accessing
//     party without a governing grant.
func (a *CareRelationshipAggregate) revokeCareRelationship(cmd RevokeCareRelationshipCmd) ([]shared.DomainEvent, error) {
	if cmd.RelationshipID == "" {
		return nil, ErrMissingRelationshipID
	}
	if cmd.Reason == "" {
		return nil, ErrMissingReason
	}

	// Invariant: a provider may only access a patient's PHI when an active care
	// relationship exists.
	if a.Status != RelationshipStatusActive || a.Inactive {
		return nil, ErrNoActiveRelationship
	}

	// Invariant: a care relationship must be revoked when the care episode ends.
	if a.EpisodeEnded {
		return nil, ErrCareEpisodeEnded
	}

	// Invariant: a relationship cannot be self-asserted by the accessing party
	// without a governing grant.
	if a.SelfAsserted {
		return nil, ErrSelfAssertedRelationship
	}

	evt := CareRelationshipRevokedEvent{
		RelationshipID: a.ID,
		Reason:         cmd.Reason,
	}

	a.applyRevoked(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// apply mutates aggregate state from a domain event. It is the single place
// state changes, so the same function serves both command handling and future
// event replay when rehydrating the aggregate from the store.
func (a *CareRelationshipAggregate) apply(evt CareRelationshipEstablishedEvent) {
	a.Status = RelationshipStatusActive
	a.ProviderID = evt.ProviderID
	a.PatientID = evt.PatientID
	a.ClinicID = evt.ClinicID
}

// applyRevoked mutates aggregate state from a CareRelationshipRevokedEvent,
// moving the relationship to its terminal revoked state. Like apply, it is the
// single place revocation state changes, so it serves both command handling and
// future event replay when rehydrating the aggregate from the store.
func (a *CareRelationshipAggregate) applyRevoked(evt CareRelationshipRevokedEvent) {
	a.Status = RelationshipStatusRevoked
}

// assignScopedRole handles AssignScopedRoleCmd: it validates the command input,
// enforces the care-relationship invariants, then emits a RoleAssignedEvent and
// buffers it on the aggregate. The role is bounded to the supplied clinic so the
// account holds it only within that clinic's scope, not platform-wide.
//
// The guards enforce, in order:
//
//   - Completeness: the account, role, and clinic must all be present.
//   - Active relationship: a provider may only access a patient's PHI when an
//     active care relationship exists.
//   - Episode revocation: a care relationship must be revoked when the care
//     episode ends.
//   - Governing grant: a relationship cannot be self-asserted by the accessing
//     party without a governing grant.
func (a *CareRelationshipAggregate) assignScopedRole(cmd AssignScopedRoleCmd) ([]shared.DomainEvent, error) {
	if cmd.AccountID == "" {
		return nil, ErrMissingAccountID
	}
	if cmd.Role == "" {
		return nil, ErrMissingRole
	}
	if cmd.ClinicID == "" {
		return nil, ErrMissingClinicID
	}

	// Invariant: a provider may only access a patient's PHI when an active care
	// relationship exists.
	if a.Status != RelationshipStatusActive || a.Inactive {
		return nil, ErrNoActiveRelationship
	}

	// Invariant: a care relationship must be revoked when the care episode ends.
	if a.EpisodeEnded {
		return nil, ErrCareEpisodeEnded
	}

	// Invariant: a relationship cannot be self-asserted by the accessing party
	// without a governing grant.
	if a.SelfAsserted {
		return nil, ErrSelfAssertedRelationship
	}

	evt := RoleAssignedEvent{
		RelationshipID: a.ID,
		AccountID:      cmd.AccountID,
		Role:           cmd.Role,
		ClinicID:       cmd.ClinicID,
	}

	a.applyRoleAssigned(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// applyRoleAssigned mutates aggregate state from a RoleAssignedEvent. Like
// apply, it is the single place this event changes state, so it serves both
// command handling and future event replay when rehydrating from the store.
func (a *CareRelationshipAggregate) applyRoleAssigned(evt RoleAssignedEvent) {
	a.ScopedRoleAccountID = evt.AccountID
	a.ScopedRole = evt.Role
	a.ScopedRoleClinicID = evt.ClinicID
}
