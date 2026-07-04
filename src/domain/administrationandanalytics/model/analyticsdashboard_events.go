package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// DashboardQueriedEventType is the stable wire name emitted when an analytics
// dashboard is successfully queried.
const DashboardQueriedEventType = "analytics.dashboard.queried"

// DashboardQueriedEvent is emitted when a QueryDashboardCmd succeeds. It records
// the clinic scope, date range, and metric type of the rollup that was
// retrieved. It carries only the aggregate query descriptor, never
// patient-identifiable data.
type DashboardQueriedEvent struct {
	// DashboardID is the identity of the AnalyticsDashboardAggregate that produced
	// the event.
	DashboardID string
	// ClinicID is the clinic the queried rollup is scoped to.
	ClinicID string
	// DateRange is the event window the rollup was computed over.
	DateRange string
	// MetricType is the aggregated metric that was retrieved.
	MetricType string
}

// Type identifies the event kind.
func (e DashboardQueriedEvent) Type() string { return DashboardQueriedEventType }

// AggregateID ties the event back to the dashboard that produced it.
func (e DashboardQueriedEvent) AggregateID() string { return e.DashboardID }

// Compile-time assertion that DashboardQueriedEvent satisfies the DomainEvent
// contract.
var _ shared.DomainEvent = DashboardQueriedEvent{}
