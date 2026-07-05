package rest

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	adminmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/administrationandanalytics/model"
	auditmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/auditandcompliance/model"
	schedmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/scheduling/model"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"
)

// doRequest issues a request through the router with the trusted identity
// headers the edge would inject, and returns the recorded response.
func doRequest(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *strings.Reader
	if body != "" {
		reader = strings.NewReader(body)
	} else {
		reader = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set(HeaderSubject, "user-123")
	req.Header.Set(HeaderRoles, "provider,admin")
	req.Header.Set(HeaderTenant, "tenant-a")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// decodeErr reads the structured error envelope from a response body.
func decodeErr(t *testing.T, rec *httptest.ResponseRecorder) ErrorResponse {
	t.Helper()
	var er ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &er); err != nil {
		t.Fatalf("decoding error envelope: %v (body=%q)", err, rec.Body.String())
	}
	return er
}

// --- success path ---

func TestAppointmentHold_Success(t *testing.T) {
	f := newFakes()
	router := NewRouter(f.deps())

	body := `{"providerId":"prov-1","timeSlot":"2026-07-10T09:00Z","patientId":"pat-1"}`
	rec := doRequest(t, router, http.MethodPost, "/api/v1/appointments", body)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body=%q)", rec.Code, rec.Body.String())
	}
	var resp appointmentResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID == "" || resp.Status != string(schedmodel.AppointmentStatusHeld) {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp.ProviderID != "prov-1" || resp.PatientID != "pat-1" {
		t.Fatalf("participants not mapped: %+v", resp)
	}

	// The created record round-trips through GET.
	get := doRequest(t, router, http.MethodGet, "/api/v1/appointments/"+resp.ID, "")
	if get.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want 200", get.Code)
	}
}

// --- validation-failure path ---

func TestAppointmentHold_ValidationFailure(t *testing.T) {
	f := newFakes()
	router := NewRouter(f.deps())

	// Missing patientId.
	rec := doRequest(t, router, http.MethodPost, "/api/v1/appointments",
		`{"providerId":"prov-1","timeSlot":"2026-07-10T09:00Z"}`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	er := decodeErr(t, rec)
	if er.Code != codeValidation || !strings.Contains(er.Message, "patientId") {
		t.Fatalf("unexpected error envelope: %+v", er)
	}
}

func TestDecode_MalformedJSON(t *testing.T) {
	f := newFakes()
	router := NewRouter(f.deps())

	rec := doRequest(t, router, http.MethodPost, "/api/v1/appointments", `{"providerId":`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestDecode_UnknownField(t *testing.T) {
	f := newFakes()
	router := NewRouter(f.deps())

	rec := doRequest(t, router, http.MethodPost, "/api/v1/appointments",
		`{"providerId":"p","timeSlot":"s","patientId":"x","surprise":true}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

// --- not-found path ---

func TestAppointmentGet_NotFound(t *testing.T) {
	f := newFakes()
	router := NewRouter(f.deps())

	rec := doRequest(t, router, http.MethodGet, "/api/v1/appointments/does-not-exist", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	er := decodeErr(t, rec)
	if er.Code != codeNotFound {
		t.Fatalf("code = %q, want %q", er.Code, codeNotFound)
	}
	// The not-found body must not echo the requested identifier.
	if strings.Contains(er.Message, "does-not-exist") {
		t.Fatalf("error message leaked the requested id: %q", er.Message)
	}
}

func TestPrescriptionGet_NotFound(t *testing.T) {
	f := newFakes()
	router := NewRouter(f.deps())

	rec := doRequest(t, router, http.MethodGet, "/api/v1/prescriptions/missing", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

// --- conflict path (optimistic concurrency) ---

func TestAppointmentHold_ConcurrencyConflict(t *testing.T) {
	f := newFakes()
	f.appts.saveErr = shared.ErrConcurrencyConflict
	router := NewRouter(f.deps())

	rec := doRequest(t, router, http.MethodPost, "/api/v1/appointments",
		`{"providerId":"prov-1","timeSlot":"slot","patientId":"pat-1"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
	er := decodeErr(t, rec)
	if er.Code != codeConflict {
		t.Fatalf("code = %q, want %q", er.Code, codeConflict)
	}
}

// --- conflict path (domain double-booking) ---

func TestAppointmentBook_DoubleBookedConflict(t *testing.T) {
	f := newFakes()
	// A held appointment whose slot has since been claimed by another booking.
	f.appts.seed(&schedmodel.AppointmentAggregate{
		ID:                "appt-1",
		Status:            schedmodel.AppointmentStatusHeld,
		SlotAlreadyBooked: true,
	})
	router := NewRouter(f.deps())

	rec := doRequest(t, router, http.MethodPost, "/api/v1/appointments/appt-1/booking",
		`{"holdToken":"tok","patientId":"pat-1","reason":"checkup"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409 (body=%q)", rec.Code, rec.Body.String())
	}
}

// --- domain rule violation maps to 422 ---

func TestAppointmentBook_RuleViolationUnprocessable(t *testing.T) {
	f := newFakes()
	// Outside the policy window is a business-rule violation, not a conflict.
	f.appts.seed(&schedmodel.AppointmentAggregate{
		ID:                  "appt-2",
		Status:              schedmodel.AppointmentStatusHeld,
		OutsidePolicyWindow: true,
	})
	router := NewRouter(f.deps())

	rec := doRequest(t, router, http.MethodPost, "/api/v1/appointments/appt-2/booking",
		`{"holdToken":"tok","patientId":"pat-1","reason":"checkup"}`)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422 (body=%q)", rec.Code, rec.Body.String())
	}
	er := decodeErr(t, rec)
	if er.Code != codeUnprocessable {
		t.Fatalf("code = %q, want %q", er.Code, codeUnprocessable)
	}
}

// --- infrastructure failure maps to 500, not 422 ---

func TestAppointmentHold_InfraErrorInternal(t *testing.T) {
	f := newFakes()
	// A repository failure that is neither not-found nor a concurrency conflict is
	// an unexpected infrastructure fault: it must surface as 500, never as a
	// client-correctable 4xx.
	f.appts.saveErr = errors.New("mongo: connection reset")
	router := NewRouter(f.deps())

	rec := doRequest(t, router, http.MethodPost, "/api/v1/appointments",
		`{"providerId":"prov-1","timeSlot":"slot","patientId":"pat-1"}`)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 (body=%q)", rec.Code, rec.Body.String())
	}
	er := decodeErr(t, rec)
	if er.Code != codeInternal {
		t.Fatalf("code = %q, want %q", er.Code, codeInternal)
	}
	// The 500 body must not echo the underlying infrastructure detail.
	if strings.Contains(er.Message, "mongo") {
		t.Fatalf("error message leaked infrastructure detail: %q", er.Message)
	}
}

// --- cross-context success coverage ---

func TestPrescriptionCompose_Success(t *testing.T) {
	f := newFakes()
	router := NewRouter(f.deps())

	rec := doRequest(t, router, http.MethodPost, "/api/v1/prescriptions",
		`{"patientId":"pat-1","providerId":"prov-1","medication":"amoxicillin","dosage":"500mg"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body=%q)", rec.Code, rec.Body.String())
	}
}

func TestInsurancePolicyRegister_Success(t *testing.T) {
	f := newFakes()
	router := NewRouter(f.deps())

	rec := doRequest(t, router, http.MethodPost, "/api/v1/insurance-policies",
		`{"patientId":"pat-1","payerIdentifier":"payer-9","effectiveDates":{"start":"2026-01-01","end":"2026-12-31"}}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body=%q)", rec.Code, rec.Body.String())
	}
}

func TestClinicDirectory_CreateAndConflict(t *testing.T) {
	f := newFakes()
	router := NewRouter(f.deps())

	create := doRequest(t, router, http.MethodPost, "/api/v1/clinic-directories", "")
	if create.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", create.Code)
	}
	var dir clinicDirectoryResponse
	if err := json.Unmarshal(create.Body.Bytes(), &dir); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// Registering a provider that reuses an existing specialty code is a conflict.
	f.dirs.seed(&adminmodel.ClinicDirectoryAggregate{
		ID:                     dir.ID,
		SpecialtyCodes:         []string{"cardio"},
		DuplicateSpecialtyCode: true,
	})
	rec := doRequest(t, router, http.MethodPost, "/api/v1/clinic-directories/"+dir.ID+"/providers",
		`{"providerId":"prov-1","specialties":["cardio"],"clinicIds":["clinic-1"]}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409 (body=%q)", rec.Code, rec.Body.String())
	}
}

// --- audit read API with pagination ---

func TestAuditTrailRead_PaginationAndFilter(t *testing.T) {
	f := newFakes()
	f.trails.seed(seededTrail(t, "trail-1"))
	router := NewRouter(f.deps())

	// Full read.
	rec := doRequest(t, router, http.MethodGet, "/api/v1/audit-trails/trail-1", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%q)", rec.Code, rec.Body.String())
	}
	var full auditTrailResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &full); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if full.EntryCount != 3 || len(full.Entries) != 3 {
		t.Fatalf("expected 3 entries, got count=%d len=%d", full.EntryCount, len(full.Entries))
	}
	if full.HeadHash == "" {
		t.Fatalf("head hash should be exposed for chain verification")
	}

	// Paginated: first page of one entry.
	page := doRequest(t, router, http.MethodGet, "/api/v1/audit-trails/trail-1/entries?limit=1&offset=0", "")
	var window []auditEntryResponse
	if err := json.Unmarshal(page.Body.Bytes(), &window); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(window) != 1 {
		t.Fatalf("expected 1 entry in window, got %d", len(window))
	}

	// Filtered by action: the seeded chain holds exactly one UPDATE entry.
	filtered := doRequest(t, router, http.MethodGet, "/api/v1/audit-trails/trail-1?filter=UPDATE", "")
	var only auditTrailResponse
	if err := json.Unmarshal(filtered.Body.Bytes(), &only); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if only.EntryCount != 1 {
		t.Fatalf("expected 1 UPDATE entry, got %d", only.EntryCount)
	}
}

func TestPagination_InvalidLimit(t *testing.T) {
	f := newFakes()
	f.trails.seed(seededTrail(t, "trail-2"))
	router := NewRouter(f.deps())

	rec := doRequest(t, router, http.MethodGet, "/api/v1/audit-trails/trail-2/entries?limit=abc", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

// --- observability + operational endpoints ---

func TestMetricsEndpoint(t *testing.T) {
	f := newFakes()
	router := NewRouter(f.deps())

	// Generate some traffic first so a counter series exists.
	doRequest(t, router, http.MethodGet, "/api/v1/appointments/none", "")

	rec := doRequest(t, router, http.MethodGet, "/metrics", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "http_requests_total") {
		t.Fatalf("metrics output missing http_requests_total:\n%s", body)
	}
}

func TestHealthAndReady(t *testing.T) {
	f := newFakes()
	router := NewRouter(f.deps())

	if rec := doRequest(t, router, http.MethodGet, "/health", ""); rec.Code != http.StatusOK {
		t.Fatalf("/health status = %d, want 200", rec.Code)
	}
	// No health checker wired -> readiness reports unavailable.
	if rec := doRequest(t, router, http.MethodGet, "/ready", ""); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("/ready status = %d, want 503", rec.Code)
	}
}

func TestReady_Healthy(t *testing.T) {
	f := newFakes()
	deps := f.deps()
	deps.Health = healthyProbe{}
	router := NewRouter(deps)

	if rec := doRequest(t, router, http.MethodGet, "/ready", ""); rec.Code != http.StatusOK {
		t.Fatalf("/ready status = %d, want 200", rec.Code)
	}
}

type healthyProbe struct{}

func (healthyProbe) HealthCheck(context.Context) error { return nil }

// --- identity extraction ---

func TestIdentityMiddleware_ParsesTrustedHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(HeaderSubject, "  user-9 ")
	req.Header.Set(HeaderRoles, "provider, , admin")
	req.Header.Set(HeaderTenant, "tenant-z")

	var got Caller
	handler := IdentityMiddleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = CallerFrom(r.Context())
	}))
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if got.Subject != "user-9" {
		t.Fatalf("subject = %q, want trimmed user-9", got.Subject)
	}
	if got.Tenant != "tenant-z" {
		t.Fatalf("tenant = %q", got.Tenant)
	}
	if len(got.Roles) != 2 || !got.HasRole("provider") || !got.HasRole("admin") {
		t.Fatalf("roles not parsed: %+v", got.Roles)
	}
}

func TestCallerFrom_Empty(t *testing.T) {
	if c := CallerFrom(context.Background()); c.Subject != "" || len(c.Roles) != 0 {
		t.Fatalf("expected zero caller, got %+v", c)
	}
}

// --- test fixtures ---

// seededTrail builds an audit trail with three sealed entries (two VIEW, one
// UPDATE) by appending through the domain so the hash chain is well-formed.
func seededTrail(t *testing.T, id string) *auditmodel.AuditTrailAggregate {
	t.Helper()
	trail := &auditmodel.AuditTrailAggregate{ID: id}
	at := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
	actions := []string{"VIEW", "UPDATE", "VIEW"}
	for i, action := range actions {
		cmd := auditmodel.AppendAuditEntryCmd{
			ActorContext: "user-123",
			ResourceRef:  "patient/pat-1",
			Action:       action,
			OccurredAt:   at.Add(time.Duration(i) * time.Minute),
			PrevHash:     trail.HeadHash(),
		}
		if _, err := trail.Execute(cmd); err != nil {
			t.Fatalf("seeding audit entry %d: %v", i, err)
		}
	}
	return trail
}
