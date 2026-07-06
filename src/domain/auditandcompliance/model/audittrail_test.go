package model

import (
	"errors"
	"testing"
	"time"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"
)

// fixedTime is a stable, non-zero timestamp for sealing audit entries.
var fixedTime = time.Date(2026, 7, 6, 9, 0, 0, 0, time.UTC)

// buildTrail appends n complete entries to a fresh trail, chaining each entry to
// the previous head, and returns the sealed aggregate. It fails the test if any
// append is rejected.
func buildTrail(t *testing.T, id string, n int) *AuditTrailAggregate {
	t.Helper()
	agg := &AuditTrailAggregate{ID: id}
	for i := 0; i < n; i++ {
		_, err := agg.Execute(AppendAuditEntryCmd{
			ActorContext: "user-1",
			ResourceRef:  "record-1",
			Action:       "record.read",
			OccurredAt:   fixedTime.Add(time.Duration(i) * time.Second),
			PrevHash:     agg.HeadHash(),
		})
		if err != nil {
			t.Fatalf("buildTrail append %d returned error: %v", i+1, err)
		}
	}
	return agg
}

func TestHeadHashEmptyTrail(t *testing.T) {
	agg := &AuditTrailAggregate{ID: "trail-1"}
	if got := agg.HeadHash(); got != "" {
		t.Fatalf("HeadHash on empty trail = %q, want empty", got)
	}
}

func TestExecuteAppendAuditEntryEmitsEvent(t *testing.T) {
	agg := &AuditTrailAggregate{ID: "trail-1"}

	events, err := agg.Execute(AppendAuditEntryCmd{
		ActorContext: "user-1",
		ResourceRef:  "record-1",
		Action:       "record.read",
		OccurredAt:   fixedTime,
		PrevHash:     "",
	})
	if err != nil {
		t.Fatalf("Execute(AppendAuditEntryCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	evt, ok := events[0].(AuditEntryAppendedEvent)
	if !ok {
		t.Fatalf("event type = %T, want AuditEntryAppendedEvent", events[0])
	}
	if evt.Type() != AuditEntryAppendedEventType || evt.Type() != "audit.entry.appended" {
		t.Fatalf("unexpected event type %q", evt.Type())
	}
	if evt.AggregateID() != "trail-1" {
		t.Fatalf("event aggregate id = %q, want trail-1", evt.AggregateID())
	}
	if evt.TrailID != "trail-1" {
		t.Fatalf("event trail id = %q, want trail-1", evt.TrailID)
	}
	if evt.Sequence != 1 {
		t.Fatalf("event sequence = %d, want 1", evt.Sequence)
	}
	if evt.ActorContext != "user-1" || evt.ResourceRef != "record-1" || evt.Action != "record.read" {
		t.Fatalf("event payload not copied from command: %+v", evt)
	}
	if !evt.OccurredAt.Equal(fixedTime) {
		t.Fatalf("event occurredAt = %v, want %v", evt.OccurredAt, fixedTime)
	}
	if evt.PrevHash != "" {
		t.Fatalf("event prevHash = %q, want empty (genesis)", evt.PrevHash)
	}
	if evt.EntryHash == "" {
		t.Fatalf("event entry hash is empty")
	}

	// Aggregate state was mutated via apply.
	if len(agg.Entries()) != 1 {
		t.Fatalf("expected 1 sealed entry, got %d", len(agg.Entries()))
	}
	if agg.HeadHash() != evt.EntryHash {
		t.Fatalf("head hash = %q, want %q", agg.HeadHash(), evt.EntryHash)
	}
	entry := agg.Entries()[0]
	if entry.Sequence != 1 || entry.ActorContext != "user-1" || entry.ResourceRef != "record-1" ||
		entry.Action != "record.read" || entry.PrevHash != "" || entry.Hash != evt.EntryHash {
		t.Fatalf("sealed entry not built from event: %+v", entry)
	}
	if agg.Version != 1 {
		t.Fatalf("version = %d, want 1", agg.Version)
	}
	if buffered := agg.Events(); len(buffered) != 1 {
		t.Fatalf("buffered %d events, want 1", len(buffered))
	}
}

func TestExecuteAppendAuditEntrySecondEntryChains(t *testing.T) {
	agg := buildTrail(t, "trail-1", 1)
	firstHead := agg.HeadHash()

	events, err := agg.Execute(AppendAuditEntryCmd{
		ActorContext: "user-2",
		ResourceRef:  "record-2",
		Action:       "record.update",
		OccurredAt:   fixedTime.Add(time.Minute),
		PrevHash:     firstHead,
	})
	if err != nil {
		t.Fatalf("Execute(AppendAuditEntryCmd) returned error: %v", err)
	}

	evt := events[0].(AuditEntryAppendedEvent)
	if evt.Sequence != 2 {
		t.Fatalf("event sequence = %d, want 2", evt.Sequence)
	}
	if evt.PrevHash != firstHead {
		t.Fatalf("event prevHash = %q, want %q", evt.PrevHash, firstHead)
	}
	if agg.Version != 2 {
		t.Fatalf("version = %d, want 2", agg.Version)
	}
	if len(agg.Entries()) != 2 {
		t.Fatalf("expected 2 sealed entries, got %d", len(agg.Entries()))
	}
}

func TestExecuteAppendAuditEntryRejectsIncompleteEntry(t *testing.T) {
	tests := []struct {
		name string
		cmd  AppendAuditEntryCmd
	}{
		{
			name: "missing actor",
			cmd:  AppendAuditEntryCmd{ResourceRef: "record-1", Action: "read", OccurredAt: fixedTime},
		},
		{
			name: "blank actor",
			cmd:  AppendAuditEntryCmd{ActorContext: "   ", ResourceRef: "record-1", Action: "read", OccurredAt: fixedTime},
		},
		{
			name: "missing resource",
			cmd:  AppendAuditEntryCmd{ActorContext: "user-1", Action: "read", OccurredAt: fixedTime},
		},
		{
			name: "missing action",
			cmd:  AppendAuditEntryCmd{ActorContext: "user-1", ResourceRef: "record-1", OccurredAt: fixedTime},
		},
		{
			name: "missing timestamp",
			cmd:  AppendAuditEntryCmd{ActorContext: "user-1", ResourceRef: "record-1", Action: "read"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := &AuditTrailAggregate{ID: "trail-1"}
			events, err := agg.Execute(tt.cmd)
			assertAuditRejected(t, agg, events, err, ErrIncompleteAuditEntry, 0)
		})
	}
}

func TestExecuteAppendAuditEntryRejectsBrokenChain(t *testing.T) {
	agg := buildTrail(t, "trail-1", 1)
	agg.ClearEvents()

	events, err := agg.Execute(AppendAuditEntryCmd{
		ActorContext: "user-1",
		ResourceRef:  "record-1",
		Action:       "record.read",
		OccurredAt:   fixedTime.Add(time.Minute),
		PrevHash:     "not-the-head",
	})
	assertAuditRejected(t, agg, events, err, ErrAuditChainBroken, 1)
}

func TestExecuteVerifyChainIntegrityVerifiesIntactWindow(t *testing.T) {
	agg := buildTrail(t, "trail-1", 3)
	agg.ClearEvents()

	events, err := agg.Execute(VerifyChainIntegrityCmd{FromSequence: 1, ToSequence: 3})
	if err != nil {
		t.Fatalf("Execute(VerifyChainIntegrityCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	evt, ok := events[0].(ChainIntegrityVerifiedEvent)
	if !ok {
		t.Fatalf("event type = %T, want ChainIntegrityVerifiedEvent", events[0])
	}
	if evt.Type() != ChainIntegrityVerifiedEventType || evt.Type() != "audit.chain.integrity.verified" {
		t.Fatalf("unexpected event type %q", evt.Type())
	}
	if evt.AggregateID() != "trail-1" || evt.TrailID != "trail-1" {
		t.Fatalf("event aggregate id = %q / trail id = %q, want trail-1", evt.AggregateID(), evt.TrailID)
	}
	if evt.FromSequence != 1 || evt.ToSequence != 3 {
		t.Fatalf("event window = [%d,%d], want [1,3]", evt.FromSequence, evt.ToSequence)
	}
	if evt.HeadHash != agg.HeadHash() {
		t.Fatalf("event head hash = %q, want %q", evt.HeadHash, agg.HeadHash())
	}
	// Read-only: version unchanged, only the verification event buffered.
	if agg.Version != 3 {
		t.Fatalf("version = %d, want 3 (unchanged)", agg.Version)
	}
	if buffered := agg.Events(); len(buffered) != 1 {
		t.Fatalf("buffered %d events, want 1", len(buffered))
	}
}

func TestExecuteVerifyChainIntegrityDetectsTampering(t *testing.T) {
	agg := buildTrail(t, "trail-1", 3)

	orig := agg.Entries()
	tampered := make([]AuditEntry, len(orig))
	copy(tampered, orig)
	// Rewrite the sealed payload of the middle entry while leaving its stored
	// Hash and the chain linkage (PrevHash values) intact, so the window stays
	// structurally sound but the recomputed hash of entry 2 diverges.
	tampered[1].Action = "record.exfiltrate"

	ta := RehydrateAuditTrail("trail-1", tampered)
	if ta.Version != 3 {
		t.Fatalf("rehydrated version = %d, want 3", ta.Version)
	}

	events, err := ta.Execute(VerifyChainIntegrityCmd{FromSequence: 1, ToSequence: 3})
	if err != nil {
		t.Fatalf("Execute(VerifyChainIntegrityCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	evt, ok := events[0].(ChainTamperingDetectedEvent)
	if !ok {
		t.Fatalf("event type = %T, want ChainTamperingDetectedEvent", events[0])
	}
	if evt.Type() != ChainTamperingDetectedEventType || evt.Type() != "audit.chain.tampering.detected" {
		t.Fatalf("unexpected event type %q", evt.Type())
	}
	if evt.AggregateID() != "trail-1" || evt.TrailID != "trail-1" {
		t.Fatalf("event aggregate id = %q / trail id = %q, want trail-1", evt.AggregateID(), evt.TrailID)
	}
	if evt.FromSequence != 1 || evt.ToSequence != 3 {
		t.Fatalf("event window = [%d,%d], want [1,3]", evt.FromSequence, evt.ToSequence)
	}
	if evt.TamperedAt != 2 {
		t.Fatalf("event tamperedAt = %d, want 2", evt.TamperedAt)
	}
	if buffered := ta.Events(); len(buffered) != 1 {
		t.Fatalf("buffered %d events, want 1", len(buffered))
	}
}

func TestExecuteVerifyChainIntegrityRejectsInvalidRange(t *testing.T) {
	tests := []struct {
		name string
		cmd  VerifyChainIntegrityCmd
	}{
		{name: "from below one", cmd: VerifyChainIntegrityCmd{FromSequence: 0, ToSequence: 1}},
		{name: "to before from", cmd: VerifyChainIntegrityCmd{FromSequence: 2, ToSequence: 1}},
		{name: "to beyond chain", cmd: VerifyChainIntegrityCmd{FromSequence: 1, ToSequence: 99}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := buildTrail(t, "trail-1", 3)
			agg.ClearEvents()
			events, err := agg.Execute(tt.cmd)
			if !errors.Is(err, ErrInvalidSequenceRange) {
				t.Fatalf("error = %v, want %v", err, ErrInvalidSequenceRange)
			}
			if len(events) != 0 {
				t.Fatalf("expected no events, got %d", len(events))
			}
			if len(agg.Events()) != 0 {
				t.Fatalf("expected no buffered events, got %d", len(agg.Events()))
			}
			if agg.Version != 3 {
				t.Fatalf("version = %d, want 3 (unchanged)", agg.Version)
			}
		})
	}
}

func TestExecuteVerifyChainIntegrityRejectsIncompleteEntry(t *testing.T) {
	entries := []AuditEntry{
		{Sequence: 1, ActorContext: "", ResourceRef: "record-1", Action: "read", OccurredAt: fixedTime, PrevHash: "", Hash: "h1"},
	}
	agg := RehydrateAuditTrail("trail-1", entries)

	events, err := agg.Execute(VerifyChainIntegrityCmd{FromSequence: 1, ToSequence: 1})
	if !errors.Is(err, ErrIncompleteAuditEntry) {
		t.Fatalf("error = %v, want %v", err, ErrIncompleteAuditEntry)
	}
	if len(events) != 0 || len(agg.Events()) != 0 {
		t.Fatalf("expected no events on rejection")
	}
}

func TestExecuteVerifyChainIntegrityRejectsBrokenChain(t *testing.T) {
	entries := []AuditEntry{
		{Sequence: 1, ActorContext: "user-1", ResourceRef: "record-1", Action: "read", OccurredAt: fixedTime, PrevHash: "", Hash: "h1"},
		{Sequence: 2, ActorContext: "user-1", ResourceRef: "record-1", Action: "read", OccurredAt: fixedTime, PrevHash: "wrong-prev", Hash: "h2"},
	}
	agg := RehydrateAuditTrail("trail-1", entries)

	events, err := agg.Execute(VerifyChainIntegrityCmd{FromSequence: 1, ToSequence: 2})
	if !errors.Is(err, ErrAuditChainBroken) {
		t.Fatalf("error = %v, want %v", err, ErrAuditChainBroken)
	}
	if len(events) != 0 || len(agg.Events()) != 0 {
		t.Fatalf("expected no events on rejection")
	}
}

func TestExecuteVerifyChainIntegrityRejectsDuplicateHash(t *testing.T) {
	entries := []AuditEntry{
		{Sequence: 1, ActorContext: "user-1", ResourceRef: "record-1", Action: "read", OccurredAt: fixedTime, PrevHash: "", Hash: "dup"},
		{Sequence: 2, ActorContext: "user-1", ResourceRef: "record-1", Action: "read", OccurredAt: fixedTime, PrevHash: "dup", Hash: "dup"},
	}
	agg := RehydrateAuditTrail("trail-1", entries)

	events, err := agg.Execute(VerifyChainIntegrityCmd{FromSequence: 1, ToSequence: 2})
	if !errors.Is(err, ErrAuditEntryImmutable) {
		t.Fatalf("error = %v, want %v", err, ErrAuditEntryImmutable)
	}
	if len(events) != 0 || len(agg.Events()) != 0 {
		t.Fatalf("expected no events on rejection")
	}
}

func TestAuditTrailExecuteUnknownCommand(t *testing.T) {
	agg := &AuditTrailAggregate{ID: "trail-1"}

	type bogusCmd struct{}
	events, err := agg.Execute(bogusCmd{})
	if !errors.Is(err, shared.ErrUnknownCommand) {
		t.Fatalf("error = %v, want %v", err, shared.ErrUnknownCommand)
	}
	if events != nil {
		t.Fatalf("expected nil events, got %v", events)
	}
	if agg.Version != 0 {
		t.Fatalf("version = %d, want 0", agg.Version)
	}
	if len(agg.Events()) != 0 {
		t.Fatalf("expected no buffered events, got %d", len(agg.Events()))
	}
}

func TestAuditTrailRootHelpers(t *testing.T) {
	agg := buildTrail(t, "trail-1", 1)

	if agg.GetVersion() != 1 {
		t.Fatalf("GetVersion = %d, want 1", agg.GetVersion())
	}
	if len(agg.Events()) != 1 {
		t.Fatalf("expected 1 buffered event, got %d", len(agg.Events()))
	}

	agg.ClearEvents()
	if len(agg.Events()) != 0 {
		t.Fatalf("expected events cleared, got %d", len(agg.Events()))
	}
	if agg.GetVersion() != 1 {
		t.Fatalf("version changed after ClearEvents: %d", agg.GetVersion())
	}
}

// assertAuditRejected checks that an append command produced the expected
// sentinel error, emitted no events, left the buffer at wantBuffered and did not
// advance the version.
func assertAuditRejected(t *testing.T, agg *AuditTrailAggregate, events []shared.DomainEvent, err, wantErr error, wantVersion int) {
	t.Helper()
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
	if len(events) != 0 {
		t.Fatalf("expected no events on rejection, got %d", len(events))
	}
	if len(agg.Events()) != 0 {
		t.Fatalf("expected no buffered events on rejection, got %d", len(agg.Events()))
	}
	if agg.Version != wantVersion {
		t.Fatalf("version = %d, want %d", agg.Version, wantVersion)
	}
}
