package model

// StartMessageThreadCmd requests that a new secure MessageThread be opened
// between a patient and their care team, capturing the thread's subject.
//
// Starting a thread is the act that brings a secure conversation into being: it
// may only join a patient with care-team members who share an active care
// relationship, its participant set gates who may post to or read the thread,
// and its message content must be PHI-encrypted at rest. PatientId identifies
// the patient, CareTeamMemberIds the care-team participants, and Subject the
// thread topic. All three are mandatory.
type StartMessageThreadCmd struct {
	// PatientId identifies the patient the thread is opened for; the patient is
	// one of the two permitted participant classes on the thread.
	PatientId string
	// CareTeamMemberIds identifies the care-team participants on the thread. They
	// must share an active care relationship with the patient, and together with
	// the patient they form the set permitted to post to or read the thread.
	CareTeamMemberIds []string
	// Subject is the topic the thread is opened to discuss.
	Subject string
}

// PostSecureMessageCmd requests that an encrypted message be posted to an
// existing secure MessageThread.
//
// Posting is the act of adding content to a live secure conversation, and it is
// bound by the same invariants that govern the thread itself: only the patient
// and care-team participants may post to or read the thread, the message content
// must be PHI-encrypted at rest, and the thread only exists where the
// participants share an active care relationship. ThreadId identifies the thread
// to post to, AuthorId the participant posting, and Body the message content.
// All three are mandatory.
type PostSecureMessageCmd struct {
	// ThreadId identifies the message thread the message is posted to.
	ThreadId string
	// AuthorId identifies the participant posting the message; they must be the
	// patient or one of the care-team participants scoped to the thread.
	AuthorId string
	// Body is the message content. It must be PHI-encrypted at rest.
	Body string
}
