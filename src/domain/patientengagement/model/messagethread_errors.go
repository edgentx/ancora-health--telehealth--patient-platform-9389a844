package model

import "errors"

var (
	// ErrMissingThreadPatient is returned when StartMessageThreadCmd omits the
	// patient id.
	ErrMissingThreadPatient = errors.New("messagethread: patient id is required")

	// ErrMissingCareTeam is returned when StartMessageThreadCmd omits the
	// care-team member ids.
	ErrMissingCareTeam = errors.New("messagethread: at least one care-team member id is required")

	// ErrMissingSubject is returned when StartMessageThreadCmd omits the subject.
	ErrMissingSubject = errors.New("messagethread: subject is required")

	// ErrParticipantAccessNotRestricted is returned when the thread's access is
	// not confined to its participant set. Invariant: only the patient and
	// care-team participants may post to or read the thread.
	ErrParticipantAccessNotRestricted = errors.New("messagethread: only the patient and care-team participants may post to or read the thread")

	// ErrContentNotEncrypted is returned when the thread's message content is not
	// PHI-encrypted at rest. Invariant: message content must be PHI-encrypted at
	// rest.
	ErrContentNotEncrypted = errors.New("messagethread: message content must be PHI-encrypted at rest")

	// ErrNoActiveCareRelationship is returned when the patient and care-team
	// participants do not share an active care relationship. Invariant: a thread
	// cannot be created without an active care relationship between participants.
	ErrNoActiveCareRelationship = errors.New("messagethread: a thread cannot be created without an active care relationship between participants")
)
