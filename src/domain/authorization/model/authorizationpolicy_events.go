package model

import "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"

// PolicyPublishedEventType is the stable wire name emitted when a new
// authorization policy ruleset version is published.
const PolicyPublishedEventType = "authz.policy.published"

// PolicyPublishedEvent is emitted when a PublishPolicyVersionCmd succeeds. It
// records the Rego bundle that was deployed as the new immutable version and
// the author accountable for the publication.
type PolicyPublishedEvent struct {
	// PolicyID is the identity of the AuthorizationPolicyAggregate that produced
	// the event.
	PolicyID string
	// RegoBundle is the immutable ruleset that was deployed as the new version.
	RegoBundle string
	// Author identifies who published the new ruleset version.
	Author string
}

// Type identifies the event kind.
func (e PolicyPublishedEvent) Type() string { return PolicyPublishedEventType }

// AggregateID ties the event back to the policy that produced it.
func (e PolicyPublishedEvent) AggregateID() string { return e.PolicyID }

// Compile-time assertion that PolicyPublishedEvent satisfies the DomainEvent
// contract.
var _ shared.DomainEvent = PolicyPublishedEvent{}

// AccessGrantedEventType is the stable wire name emitted when an access
// evaluation resolves to allow.
const AccessGrantedEventType = "authz.access.granted"

// AccessDeniedEventType is the stable wire name emitted when an access
// evaluation resolves to deny, including the default-deny outcome when no
// matching allow rule exists.
const AccessDeniedEventType = "authz.access.denied"

// AccessGrantedEvent is emitted when an EvaluateAccessCmd resolves to allow. It
// records the subject-resource pair and the action that was permitted, together
// with the care context the decision was rendered within.
type AccessGrantedEvent struct {
	// PolicyID is the identity of the AuthorizationPolicyAggregate that rendered
	// the decision.
	PolicyID string
	// SubjectAttrs are the subject attributes the allow decision was rendered for.
	SubjectAttrs map[string]string
	// ResourceRef identifies the resource access was granted to.
	ResourceRef string
	// Action is the operation that was permitted on the resource.
	Action string
	// CareContext is the clinical relationship the decision was evaluated within.
	CareContext string
}

// Type identifies the event kind.
func (e AccessGrantedEvent) Type() string { return AccessGrantedEventType }

// AggregateID ties the event back to the policy that produced it.
func (e AccessGrantedEvent) AggregateID() string { return e.PolicyID }

// Compile-time assertion that AccessGrantedEvent satisfies the DomainEvent
// contract.
var _ shared.DomainEvent = AccessGrantedEvent{}

// AccessDeniedEvent is emitted when an EvaluateAccessCmd resolves to deny. It
// records the subject-resource pair and the action that was refused, the care
// context the decision was rendered within, and the reason the request was
// denied (e.g. the default-deny fallthrough).
type AccessDeniedEvent struct {
	// PolicyID is the identity of the AuthorizationPolicyAggregate that rendered
	// the decision.
	PolicyID string
	// SubjectAttrs are the subject attributes the deny decision was rendered for.
	SubjectAttrs map[string]string
	// ResourceRef identifies the resource access was denied to.
	ResourceRef string
	// Action is the operation that was refused on the resource.
	Action string
	// CareContext is the clinical relationship the decision was evaluated within.
	CareContext string
	// Reason records why access was denied, for the audit trail.
	Reason string
}

// Type identifies the event kind.
func (e AccessDeniedEvent) Type() string { return AccessDeniedEventType }

// AggregateID ties the event back to the policy that produced it.
func (e AccessDeniedEvent) AggregateID() string { return e.PolicyID }

// Compile-time assertion that AccessDeniedEvent satisfies the DomainEvent
// contract.
var _ shared.DomainEvent = AccessDeniedEvent{}
