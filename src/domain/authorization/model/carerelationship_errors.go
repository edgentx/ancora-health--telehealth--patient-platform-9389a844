package model

import "errors"

var (
	// ErrMissingRelationshipID is returned when RevokeCareRelationshipCmd omits the
	// identity of the relationship to revoke.
	ErrMissingRelationshipID = errors.New("carerelationship: relationship id is required")

	// ErrMissingRevocationReason is returned when RevokeCareRelationshipCmd omits
	// the reason the relationship is being revoked.
	ErrMissingRevocationReason = errors.New("carerelationship: revocation reason is required")

	// ErrNoActiveCareRelationship is returned when there is no active relationship
	// to revoke. Invariant: a provider may only access a patient's PHI when an
	// active care relationship exists.
	ErrNoActiveCareRelationship = errors.New("carerelationship: a provider may only access a patient's PHI when an active care relationship exists")

	// ErrCareEpisodeUnresolved is returned when the care episode's end is out of
	// step with the relationship's revocation state. Invariant: a care relationship
	// must be revoked when the care episode ends.
	ErrCareEpisodeUnresolved = errors.New("carerelationship: a care relationship must be revoked when the care episode ends")

	// ErrSelfAssertedRelationship is returned when the relationship was asserted by
	// the accessing party without a governing grant. Invariant: a relationship
	// cannot be self-asserted by the accessing party without a governing grant.
	ErrSelfAssertedRelationship = errors.New("carerelationship: a relationship cannot be self-asserted by the accessing party without a governing grant")
)
