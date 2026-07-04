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
