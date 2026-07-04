// Package model holds the aggregates for the patient-engagement bounded
// context. MessageThreadAggregate is a secure patient/care-team messaging
// thread; StartMessageThreadCmd opens one.
package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// MessageThreadStatus is the lifecycle state of a message thread. The zero value
// is an unopened thread, which is what StartMessageThreadCmd acts on.
type MessageThreadStatus string

const (
	// MessageThreadStatusNew is a thread that has not yet been opened. It is the
	// zero value, so a freshly constructed aggregate is new.
	MessageThreadStatusNew MessageThreadStatus = ""
	// MessageThreadStatusOpen is a thread that has been started and is now open
	// for secure messaging between the patient and their care team.
	MessageThreadStatusOpen MessageThreadStatus = "open"
)

// MessageThreadAggregate is the aggregate root for a patient-engagement message
// thread. It embeds shared.AggregateRoot for version tracking and an
// uncommitted-event buffer, and carries its own identity in ID.
//
// Beyond identity it tracks the state that command invariants read: its
// lifecycle status, the patient and care-team participants scoped to it, the
// thread subject, and the flags describing whether access is confined to the
// participant set, whether message content is PHI-encrypted at rest, and whether
// the participants share an active care relationship.
//
// The invariant flags follow the repository convention that a freshly
// constructed aggregate is valid: their zero value is the compliant state, and a
// non-zero value marks a violation the guards reject.
type MessageThreadAggregate struct {
	shared.AggregateRoot
	ID string

	// Status is the thread's lifecycle state.
	Status MessageThreadStatus

	// ScopedPatientID and ScopedCareTeamMemberIDs are the participants the thread
	// is bound to. They are empty until the thread is started, at which point it
	// is scoped to the patient and their care-team participants.
	ScopedPatientID         string
	ScopedCareTeamMemberIDs []string

	// Subject is the thread topic. It is empty until the thread is started.
	Subject string

	// AccessNotRestricted reports that access to the thread is not confined to its
	// participant set. Invariant: only the patient and care-team participants may
	// post to or read the thread.
	AccessNotRestricted bool

	// ContentNotEncrypted reports that the thread's message content is not
	// PHI-encrypted at rest. Invariant: message content must be PHI-encrypted at
	// rest.
	ContentNotEncrypted bool

	// NoActiveCareRelationship reports that the patient and care-team participants
	// do not share an active care relationship. Invariant: a thread cannot be
	// created without an active care relationship between participants.
	NoActiveCareRelationship bool
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. Unrecognized command types return shared.ErrUnknownCommand.
func (a *MessageThreadAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch c := cmd.(type) {
	case StartMessageThreadCmd:
		return a.startMessageThread(c)
	default:
		return nil, shared.ErrUnknownCommand
	}
}

// startMessageThread handles StartMessageThreadCmd: it validates the command
// input, enforces the message-thread invariants, then emits a
// MessageThreadStartedEvent and buffers it on the aggregate.
//
// The guards enforce, in order:
//
//   - Completeness: patient, care-team participants and subject must all be
//     present.
//   - Participant access: only the patient and care-team participants may post to
//     or read the thread, so access must be confined to that set.
//   - Encryption: the thread's message content must be PHI-encrypted at rest.
//   - Care relationship: a thread cannot be created without an active care
//     relationship between the participants.
func (a *MessageThreadAggregate) startMessageThread(cmd StartMessageThreadCmd) ([]shared.DomainEvent, error) {
	if cmd.PatientId == "" {
		return nil, ErrMissingThreadPatient
	}
	if len(cmd.CareTeamMemberIds) == 0 {
		return nil, ErrMissingCareTeam
	}
	if cmd.Subject == "" {
		return nil, ErrMissingSubject
	}

	// Invariant: only the patient and care-team participants may post to or read
	// the thread.
	if a.AccessNotRestricted {
		return nil, ErrParticipantAccessNotRestricted
	}

	// Invariant: message content must be PHI-encrypted at rest.
	if a.ContentNotEncrypted {
		return nil, ErrContentNotEncrypted
	}

	// Invariant: a thread cannot be created without an active care relationship
	// between the participants.
	if a.NoActiveCareRelationship {
		return nil, ErrNoActiveCareRelationship
	}

	evt := MessageThreadStartedEvent{
		ThreadID:          a.ID,
		PatientID:         cmd.PatientId,
		CareTeamMemberIDs: cmd.CareTeamMemberIds,
		Subject:           cmd.Subject,
	}

	a.apply(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// apply mutates aggregate state from a domain event. It is the single place
// state changes, so the same function serves both command handling and future
// event replay when rehydrating the aggregate from the store.
func (a *MessageThreadAggregate) apply(evt MessageThreadStartedEvent) {
	a.Status = MessageThreadStatusOpen
	a.ScopedPatientID = evt.PatientID
	a.ScopedCareTeamMemberIDs = evt.CareTeamMemberIDs
	a.Subject = evt.Subject
}
