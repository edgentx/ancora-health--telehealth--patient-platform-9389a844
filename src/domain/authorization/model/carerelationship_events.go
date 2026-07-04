package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// CareRelationshipEstablishedEventType is the stable wire name emitted when a
// care-team-to-patient grant is established.
const CareRelationshipEstablishedEventType = "care.relationship.established"

// CareRelationshipEstablishedEvent is emitted when an EstablishCareRelationshipCmd
// succeeds. It records the provider, patient, and clinic that scope the newly
// created care grant.
type CareRelationshipEstablishedEvent struct {
	// RelationshipID is the identity of the CareRelationshipAggregate that
	// produced the event.
	RelationshipID string
	// ProviderID identifies the care-team member granted access to the patient's
	// PHI.
	ProviderID string
	// PatientID identifies the patient whose PHI the grant authorizes access to.
	PatientID string
	// ClinicID identifies the clinic the care relationship is scoped within.
	ClinicID string
}

// Type identifies the event kind.
func (e CareRelationshipEstablishedEvent) Type() string { return CareRelationshipEstablishedEventType }

// AggregateID ties the event back to the care relationship that produced it.
func (e CareRelationshipEstablishedEvent) AggregateID() string { return e.RelationshipID }

// Compile-time assertion that CareRelationshipEstablishedEvent satisfies the
// DomainEvent contract.
var _ shared.DomainEvent = CareRelationshipEstablishedEvent{}

// CareRelationshipRevokedEventType is the stable wire name emitted when a
// care-team-to-patient grant is revoked.
const CareRelationshipRevokedEventType = "care.relationship.revoked"

// CareRelationshipRevokedEvent is emitted when a RevokeCareRelationshipCmd
// succeeds. It records the relationship whose grant was revoked and the reason
// the care relationship was ended.
type CareRelationshipRevokedEvent struct {
	// RelationshipID is the identity of the CareRelationshipAggregate that
	// produced the event.
	RelationshipID string
	// Reason records why the care relationship was ended.
	Reason string
}

// Type identifies the event kind.
func (e CareRelationshipRevokedEvent) Type() string { return CareRelationshipRevokedEventType }

// AggregateID ties the event back to the care relationship that produced it.
func (e CareRelationshipRevokedEvent) AggregateID() string { return e.RelationshipID }

// Compile-time assertion that CareRelationshipRevokedEvent satisfies the
// DomainEvent contract.
var _ shared.DomainEvent = CareRelationshipRevokedEvent{}

// RoleAssignedEventType is the stable wire name emitted when a role is granted
// to an account, bounded to a clinic scope.
const RoleAssignedEventType = "authz.role.assigned"

// RoleAssignedEvent is emitted when an AssignScopedRoleCmd succeeds. It records
// the account granted the role, the role itself, and the clinic the assignment
// is bounded to.
type RoleAssignedEvent struct {
	// RelationshipID is the identity of the CareRelationshipAggregate that
	// produced the event.
	RelationshipID string
	// AccountID identifies the account granted the role.
	AccountID string
	// Role is the role granted to the account within the clinic scope.
	Role string
	// ClinicID identifies the clinic the role assignment is bounded to.
	ClinicID string
}

// Type identifies the event kind.
func (e RoleAssignedEvent) Type() string { return RoleAssignedEventType }

// AggregateID ties the event back to the care relationship that produced it.
func (e RoleAssignedEvent) AggregateID() string { return e.RelationshipID }

// Compile-time assertion that RoleAssignedEvent satisfies the DomainEvent
// contract.
var _ shared.DomainEvent = RoleAssignedEvent{}
