package realtime

import (
	"context"
	"sync"
	"time"

	auditmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/auditandcompliance/model"
)

// AuditRecorder is the port the gateways use to record PHI-relevant events —
// connection lifecycle (session opened/closed) and PHI message access — to the
// tamper-evident audit trail. It is the seam that keeps the gateways decoupled
// from the audit aggregate's chaining mechanics.
type AuditRecorder interface {
	// Record seals an access event: who acted (actor), on what (resource), doing
	// what (action), at when. Implementations append it to the audit trail.
	Record(ctx context.Context, actor, resource, action string, at time.Time) error
}

// AuditTrailStore loads and persists the audit-trail aggregate the recorder
// appends to. Load returns (nil, nil) when the trail does not exist yet so the
// recorder can begin a fresh chain without coupling to a store-specific
// not-found sentinel; a genuine load failure returns a non-nil error.
type AuditTrailStore interface {
	Load(ctx context.Context, trailID string) (*auditmodel.AuditTrailAggregate, error)
	Save(ctx context.Context, trail *auditmodel.AuditTrailAggregate) error
}

// TrailAuditRecorder is the production AuditRecorder. It appends each event to a
// single named audit trail, extending the hash chain from its current head so
// the realtime access history is tamper-evident alongside the rest of the
// platform's audit trail.
type TrailAuditRecorder struct {
	store   AuditTrailStore
	trailID string

	// mu serializes append-then-save so concurrent gateway connections extend the
	// chain from a consistent head rather than racing on the same trail.
	mu sync.Mutex
}

// NewTrailAuditRecorder builds a recorder that appends to the given trail. An
// empty trailID falls back to a stable default so realtime events always land in
// a well-known trail.
func NewTrailAuditRecorder(store AuditTrailStore, trailID string) *TrailAuditRecorder {
	if trailID == "" {
		trailID = "realtime-access"
	}
	return &TrailAuditRecorder{store: store, trailID: trailID}
}

// Record loads the trail (or begins one), appends the sealed entry chained to
// the current head, and persists it.
func (r *TrailAuditRecorder) Record(ctx context.Context, actor, resource, action string, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	trail, err := r.store.Load(ctx, r.trailID)
	if err != nil {
		return err
	}
	if trail == nil {
		trail = &auditmodel.AuditTrailAggregate{ID: r.trailID}
	}

	if _, err := trail.Execute(auditmodel.AppendAuditEntryCmd{
		ActorContext: actor,
		ResourceRef:  resource,
		Action:       action,
		OccurredAt:   at,
		PrevHash:     trail.HeadHash(),
	}); err != nil {
		return err
	}
	return r.store.Save(ctx, trail)
}

// MemoryAuditTrailStore is an in-process AuditTrailStore for tests and local
// runs without a database. It keeps rehydratable snapshots of each trail's
// sealed entries keyed by id.
type MemoryAuditTrailStore struct {
	mu      sync.Mutex
	entries map[string][]auditmodel.AuditEntry
}

// NewMemoryAuditTrailStore builds an empty in-process trail store.
func NewMemoryAuditTrailStore() *MemoryAuditTrailStore {
	return &MemoryAuditTrailStore{entries: make(map[string][]auditmodel.AuditEntry)}
}

// Load rehydrates the trail from its stored entries, or returns (nil, nil) when
// it has never been saved.
func (s *MemoryAuditTrailStore) Load(_ context.Context, trailID string) (*auditmodel.AuditTrailAggregate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	stored, ok := s.entries[trailID]
	if !ok {
		return nil, nil
	}
	cp := make([]auditmodel.AuditEntry, len(stored))
	copy(cp, stored)
	return auditmodel.RehydrateAuditTrail(trailID, cp), nil
}

// Save snapshots the trail's sealed entries.
func (s *MemoryAuditTrailStore) Save(_ context.Context, trail *auditmodel.AuditTrailAggregate) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries := trail.Entries()
	cp := make([]auditmodel.AuditEntry, len(entries))
	copy(cp, entries)
	s.entries[trail.ID] = cp
	return nil
}

// Compile-time assertions.
var (
	_ AuditRecorder   = (*TrailAuditRecorder)(nil)
	_ AuditTrailStore = (*MemoryAuditTrailStore)(nil)
)
