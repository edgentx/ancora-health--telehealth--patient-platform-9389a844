package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// IntakeChartSeededEventType is the stable wire name emitted when a validated
// intake form is projected into the patient's chart.
const IntakeChartSeededEventType = "intake.chart.seeded"

// IntakeChartSeededEvent is emitted when a SeedChartFromIntakeCmd succeeds. It
// records the patient whose chart was seeded from the intake submission. Its
// emission marks the intake as consumed: the submitted data is immutable
// thereafter and may only be corrected via a new submission.
type IntakeChartSeededEvent struct {
	// IntakeID is the identity of the IntakeFormAggregate that produced the event.
	IntakeID string
	// PatientID is the patient whose chart was seeded from the intake.
	PatientID string
}

// Type identifies the event kind.
func (e IntakeChartSeededEvent) Type() string { return IntakeChartSeededEventType }

// AggregateID ties the event back to the intake form that produced it.
func (e IntakeChartSeededEvent) AggregateID() string { return e.IntakeID }

// Compile-time assertion that IntakeChartSeededEvent satisfies the DomainEvent
// contract.
var _ shared.DomainEvent = IntakeChartSeededEvent{}
