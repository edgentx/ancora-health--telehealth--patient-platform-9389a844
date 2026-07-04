package model

// PostSecureMessageCmd requests that an encrypted message be posted to a secure
// patient/care-team messaging thread.
//
// Posting is gated by the thread's participation and compliance invariants: the
// thread may only exist where there is an active care relationship between its
// participants, only the patient and care-team participants may post to (or read)
// the thread, and the message content must be PHI-encrypted at rest. AuthorId
// identifies the participant posting and Body carries the message content;
// ThreadId identifies the thread being posted to. All three are mandatory.
type PostSecureMessageCmd struct {
	// ThreadId identifies the message thread the message is posted to.
	ThreadId string
	// AuthorId identifies the participant posting the message; they must be the
	// patient or a care-team participant on the thread.
	AuthorId string
	// Body is the message content being posted; it must be PHI-encrypted at rest.
	Body string
}
