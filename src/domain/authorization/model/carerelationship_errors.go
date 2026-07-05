package model

import "errors"

var (
	// ErrMissingProviderID is returned when EstablishCareRelationshipCmd omits the
	// provider being granted access to the patient's PHI.
	ErrMissingProviderID = errors.New("carerelationship: provider id is required")

	// ErrMissingPatientID is returned when EstablishCareRelationshipCmd omits the
	// patient whose PHI the grant authorizes access to.
	ErrMissingPatientID = errors.New("carerelationship: patient id is required")

	// ErrMissingClinicID is returned when EstablishCareRelationshipCmd or
	// AssignScopedRoleCmd omits the clinic the grant is scoped within.
	ErrMissingClinicID = errors.New("carerelationship: clinic id is required")

	// ErrMissingRelationshipID is returned when RevokeCareRelationshipCmd omits the
	// relationship whose grant is being revoked.
	ErrMissingRelationshipID = errors.New("carerelationship: relationship id is required")

	// ErrMissingReason is returned when RevokeCareRelationshipCmd omits the reason
	// the care relationship is being ended.
	ErrMissingReason = errors.New("carerelationship: reason is required")

	// ErrMissingAccountID is returned when AssignScopedRoleCmd omits the account
	// being granted the role.
	ErrMissingAccountID = errors.New("carerelationship: account id is required")

	// ErrMissingRole is returned when AssignScopedRoleCmd omits the role being
	// granted to the account.
	ErrMissingRole = errors.New("carerelationship: role is required")

	// ErrNoActiveRelationship is returned when the grant would not be active.
	// Invariant: a provider may only access a patient's PHI when an active care
	// relationship exists.
	ErrNoActiveRelationship = errors.New("carerelationship: a provider may only access a patient's PHI when an active care relationship exists")

	// ErrCareEpisodeEnded is returned when the care episode has ended but the
	// relationship has not been revoked. Invariant: a care relationship must be
	// revoked when the care episode ends.
	ErrCareEpisodeEnded = errors.New("carerelationship: a care relationship must be revoked when the care episode ends")

	// ErrSelfAssertedRelationship is returned when the relationship is asserted by
	// the accessing party without a governing grant. Invariant: a relationship
	// cannot be self-asserted by the accessing party without a governing grant.
	ErrSelfAssertedRelationship = errors.New("carerelationship: a relationship cannot be self-asserted by the accessing party without a governing grant")
)
