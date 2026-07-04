package model

import (
	"errors"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"
)

// Domain errors raised by VerifyChainIntegrityCmd when the aggregate is not in a
// verifiable state. Each corresponds to a structural invariant that must hold
// before the hash chain can be recomputed.
var (
	// ErrBrokenChain means an entry that has a predecessor is missing the
	// PrevHash that should reference it — the chain was never fully linked.
	ErrBrokenChain = errors.New("audittrail: entry does not reference the preceding entry's hash")

	// ErrEntryMutated means a persisted entry was updated or deleted in place,
	// violating the immutability invariant.
	ErrEntryMutated = errors.New("audittrail: entry was mutated after being sealed")

	// ErrEntryNotSealed means an entry is missing actor identity, resource,
	// action, or timestamp and therefore was never validly sealed.
	ErrEntryNotSealed = errors.New("audittrail: entry is missing actor, resource, action, or timestamp")

	// ErrInvalidRange means the requested sequence bounds are inconsistent
	// (fromSequence greater than toSequence, or negative).
	ErrInvalidRange = errors.New("audittrail: invalid sequence range")

	// ErrRangeEmpty means the requested range contains no entries to verify.
	ErrRangeEmpty = errors.New("audittrail: sequence range contains no entries")
)

// Event type identifiers emitted by VerifyChainIntegrityCmd.
const (
	EventChainIntegrityVerified = "audit.chain.integrity.verified"
	EventChainTamperingDetected = "audit.chain.tampering.detected"
)

// VerifyChainIntegrityCmd asks the aggregate to recompute the hashes of every
// entry whose sequence falls within [FromSequence, ToSequence] and confirm that
// the chain is intact. Any entry whose recomputed hash or PrevHash link no
// longer matches what was stored is reported as tampering.
type VerifyChainIntegrityCmd struct {
	FromSequence int
	ToSequence   int
}

// ChainIntegrityVerified is emitted once per verification run, summarizing the
// range that was checked and how many entries were found to be tampered with.
type ChainIntegrityVerified struct {
	AuditTrailID   string
	FromSequence   int
	ToSequence     int
	EntriesChecked int
	TamperedCount  int
}

// Type identifies the event kind.
func (e ChainIntegrityVerified) Type() string { return EventChainIntegrityVerified }

// AggregateID ties the event back to the aggregate that produced it.
func (e ChainIntegrityVerified) AggregateID() string { return e.AuditTrailID }

// ChainTamperingDetected is emitted for each entry whose stored hash no longer
// matches its recomputed value, pinpointing where the chain was broken.
type ChainTamperingDetected struct {
	AuditTrailID string
	Sequence     int
	ExpectedHash string
	ActualHash   string
}

// Type identifies the event kind.
func (e ChainTamperingDetected) Type() string { return EventChainTamperingDetected }

// AggregateID ties the event back to the aggregate that produced it.
func (e ChainTamperingDetected) AggregateID() string { return e.AuditTrailID }

// verifyChainIntegrity handles VerifyChainIntegrityCmd. It first enforces the
// structural invariants that must hold before verification is meaningful
// (rejecting with a domain error if any is violated), then recomputes the hash
// chain over the requested range and emits the resulting events.
func (a *AuditTrailAggregate) verifyChainIntegrity(cmd VerifyChainIntegrityCmd) ([]shared.DomainEvent, error) {
	if cmd.FromSequence < 0 || cmd.ToSequence < cmd.FromSequence {
		return nil, ErrInvalidRange
	}

	// Index every entry by sequence so a range entry can resolve the hash of
	// its immediate predecessor even when that predecessor lies before
	// FromSequence.
	bySeq := make(map[int]AuditTrailEntry, len(a.Entries))
	for _, entry := range a.Entries {
		bySeq[entry.Sequence] = entry
	}

	inRange := make([]AuditTrailEntry, 0, len(a.Entries))
	for _, entry := range a.Entries {
		if entry.Sequence >= cmd.FromSequence && entry.Sequence <= cmd.ToSequence {
			inRange = append(inRange, entry)
		}
	}
	if len(inRange) == 0 {
		return nil, ErrRangeEmpty
	}

	// Precondition invariants, checked in acceptance-criteria order. A violation
	// means the aggregate cannot be verified, so the command is rejected.
	for _, entry := range inRange {
		_, hasPredecessor := bySeq[entry.Sequence-1]

		// Every appended entry must reference the hash of the immediately
		// preceding entry, forming an unbroken chain.
		if hasPredecessor && entry.PrevHash == "" {
			return nil, ErrBrokenChain
		}

		// Entries are immutable — no update or delete may ever be executed
		// against a persisted entry.
		if entry.Mutated {
			return nil, ErrEntryMutated
		}

		// Each entry must carry actor identity, resource, action, and timestamp
		// before it can be sealed.
		if !entry.hasIdentity() {
			return nil, ErrEntryNotSealed
		}
	}

	// Recompute the chain and emit events. A tampering event is raised for each
	// entry whose recomputed content hash or PrevHash link no longer matches
	// what was stored; a single verification event summarizes the run.
	events := make([]shared.DomainEvent, 0, len(inRange)+1)
	tampered := 0
	for _, entry := range inRange {
		expectedPrevHash := ""
		if predecessor, ok := bySeq[entry.Sequence-1]; ok {
			expectedPrevHash = predecessor.Hash
		}

		recomputed := entry.ComputeHash(expectedPrevHash)
		if recomputed != entry.Hash || entry.PrevHash != expectedPrevHash {
			tampered++
			events = append(events, ChainTamperingDetected{
				AuditTrailID: a.ID,
				Sequence:     entry.Sequence,
				ExpectedHash: recomputed,
				ActualHash:   entry.Hash,
			})
		}
	}

	events = append(events, ChainIntegrityVerified{
		AuditTrailID:   a.ID,
		FromSequence:   cmd.FromSequence,
		ToSequence:     cmd.ToSequence,
		EntriesChecked: len(inRange),
		TamperedCount:  tampered,
	})

	for _, event := range events {
		a.AddEvent(event)
	}
	return events, nil
}
