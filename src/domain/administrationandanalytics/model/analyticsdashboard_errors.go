package model

import "errors"

var (
	// ErrMissingClinic is returned when ComputeRollupCmd omits the clinic id.
	ErrMissingClinic = errors.New("analyticsdashboard: clinic id is required")

	// ErrMissingDateRange is returned when ComputeRollupCmd supplies an incomplete
	// date range (either end missing).
	ErrMissingDateRange = errors.New("analyticsdashboard: a complete date range is required")

	// ErrMissingMetricType is returned when ComputeRollupCmd omits the metric type.
	ErrMissingMetricType = errors.New("analyticsdashboard: metric type is required")

	// ErrRollupOutOfScope is returned when a rollup would draw on events outside
	// its declared clinic/date scope. Invariant: a rollup must be computed only
	// from events within its declared clinic/date scope.
	ErrRollupOutOfScope = errors.New("analyticsdashboard: a rollup must be computed only from events within its declared clinic/date scope")

	// ErrPHIExposed is returned when a rollup would expose patient-identifiable
	// PHI rather than an aggregate. Invariant: dashboards must never expose
	// patient-identifiable PHI, only aggregates.
	ErrPHIExposed = errors.New("analyticsdashboard: dashboards must never expose patient-identifiable PHI, only aggregates")

	// ErrRollupNotReproducible is returned when a rollup's totals cannot be
	// reproduced from its source event window. Invariant: a rollup's totals must
	// be reproducible from its source event window.
	ErrRollupNotReproducible = errors.New("analyticsdashboard: a rollup's totals must be reproducible from its source event window")
)
