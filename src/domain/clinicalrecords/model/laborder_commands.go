package model

import "time"

// AttachLabResultCmd requests that a returned result be recorded against a lab
// order, transitioning it from ordered to resulted.
//
// OrderId identifies the lab order the result belongs to; ResultDocumentRef
// points at the stored result document (e.g., the reference lab's PDF or the
// discrete-results payload); ResultedAt is the instant the result was reported.
// All three are mandatory. The order must exist and be non-cancelled, the
// ordering provider must retain an active care relationship with the patient,
// and an order that has already been resulted may not be walked back to the
// ordered state.
type AttachLabResultCmd struct {
	// OrderId is the identity of the lab order the result is attached to.
	OrderId string
	// ResultDocumentRef references the stored result document being attached.
	ResultDocumentRef string
	// ResultedAt is the instant the result was reported.
	ResultedAt time.Time
}
