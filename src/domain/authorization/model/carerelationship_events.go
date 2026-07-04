package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// CareRelationshipRevokedEventType is the stable wire name emitted when an
// active care relationship is revoked.
const CareRelationshipRevokedEventType = "care.relationship.revoked"

// CareRelationshipRevokedEvent is emitted when a RevokeCareRelationshipCmd
// succeeds. It records the relationship that was ended and the reason its grant
// to the patient's PHI was withdrawn.
type CareRelationshipRevokedEvent struct {
	// RelationshipID is the identity of the CareRelationshipAggregate that produced
	// the event.
	RelationshipID string
	// Reason records why the relationship was revoked, for the audit trail.
	Reason string
}

// Type identifies the event kind.
func (e CareRelationshipRevokedEvent) Type() string { return CareRelationshipRevokedEventType }

// AggregateID ties the event back to the relationship that produced it.
func (e CareRelationshipRevokedEvent) AggregateID() string { return e.RelationshipID }

// Compile-time assertion that CareRelationshipRevokedEvent satisfies the
// DomainEvent contract.
var _ shared.DomainEvent = CareRelationshipRevokedEvent{}
