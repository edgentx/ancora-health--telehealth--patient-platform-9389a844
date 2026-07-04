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

// AppendAddendumCmd requests that a correction be appended to an encounter's
// signed SOAP note. A signed note is immutable, so corrections are recorded as
// addenda rather than edits.
//
// EncounterId identifies the encounter to amend; AddendumText carries the
// correction; AuthorId identifies the author recording it. All three are
// mandatory, and the author must be one of the participants the encounter is
// scoped to — only scoped participants may view or amend in-call notes.
type AppendAddendumCmd struct {
	// EncounterId is the identity of the encounter whose note is being amended.
	EncounterId string
	// AddendumText is the body of the correction being appended.
	AddendumText string
	// AuthorId identifies the participant authoring the addendum.
	AuthorId string
}
