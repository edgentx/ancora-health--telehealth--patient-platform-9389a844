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
	// that has already been sealed into the chain, or when a chain-integrity
	// check finds the same sealed hash twice. Persisted entries are immutable
	// and may never be updated or deleted.
	ErrAuditEntryImmutable = errors.New("audit entry immutable: a sealed entry may not be rewritten")

	// ErrInvalidSequenceRange is returned when a VerifyChainIntegrityCmd names a
	// window that is not a non-empty, in-bounds slice of the chain: FromSequence
	// must be >= 1, ToSequence must be >= FromSequence, and ToSequence must not
	// exceed the number of sealed entries.
	ErrInvalidSequenceRange = errors.New("invalid sequence range: must be 1-based, ordered, and within the sealed chain")
)
