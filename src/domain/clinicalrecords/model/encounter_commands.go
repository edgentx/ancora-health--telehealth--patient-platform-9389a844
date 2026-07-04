package model

// OpenEncounterCmd requests that a scheduled Encounter be started and its video
// room provisioned so the scoped provider and patient can join.
//
// AppointmentId ties the encounter back to the booking it realizes; ProviderId
// and PatientId identify the two participants scoped to the encounter. All three
// are mandatory, and the provider/patient must match any participants the
// encounter was already bound to — only scoped participants may join the video
// room or view in-call notes.
type OpenEncounterCmd struct {
	// AppointmentId is the identity of the appointment this encounter realizes.
	AppointmentId string
	// ProviderId identifies the rendering provider joining the encounter.
	ProviderId string
	// PatientId identifies the patient joining the encounter.
	PatientId string
}

// SignSoapNoteCmd requests that the encounter's SOAP note be signed and sealed
// by the rendering provider, recording the coded diagnoses reached during the
// encounter.
//
// Signing is the act that makes the note authoritative: once signed the note is
// immutable and may only be corrected by appended addenda. ProviderId names the
// signing provider, who must be the participant the encounter is scoped to —
// only scoped participants may view or act on the in-call note. Every diagnosis
// must reference a coded terminology entry (e.g., an ICD-10 code).
type SignSoapNoteCmd struct {
	// EncounterId is the identity of the encounter whose note is being signed.
	EncounterId string
	// ProviderId identifies the provider signing the note; it must match the
	// provider the encounter is scoped to.
	ProviderId string
	// SoapNote is the body of the SOAP note being signed and sealed.
	SoapNote string
	// Diagnoses are the coded findings recorded on the encounter. Each must
	// carry a coded-terminology reference.
	Diagnoses []Diagnosis
}

// CompleteEncounterCmd requests that a documented encounter be closed, marking
// it complete once its SOAP note has been signed.
//
// Completion is the act that finalizes the encounter: it may only proceed when
// the encounter carries a signed note, an encounter cannot be marked complete
// without one. ProviderId names the completing provider, who must be the
// participant the encounter is scoped to — only scoped participants may act on
// the encounter — and every recorded diagnosis must reference a coded
// terminology entry (e.g., an ICD-10 code).
type CompleteEncounterCmd struct {
	// EncounterId is the identity of the encounter being closed.
	EncounterId string
	// ProviderId identifies the provider completing the encounter; it must match
	// the provider the encounter is scoped to.
	ProviderId string
}
