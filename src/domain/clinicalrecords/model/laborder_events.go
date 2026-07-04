package model

import (
	"time"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"
)

// LabResultReadyEventType is the stable wire name emitted when a result is
// attached to a lab order and the order becomes resulted.
const LabResultReadyEventType = "lab.result.ready"

// LabResultReadyEvent is emitted when an AttachLabResultCmd succeeds. It records
// the result document attached to the order and the instant it was reported.
type LabResultReadyEvent struct {
	// OrderID is the identity of the LabOrderAggregate that produced the event.
	OrderID string
	// ResultDocumentRef references the stored result document attached to the
	// order.
	ResultDocumentRef string
	// ResultedAt is the instant the result was reported.
	ResultedAt time.Time
}

// Type identifies the event kind.
func (e LabResultReadyEvent) Type() string { return LabResultReadyEventType }

// AggregateID ties the event back to the lab order that produced it.
func (e LabResultReadyEvent) AggregateID() string { return e.OrderID }

// Compile-time assertion that LabResultReadyEvent satisfies the DomainEvent
// contract.
var _ shared.DomainEvent = LabResultReadyEvent{}
