package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// RoleAssignedEventType is the stable wire name emitted when a scoped role is
// assigned on a care relationship.
const RoleAssignedEventType = "authz.role.assigned"

// RoleAssignedEvent is emitted when an AssignScopedRoleCmd succeeds. It records
// the account granted the role, the role that was granted, and the clinic scope
// the grant is bounded to.
type RoleAssignedEvent struct {
	// CareRelationshipID is the identity of the CareRelationshipAggregate that
	// produced the event.
	CareRelationshipID string
	// AccountID identifies the account the scoped role was granted to.
	AccountID string
	// Role is the role that was granted on the care relationship.
	Role string
	// ClinicID is the clinic scope the granted role is bounded to.
	ClinicID string
}

// Type identifies the event kind.
func (e RoleAssignedEvent) Type() string { return RoleAssignedEventType }

// AggregateID ties the event back to the care relationship that produced it.
func (e RoleAssignedEvent) AggregateID() string { return e.CareRelationshipID }

// Compile-time assertion that RoleAssignedEvent satisfies the DomainEvent
// contract.
var _ shared.DomainEvent = RoleAssignedEvent{}
