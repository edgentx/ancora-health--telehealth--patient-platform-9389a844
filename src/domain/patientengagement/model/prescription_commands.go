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

// TransmitPrescriptionCmd requests that a drafted Prescription be sent to the
// pharmacy gateway for fulfillment.
//
// Transmission is the sealing act: it dispatches the order to a pharmacy and,
// once done, the prescription is immutable and may only be superseded by a
// cancellation. The same issuing invariants that gate composition gate
// transmission — the prescription must have been issued by an authenticated
// provider with an active care relationship, and it must clear (or have an
// acknowledged override for) its allergy and interaction checks before it can
// leave for the pharmacy. PrescriptionId identifies the order to transmit and
// PharmacyId the destination gateway; both are mandatory.
type TransmitPrescriptionCmd struct {
	// PrescriptionId identifies the prescription being transmitted.
	PrescriptionId string
	// PharmacyId identifies the pharmacy gateway the prescription is sent to.
	PharmacyId string
}

// RunSafetyCheckCmd requests that a drafted Prescription be run through allergy
// and interaction verification before it is transmitted.
//
// The safety check is a gating act: it verifies the drafted order against the
// patient's allergies and the interaction profile of their other medications.
// The same issuing invariants that gate composition and transmission gate the
// check — the prescription must have been issued by an authenticated provider
// with an active care relationship, a prior allergy/interaction failure must
// have been acknowledged or overridden before the check may proceed, and a
// prescription that has already been transmitted is immutable and may only be
// superseded by a cancellation. PrescriptionId identifies the order to check and
// is mandatory.
type RunSafetyCheckCmd struct {
	// PrescriptionId identifies the prescription the safety check is run against.
	PrescriptionId string
}
