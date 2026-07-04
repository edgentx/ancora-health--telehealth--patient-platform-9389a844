// Package model holds the aggregates for the authorization bounded context.
package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// CareRelationshipAggregate is the authorization aggregate that tracks the
// relationship granting a provider access to a patient's care. It embeds
// shared.AggregateRoot for version tracking and event buffering, and carries its
// own identity in ID.
//
// Beyond identity it tracks the scoped role most recently granted and the flags
// that command invariants read. The invariant flags follow the repository
// convention that a freshly constructed aggregate is valid: their zero value is
// the compliant state, and a non-zero value marks a violation the guards reject.
type CareRelationshipAggregate struct {
	shared.AggregateRoot
	ID string

	// AccountID is the account that holds the most recently assigned scoped role.
	// It is empty until a role has been assigned.
	AccountID string

	// Role is the most recently assigned role. It is empty until a role has been
	// assigned.
	Role string

	// ClinicID is the clinic scope the most recently assigned role is bounded to.
	// It is empty until a role has been assigned.
	ClinicID string

	// NoActiveRelationship reports that no active care relationship exists between
	// the provider and the patient. Invariant: a provider may only access a
	// patient's PHI when an active care relationship exists.
	NoActiveRelationship bool

	// EpisodeEndedNotRevoked reports that the care episode has ended while the
	// relationship remains in force. Invariant: a care relationship must be
	// revoked when the care episode ends.
	EpisodeEndedNotRevoked bool

	// SelfAssertedWithoutGrant reports that the relationship was self-asserted by
	// the accessing party without a governing grant. Invariant: a relationship
	// cannot be self-asserted by the accessing party without a governing grant.
	SelfAssertedWithoutGrant bool
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. Unrecognized command types return shared.ErrUnknownCommand.
func (a *CareRelationshipAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch c := cmd.(type) {
	case AssignScopedRoleCmd:
		return a.assignScopedRole(c)
	default:
		return nil, shared.ErrUnknownCommand
	}
}

// assignScopedRole handles AssignScopedRoleCmd: it validates the command input,
// enforces the care-relationship invariants, then emits a RoleAssignedEvent and
// buffers it on the aggregate.
//
// The guards enforce, in order:
//
//   - Completeness: the account, role, and clinic scope must all be present.
//   - Active relationship: a provider may only access a patient's PHI when an
//     active care relationship exists.
//   - Revocation on episode end: a care relationship must be revoked when the
//     care episode ends.
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
	if a.NoActiveRelationship {
		return nil, ErrNoActiveCareRelationship
	}

	// Invariant: a care relationship must be revoked when the care episode ends.
	if a.EpisodeEndedNotRevoked {
		return nil, ErrRelationshipNotRevoked
	}

	// Invariant: a relationship cannot be self-asserted by the accessing party
	// without a governing grant.
	if a.SelfAssertedWithoutGrant {
		return nil, ErrSelfAssertedWithoutGrant
	}

	evt := RoleAssignedEvent{
		CareRelationshipID: a.ID,
		AccountID:          cmd.AccountID,
		Role:               cmd.Role,
		ClinicID:           cmd.ClinicID,
	}

	a.apply(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// apply mutates aggregate state from a domain event. It is the single place
// state changes, so the same function serves both command handling and future
// event replay when rehydrating the aggregate from the store.
func (a *CareRelationshipAggregate) apply(evt RoleAssignedEvent) {
	a.AccountID = evt.AccountID
	a.Role = evt.Role
	a.ClinicID = evt.ClinicID
}
