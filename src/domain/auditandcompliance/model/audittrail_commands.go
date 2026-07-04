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
