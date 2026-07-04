// Package model holds the aggregates for the clinical-records bounded context.
package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// LabOrderStatus is the lifecycle state of a lab order. The zero value is an
// unplaced order, which is what PlaceLabOrderCmd acts on.
type LabOrderStatus string

const (
	// LabOrderStatusUnplaced is a not-yet-placed order. It is the zero value, so a
	// freshly constructed aggregate is unplaced.
	LabOrderStatusUnplaced LabOrderStatus = ""
	// LabOrderStatusOrdered is a placed order awaiting results.
	LabOrderStatusOrdered LabOrderStatus = "ordered"
	// LabOrderStatusResulted is an order whose results have been attached.
	LabOrderStatusResulted LabOrderStatus = "resulted"
	// LabOrderStatusCancelled is a cancelled order; results may not be attached to
	// it.
	LabOrderStatusCancelled LabOrderStatus = "cancelled"
)

// LabOrderAggregate is the clinical-records aggregate that tracks a lab order
// through its lifecycle. It embeds shared.AggregateRoot for version tracking and
// event buffering, and carries its own string identity.
//
// Beyond identity it tracks the state that command invariants read: its
// lifecycle status, the patient/provider scoped to it, and whether that
// provider's care relationship to the patient is active.
type LabOrderAggregate struct {
	shared.AggregateRoot
	ID string

	// Status is the lab order's lifecycle state.
	Status LabOrderStatus

	// ScopedPatientID and ScopedProviderID are the participants the order is bound
	// to once placed. When non-empty they constrain who may place or act on the
	// order; an empty value means the participant is bound on placement.
	ScopedPatientID  string
	ScopedProviderID string

	// CareRelationshipActive reports whether the scoped provider holds an active
	// care relationship to the scoped patient. It is meaningful only once the
	// order has been scoped to a provider.
	CareRelationshipActive bool

	// TestCode is the coded-terminology reference for the ordered test. It is
	// empty until the order is placed.
	TestCode string
}

// Execute applies a command to the aggregate and returns the domain events it
// produced. Unrecognized command types return shared.ErrUnknownCommand.
func (a *LabOrderAggregate) Execute(cmd interface{}) ([]shared.DomainEvent, error) {
	switch c := cmd.(type) {
	case PlaceLabOrderCmd:
		return a.placeLabOrder(c)
	default:
		return nil, shared.ErrUnknownCommand
	}
}

// placeLabOrder handles PlaceLabOrderCmd: it validates the command input,
// enforces the lab-order invariants, then emits a LabOrderPlacedEvent and
// buffers it on the aggregate.
//
// The guards enforce, in order:
//
//   - Completeness: patient, provider and test code must all be present.
//   - Active care relationship: a lab order must be placed by a provider with an
//     active care relationship to the patient.
//   - Non-cancelled order: results may only be attached to an existing,
//     non-cancelled order, so a cancelled order may not be acted on.
//   - Result integrity: a resulted order cannot be reverted to the ordered
//     state.
func (a *LabOrderAggregate) placeLabOrder(cmd PlaceLabOrderCmd) ([]shared.DomainEvent, error) {
	if cmd.PatientId == "" {
		return nil, ErrMissingLabPatient
	}
	if cmd.ProviderId == "" {
		return nil, ErrMissingLabProvider
	}
	if cmd.TestCode == "" {
		return nil, ErrMissingTestCode
	}

	// Invariant: a lab order must be placed by a provider with an active care
	// relationship to the patient. When the order is already scoped, the command
	// must name the same participants, and that provider's care relationship must
	// still be active.
	if a.ScopedProviderID != "" && a.ScopedProviderID != cmd.ProviderId {
		return nil, ErrProviderNotInCare
	}
	if a.ScopedPatientID != "" && a.ScopedPatientID != cmd.PatientId {
		return nil, ErrProviderNotInCare
	}
	if a.ScopedProviderID != "" && !a.CareRelationshipActive {
		return nil, ErrProviderNotInCare
	}

	// Invariant: results may only be attached to an existing, non-cancelled order.
	// A cancelled order is terminal and may not be acted on.
	if a.Status == LabOrderStatusCancelled {
		return nil, ErrOrderCancelled
	}

	// Invariant: a resulted order cannot be reverted to the ordered state. Placing
	// an already-resulted order would push it back to ordered, so it is rejected.
	if a.Status == LabOrderStatusResulted {
		return nil, ErrResultedCannotRevert
	}

	evt := LabOrderPlacedEvent{
		LabOrderID: a.ID,
		PatientID:  cmd.PatientId,
		ProviderID: cmd.ProviderId,
		TestCode:   cmd.TestCode,
	}

	a.apply(evt)
	a.AddEvent(evt)
	a.Version++

	return []shared.DomainEvent{evt}, nil
}

// apply mutates aggregate state from a LabOrderPlacedEvent. It is the single
// place these state changes happen, so the same function serves both command
// handling and future event replay when rehydrating the aggregate from the
// store.
func (a *LabOrderAggregate) apply(evt LabOrderPlacedEvent) {
	a.Status = LabOrderStatusOrdered
	a.ScopedPatientID = evt.PatientID
	a.ScopedProviderID = evt.ProviderID
	a.CareRelationshipActive = true
	a.TestCode = evt.TestCode
}
