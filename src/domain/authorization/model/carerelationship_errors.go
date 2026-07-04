package model

import "errors"

var (
	// ErrMissingAccountID is returned when AssignScopedRoleCmd omits the account
	// the scoped role is granted to.
	ErrMissingAccountID = errors.New("carerelationship: account id is required")

	// ErrMissingRole is returned when AssignScopedRoleCmd omits the role being
	// granted.
	ErrMissingRole = errors.New("carerelationship: role is required")

	// ErrMissingClinicID is returned when AssignScopedRoleCmd omits the clinic
	// scope the granted role is bounded to.
	ErrMissingClinicID = errors.New("carerelationship: clinic id is required")

	// ErrNoActiveCareRelationship is returned when no active care relationship
	// exists. Invariant: a provider may only access a patient's PHI when an active
	// care relationship exists.
	ErrNoActiveCareRelationship = errors.New("carerelationship: a provider may only access a patient's PHI when an active care relationship exists")

	// ErrRelationshipNotRevoked is returned when the care episode has ended but the
	// relationship remains in force. Invariant: a care relationship must be revoked
	// when the care episode ends.
	ErrRelationshipNotRevoked = errors.New("carerelationship: a care relationship must be revoked when the care episode ends")

	// ErrSelfAssertedWithoutGrant is returned when the relationship was
	// self-asserted by the accessing party without a governing grant. Invariant: a
	// relationship cannot be self-asserted by the accessing party without a
	// governing grant.
	ErrSelfAssertedWithoutGrant = errors.New("carerelationship: a relationship cannot be self-asserted by the accessing party without a governing grant")
)
