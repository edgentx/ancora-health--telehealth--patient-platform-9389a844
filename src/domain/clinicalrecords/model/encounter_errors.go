package model

import "errors"

var (
	// ErrMissingAppointment is returned when OpenEncounterCmd omits the
	// appointment id.
	ErrMissingAppointment = errors.New("encounter: appointment id is required")

	// ErrMissingProvider is returned when OpenEncounterCmd omits the provider id.
	ErrMissingProvider = errors.New("encounter: provider id is required")

	// ErrMissingPatient is returned when OpenEncounterCmd omits the patient id.
	ErrMissingPatient = errors.New("encounter: patient id is required")

	// ErrMissingEncounter is returned when AppendAddendumCmd omits the encounter
	// id.
	ErrMissingEncounter = errors.New("encounter: encounter id is required")

	// ErrMissingAddendumText is returned when AppendAddendumCmd omits the
	// addendum text.
	ErrMissingAddendumText = errors.New("encounter: addendum text is required")

	// ErrMissingAuthor is returned when AppendAddendumCmd omits the author id.
	ErrMissingAuthor = errors.New("encounter: author id is required")

	// ErrNoSignedNoteToAmend is returned when AppendAddendumCmd targets an
	// encounter whose SOAP note is absent or unsigned. Invariant: a signed SOAP
	// note is immutable — corrections must be appended as addenda, not edits — so
	// there must be a signed note for an addendum to amend.
	ErrNoSignedNoteToAmend = errors.New("encounter: an addendum may only be appended to a signed note")

	// ErrParticipantNotScoped is returned when the command's provider or patient
	// is not one of the participants the encounter is scoped to. Invariant: only
	// participants scoped to the encounter may join the video room or view
	// in-call notes.
	ErrParticipantNotScoped = errors.New("encounter: only participants scoped to the encounter may join the video room or view in-call notes")

	// ErrSignedNoteImmutable is returned when opening would reopen an encounter
	// whose SOAP note is already signed. Invariant: a signed SOAP note is
	// immutable — corrections must be appended as addenda, not edits.
	ErrSignedNoteImmutable = errors.New("encounter: a signed SOAP note is immutable; corrections must be appended as addenda")

	// ErrDiagnosisUncoded is returned when the encounter carries a diagnosis that
	// does not reference a coded terminology entry. Invariant: a diagnosis must
	// reference a valid coded terminology entry (e.g., ICD-10).
	ErrDiagnosisUncoded = errors.New("encounter: a diagnosis must reference a valid coded terminology entry")

	// ErrIncompleteWithoutSignedNote is returned when the encounter is marked
	// complete but has no signed note. Invariant: an encounter cannot be marked
	// complete without a signed note.
	ErrIncompleteWithoutSignedNote = errors.New("encounter: an encounter cannot be marked complete without a signed note")
)
