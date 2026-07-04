package model

import "errors"

var (
	// ErrMissingIntakePatient is returned when SubmitIntakeFormCmd or
	// SeedChartFromIntakeCmd omits the patient id.
	ErrMissingIntakePatient = errors.New("intakeform: patient id is required")

	// ErrMissingIntakeID is returned when SeedChartFromIntakeCmd omits the intake
	// id identifying the intake to project into the chart.
	ErrMissingIntakeID = errors.New("intakeform: intake id is required")

	// ErrMissingIntakeHistory is returned when SubmitIntakeFormCmd omits the
	// clinical history.
	ErrMissingIntakeHistory = errors.New("intakeform: history is required")

	// ErrMissingIntakeDemographics is returned when SubmitIntakeFormCmd omits the
	// demographics.
	ErrMissingIntakeDemographics = errors.New("intakeform: demographics is required")

	// ErrIntakeFormIncomplete is returned when the intake form has not been
	// completed and validated. Invariant: an intake form must be complete and
	// validated before it can seed the chart.
	ErrIntakeFormIncomplete = errors.New("intakeform: an intake form must be complete and validated before it can seed the chart")

	// ErrIntakeFormSubmittedImmutable is returned when a command would alter intake
	// data that has already been sealed — re-submitting a submitted form or
	// re-seeding an already-seeded chart. Invariant: submitted intake data is
	// immutable and may only be corrected via a new submission.
	ErrIntakeFormSubmittedImmutable = errors.New("intakeform: submitted intake data is immutable and may only be corrected via a new submission")
)
