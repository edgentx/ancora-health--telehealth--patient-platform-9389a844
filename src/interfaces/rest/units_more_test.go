package rest

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/persistence/mongodb"
)

// --- respond.go ---

func TestWriteJSON_NilBody(t *testing.T) {
	rec := httptest.NewRecorder()
	writeJSON(rec, http.StatusNoContent, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("code = %d, want 204", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("expected empty body, got %q", rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type = %q", ct)
	}
}

func TestWriteJSON_Body(t *testing.T) {
	rec := httptest.NewRecorder()
	writeJSON(rec, http.StatusOK, map[string]string{"a": "b"})
	var got map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got["a"] != "b" {
		t.Fatalf("body = %v", got)
	}
}

func TestDecodeJSON_NilBody(t *testing.T) {
	req := &http.Request{Body: nil}
	var dst struct{}
	err := decodeJSON(req, &dst)
	if err == nil || !strings.Contains(err.Error(), "request body is required") {
		t.Fatalf("err = %v, want required", err)
	}
}

func TestDecodeJSON_TrailingContent(t *testing.T) {
	f := newFakes()
	router := NewRouter(f.deps())
	// Two JSON objects in one body: the strict decoder rejects trailing content.
	rec := doRequest(t, router, http.MethodPost, "/api/v1/appointments",
		`{"providerId":"p","timeSlot":"s","patientId":"x"}{}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	er := decodeErr(t, rec)
	if !strings.Contains(er.Message, "single JSON object") {
		t.Fatalf("message = %q, want single JSON object", er.Message)
	}
}

func TestDecodeJSON_EmptyBodyRequired(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
	var dst struct{}
	err := decodeJSON(req, &dst)
	if err == nil || !strings.Contains(err.Error(), "request body is required") {
		t.Fatalf("err = %v", err)
	}
}

// --- errors.go / statusForError ---

func TestStatusForError_Classification(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"validation", badRequest("bad field"), http.StatusBadRequest, codeValidation},
		{"not found", mongodb.ErrDocumentNotFound, http.StatusNotFound, codeNotFound},
		{"duplicate key conflict", mongodb.ErrDuplicateKey, http.StatusConflict, codeConflict},
		{"concurrency conflict", shared.ErrConcurrencyConflict, http.StatusConflict, codeConflict},
		{"tagged conflict", asConflict(errors.New("double booked")), http.StatusConflict, codeConflict},
		{"unknown command", shared.ErrUnknownCommand, http.StatusBadRequest, codeValidation},
		{"domain rule", domainError{err: errors.New("rule broken")}, http.StatusUnprocessableEntity, codeUnprocessable},
		{"infra", errors.New("mongo: connection reset"), http.StatusInternalServerError, codeInternal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, body := statusForError(tc.err)
			if status != tc.wantStatus {
				t.Fatalf("status = %d, want %d", status, tc.wantStatus)
			}
			if body.Code != tc.wantCode {
				t.Fatalf("code = %q, want %q", body.Code, tc.wantCode)
			}
			if body.Status != tc.wantStatus {
				t.Fatalf("body.Status = %d, want %d", body.Status, tc.wantStatus)
			}
		})
	}
}

func TestStatusForError_NotFoundAndInternalHideCause(t *testing.T) {
	_, nf := statusForError(mongodb.ErrDocumentNotFound)
	if nf.Message != notFoundMessage.Error() {
		t.Fatalf("not-found message = %q", nf.Message)
	}
	_, in := statusForError(errors.New("secret infra detail"))
	if strings.Contains(in.Message, "secret") {
		t.Fatalf("internal message leaked cause: %q", in.Message)
	}
}

func TestExecErr(t *testing.T) {
	t.Run("nil passes through", func(t *testing.T) {
		if execErr(nil) != nil {
			t.Fatal("expected nil")
		}
	})
	t.Run("conflict sentinel tagged", func(t *testing.T) {
		sentinel := errors.New("dup")
		err := execErr(sentinel, sentinel)
		if !isConflict(err) {
			t.Fatalf("expected conflict, got %v", err)
		}
	})
	t.Run("unknown command stays itself", func(t *testing.T) {
		err := execErr(shared.ErrUnknownCommand)
		if !errors.Is(err, shared.ErrUnknownCommand) {
			t.Fatalf("expected unknown command, got %v", err)
		}
		var d domainError
		if errors.As(err, &d) {
			t.Fatalf("unknown command must not be tagged domain")
		}
	})
	t.Run("other becomes domain error", func(t *testing.T) {
		err := execErr(errors.New("rule violation"))
		var d domainError
		if !errors.As(err, &d) {
			t.Fatalf("expected domain error, got %v", err)
		}
	})
}

func TestConflictError_Unwrap(t *testing.T) {
	base := errors.New("base")
	c := asConflict(base)
	if !errors.Is(c, base) {
		t.Fatalf("conflictError should unwrap to base")
	}
	if c.Error() != "base" {
		t.Fatalf("Error() = %q", c.Error())
	}
}

func TestDomainError_Unwrap(t *testing.T) {
	base := errors.New("dbase")
	d := domainError{err: base}
	if !errors.Is(d, base) {
		t.Fatalf("domainError should unwrap to base")
	}
	if d.Error() != "dbase" {
		t.Fatalf("Error() = %q", d.Error())
	}
}

func TestWriteError_WritesEnvelope(t *testing.T) {
	rec := httptest.NewRecorder()
	writeError(rec, badRequest("nope"))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code = %d, want 400", rec.Code)
	}
	var er ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &er); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if er.Code != codeValidation || er.Message != "nope" {
		t.Fatalf("envelope = %+v", er)
	}
}

// --- helpers.go ---

func TestNewID_Format(t *testing.T) {
	id := newID("appt")
	if !strings.HasPrefix(id, "appt_") {
		t.Fatalf("id = %q, want appt_ prefix", id)
	}
	if len(id) != len("appt_")+32 {
		t.Fatalf("id length = %d, want %d", len(id), len("appt_")+32)
	}
	if newID("appt") == id {
		t.Fatalf("ids should be unique")
	}
}

func TestRequireField(t *testing.T) {
	if err := requireField("value", "field"); err != nil {
		t.Fatalf("non-empty should pass: %v", err)
	}
	if err := requireField("   ", "field"); err == nil || !strings.Contains(err.Error(), "field is required") {
		t.Fatalf("whitespace should fail: %v", err)
	}
}

func TestPathID_TrimAndReject(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// No chi route context -> URLParam returns "" -> rejected.
	if _, err := pathID(req); err == nil {
		t.Fatalf("expected error for empty id")
	}
}

func TestRecordAudit_NilSinkNoop(t *testing.T) {
	if err := recordAudit(nil, nil, "a", "r", "act"); err != nil {
		t.Fatalf("nil sink should be noop, got %v", err)
	}
}

func TestRecordAudit_ForwardsError(t *testing.T) {
	sink := &fakeAudit{err: errors.New("boom")}
	if err := recordAudit(nil, sink, "a", "r", "act"); err == nil {
		t.Fatalf("expected forwarded error")
	}
	if sink.calls != 1 {
		t.Fatalf("calls = %d, want 1", sink.calls)
	}
}

// --- pagination.go ---

func TestParsePage_Defaults(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	p, err := parsePage(req)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if p.Limit != defaultPageLimit || p.Offset != 0 || p.Filter != "" {
		t.Fatalf("page = %+v", p)
	}
}

func TestParsePage_Cases(t *testing.T) {
	cases := []struct {
		name      string
		query     string
		wantErr   bool
		wantLimit int
	}{
		{"valid", "limit=10&offset=5&filter=VIEW", false, 10},
		{"limit zero clamps to max", "limit=0", false, maxPageLimit},
		{"limit over max clamps", "limit=9999", false, maxPageLimit},
		{"invalid limit", "limit=abc", true, 0},
		{"negative limit", "limit=-1", true, 0},
		{"invalid offset", "offset=abc", true, 0},
		{"negative offset", "offset=-2", true, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/x?"+tc.query, nil)
			p, err := parsePage(req)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("err = %v", err)
			}
			if p.Limit != tc.wantLimit {
				t.Fatalf("limit = %d, want %d", p.Limit, tc.wantLimit)
			}
		})
	}
}

func TestParsePage_ValidFilterAndOffset(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x?limit=10&offset=5&filter=VIEW", nil)
	p, _ := parsePage(req)
	if p.Offset != 5 || p.Filter != "VIEW" {
		t.Fatalf("page = %+v", p)
	}
}

func TestPageWindow(t *testing.T) {
	cases := []struct {
		name               string
		p                  Page
		n                  int
		wantStart, wantEnd int
	}{
		{"normal", Page{Offset: 1, Limit: 2}, 5, 1, 3},
		{"end clamped", Page{Offset: 3, Limit: 10}, 5, 3, 5},
		{"offset beyond", Page{Offset: 100, Limit: 10}, 5, 5, 5},
		{"empty collection", Page{Offset: 0, Limit: 10}, 0, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, e := tc.p.window(tc.n)
			if s != tc.wantStart || e != tc.wantEnd {
				t.Fatalf("window = (%d,%d), want (%d,%d)", s, e, tc.wantStart, tc.wantEnd)
			}
		})
	}
}

// --- identity.go ---

func TestParseRoles(t *testing.T) {
	cases := []struct {
		raw  string
		want int
	}{
		{"", 0},
		{"   ", 0},
		{"provider", 1},
		{"provider, admin", 2},
		{"provider, , admin ,", 2},
	}
	for _, tc := range cases {
		if got := parseRoles(tc.raw); len(got) != tc.want {
			t.Fatalf("parseRoles(%q) len = %d, want %d", tc.raw, len(got), tc.want)
		}
	}
}

func TestHasRole(t *testing.T) {
	c := Caller{Roles: []string{"provider"}}
	if !c.HasRole("provider") {
		t.Fatalf("expected provider role")
	}
	if c.HasRole("admin") {
		t.Fatalf("did not expect admin role")
	}
}

func TestCallerSubject_FromRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(HeaderSubject, "user-77")
	var got string
	IdentityMiddleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = callerSubject(r)
	})).ServeHTTP(httptest.NewRecorder(), req)
	if got != "user-77" {
		t.Fatalf("subject = %q", got)
	}
}

// --- router.go operational endpoints ---

func TestVersionEndpoint(t *testing.T) {
	f := newFakes()
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodGet, "/version", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestReadyUnhealthyProbe(t *testing.T) {
	f := newFakes()
	deps := f.deps()
	deps.Health = failingProbe{}
	router := NewRouter(deps)
	rec := doRequest(t, router, http.MethodGet, "/ready", "")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	var body map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body["reason"] != "database unreachable" {
		t.Fatalf("reason = %q", body["reason"])
	}
}

type failingProbe struct{}

func (failingProbe) HealthCheck(context.Context) error { return errors.New("db down") }
