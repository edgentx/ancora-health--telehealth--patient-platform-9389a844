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

// EncounterAddendumAppendedEventType is the stable wire name emitted when a
// correction is appended to an encounter's signed SOAP note.
const EncounterAddendumAppendedEventType = "encounter.addendum.appended"

// EncounterAddendumAppendedEvent is emitted when an AppendAddendumCmd succeeds.
// It records the author of the correction and its text; the signed note it
// amends is left untouched, preserving its immutability.
type EncounterAddendumAppendedEvent struct {
	// EncounterID is the identity of the EncounterAggregate that produced the
	// event.
	EncounterID string
	// AuthorID is the participant who authored the addendum.
	AuthorID string
	// AddendumText is the body of the appended correction.
	AddendumText string
}

// Type identifies the event kind.
func (e EncounterAddendumAppendedEvent) Type() string { return EncounterAddendumAppendedEventType }

// AggregateID ties the event back to the encounter that produced it.
func (e EncounterAddendumAppendedEvent) AggregateID() string { return e.EncounterID }

// Compile-time assertion that EncounterAddendumAppendedEvent satisfies the
// DomainEvent contract.
var _ shared.DomainEvent = EncounterAddendumAppendedEvent{}
