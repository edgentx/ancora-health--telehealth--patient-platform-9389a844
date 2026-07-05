package rest

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// recordAudit appends a compliance entry through sink when one is wired. It
// returns any error the sink reports, so a flow that must produce an audit entry
// (a clinical, prescribing, or billing action) surfaces a broken chain as a 500
// rather than passing silently. A nil sink is a clean no-op, which is what lets
// the handler unit suite run without an audit backend.
func recordAudit(ctx context.Context, sink AuditSink, actor, resourceRef, action string) error {
	if sink == nil {
		return nil
	}
	return sink.Record(ctx, actor, resourceRef, action)
}

// newID mints a fresh aggregate identity as "<prefix>_<16 random hex bytes>".
// Identities are minted server-side rather than accepted from the client so a
// caller can neither collide with nor guess another tenant's record ids. The
// prefix keeps ids self-describing in logs and traces.
func newID(prefix string) string {
	var buf [16]byte
	// crypto/rand.Read never returns a short read; the only possible error is a
	// dead entropy source, which is unrecoverable, so a failure is not something
	// a request handler can meaningfully act on.
	_, _ = rand.Read(buf[:])
	return prefix + "_" + hex.EncodeToString(buf[:])
}

// pathID reads and trims the {id} path parameter, rejecting an empty value as a
// validation error so a handler never loads on a blank identity.
func pathID(r *http.Request) (string, error) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		return "", badRequest("id path parameter is required")
	}
	return id, nil
}

// requireField returns a validation error naming field when value is empty. It
// is the building block every handler uses to reject an incomplete DTO with a
// 400 before a domain command is ever constructed — the message names the field,
// never its value, so request data cannot leak.
func requireField(value, field string) error {
	if strings.TrimSpace(value) == "" {
		return badRequest(field + " is required")
	}
	return nil
}
