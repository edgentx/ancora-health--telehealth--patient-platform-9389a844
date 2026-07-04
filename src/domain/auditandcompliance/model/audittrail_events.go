package model

import (
	"time"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"
)

// AuditEntryAppendedEventType is the stable type name emitted when an entry is
// sealed and appended to an audit trail.
const AuditEntryAppendedEventType = "audit.entry.appended"

// AuditEntryAppendedEvent is emitted when an AppendAuditEntryCmd succeeds. It
// carries the sealed entry so downstream projections can reconstruct the chain.
type AuditEntryAppendedEvent struct {
	// TrailID is the identity of the AuditTrailAggregate that produced the event.
	TrailID string
	// Sequence is the 1-based position of the entry within the chain.
	Sequence int
	// ActorContext, ResourceRef, Action and OccurredAt are the sealed payload.
	ActorContext string
	ResourceRef  string
	Action       string
	OccurredAt   time.Time
	// PrevHash is the hash of the entry immediately preceding this one.
	PrevHash string
	// EntryHash is this entry's own hash, becoming the new chain head.
	EntryHash string
}

// Type identifies the event kind.
func (e AuditEntryAppendedEvent) Type() string { return AuditEntryAppendedEventType }

// AggregateID ties the event back to the audit trail that produced it.
func (e AuditEntryAppendedEvent) AggregateID() string { return e.TrailID }

// Compile-time assertion that AuditEntryAppendedEvent satisfies the DomainEvent
// contract.
var _ shared.DomainEvent = AuditEntryAppendedEvent{}
