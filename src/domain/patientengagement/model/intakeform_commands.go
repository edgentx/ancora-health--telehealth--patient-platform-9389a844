package model

// SeedChartFromIntakeCmd requests that a completed, validated IntakeForm be
// projected into the patient's clinical chart.
//
// Seeding is the projecting act: it copies the validated intake submission into
// the chart so downstream care can rely on it. Two invariants gate it — the
// intake form must be complete and validated before it can seed the chart, and a
// submitted intake is immutable, so once its data has seeded the chart it may
// only be corrected via a new submission rather than re-seeded. IntakeId
// identifies the form being projected and PatientId the patient whose chart is
// seeded; both are mandatory.
type SeedChartFromIntakeCmd struct {
	// IntakeId identifies the intake form being projected into the chart.
	IntakeId string
	// PatientId identifies the patient whose chart is seeded from the intake.
	PatientId string
}
