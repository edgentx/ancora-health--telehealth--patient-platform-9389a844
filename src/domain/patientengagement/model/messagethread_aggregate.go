// Package model holds the aggregates for the patient-engagement bounded
// context. MessageThreadAggregate is the aggregate for a secure patient/provider
// messaging thread and enforces its posting invariants via Execute(cmd).
package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// MessageThreadAggregate is the aggregate root for a patient-engagement message
// thread. It embeds shared.AggregateRoot for version tracking and an
// uncommitted-event buffer, and carries its own identity in ID.
//
// Beyond identity it tracks the state the posting invariants read: a running
// count of posted messages, the last author, and the invariant flags describing
// whether the author is a permitted participant, whether the content is
// PHI-encrypted, and whether the participants have an active care relationship.
//
// The invariant flags follow the repository convention that a freshly
// constructed aggregate is valid: their zero value is the compliant state, and a
// non-zero value marks a violation the guards reject.
type MessageThreadAggregate struct {
	shared.AggregateRoot
	ID string

	// MessageCount is the number of messages posted to the thread, and
	// LastAuthorID is the participant who authored the most recent one. Both are
	// updated as messages are posted.
	MessageCount int
	LastAuthorID string

	// AuthorNotParticipant reports that the posting author is not the patient or a
	// care-team participant on the thread. Invariant: only the patient and
	// care-team participants may post to or read the thread.
	AuthorNotParticipant bool

	// ContentNotEncrypted reports that the message content is not PHI-encrypted.
	// Invariant: message content must be PHI-encrypted at rest.
	ContentNotEncrypted bool

	// NoActiveCareRelationship reports that the thread's participants have no
	// active care relationship. Invariant: a thread cannot be created without an
	// active care relationship between participants.
	NoActiveCareRelationship bool
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. Unrecognized command types return shared.ErrUnknownCommand.
func (a *MessageThreadAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch c := cmd.(type) {
	case PostSecureMessageCmd:
		return a.postSecureMessage(c)
	default:
		return nil, shared.ErrUnknownCommand
	}
}

// postSecureMessage handles PostSecureMessageCmd: it validates the command
// input, enforces the thread invariants, then emits a MessageSecurePostedEvent
// and buffers it on the aggregate.
//
// The guards enforce, in order:
//
//   - Completeness: thread, author and body must all be present.
//   - Care relationship: a thread cannot exist without an active care
//     relationship between its participants, so it may not be posted to.
//   - Participation: only the patient and care-team participants may post to (or
//     read) the thread.
//   - Encryption: message content must be PHI-encrypted at rest.
func (a *MessageThreadAggregate) postSecureMessage(cmd PostSecureMessageCmd) ([]shared.DomainEvent, error) {
	if cmd.ThreadId == "" {
		return nil, ErrMissingThread
	}
	if cmd.AuthorId == "" {
		return nil, ErrMissingAuthor
	}
	if cmd.Body == "" {
		return nil, ErrMissingBody
	}

	// Invariant: a thread cannot be created without an active care relationship
	// between its participants, so no message may be posted to one that lacks it.
	if a.NoActiveCareRelationship {
		return nil, ErrNoActiveCareRelationship
	}

	// Invariant: only the patient and care-team participants may post to or read
	// the thread.
	if a.AuthorNotParticipant {
		return nil, ErrAuthorNotParticipant
	}

	// Invariant: message content must be PHI-encrypted at rest.
	if a.ContentNotEncrypted {
		return nil, ErrContentNotEncrypted
	}

	evt := MessageSecurePostedEvent{
		ThreadID: a.ID,
		AuthorID: cmd.AuthorId,
		Body:     cmd.Body,
	}

	a.apply(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// apply mutates aggregate state from a domain event. It is the single place
// state changes, so the same function serves both command handling and future
// event replay when rehydrating the aggregate from the store.
func (a *MessageThreadAggregate) apply(evt MessageSecurePostedEvent) {
	a.MessageCount++
	a.LastAuthorID = evt.AuthorID
}
