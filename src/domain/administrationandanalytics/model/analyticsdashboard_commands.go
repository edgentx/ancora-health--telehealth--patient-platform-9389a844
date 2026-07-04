package model

// DateRange is the closed date window a rollup is computed over. Both ends are
// required; the aggregate treats the range as the declared scope a rollup's
// source events must fall within.
type DateRange struct {
	// Start is the inclusive first day of the rollup window.
	Start string
	// End is the inclusive last day of the rollup window.
	End string
}

// ComputeRollupCmd requests that an analytics rollup be computed for a scope,
// capturing the clinic the rollup is scoped to, the date window it covers, and
// the metric being aggregated.
//
// Computing a rollup reduces a window of source events into an aggregate figure
// for a dashboard. It carries three invariants: a rollup must be computed only
// from events within its declared clinic/date scope, dashboards must never
// expose patient-identifiable PHI (only aggregates), and a rollup's totals must
// be reproducible from its source event window. ClinicId identifies the clinic
// the rollup is scoped to, DateRange the window it covers, and MetricType the
// metric being aggregated. All three are mandatory.
type ComputeRollupCmd struct {
	// ClinicId identifies the clinic the rollup is scoped to.
	ClinicId string
	// DateRange is the date window the rollup is computed over.
	DateRange DateRange
	// MetricType is the metric being aggregated (for example a visit count or a
	// revenue total).
	MetricType string
}
