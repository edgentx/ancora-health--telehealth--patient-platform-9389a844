package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// AccessGrantedEventType is the stable wire name emitted when an evaluated
// access decision resolves to allow.
const AccessGrantedEventType = "authz.access.granted"

// AccessGrantedEvent is emitted when EvaluateAccessCmd renders an allow
// decision — a matching allow rule exists within the subject's least-privilege
// ceiling. It records the resource, action and care context the decision was
// rendered for.
type AccessGrantedEvent struct {
	// PolicyID is the identity of the AuthorizationPolicyAggregate that produced
	// the event.
	PolicyID string
	// ResourceRef is the resource the decision was rendered for.
	ResourceRef string
	// Action is the operation the decision was rendered for.
	Action string
	// CareContext is the care relationship the decision was scoped to.
	CareContext string
}

// Type identifies the event kind.
func (e AccessGrantedEvent) Type() string { return AccessGrantedEventType }

// AggregateID ties the event back to the policy that produced it.
func (e AccessGrantedEvent) AggregateID() string { return e.PolicyID }

// Compile-time assertion that AccessGrantedEvent satisfies the DomainEvent
// contract.
var _ shared.DomainEvent = AccessGrantedEvent{}

// AccessDeniedEventType is the stable wire name emitted when an evaluated
// access decision resolves to deny.
const AccessDeniedEventType = "authz.access.denied"

// AccessDeniedEvent is emitted when EvaluateAccessCmd renders a deny decision —
// the default outcome when no matching allow rule exists. It records the
// resource, action and care context the decision was rendered for.
type AccessDeniedEvent struct {
	// PolicyID is the identity of the AuthorizationPolicyAggregate that produced
	// the event.
	PolicyID string
	// ResourceRef is the resource the decision was rendered for.
	ResourceRef string
	// Action is the operation the decision was rendered for.
	Action string
	// CareContext is the care relationship the decision was scoped to.
	CareContext string
}

// Type identifies the event kind.
func (e AccessDeniedEvent) Type() string { return AccessDeniedEventType }

// AggregateID ties the event back to the policy that produced it.
func (e AccessDeniedEvent) AggregateID() string { return e.PolicyID }

// Compile-time assertion that AccessDeniedEvent satisfies the DomainEvent
// contract.
var _ shared.DomainEvent = AccessDeniedEvent{}
