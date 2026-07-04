// Package model holds the aggregates for the audit-and-compliance bounded
// context. AuditTrailAggregate is an immutable, append-only hash chain of audit
// entries; commands are dispatched through Execute.
package model

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"
)

// AuditEntry is a single sealed access event within the trail. Once appended an
// entry is immutable; its Hash chains it to the entry before it via PrevHash.
type AuditEntry struct {
	Sequence     int
	ActorContext string
	ResourceRef  string
	Action       string
	OccurredAt   time.Time
	PrevHash     string
	Hash         string
}

// AuditTrailAggregate is the aggregate root for an audit-and-compliance audit
// trail. It embeds shared.AggregateRoot for version tracking and an
// uncommitted-event buffer, carries its own identity in ID, and maintains the
// ordered chain of sealed entries.
type AuditTrailAggregate struct {
	shared.AggregateRoot
	ID string

	entries []AuditEntry
}

// Entries returns a copy-safe view of the sealed entries in chain order.
func (a *AuditTrailAggregate) Entries() []AuditEntry {
	return a.entries
}

// HeadHash returns the hash of the current chain head, or the empty string when
// the trail has no entries yet (the genesis reference).
func (a *AuditTrailAggregate) HeadHash() string {
	if len(a.entries) == 0 {
		return ""
	}
	return a.entries[len(a.entries)-1].Hash
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. Unrecognized commands yield shared.ErrUnknownCommand.
func (a *AuditTrailAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch c := cmd.(type) {
	case AppendAuditEntryCmd:
		return a.appendAuditEntry(c)
	case VerifyChainIntegrityCmd:
		return a.verifyChainIntegrity(c)
	default:
		return nil, shared.ErrUnknownCommand
	}
}

// verifyChainIntegrity re-derives the hash of every entry in the inclusive
// [FromSequence, ToSequence] window and compares it against the stored hash to
// detect tampering. It is read-only: it never mutates the chain or bumps the
// aggregate version.
//
// Structural invariants are enforced first, and a violation rejects the command
// with a domain error because the window is malformed and cannot be assessed:
//
//   - Completeness: each entry must carry actor identity, resource, action and
//     timestamp (ErrIncompleteAuditEntry).
//   - Unbroken chain: each entry must reference the hash of its immediate
//     predecessor, "" for the genesis entry (ErrAuditChainBroken).
//   - Immutability: a given sealed hash may appear at most once; a repeat means
//     an entry was rewritten in place (ErrAuditEntryImmutable).
//
// Only once the window is structurally sound does it assess content tampering:
// recomputing each entry's hash from its sealed payload and reporting the first
// divergence via a ChainTamperingDetectedEvent. An untampered window yields a
// ChainIntegrityVerifiedEvent.
func (a *AuditTrailAggregate) verifyChainIntegrity(cmd VerifyChainIntegrityCmd) ([]shared.DomainEvent, error) {
	if cmd.FromSequence < 1 || cmd.ToSequence < cmd.FromSequence || cmd.ToSequence > len(a.entries) {
		return nil, ErrInvalidSequenceRange
	}

	tamperedAt := 0
	seen := make(map[string]struct{}, cmd.ToSequence-cmd.FromSequence+1)
	for seq := cmd.FromSequence; seq <= cmd.ToSequence; seq++ {
		entry := a.entries[seq-1]

		if strings.TrimSpace(entry.ActorContext) == "" ||
			strings.TrimSpace(entry.ResourceRef) == "" ||
			strings.TrimSpace(entry.Action) == "" ||
			entry.OccurredAt.IsZero() {
			return nil, ErrIncompleteAuditEntry
		}

		// The predecessor of sequence 1 is the genesis reference ("").
		var predecessorHash string
		if seq > 1 {
			predecessorHash = a.entries[seq-2].Hash
		}
		if entry.PrevHash != predecessorHash {
			return nil, ErrAuditChainBroken
		}

		if _, dup := seen[entry.Hash]; dup {
			return nil, ErrAuditEntryImmutable
		}
		seen[entry.Hash] = struct{}{}

		recomputed := computeEntryHash(entry.PrevHash, entry.Sequence, entry.ActorContext, entry.ResourceRef, entry.Action, entry.OccurredAt)
		if recomputed != entry.Hash && tamperedAt == 0 {
			tamperedAt = seq
		}
	}

	if tamperedAt != 0 {
		evt := ChainTamperingDetectedEvent{
			TrailID:      a.ID,
			FromSequence: cmd.FromSequence,
			ToSequence:   cmd.ToSequence,
			TamperedAt:   tamperedAt,
		}
		a.AddEvent(evt)
		return []shared.DomainEvent{evt}, nil
	}

	evt := ChainIntegrityVerifiedEvent{
		TrailID:      a.ID,
		FromSequence: cmd.FromSequence,
		ToSequence:   cmd.ToSequence,
		HeadHash:     a.entries[cmd.ToSequence-1].Hash,
	}
	a.AddEvent(evt)
	return []shared.DomainEvent{evt}, nil
}

// appendAuditEntry seals a new access event and appends it to the chain,
// enforcing the audit-trail invariants:
//
//   - Completeness: actor identity, resource, action and timestamp must all be
//     present before an entry can be sealed.
//   - Unbroken chain: the command must reference the hash of the current head.
//   - Immutability: a sealed entry may never be rewritten.
func (a *AuditTrailAggregate) appendAuditEntry(cmd AppendAuditEntryCmd) ([]shared.DomainEvent, error) {
	if strings.TrimSpace(cmd.ActorContext) == "" ||
		strings.TrimSpace(cmd.ResourceRef) == "" ||
		strings.TrimSpace(cmd.Action) == "" ||
		cmd.OccurredAt.IsZero() {
		return nil, ErrIncompleteAuditEntry
	}

	if cmd.PrevHash != a.HeadHash() {
		return nil, ErrAuditChainBroken
	}

	seq := len(a.entries) + 1
	hash := computeEntryHash(cmd.PrevHash, seq, cmd.ActorContext, cmd.ResourceRef, cmd.Action, cmd.OccurredAt)

	if a.hasEntryHash(hash) {
		return nil, ErrAuditEntryImmutable
	}

	evt := AuditEntryAppendedEvent{
		TrailID:      a.ID,
		Sequence:     seq,
		ActorContext: cmd.ActorContext,
		ResourceRef:  cmd.ResourceRef,
		Action:       cmd.Action,
		OccurredAt:   cmd.OccurredAt,
		PrevHash:     cmd.PrevHash,
		EntryHash:    hash,
	}

	a.apply(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// apply mutates aggregate state from a domain event. It is the single place
// state changes, so the same function serves both command handling and future
// event replay when rehydrating the aggregate from the store.
func (a *AuditTrailAggregate) apply(evt AuditEntryAppendedEvent) {
	a.entries = append(a.entries, AuditEntry{
		Sequence:     evt.Sequence,
		ActorContext: evt.ActorContext,
		ResourceRef:  evt.ResourceRef,
		Action:       evt.Action,
		OccurredAt:   evt.OccurredAt,
		PrevHash:     evt.PrevHash,
		Hash:         evt.EntryHash,
	})
}

// hasEntryHash reports whether an entry with the given hash is already sealed
// into the chain.
func (a *AuditTrailAggregate) hasEntryHash(hash string) bool {
	for _, e := range a.entries {
		if e.Hash == hash {
			return true
		}
	}
	return false
}

// computeEntryHash derives an entry's tamper-evident hash from its predecessor's
// hash and its sealed payload, so any change to history breaks the chain.
func computeEntryHash(prevHash string, seq int, actor, resource, action string, occurredAt time.Time) string {
	h := sha256.New()
	// A field separator that cannot appear in the encoded fields keeps the
	// canonical form unambiguous.
	const sep = "\x1f"
	h.Write([]byte(prevHash + sep +
		strconv.Itoa(seq) + sep +
		actor + sep +
		resource + sep +
		action + sep +
		occurredAt.UTC().Format(time.RFC3339Nano)))
	return hex.EncodeToString(h.Sum(nil))
}
