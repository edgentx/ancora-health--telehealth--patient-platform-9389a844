package model

// QueryDashboardCmd requests the filtered metrics for an analytics dashboard,
// capturing the clinic the rollup is scoped to, the date range it covers, and
// the metric type being retrieved.
//
// Querying a dashboard returns aggregated rollups only, never patient-level
// data: a rollup must be computed only from events within its declared
// clinic/date scope, a dashboard must never expose patient-identifiable PHI
// (only aggregates), and a rollup's totals must be reproducible from its source
// event window. ClinicId scopes the rollup to a clinic, DateRange bounds the
// event window it is computed over, and MetricType selects the metric being
// retrieved. All three are mandatory.
type QueryDashboardCmd struct {
	// ClinicId scopes the dashboard rollup to a single clinic.
	ClinicId string
	// DateRange bounds the event window the rollup is computed over.
	DateRange string
	// MetricType selects which aggregated metric is retrieved.
	MetricType string
}
