// Package model holds the aggregates for the clinical-records bounded context.
package model

import (
	"time"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"
)

// LabOrderStatus is the lifecycle state of a lab order. The zero value is an
// ordered (placed, awaiting results) order, which is what AttachLabResultCmd
// acts on.
type LabOrderStatus string

const (
	// LabOrderStatusOrdered is a placed order awaiting results. It is the zero
	// value, so a freshly constructed aggregate is ordered.
	LabOrderStatusOrdered LabOrderStatus = ""
	// LabOrderStatusResulted is an order that has had a result attached.
	LabOrderStatusResulted LabOrderStatus = "resulted"
	// LabOrderStatusCancelled is an order that was cancelled before results
	// returned; no result may be attached to it.
	LabOrderStatusCancelled LabOrderStatus = "cancelled"
)

// LabOrderAggregate is the clinical-records aggregate that tracks a lab order
// through its lifecycle. It embeds shared.AggregateRoot for version tracking and
// event buffering, and carries its own string identity.
//
// Beyond identity it tracks the state that command invariants read: its
// lifecycle status, the provider/patient the order was placed for and whether an
// active care relationship binds them, and the attached result.
type LabOrderAggregate struct {
	shared.AggregateRoot
	ID string

	// Status is the order's lifecycle state.
	Status LabOrderStatus

	// ProviderID and PatientID are the ordering provider and the patient the
	// order was placed for.
	ProviderID string
	PatientID  string

	// CareRelationshipActive reports whether the ordering provider retains an
	// active care relationship with the patient. A lab order may only be placed —
	// and its results attached — while this holds.
	CareRelationshipActive bool

	// ResultDocumentRef references the attached result document; it is empty until
	// a result is attached.
	ResultDocumentRef string

	// ResultedAt is the instant the attached result was reported; it is the zero
	// time until a result is attached.
	ResultedAt time.Time
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. Unrecognized command types return shared.ErrUnknownCommand.
func (a *LabOrderAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch c := cmd.(type) {
	case AttachLabResultCmd:
		return a.attachLabResult(c)
	default:
		return nil, shared.ErrUnknownCommand
	}
}

// attachLabResult handles AttachLabResultCmd: it validates the command input,
// enforces the lab-order invariants, then emits a LabResultReadyEvent and
// buffers it on the aggregate.
//
// The guards enforce, in order:
//
//   - Completeness: order id, result document ref and resulted timestamp must
//     all be present.
//   - Care relationship: the ordering provider must retain an active care
//     relationship with the patient.
//   - Existing, non-cancelled order: a result may not be attached to a cancelled
//     order.
//   - No revert: an order that is already resulted may not be walked back to the
//     ordered state by re-attaching a result.
func (a *LabOrderAggregate) attachLabResult(cmd AttachLabResultCmd) ([]shared.DomainEvent, error) {
	if cmd.OrderId == "" {
		return nil, ErrMissingOrder
	}
	if cmd.ResultDocumentRef == "" {
		return nil, ErrMissingResultDocumentRef
	}
	if cmd.ResultedAt.IsZero() {
		return nil, ErrMissingResultedAt
	}

	// Invariant: a lab order must be placed by a provider with an active care
	// relationship to the patient. Once that relationship lapses, results may no
	// longer be attached.
	if !a.CareRelationshipActive {
		return nil, ErrNoActiveCareRelationship
	}

	// Invariant: results may only be attached to an existing, non-cancelled
	// order. A cancelled order is terminal and accepts no results.
	if a.Status == LabOrderStatusCancelled {
		return nil, ErrOrderCancelled
	}

	// Invariant: a resulted order cannot be reverted to the ordered state.
	// Re-attaching a result to an already-resulted order would do exactly that,
	// so it is rejected.
	if a.Status == LabOrderStatusResulted {
		return nil, ErrOrderAlreadyResulted
	}

	evt := LabResultReadyEvent{
		OrderID:           a.ID,
		ResultDocumentRef: cmd.ResultDocumentRef,
		ResultedAt:        cmd.ResultedAt,
	}

	a.applyLabResultReady(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// applyLabResultReady mutates aggregate state from a LabResultReadyEvent. It is
// the single place these state changes happen, so it serves both command
// handling and future event replay when rehydrating the aggregate from the
// store.
func (a *LabOrderAggregate) applyLabResultReady(evt LabResultReadyEvent) {
	a.Status = LabOrderStatusResulted
	a.ResultDocumentRef = evt.ResultDocumentRef
	a.ResultedAt = evt.ResultedAt
}
