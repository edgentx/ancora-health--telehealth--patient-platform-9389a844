package model

// RunSafetyCheckCmd requests that allergy and drug-interaction verification be
// run against a prescription before it can move toward transmission.
//
// PrescriptionId identifies the prescription being checked. ProviderId names the
// issuing provider, who must be authenticated and hold an active care
// relationship to the patient — only such a provider may issue or act on a
// prescription. PatientId identifies the patient the prescription is written
// for. All three are mandatory.
type RunSafetyCheckCmd struct {
	// PrescriptionId is the identity of the prescription the safety check runs
	// against.
	PrescriptionId string
	// ProviderId identifies the issuing provider; the provider must be
	// authenticated and hold an active care relationship to the patient.
	ProviderId string
	// PatientId identifies the patient the prescription is written for.
	PatientId string
}
