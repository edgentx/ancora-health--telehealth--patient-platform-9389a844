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
	default:
		return nil, shared.ErrUnknownCommand
	}
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
