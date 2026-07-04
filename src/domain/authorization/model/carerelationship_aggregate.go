// Package model holds the aggregates for the authorization bounded context.
package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// CareRelationshipStatus is the lifecycle state of a care relationship. The zero
// value is an active relationship, so a freshly established aggregate is one a
// provider may act under — and the one RevokeCareRelationshipCmd ends.
type CareRelationshipStatus string

const (
	// CareRelationshipStatusActive is a relationship currently granting a provider
	// access to a patient's care. It is the zero value, so a freshly constructed
	// aggregate is active.
	CareRelationshipStatusActive CareRelationshipStatus = ""
	// CareRelationshipStatusRevoked is a relationship that has been ended; its
	// provider no longer has a live grant to the patient's PHI.
	CareRelationshipStatusRevoked CareRelationshipStatus = "revoked"
)

// CareRelationshipAggregate is the authorization aggregate that tracks the
// relationship granting a provider access to a patient's care. It embeds
// shared.AggregateRoot for version tracking and event buffering, and carries its
// own identity in ID.
//
// Beyond identity it tracks the lifecycle Status the command handlers read, the
// RevocationReason recorded when the relationship is ended, and the invariant
// flags describing whether the relationship violates one of the care-access
// invariants.
//
// The invariant flags follow the repository convention that a freshly
// constructed aggregate is valid: their zero value is the compliant state, and a
// non-zero value marks a violation the guards reject.
type CareRelationshipAggregate struct {
	shared.AggregateRoot
	ID string

	// Status is the relationship's lifecycle state.
	Status CareRelationshipStatus

	// RevocationReason records why the relationship was revoked. It is empty until
	// a RevokeCareRelationshipCmd is handled.
	RevocationReason string

	// NoActiveRelationship reports that no active care relationship exists for the
	// provider-patient pair. Invariant: a provider may only access a patient's PHI
	// when an active care relationship exists.
	NoActiveRelationship bool

	// CareEpisodeUnresolved reports that the governing care episode's end is out of
	// step with the relationship's revocation state. Invariant: a care relationship
	// must be revoked when the care episode ends.
	CareEpisodeUnresolved bool

	// SelfAsserted reports that the relationship was asserted by the accessing
	// party itself without a governing grant to back it. Invariant: a relationship
	// cannot be self-asserted by the accessing party without a governing grant.
	SelfAsserted bool
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. Unrecognized command types return shared.ErrUnknownCommand.
func (a *CareRelationshipAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch c := cmd.(type) {
	case RevokeCareRelationshipCmd:
		return a.revokeCareRelationship(c)
	default:
		return nil, shared.ErrUnknownCommand
	}
}

// revokeCareRelationship handles RevokeCareRelationshipCmd: it validates the
// command input, enforces the care-access invariants, then emits a
// CareRelationshipRevokedEvent and buffers it on the aggregate.
//
// The guards enforce, in order:
//
//   - Completeness: the relationship id and the revocation reason must both be
//     present.
//   - Active relationship: a provider may only access a patient's PHI when an
//     active care relationship exists, so there must be a live relationship to
//     revoke.
//   - Care-episode coupling: a care relationship must be revoked when the care
//     episode ends; the episode and revocation state must not be out of step.
//   - Governing grant: a relationship cannot be self-asserted by the accessing
//     party without a governing grant.
func (a *CareRelationshipAggregate) revokeCareRelationship(cmd RevokeCareRelationshipCmd) ([]shared.DomainEvent, error) {
	if cmd.RelationshipID == "" {
		return nil, ErrMissingRelationshipID
	}
	if cmd.Reason == "" {
		return nil, ErrMissingRevocationReason
	}

	// Invariant: a provider may only access a patient's PHI when an active care
	// relationship exists.
	if a.NoActiveRelationship {
		return nil, ErrNoActiveCareRelationship
	}

	// Invariant: a care relationship must be revoked when the care episode ends.
	if a.CareEpisodeUnresolved {
		return nil, ErrCareEpisodeUnresolved
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

	a.apply(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// apply mutates aggregate state from a CareRelationshipRevokedEvent. It is the
// single place state changes, so the same function serves both command handling
// and future event replay when rehydrating the aggregate from the store.
func (a *CareRelationshipAggregate) apply(evt CareRelationshipRevokedEvent) {
	a.Status = CareRelationshipStatusRevoked
	a.RevocationReason = evt.Reason
}
