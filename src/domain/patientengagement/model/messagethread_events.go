package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// MessageThreadStartedEventType is the stable wire name emitted when a secure
// message thread is opened between a patient and their care team.
const MessageThreadStartedEventType = "message.thread.started"

// MessageThreadStartedEvent is emitted when a StartMessageThreadCmd succeeds. It
// records the patient and care-team participants the thread is scoped to, along
// with the subject the thread was opened to discuss. The participant set it
// carries is the set permitted to post to or read the thread thereafter.
type MessageThreadStartedEvent struct {
	// ThreadID is the identity of the MessageThreadAggregate that produced the
	// event.
	ThreadID string
	// PatientID is the patient the thread was opened for.
	PatientID string
	// CareTeamMemberIDs are the care-team participants the thread is scoped to.
	CareTeamMemberIDs []string
	// Subject is the topic the thread was opened to discuss.
	Subject string
}

// Type identifies the event kind.
func (e MessageThreadStartedEvent) Type() string { return MessageThreadStartedEventType }

// AggregateID ties the event back to the thread that produced it.
func (e MessageThreadStartedEvent) AggregateID() string { return e.ThreadID }

// Compile-time assertion that MessageThreadStartedEvent satisfies the
// DomainEvent contract.
var _ shared.DomainEvent = MessageThreadStartedEvent{}
