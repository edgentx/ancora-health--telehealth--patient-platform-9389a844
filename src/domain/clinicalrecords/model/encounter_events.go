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

// SoapNoteSignedEventType is the stable wire name emitted when an encounter's
// SOAP note is signed and sealed.
const SoapNoteSignedEventType = "encounter.note.signed"

// SoapNoteSignedEvent is emitted when a SignSoapNoteCmd succeeds. It records the
// provider who signed the note, the sealed note body, and the coded diagnoses
// captured for the encounter.
type SoapNoteSignedEvent struct {
	// EncounterID is the identity of the EncounterAggregate that produced the
	// event.
	EncounterID string
	// ProviderID is the provider who signed the note; it is the participant the
	// encounter is scoped to.
	ProviderID string
	// SoapNote is the sealed body of the SOAP note.
	SoapNote string
	// Diagnoses are the coded findings recorded on the encounter.
	Diagnoses []Diagnosis
}

// Type identifies the event kind.
func (e SoapNoteSignedEvent) Type() string { return SoapNoteSignedEventType }

// AggregateID ties the event back to the encounter that produced it.
func (e SoapNoteSignedEvent) AggregateID() string { return e.EncounterID }

// Compile-time assertion that SoapNoteSignedEvent satisfies the DomainEvent
// contract.
var _ shared.DomainEvent = SoapNoteSignedEvent{}

// EncounterCompletedEventType is the stable wire name emitted when a documented
// encounter is closed and marked complete.
const EncounterCompletedEventType = "encounter.completed"

// EncounterCompletedEvent is emitted when a CompleteEncounterCmd succeeds. It
// records the provider who closed the encounter; a completed encounter always
// carries a signed note.
type EncounterCompletedEvent struct {
	// EncounterID is the identity of the EncounterAggregate that produced the
	// event.
	EncounterID string
	// ProviderID is the provider who completed the encounter; it is the
	// participant the encounter is scoped to.
	ProviderID string
}

// Type identifies the event kind.
func (e EncounterCompletedEvent) Type() string { return EncounterCompletedEventType }

// AggregateID ties the event back to the encounter that produced it.
func (e EncounterCompletedEvent) AggregateID() string { return e.EncounterID }

// Compile-time assertion that EncounterCompletedEvent satisfies the DomainEvent
// contract.
var _ shared.DomainEvent = EncounterCompletedEvent{}
