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

// Addendum is a correction appended to a signed SOAP note. Because a signed note
// is immutable, corrections are recorded as addenda rather than edits to the
// note body.
type Addendum struct {
	// AuthorID is the participant who authored the addendum.
	AuthorID string
	// Text is the body of the correction.
	Text string
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

	// Addenda are the corrections appended to the encounter's signed note, in the
	// order they were authored.
	Addenda []Addendum
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. Unrecognized command types return shared.ErrUnknownCommand.
func (a *EncounterAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch c := cmd.(type) {
	case OpenEncounterCmd:
		return a.openEncounter(c)
	case AppendAddendumCmd:
		return a.appendAddendum(c)
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

// appendAddendum handles AppendAddendumCmd: it validates the command input,
// enforces the encounter invariants, then emits an
// EncounterAddendumAppendedEvent and buffers it on the aggregate.
//
// The guards enforce, in order:
//
//   - Completeness: encounter, addendum text and author must all be present.
//   - Participant scope: only participants the encounter is scoped to may amend
//     the in-call notes.
//   - Note immutability: an addendum may only be appended to a signed note —
//     corrections are addenda, not edits, so an unsigned or absent note has
//     nothing to amend.
//   - Coded diagnoses: every recorded diagnosis must reference a coded
//     terminology entry.
//   - Completion integrity: a completed encounter must carry a signed note.
func (a *EncounterAggregate) appendAddendum(cmd AppendAddendumCmd) ([]shared.DomainEvent, error) {
	if cmd.EncounterId == "" {
		return nil, ErrMissingEncounter
	}
	if cmd.AddendumText == "" {
		return nil, ErrMissingAddendumText
	}
	if cmd.AuthorId == "" {
		return nil, ErrMissingAuthor
	}

	// Invariant: only participants scoped to the encounter may view or amend the
	// in-call notes, so the author must be one of them.
	if a.ScopedProviderID != cmd.AuthorId && a.ScopedPatientID != cmd.AuthorId {
		return nil, ErrParticipantNotScoped
	}

	// Invariant: a signed SOAP note is immutable — corrections must be appended as
	// addenda, not edits. That mechanism only applies once a note is signed; an
	// unsigned or absent note has nothing to amend.
	if a.Note == nil || !a.Note.Signed {
		return nil, ErrNoSignedNoteToAmend
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

	evt := EncounterAddendumAppendedEvent{
		EncounterID:  a.ID,
		AuthorID:     cmd.AuthorId,
		AddendumText: cmd.AddendumText,
	}

	a.applyAddendumAppended(evt)
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

// applyAddendumAppended mutates aggregate state from an
// EncounterAddendumAppendedEvent by recording the correction. Go has no method
// overloading, so each event kind gets its own apply* method; keeping the
// mutation here means command handling and future event replay share one path.
func (a *EncounterAggregate) applyAddendumAppended(evt EncounterAddendumAppendedEvent) {
	a.Addenda = append(a.Addenda, Addendum{
		AuthorID: evt.AuthorID,
		Text:     evt.AddendumText,
	})
}

// provisionVideoRoomID derives the deterministic identifier of the video room
// provisioned for an encounter, so re-provisioning the same encounter always
// yields the same room.
func provisionVideoRoomID(encounterID string) string {
	return "vr-" + encounterID
}
