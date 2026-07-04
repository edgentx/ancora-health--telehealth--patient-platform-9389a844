package model

import "errors"

var (
	// ErrMissingClinicID is returned when QueryDashboardCmd omits the clinic id
	// the rollup must be scoped to.
	ErrMissingClinicID = errors.New("analyticsdashboard: clinic id is required")

	// ErrMissingDateRange is returned when QueryDashboardCmd omits the date range
	// bounding the rollup's event window.
	ErrMissingDateRange = errors.New("analyticsdashboard: date range is required")

	// ErrMissingMetricType is returned when QueryDashboardCmd omits the metric
	// type being retrieved.
	ErrMissingMetricType = errors.New("analyticsdashboard: metric type is required")

	// ErrRollupOutOfScope is returned when a rollup would include events outside
	// its declared clinic/date scope. Invariant: a rollup must be computed only
	// from events within its declared clinic/date scope.
	ErrRollupOutOfScope = errors.New("analyticsdashboard: a rollup must be computed only from events within its declared clinic/date scope")

	// ErrDashboardExposesPHI is returned when a dashboard query would expose
	// patient-identifiable PHI. Invariant: dashboards must never expose
	// patient-identifiable PHI, only aggregates.
	ErrDashboardExposesPHI = errors.New("analyticsdashboard: dashboards must never expose patient-identifiable PHI, only aggregates")

	// ErrRollupNotReproducible is returned when a rollup's totals cannot be
	// reproduced from its source event window. Invariant: a rollup's totals must
	// be reproducible from its source event window.
	ErrRollupNotReproducible = errors.New("analyticsdashboard: a rollup's totals must be reproducible from its source event window")
)
