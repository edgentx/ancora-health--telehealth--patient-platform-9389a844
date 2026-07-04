package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// LabOrderPlacedEventType is the stable wire name emitted when a lab order is
// placed for a patient.
const LabOrderPlacedEventType = "lab.order.placed"

// LabOrderPlacedEvent is emitted when a PlaceLabOrderCmd succeeds. It records the
// patient the order is for, the ordering provider scoped to it, and the coded
// test that was ordered.
type LabOrderPlacedEvent struct {
	// LabOrderID is the identity of the LabOrderAggregate that produced the event.
	LabOrderID string
	// PatientID is the patient the lab order was placed for.
	PatientID string
	// ProviderID is the ordering provider; it holds an active care relationship
	// to the patient.
	ProviderID string
	// TestCode is the coded-terminology reference for the ordered test.
	TestCode string
}

// Type identifies the event kind.
func (e LabOrderPlacedEvent) Type() string { return LabOrderPlacedEventType }

// AggregateID ties the event back to the lab order that produced it.
func (e LabOrderPlacedEvent) AggregateID() string { return e.LabOrderID }

// Compile-time assertion that LabOrderPlacedEvent satisfies the DomainEvent
// contract.
var _ shared.DomainEvent = LabOrderPlacedEvent{}
