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

// AttachLabResultCmd records the returned results against an existing lab order,
// moving it into the resulted state.
//
// OrderId identifies the placed order the results belong to. ResultDocumentRef
// is the reference to the stored result document (e.g., an HL7/FHIR
// DiagnosticReport or document-store handle). ResultedAt is the time the result
// was made available, carried as an RFC 3339 reference to match the codebase's
// string-valued field convention. All three are mandatory.
type AttachLabResultCmd struct {
	// OrderId identifies the placed lab order the results are attached to.
	OrderId string
	// ResultDocumentRef references the stored result document being attached.
	ResultDocumentRef string
	// ResultedAt is when the result became available (RFC 3339).
	ResultedAt string
}
