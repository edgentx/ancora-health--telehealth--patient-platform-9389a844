package temporal

import (
	"context"
	"sync"
	"time"
)

// Notification is a single outbound patient/provider notification an activity
// emits. Notifications route through the realtime WebSocket gateway /
// patient-engagement service, so the shape carries only a recipient, a routing
// kind, and a human-readable subject/body — never PHI beyond what the recipient
// is already entitled to see.
//
// DedupeKey is what makes delivery idempotent on retry: an at-least-once
// activity may run more than once, so a Notifier keyed on DedupeKey collapses
// repeated deliveries of the same logical notification into one.
type Notification struct {
	// Kind is the routing/category discriminator, e.g. "appointment_reminder"
	// or "results_ready".
	Kind string
	// RecipientID is the user the notification is delivered to.
	RecipientID string
	// Subject is the short headline of the notification.
	Subject string
	// Body is the notification's human-readable body.
	Body string
	// DedupeKey uniquely identifies the logical notification. Two notifications
	// sharing a key are the same delivery and must not be sent twice.
	DedupeKey string
}

// Notifier is the outbound port a reminder/results-ready activity delivers
// through. Production wiring fans out via the realtime pub/sub broker
// (BrokerNotifier); tests use MemNotifier. Implementations must be idempotent
// on DedupeKey.
type Notifier interface {
	Notify(ctx context.Context, n Notification) error
}

// MemNotifier is an in-memory, idempotent Notifier for local development and
// tests. It records each unique notification exactly once, keyed on DedupeKey,
// mirroring the deduplicating delivery a production notifier guarantees so a
// retried activity never double-notifies.
type MemNotifier struct {
	mu       sync.Mutex
	seen     map[string]struct{}
	delivered []Notification
}

// Notify records the notification if its DedupeKey has not been seen. A repeat
// of an already-delivered key is a no-op that still reports success, which is
// exactly the idempotent-on-retry contract.
func (m *MemNotifier) Notify(_ context.Context, n Notification) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.seen == nil {
		m.seen = make(map[string]struct{})
	}
	key := n.DedupeKey
	if key == "" {
		key = n.Kind + ":" + n.RecipientID
	}
	if _, ok := m.seen[key]; ok {
		return nil
	}
	m.seen[key] = struct{}{}
	m.delivered = append(m.delivered, n)
	return nil
}

// Delivered returns a copy of the notifications delivered so far. It lets tests
// and local tooling assert exactly-once delivery.
func (m *MemNotifier) Delivered() []Notification {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Notification, len(m.delivered))
	copy(out, m.delivered)
	return out
}

// RollupMetrics is the computed analytics rollup the scheduled workflow persists
// for the AnalyticsDashboard: the operational metrics for one clinic over one
// reporting window. Utilization and no-show are fractions in [0,1]; revenue is
// in whole cents to match the billing model.
type RollupMetrics struct {
	// DashboardID is the analytics dashboard the rollup is written for.
	DashboardID string
	// ClinicID is the clinic the metrics are scoped to.
	ClinicID string
	// RangeStart and RangeEnd are the inclusive RFC-3339 date bounds of the
	// reporting window.
	RangeStart string
	RangeEnd   string
	// TotalSlots is the number of offered (non-cancelled) appointment slots the
	// utilization and no-show rates are computed over.
	TotalSlots int
	// UtilizationRate is the share of offered capacity that was claimed
	// (booked + completed + no-show) / TotalSlots.
	UtilizationRate float64
	// NoShowRate is no-shows / scheduled (booked + completed + no-show).
	NoShowRate float64
	// RevenueCents is the summed captured revenue over the window, in whole
	// cents.
	RevenueCents int64
	// ComputedAt is when the rollup was computed.
	ComputedAt time.Time
}

// RollupStore is the outbound port the rollup activity persists computed metrics
// to for the AnalyticsDashboard read model. Isolating it as a port keeps the
// rollup arithmetic testable against MemRollupStore while production writes to a
// dashboard-metrics collection.
type RollupStore interface {
	SaveRollup(ctx context.Context, m RollupMetrics) error
}

// MemRollupStore is an in-memory RollupStore for local development and tests. A
// rollup for the same dashboard + window overwrites the prior one, so a
// re-computed window is idempotent rather than accumulating duplicates.
type MemRollupStore struct {
	mu      sync.Mutex
	rollups map[string]RollupMetrics
}

// SaveRollup upserts the metrics keyed on dashboard + window.
func (s *MemRollupStore) SaveRollup(_ context.Context, m RollupMetrics) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.rollups == nil {
		s.rollups = make(map[string]RollupMetrics)
	}
	s.rollups[m.DashboardID+"|"+m.RangeStart+"|"+m.RangeEnd] = m
	return nil
}

// All returns the rollups persisted so far, for test assertions and local
// inspection.
func (s *MemRollupStore) All() []RollupMetrics {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]RollupMetrics, 0, len(s.rollups))
	for _, m := range s.rollups {
		out = append(out, m)
	}
	return out
}

// Compile-time assertions that the in-memory doubles satisfy their ports.
var (
	_ Notifier    = (*MemNotifier)(nil)
	_ RollupStore = (*MemRollupStore)(nil)
)
