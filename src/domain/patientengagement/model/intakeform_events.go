package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// IntakeFormSubmittedEventType is the stable wire name emitted when a completed
// intake form is submitted to seed the patient chart.
const IntakeFormSubmittedEventType = "intake.form.submitted"

// IntakeFormSubmittedEvent is emitted when a SubmitIntakeFormCmd succeeds. It
// records the patient the intake was completed for along with the captured
// clinical history and demographic detail. Its emission seals the form: the
// submitted data is thereafter immutable and may only be corrected via a new
// submission.
type IntakeFormSubmittedEvent struct {
	// IntakeFormID is the identity of the IntakeFormAggregate that produced the
	// event.
	IntakeFormID string
	// PatientID is the patient the intake form was completed for.
	PatientID string
	// History is the captured clinical/medical history submitted for the patient.
	History string
	// Demographics is the captured demographic detail submitted for the patient.
	Demographics string
}

// Type identifies the event kind.
func (e IntakeFormSubmittedEvent) Type() string { return IntakeFormSubmittedEventType }

// AggregateID ties the event back to the intake form that produced it.
func (e IntakeFormSubmittedEvent) AggregateID() string { return e.IntakeFormID }

// Compile-time assertion that IntakeFormSubmittedEvent satisfies the DomainEvent
// contract.
var _ shared.DomainEvent = IntakeFormSubmittedEvent{}

// IntakeChartSeededEventType is the stable wire name emitted when a validated
// intake form is projected into the patient chart.
const IntakeChartSeededEventType = "intake.chart.seeded"

// IntakeChartSeededEvent is emitted when a SeedChartFromIntakeCmd succeeds. It
// records the intake that was projected and the patient whose chart it seeded.
// Its emission seals the projection: the seeded chart data is thereafter
// immutable and may only be corrected via a new submission.
type IntakeChartSeededEvent struct {
	// IntakeFormID is the identity of the IntakeFormAggregate that produced the
	// event.
	IntakeFormID string
	// IntakeID is the intake whose validated data was projected into the chart.
	IntakeID string
	// PatientID is the patient whose chart was seeded from the intake.
	PatientID string
}

// Type identifies the event kind.
func (e IntakeChartSeededEvent) Type() string { return IntakeChartSeededEventType }

// AggregateID ties the event back to the intake form that produced it.
func (e IntakeChartSeededEvent) AggregateID() string { return e.IntakeFormID }

// Compile-time assertion that IntakeChartSeededEvent satisfies the DomainEvent
// contract.
var _ shared.DomainEvent = IntakeChartSeededEvent{}
