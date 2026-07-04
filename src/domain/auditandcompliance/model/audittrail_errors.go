package model

import "errors"

var (
	// ErrIncompleteAuditEntry is returned when an AppendAuditEntryCmd is missing
	// any of the fields required before an entry can be sealed: actor identity,
	// resource, action, or timestamp.
	ErrIncompleteAuditEntry = errors.New("audit entry incomplete: actor, resource, action and timestamp are all required")

	// ErrAuditChainBroken is returned when the command's PrevHash does not
	// reference the current head of the chain, i.e. the entry would not form an
	// unbroken chain with its immediate predecessor.
	ErrAuditChainBroken = errors.New("audit chain broken: entry must reference the hash of the current head")

	// ErrAuditEntryImmutable is returned when appending would rewrite an entry
	// that has already been sealed into the chain. Persisted entries are
	// immutable and may never be updated or deleted.
	ErrAuditEntryImmutable = errors.New("audit entry immutable: a sealed entry may not be rewritten")
)
