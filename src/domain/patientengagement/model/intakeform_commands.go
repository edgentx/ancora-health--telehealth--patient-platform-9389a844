package model

// SubmitIntakeFormCmd requests that a completed IntakeForm be submitted so its
// data can seed the patient chart.
//
// Submission is the sealing act for intake: the form must be complete and
// validated before it may seed the chart, and once submitted the captured data
// is immutable and may only be corrected by starting a new submission. PatientId
// identifies the patient the intake belongs to; History and Demographics are the
// captured clinical history and demographic details. All three are mandatory.
type SubmitIntakeFormCmd struct {
	// PatientId identifies the patient this intake form was completed for.
	PatientId string
	// History is the captured clinical/medical history for the patient.
	History string
	// Demographics is the captured demographic detail for the patient.
	Demographics string
}
