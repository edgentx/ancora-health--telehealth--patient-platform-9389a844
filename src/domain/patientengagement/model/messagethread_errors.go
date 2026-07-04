package model

import "errors"

var (
	// ErrMissingThread is returned when PostSecureMessageCmd omits the thread id.
	ErrMissingThread = errors.New("messagethread: thread id is required")

	// ErrMissingAuthor is returned when PostSecureMessageCmd omits the author id.
	ErrMissingAuthor = errors.New("messagethread: author id is required")

	// ErrMissingBody is returned when PostSecureMessageCmd omits the message body.
	ErrMissingBody = errors.New("messagethread: message body is required")

	// ErrNoActiveCareRelationship is returned when the thread's participants have
	// no active care relationship. Invariant: a thread cannot be created without an
	// active care relationship between participants.
	ErrNoActiveCareRelationship = errors.New("messagethread: a thread cannot be created without an active care relationship between participants")

	// ErrAuthorNotParticipant is returned when the author is not the patient or a
	// care-team participant on the thread. Invariant: only the patient and
	// care-team participants may post to or read the thread.
	ErrAuthorNotParticipant = errors.New("messagethread: only the patient and care-team participants may post to or read the thread")

	// ErrContentNotEncrypted is returned when the message content is not
	// PHI-encrypted. Invariant: message content must be PHI-encrypted at rest.
	ErrContentNotEncrypted = errors.New("messagethread: message content must be PHI-encrypted at rest")
)
