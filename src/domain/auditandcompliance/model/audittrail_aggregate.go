// Package model holds the aggregates for the audit-and-compliance bounded
// context. AuditTrailAggregate is an immutable, hash-chained audit trail:
// every entry references the hash of the entry immediately preceding it, so any
// retroactive edit breaks the chain and is detectable.
package model

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"
)

// AuditTrailEntry is a single sealed record in the audit trail. Entries are
// immutable once appended; VerifyChainIntegrityCmd recomputes their hashes to
// detect tampering. PrevHash links each entry to the one before it, forming an
// unbroken chain from the genesis entry (which carries an empty PrevHash).
type AuditTrailEntry struct {
	Sequence  int
	ActorID   string
	Resource  string
	Action    string
	Timestamp time.Time
	PrevHash  string
	Hash      string

	// Mutated is a defensive marker used to model an immutability violation:
	// a persisted entry that was updated or deleted in place. A well-formed
	// trail never sets it; verification rejects any range that contains one.
	Mutated bool
}

// hasIdentity reports whether the entry carries actor identity, resource,
// action, and timestamp — the fields required before an entry may be sealed.
func (e AuditTrailEntry) hasIdentity() bool {
	return e.ActorID != "" && e.Resource != "" && e.Action != "" && !e.Timestamp.IsZero()
}

// ComputeHash derives the canonical SHA-256 hash of the entry's sealed content
// chained onto prevHash. Verification recomputes this and compares it against
// the stored Hash: a mismatch means the content was altered after sealing.
func (e AuditTrailEntry) ComputeHash(prevHash string) string {
	canonical := strings.Join([]string{
		strconv.Itoa(e.Sequence),
		e.ActorID,
		e.Resource,
		e.Action,
		e.Timestamp.UTC().Format(time.RFC3339Nano),
		prevHash,
	}, "|")
	sum := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(sum[:])
}

// AuditTrailAggregate is the aggregate root for an audit-and-compliance audit
// trail. It embeds shared.AggregateRoot for version tracking and an
// uncommitted-event buffer, carries its own identity in ID, and holds the
// ordered, hash-chained Entries the trail has sealed.
type AuditTrailAggregate struct {
	shared.AggregateRoot
	ID      string
	Entries []AuditTrailEntry
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. Recognized commands are dispatched to their handler; anything else
// returns shared.ErrUnknownCommand.
func (a *AuditTrailAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch c := cmd.(type) {
	case VerifyChainIntegrityCmd:
		return a.verifyChainIntegrity(c)
	default:
		return nil, shared.ErrUnknownCommand
	}
}
