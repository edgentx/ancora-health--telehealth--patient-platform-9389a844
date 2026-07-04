package model

// ComposePrescriptionCmd requests that a new Prescription be drafted for a
// patient, capturing the medication and its dosage.
//
// Composing is the drafting act that brings a prescription into being: the
// issuing provider must be an authenticated provider with an active care
// relationship to the patient, the prescription must clear (or have an
// acknowledged override for) its allergy and interaction checks, and a
// prescription that has already been transmitted is immutable and may only be
// superseded by a cancellation. PatientId and ProviderId identify the two
// parties; Medication and Dosage are the drafted order. All four are mandatory.
type ComposePrescriptionCmd struct {
	// PatientId identifies the patient the prescription is drafted for.
	PatientId string
	// ProviderId identifies the provider issuing the prescription; they must be
	// an authenticated provider with an active care relationship to the patient.
	ProviderId string
	// Medication is the medication being prescribed.
	Medication string
	// Dosage is the dosage instruction drafted for the medication.
	Dosage string
}
