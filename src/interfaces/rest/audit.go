package rest

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	auditmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/auditandcompliance/model"
	auditrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/auditandcompliance/repository"
)

// auditAPI adapts the audit-and-compliance bounded context to HTTP as a
// read-only surface. Audit trails are an append-only hash chain written by the
// domain elsewhere; this API only ever reads them, so it exposes GET routes and
// no mutation. Pagination and an Action filter let a compliance reviewer page
// through a long chain.
type auditAPI struct {
	trails auditrepo.AuditTrailRepository
}

func (h auditAPI) mount(r chi.Router) {
	r.Route("/audit-trails", func(r chi.Router) {
		r.Get("/{id}", h.get)
		r.Get("/{id}/entries", h.listEntries)
	})
}

// auditEntryResponse is the outward projection of a sealed audit entry. The
// hash-chain fields are surfaced so a reviewer can independently verify chain
// integrity; the actor/resource references are the same identifiers the chain
// sealed and carry no encrypted payload.
type auditEntryResponse struct {
	Sequence     int    `json:"sequence"`
	ActorContext string `json:"actorContext"`
	ResourceRef  string `json:"resourceRef"`
	Action       string `json:"action"`
	OccurredAt   string `json:"occurredAt"`
	PrevHash     string `json:"prevHash"`
	Hash         string `json:"hash"`
}

type auditTrailResponse struct {
	ID         string               `json:"id"`
	HeadHash   string               `json:"headHash"`
	EntryCount int                  `json:"entryCount"`
	Entries    []auditEntryResponse `json:"entries"`
	Version    int                  `json:"version"`
}

func toAuditEntryResponse(e auditmodel.AuditEntry) auditEntryResponse {
	return auditEntryResponse{
		Sequence:     e.Sequence,
		ActorContext: e.ActorContext,
		ResourceRef:  e.ResourceRef,
		Action:       e.Action,
		OccurredAt:   e.OccurredAt.UTC().Format(time.RFC3339),
		PrevHash:     e.PrevHash,
		Hash:         e.Hash,
	}
}

// filterEntries narrows entries to those whose Action equals the page filter.
// An empty filter is a pass-through, so unfiltered reads return the whole chain.
func filterEntries(entries []auditmodel.AuditEntry, filter string) []auditmodel.AuditEntry {
	if filter == "" {
		return entries
	}
	kept := make([]auditmodel.AuditEntry, 0, len(entries))
	for _, e := range entries {
		if e.Action == filter {
			kept = append(kept, e)
		}
	}
	return kept
}

// get returns the trail head hash plus a paginated, optionally filtered window
// of its entries.
func (h auditAPI) get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	page, err := parsePage(r)
	if err != nil {
		writeError(w, err)
		return
	}
	agg, err := h.trails.FindByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	entries := filterEntries(agg.Entries(), page.Filter)
	start, end := page.window(len(entries))
	window := entries[start:end]

	resp := auditTrailResponse{
		ID:         agg.ID,
		HeadHash:   agg.HeadHash(),
		EntryCount: len(entries),
		Entries:    make([]auditEntryResponse, 0, len(window)),
		Version:    agg.GetVersion(),
	}
	for _, e := range window {
		resp.Entries = append(resp.Entries, toAuditEntryResponse(e))
	}
	writeJSON(w, http.StatusOK, resp)
}

// listEntries returns only the paginated entry window, without the trail
// envelope — the lighter read a reviewer pages through.
func (h auditAPI) listEntries(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	page, err := parsePage(r)
	if err != nil {
		writeError(w, err)
		return
	}
	agg, err := h.trails.FindByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	entries := filterEntries(agg.Entries(), page.Filter)
	start, end := page.window(len(entries))
	window := entries[start:end]

	out := make([]auditEntryResponse, 0, len(window))
	for _, e := range window {
		out = append(out, toAuditEntryResponse(e))
	}
	writeJSON(w, http.StatusOK, out)
}
