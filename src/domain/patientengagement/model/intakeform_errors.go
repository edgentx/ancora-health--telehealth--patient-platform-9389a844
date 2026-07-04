package model

import "errors"

var (
	// ErrMissingIntake is returned when SeedChartFromIntakeCmd omits the intake
	// id.
	ErrMissingIntake = errors.New("intake: intake id is required")

	// ErrMissingIntakePatient is returned when SeedChartFromIntakeCmd omits the
	// patient id.
	ErrMissingIntakePatient = errors.New("intake: patient id is required")

	// ErrIntakeNotValidated is returned when the intake form has not been
	// completed and validated. Invariant: an intake form must be complete and
	// validated before it can seed the chart.
	ErrIntakeNotValidated = errors.New("intake: an intake form must be complete and validated before it can seed the chart")

	// ErrIntakeImmutable is returned when seeding would re-consume an intake whose
	// data has already seeded the chart. Invariant: submitted intake data is
	// immutable and may only be corrected via a new submission.
	ErrIntakeImmutable = errors.New("intake: submitted intake data is immutable and may only be corrected via a new submission")
)
