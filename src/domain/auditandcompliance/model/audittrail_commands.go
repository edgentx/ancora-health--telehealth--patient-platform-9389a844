package model

import "time"

// AppendAuditEntryCmd requests that a new access event be sealed and appended to
// the audit trail's hash chain.
//
// PrevHash is the hash of the immediately preceding entry the caller believes to
// be the current head of the chain. The aggregate rejects the command unless
// PrevHash matches its actual head, which is how the unbroken-chain invariant is
// enforced: a caller can only extend the chain from its current tip, never fork
// or rewrite it. For the very first entry PrevHash must be the empty string
// (the genesis reference).
//
// ActorContext, ResourceRef, Action and OccurredAt together form the sealed
// payload of the entry; all four are mandatory.
type AppendAuditEntryCmd struct {
	// ActorContext identifies who performed the action (e.g. user/service id).
	ActorContext string
	// ResourceRef identifies the resource the action was performed against.
	ResourceRef string
	// Action names the operation that occurred (e.g. "record.read").
	Action string
	// OccurredAt is the instant the action took place.
	OccurredAt time.Time
	// PrevHash references the hash of the current chain head ("" for genesis).
	PrevHash string
}

// VerifyChainIntegrityCmd requests a tamper check over a contiguous window of
// the trail. The aggregate recomputes each entry's hash from its sealed payload
// across the inclusive [FromSequence, ToSequence] range and compares it against
// the stored hash: any divergence means the entry's contents were altered after
// it was sealed.
//
// Both bounds are 1-based sequence numbers. FromSequence must be at least 1,
// ToSequence must be at least FromSequence, and ToSequence must not exceed the
// number of sealed entries; otherwise the command is rejected as an invalid
// range. The check is read-only — it never mutates the chain — and yields either
// a ChainIntegrityVerifiedEvent or, when tampering is found, a
// ChainTamperingDetectedEvent.
type VerifyChainIntegrityCmd struct {
	// FromSequence is the 1-based sequence number of the first entry to verify.
	FromSequence int
	// ToSequence is the 1-based sequence number of the last entry to verify.
	ToSequence int
}
