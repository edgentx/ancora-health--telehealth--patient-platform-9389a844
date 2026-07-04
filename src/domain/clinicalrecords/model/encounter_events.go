package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// EncounterOpenedEventType is the stable wire name emitted when an encounter is
// started and its video room provisioned.
const EncounterOpenedEventType = "encounter.opened"

// EncounterOpenedEvent is emitted when an OpenEncounterCmd succeeds. It records
// the participants scoped to the encounter and the identifier of the video room
// provisioned for them to join.
type EncounterOpenedEvent struct {
	// EncounterID is the identity of the EncounterAggregate that produced the
	// event.
	EncounterID string
	// AppointmentID is the appointment this encounter realizes.
	AppointmentID string
	// ProviderID and PatientID are the participants scoped to the encounter.
	ProviderID string
	PatientID  string
	// VideoRoomID is the identifier of the video room provisioned on open; only
	// the scoped participants may join it.
	VideoRoomID string
}

// Type identifies the event kind.
func (e EncounterOpenedEvent) Type() string { return EncounterOpenedEventType }

// AggregateID ties the event back to the encounter that produced it.
func (e EncounterOpenedEvent) AggregateID() string { return e.EncounterID }

// Compile-time assertion that EncounterOpenedEvent satisfies the DomainEvent
// contract.
var _ shared.DomainEvent = EncounterOpenedEvent{}
