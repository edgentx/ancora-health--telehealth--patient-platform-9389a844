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
