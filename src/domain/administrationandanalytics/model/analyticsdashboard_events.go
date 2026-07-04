package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// RollupComputedEventType is the stable wire name emitted when an analytics
// rollup is computed for a scope.
const RollupComputedEventType = "analytics.rollup.computed"

// RollupComputedEvent is emitted when a ComputeRollupCmd succeeds. It records the
// clinic the rollup was scoped to, the date window it covered, and the metric
// that was aggregated.
type RollupComputedEvent struct {
	// DashboardID is the identity of the AnalyticsDashboardAggregate that produced
	// the event.
	DashboardID string
	// ClinicID is the clinic the rollup was scoped to.
	ClinicID string
	// RangeStart is the inclusive first day of the rollup window.
	RangeStart string
	// RangeEnd is the inclusive last day of the rollup window.
	RangeEnd string
	// MetricType is the metric that was aggregated.
	MetricType string
}

// Type identifies the event kind.
func (e RollupComputedEvent) Type() string { return RollupComputedEventType }

// AggregateID ties the event back to the dashboard that produced it.
func (e RollupComputedEvent) AggregateID() string { return e.DashboardID }

// Compile-time assertion that RollupComputedEvent satisfies the DomainEvent
// contract.
var _ shared.DomainEvent = RollupComputedEvent{}
