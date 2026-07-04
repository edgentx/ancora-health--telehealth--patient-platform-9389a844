package model

// PlaceLabOrderCmd requests that a new lab order be placed for a patient by a
// provider, for a specific coded test.
//
// PatientId identifies the patient the specimen is drawn from; ProviderId names
// the ordering provider, who must hold an active care relationship to that
// patient — only such a provider may place an order. TestCode is the coded
// terminology reference (e.g., a LOINC code) for the test being ordered. All
// three are mandatory.
type PlaceLabOrderCmd struct {
	// PatientId identifies the patient the lab order is placed for.
	PatientId string
	// ProviderId identifies the ordering provider; the provider must hold an
	// active care relationship to the patient.
	ProviderId string
	// TestCode is the coded-terminology reference for the ordered test.
	TestCode string
}
