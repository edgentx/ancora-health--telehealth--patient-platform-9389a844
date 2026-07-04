// Package model holds the Encounter aggregate for the clinical-records bounded
// context.
package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// EncounterStatus is the lifecycle state of an encounter. The zero value is an
// unopened (scheduled) encounter, which is what OpenEncounterCmd acts on.
type EncounterStatus string

const (
	// EncounterStatusScheduled is a booked but not-yet-started encounter. It is
	// the zero value, so a freshly constructed aggregate is scheduled.
	EncounterStatusScheduled EncounterStatus = ""
	// EncounterStatusOpen is an in-progress encounter whose video room has been
	// provisioned.
	EncounterStatusOpen EncounterStatus = "open"
	// EncounterStatusCompleted is a finished encounter.
	EncounterStatusCompleted EncounterStatus = "completed"
)

// ClinicalNote is the SOAP note attached to an encounter. Once Signed it is
// immutable and may only be corrected by appending addenda.
type ClinicalNote struct {
	// Content is the body of the note.
	Content string
	// Signed reports whether the note has been signed by the provider.
	Signed bool
}

// Diagnosis is a coded finding recorded on the encounter. Code must reference a
// valid coded terminology entry (e.g., an ICD-10 code).
type Diagnosis struct {
	// Code is the coded-terminology reference for the diagnosis (e.g. "J06.9").
	Code string
	// Description is the human-readable label for the diagnosis.
	Description string
}

// EncounterAggregate is the clinical-records Encounter aggregate. It embeds
// shared.AggregateRoot for version tracking and an uncommitted-event buffer,
// and carries its own string identity.
//
// Beyond identity it tracks the state that command invariants read: its
// lifecycle status, the provider/patient scoped to it, the SOAP note, and any
// coded diagnoses.
type EncounterAggregate struct {
	shared.AggregateRoot
	ID string

	// Status is the encounter's lifecycle state.
	Status EncounterStatus

	// ScopedProviderID and ScopedPatientID are the participants the encounter is
	// bound to. When non-empty they constrain who may open and join the video
	// room; an empty value means the participant is bound on open.
	ScopedProviderID string
	ScopedPatientID  string

	// VideoRoomID is the identifier of the video room provisioned on open. It is
	// empty until the encounter is opened.
	VideoRoomID string

	// Note is the encounter's SOAP note, or nil if none has been drafted yet.
	Note *ClinicalNote

	// Diagnoses are the coded diagnoses recorded on the encounter.
	Diagnoses []Diagnosis
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. Unrecognized command types return shared.ErrUnknownCommand.
func (a *EncounterAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch c := cmd.(type) {
	case OpenEncounterCmd:
		return a.openEncounter(c)
	case SignSoapNoteCmd:
		return a.signSoapNote(c)
	default:
		return nil, shared.ErrUnknownCommand
	}
}

// openEncounter handles OpenEncounterCmd: it validates the command input,
// enforces the encounter invariants, then emits an EncounterOpenedEvent and
// buffers it on the aggregate.
//
// The guards enforce, in order:
//
//   - Completeness: appointment, provider and patient must all be present.
//   - Participant scope: only participants the encounter is scoped to may join
//     the video room or view in-call notes.
//   - Note immutability: a signed SOAP note may not be reopened for editing.
//   - Coded diagnoses: every recorded diagnosis must reference a coded
//     terminology entry.
//   - Completion integrity: a completed encounter must carry a signed note.
func (a *EncounterAggregate) openEncounter(cmd OpenEncounterCmd) ([]shared.DomainEvent, error) {
	if cmd.AppointmentId == "" {
		return nil, ErrMissingAppointment
	}
	if cmd.ProviderId == "" {
		return nil, ErrMissingProvider
	}
	if cmd.PatientId == "" {
		return nil, ErrMissingPatient
	}

	// Invariant: only participants scoped to the encounter may join the video
	// room. When the encounter is already bound to a provider/patient, the
	// command must name those same participants.
	if a.ScopedProviderID != "" && a.ScopedProviderID != cmd.ProviderId {
		return nil, ErrParticipantNotScoped
	}
	if a.ScopedPatientID != "" && a.ScopedPatientID != cmd.PatientId {
		return nil, ErrParticipantNotScoped
	}

	// Invariant: a signed SOAP note is immutable. Reopening an encounter whose
	// note is signed would expose it to edits, so it is rejected — corrections
	// belong in appended addenda.
	if a.Note != nil && a.Note.Signed {
		return nil, ErrSignedNoteImmutable
	}

	// Invariant: every diagnosis must reference a coded terminology entry.
	for _, d := range a.Diagnoses {
		if d.Code == "" {
			return nil, ErrDiagnosisUncoded
		}
	}

	// Invariant: an encounter cannot be complete without a signed note.
	if a.Status == EncounterStatusCompleted && (a.Note == nil || !a.Note.Signed) {
		return nil, ErrIncompleteWithoutSignedNote
	}

	evt := EncounterOpenedEvent{
		EncounterID:   a.ID,
		AppointmentID: cmd.AppointmentId,
		ProviderID:    cmd.ProviderId,
		PatientID:     cmd.PatientId,
		VideoRoomID:   provisionVideoRoomID(a.ID),
	}

	a.apply(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// apply mutates aggregate state from a domain event. It is the single place
// state changes, so the same function serves both command handling and future
// event replay when rehydrating the aggregate from the store.
func (a *EncounterAggregate) apply(evt EncounterOpenedEvent) {
	a.Status = EncounterStatusOpen
	a.ScopedProviderID = evt.ProviderID
	a.ScopedPatientID = evt.PatientID
	a.VideoRoomID = evt.VideoRoomID
}

// signSoapNote handles SignSoapNoteCmd: it validates the command input, enforces
// the encounter invariants, then emits a SoapNoteSignedEvent and buffers it on
// the aggregate.
//
// The guards enforce, in order:
//
//   - Completeness: encounter id, note body and at least one diagnosis must all
//     be present.
//   - Participant scope: only the provider the encounter is scoped to may sign
//     the in-call note.
//   - Note immutability: an already-signed note may not be re-signed; a
//     correction must be an appended addendum, not a fresh signature.
//   - Coded diagnoses: every diagnosis being recorded must reference a coded
//     terminology entry.
//   - Completion integrity: the aggregate must not already be a completed
//     encounter that is somehow missing its signed note.
func (a *EncounterAggregate) signSoapNote(cmd SignSoapNoteCmd) ([]shared.DomainEvent, error) {
	if cmd.EncounterId == "" {
		return nil, ErrMissingEncounter
	}
	if cmd.SoapNote == "" {
		return nil, ErrMissingSoapNote
	}
	if len(cmd.Diagnoses) == 0 {
		return nil, ErrMissingDiagnoses
	}

	// Invariant: only participants scoped to the encounter may view or act on
	// the in-call note. When the encounter is bound to a provider, only that
	// provider may sign its note.
	if a.ScopedProviderID != "" && a.ScopedProviderID != cmd.ProviderId {
		return nil, ErrParticipantNotScoped
	}

	// Invariant: a signed SOAP note is immutable. Re-signing an already-signed
	// note would let it be edited under a new signature, so it is rejected —
	// corrections belong in appended addenda.
	if a.Note != nil && a.Note.Signed {
		return nil, ErrSignedNoteImmutable
	}

	// Invariant: every diagnosis must reference a coded terminology entry.
	for _, d := range cmd.Diagnoses {
		if d.Code == "" {
			return nil, ErrDiagnosisUncoded
		}
	}

	// Invariant: an encounter cannot be complete without a signed note. Guard
	// against acting on an aggregate that is already completed yet inconsistent.
	if a.Status == EncounterStatusCompleted && (a.Note == nil || !a.Note.Signed) {
		return nil, ErrIncompleteWithoutSignedNote
	}

	evt := SoapNoteSignedEvent{
		EncounterID: a.ID,
		ProviderID:  cmd.ProviderId,
		SoapNote:    cmd.SoapNote,
		Diagnoses:   cmd.Diagnoses,
	}

	a.applySoapNoteSigned(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// applySoapNoteSigned mutates aggregate state from a SoapNoteSignedEvent. Like
// apply it is the single place these state changes happen, so it serves both
// command handling and future event replay.
func (a *EncounterAggregate) applySoapNoteSigned(evt SoapNoteSignedEvent) {
	a.Note = &ClinicalNote{Content: evt.SoapNote, Signed: true}
	a.Diagnoses = evt.Diagnoses
}

// provisionVideoRoomID derives the deterministic identifier of the video room
// provisioned for an encounter, so re-provisioning the same encounter always
// yields the same room.
func provisionVideoRoomID(encounterID string) string {
	return "vr-" + encounterID
}
