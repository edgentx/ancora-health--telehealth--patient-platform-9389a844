package integration

import (
	"context"
	"time"
)

// OutboundAccess describes a single outbound access to a third-party service
// that touched protected health information (PHI). It is deliberately a
// reference record, not a payload: it names the actor, the resource, and the
// action, but never carries the PHI itself (medication, dosage, member data,
// document contents). That keeps the audit trail useful for compliance without
// widening the PHI surface — the same discipline the persistence codec applies
// at rest.
type OutboundAccess struct {
	// ActorContext identifies who or what initiated the access — typically the
	// authenticated provider or service principal passed through from trusted
	// headers.
	ActorContext string
	// ResourceRef is a stable, non-PHI reference to the resource accessed, e.g. a
	// prescription id, policy id, or bucket/key — never the resource's contents.
	ResourceRef string
	// Action is the outbound operation performed, e.g. "eprescribe.submit" or
	// "eligibility.check".
	Action string
	// Destination names the upstream service the access went to, e.g. the
	// pharmacy gateway or object store endpoint.
	Destination string
	// OccurredAt is when the access was made.
	OccurredAt time.Time
}

// AuditRecorder is the port the adapters record outbound PHI access through. It
// is intentionally narrow so any sink — the audit-and-compliance hash chain, a
// SIEM shipper, or a test spy — can satisfy it. A nil recorder is tolerated by
// the adapters (audit becomes a no-op) so the transport can be used in contexts
// where audit is wired separately.
type AuditRecorder interface {
	RecordOutboundAccess(ctx context.Context, access OutboundAccess) error
}

// RecordIfSet records access on rec when rec is non-nil, returning any error the
// sink reports. It centralises the nil-tolerant guard every adapter would
// otherwise repeat.
func RecordIfSet(ctx context.Context, rec AuditRecorder, access OutboundAccess) error {
	if rec == nil {
		return nil
	}
	return rec.RecordOutboundAccess(ctx, access)
}
