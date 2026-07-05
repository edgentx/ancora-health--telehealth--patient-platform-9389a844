// Package audit provides the concrete audit sink that turns a flow-level or
// outbound-adapter access into a sealed entry on the audit-and-compliance hash
// chain. It is the bridge between the thin recording seams the REST handlers
// and integration adapters depend on and the append-only AuditTrailRepository
// that owns the tamper-evident chain.
//
// A recorder keys each resource's history under its own trail ("audit_<ref>"),
// loads-or-starts that trail, and appends an entry referencing the current
// chain head so the hash linkage stays unbroken. It records only references —
// actor, resource id, action — never the PHI the resource carries, the same
// discipline the persistence codec applies at rest.
package audit

import (
	"context"
	"errors"
	"strings"
	"time"

	auditmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/auditandcompliance/model"
	auditrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/auditandcompliance/repository"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/integration"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/persistence/mongodb"
)

// systemActor is the fallback actor stamped on an entry when the caller supplied
// no subject (an unauthenticated or system-initiated flow). An audit entry must
// name an actor, so a blank one is normalised rather than rejected.
const systemActor = "system"

// trailPrefix namespaces a resource's audit trail id, keeping per-resource
// chains from colliding with any other id space in the collection.
const trailPrefix = "audit_"

// TrailRecorder appends flow and outbound-access events onto the per-resource
// audit hash chain. It satisfies both the narrow recording seam the REST
// handlers use and integration.AuditRecorder, so a single sink audits in-process
// mutations and outbound PHI access alike.
type TrailRecorder struct {
	trails auditrepo.AuditTrailRepository
	now    func() time.Time
}

// NewTrailRecorder builds a recorder over the audit-trail repository. The clock
// is injectable so a test can seal entries at deterministic instants.
func NewTrailRecorder(trails auditrepo.AuditTrailRepository) *TrailRecorder {
	return &TrailRecorder{trails: trails, now: time.Now}
}

// Record seals one entry describing actor performing action against
// resourceRef. It loads the resource's trail (starting a fresh one on first
// touch), appends referencing the current head so the chain stays linked, and
// persists. A blank actor is normalised to the system actor.
func (r *TrailRecorder) Record(ctx context.Context, actor, resourceRef, action string) error {
	if strings.TrimSpace(resourceRef) == "" || strings.TrimSpace(action) == "" {
		// Nothing meaningful to seal; treat as a no-op rather than corrupt the
		// chain with an incomplete entry.
		return nil
	}
	if strings.TrimSpace(actor) == "" {
		actor = systemActor
	}

	trailID := trailPrefix + resourceRef
	trail, err := r.trails.FindByID(ctx, trailID)
	if err != nil {
		if !errors.Is(err, mongodb.ErrDocumentNotFound) {
			return err
		}
		trail = &auditmodel.AuditTrailAggregate{ID: trailID}
	}

	cmd := auditmodel.AppendAuditEntryCmd{
		ActorContext: actor,
		ResourceRef:  resourceRef,
		Action:       action,
		OccurredAt:   r.now(),
		PrevHash:     trail.HeadHash(),
	}
	if _, err := trail.Execute(cmd); err != nil {
		return err
	}
	return r.trails.Save(ctx, trail)
}

// RecordOutboundAccess adapts an adapter's OutboundAccess record onto the same
// per-resource chain, so an outbound PHI access (e.g. a pharmacy transmission)
// is audited exactly like an in-process mutation. It satisfies
// integration.AuditRecorder.
func (r *TrailRecorder) RecordOutboundAccess(ctx context.Context, access integration.OutboundAccess) error {
	return r.Record(ctx, access.ActorContext, access.ResourceRef, access.Action)
}

// Compile-time assertion that the recorder is a valid outbound audit sink.
var _ integration.AuditRecorder = (*TrailRecorder)(nil)
