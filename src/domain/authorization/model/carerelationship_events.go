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
