package apiintegration

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/locking"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/interfaces/rest"
)

// Compile-time assertion that the minimal RESP client satisfies the slot
// locker's Redis port, so the production RedisSlotLocker can be wired over it.
var _ locking.RedisConn = (*redisConn)(nil)

// newStringBody wraps a string as an HTTP response body, used by the stubbed
// upstream to return a canned acknowledgement.
func newStringBody(s string) io.ReadCloser {
	return io.NopCloser(strings.NewReader(s))
}

// request issues a request through the booted router with the trusted-edge
// identity headers the Kong+OPA gateway would inject, applying any per-call
// header overrides last (so a test can vary the acting role or tenant). It
// returns the recorded response.
func (e *environment) request(method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	var rdr io.Reader = http.NoBody
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, rdr)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set(rest.HeaderSubject, defaultSubject)
	req.Header.Set(rest.HeaderRoles, defaultRoles)
	req.Header.Set(rest.HeaderTenant, defaultTenant)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// auditEntryCount reads a resource's audit trail through the public read API and
// reports how many sealed entries it holds. A resource that has produced no
// audit entry has no trail, which the API reports as 404 — surfaced here as a
// zero count with found=false so a test can distinguish "no trail" from "empty".
func (e *environment) auditEntryCount(resourceRef string) (count int, found bool, status int) {
	rec := e.request(http.MethodGet, "/api/v1/audit-trails/audit_"+resourceRef, "", nil)
	if rec.Code != http.StatusOK {
		return 0, false, rec.Code
	}
	var trail struct {
		EntryCount int `json:"entryCount"`
	}
	if err := decodeInto(rec.Body.Bytes(), &trail); err != nil {
		return 0, false, rec.Code
	}
	return trail.EntryCount, true, rec.Code
}
