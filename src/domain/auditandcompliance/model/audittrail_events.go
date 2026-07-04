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

// ChainIntegrityVerifiedEventType is the stable type name emitted when a range
// of the audit trail passes a tamper check with every hash intact.
const ChainIntegrityVerifiedEventType = "audit.chain.integrity.verified"

// ChainIntegrityVerifiedEvent is emitted when a VerifyChainIntegrityCmd finds
// the requested range structurally sound and free of tampering. It records the
// window that was checked and the head hash at the end of that window so
// downstream auditors can attest to exactly what was verified.
type ChainIntegrityVerifiedEvent struct {
	// TrailID is the identity of the AuditTrailAggregate that was verified.
	TrailID string
	// FromSequence and ToSequence bound the inclusive range that was checked.
	FromSequence int
	ToSequence   int
	// HeadHash is the stored hash of the entry at ToSequence, the tip of the
	// verified window.
	HeadHash string
}

// Type identifies the event kind.
func (e ChainIntegrityVerifiedEvent) Type() string { return ChainIntegrityVerifiedEventType }

// AggregateID ties the event back to the audit trail that produced it.
func (e ChainIntegrityVerifiedEvent) AggregateID() string { return e.TrailID }

// ChainTamperingDetectedEventType is the stable type name emitted when a tamper
// check finds an entry whose recomputed hash no longer matches its stored hash.
const ChainTamperingDetectedEventType = "audit.chain.tampering.detected"

// ChainTamperingDetectedEvent is emitted when a VerifyChainIntegrityCmd detects
// that an entry's sealed contents were altered after the fact. TamperedAt marks
// the earliest sequence in the checked window whose recomputed hash diverged, so
// investigators can pinpoint where history was rewritten.
type ChainTamperingDetectedEvent struct {
	// TrailID is the identity of the AuditTrailAggregate that was verified.
	TrailID string
	// FromSequence and ToSequence bound the inclusive range that was checked.
	FromSequence int
	ToSequence   int
	// TamperedAt is the 1-based sequence of the first entry found to be tampered.
	TamperedAt int
}

// Type identifies the event kind.
func (e ChainTamperingDetectedEvent) Type() string { return ChainTamperingDetectedEventType }

// AggregateID ties the event back to the audit trail that produced it.
func (e ChainTamperingDetectedEvent) AggregateID() string { return e.TrailID }

// Compile-time assertions that the chain-integrity events satisfy DomainEvent.
var (
	_ shared.DomainEvent = ChainIntegrityVerifiedEvent{}
	_ shared.DomainEvent = ChainTamperingDetectedEvent{}
)
