package model

// RevokeCareRelationshipCmd requests that an active care relationship be ended,
// withdrawing the provider's grant to the patient's PHI. It carries the identity
// of the relationship to revoke and the reason the grant is being withdrawn.
//
// Revoking is the act that ends a live relationship. The command is rejected
// with a domain error when it is malformed (a missing relationship id or reason)
// or when the relationship violates a care-access invariant: a provider may only
// access a patient's PHI when an active care relationship exists, a care
// relationship must be revoked when the care episode ends, and a relationship
// cannot be self-asserted by the accessing party without a governing grant.
type RevokeCareRelationshipCmd struct {
	// RelationshipID identifies the care relationship to revoke.
	RelationshipID string
	// Reason records why the relationship is being revoked, for the audit trail.
	Reason string
}
