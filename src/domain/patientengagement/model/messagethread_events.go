package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// MessageSecurePostedEventType is the stable wire name emitted when an encrypted
// message is posted to a secure thread.
const MessageSecurePostedEventType = "message.secure.posted"

// MessageSecurePostedEvent is emitted when a PostSecureMessageCmd succeeds. It
// records the thread the message was posted to, the participant who authored it,
// and the (PHI-encrypted) message body.
type MessageSecurePostedEvent struct {
	// ThreadID is the identity of the MessageThreadAggregate that produced the
	// event.
	ThreadID string
	// AuthorID is the participant who posted the message.
	AuthorID string
	// Body is the PHI-encrypted message content that was posted.
	Body string
}

// Type identifies the event kind.
func (e MessageSecurePostedEvent) Type() string { return MessageSecurePostedEventType }

// AggregateID ties the event back to the thread that produced it.
func (e MessageSecurePostedEvent) AggregateID() string { return e.ThreadID }

// Compile-time assertion that MessageSecurePostedEvent satisfies the DomainEvent
// contract.
var _ shared.DomainEvent = MessageSecurePostedEvent{}
