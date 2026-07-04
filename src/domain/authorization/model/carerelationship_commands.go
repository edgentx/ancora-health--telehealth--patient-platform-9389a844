package model

// AssignScopedRoleCmd requests that a role be granted on the care relationship,
// bounded to a single clinic scope. It carries the account receiving the role,
// the role being granted, and the clinic the grant is scoped to.
//
// Assigning a scoped role is only permitted while the care relationship holds:
// a provider may only access a patient's PHI when an active care relationship
// exists, a care relationship must be revoked when the care episode ends, and a
// relationship cannot be self-asserted by the accessing party without a
// governing grant. AccountID, Role, and ClinicID are all mandatory.
type AssignScopedRoleCmd struct {
	// AccountID identifies the account the scoped role is granted to.
	AccountID string
	// Role is the role being granted on the care relationship.
	Role string
	// ClinicID is the clinic scope the granted role is bounded to.
	ClinicID string
}
