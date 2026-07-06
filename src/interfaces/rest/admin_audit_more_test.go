package rest

import (
	"encoding/json"
	"net/http"
	"testing"

	adminmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/administrationandanalytics/model"
)

// --- clinic directory (administration) ---

func TestClinicDirectory_GetSuccessAndNotFound(t *testing.T) {
	f := newFakes()
	f.dirs.seed(&adminmodel.ClinicDirectoryAggregate{ID: "dir-1", ProviderIDs: []string{"prov-1"}})
	router := NewRouter(f.deps())
	if rec := doRequest(t, router, http.MethodGet, "/api/v1/clinic-directories/dir-1", ""); rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if rec := doRequest(t, router, http.MethodGet, "/api/v1/clinic-directories/missing", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestClinicDirectory_RegisterProviderSuccess(t *testing.T) {
	f := newFakes()
	f.dirs.seed(&adminmodel.ClinicDirectoryAggregate{ID: "dir-r"})
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodPost, "/api/v1/clinic-directories/dir-r/providers",
		`{"providerId":"prov-1","specialties":["cardio"],"clinicIds":["clinic-1"]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%q)", rec.Code, rec.Body.String())
	}
	var resp clinicDirectoryResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if len(resp.ProviderIDs) != 1 {
		t.Fatalf("providerIds = %v", resp.ProviderIDs)
	}
}

func TestClinicDirectory_RegisterProviderValidation(t *testing.T) {
	cases := []struct{ name, body string }{
		{"missing provider", `{"specialties":["c"],"clinicIds":["cl"]}`},
		{"empty specialties", `{"providerId":"p","specialties":[],"clinicIds":["cl"]}`},
		{"empty clinics", `{"providerId":"p","specialties":["c"],"clinicIds":[]}`},
		{"malformed", `{`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakes()
			f.dirs.seed(&adminmodel.ClinicDirectoryAggregate{ID: "dir-v"})
			router := NewRouter(f.deps())
			rec := doRequest(t, router, http.MethodPost, "/api/v1/clinic-directories/dir-v/providers", tc.body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", rec.Code)
			}
		})
	}
}

func TestClinicDirectory_RegisterProviderNotFound(t *testing.T) {
	f := newFakes()
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodPost, "/api/v1/clinic-directories/missing/providers",
		`{"providerId":"p","specialties":["c"],"clinicIds":["cl"]}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestClinicDirectory_ManageSpecialtySuccess(t *testing.T) {
	f := newFakes()
	f.dirs.seed(&adminmodel.ClinicDirectoryAggregate{ID: "dir-s"})
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodPost, "/api/v1/clinic-directories/dir-s/specialties",
		`{"specialtyCode":"cardio","displayName":"Cardiology"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%q)", rec.Code, rec.Body.String())
	}
	var resp clinicDirectoryResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if len(resp.SpecialtyCodes) != 1 {
		t.Fatalf("specialtyCodes = %v", resp.SpecialtyCodes)
	}
}

func TestClinicDirectory_ManageSpecialtyValidation(t *testing.T) {
	cases := []struct{ name, body string }{
		{"missing code", `{"displayName":"n"}`},
		{"missing displayName", `{"specialtyCode":"c"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakes()
			f.dirs.seed(&adminmodel.ClinicDirectoryAggregate{ID: "dir-sv"})
			router := NewRouter(f.deps())
			rec := doRequest(t, router, http.MethodPost, "/api/v1/clinic-directories/dir-sv/specialties", tc.body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", rec.Code)
			}
		})
	}
}

func TestClinicDirectory_ManageSpecialtyNotFound(t *testing.T) {
	f := newFakes()
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodPost, "/api/v1/clinic-directories/missing/specialties",
		`{"specialtyCode":"c","displayName":"n"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestClinicDirectory_ConfigureClinicSuccess(t *testing.T) {
	f := newFakes()
	f.dirs.seed(&adminmodel.ClinicDirectoryAggregate{ID: "dir-cc"})
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodPost, "/api/v1/clinic-directories/dir-cc/clinics",
		`{"clinicIdentity":"clinic-1","operatingHours":"09:00-17:00"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%q)", rec.Code, rec.Body.String())
	}
	var resp clinicDirectoryResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if len(resp.ClinicIDs) != 1 {
		t.Fatalf("clinicIds = %v", resp.ClinicIDs)
	}
}

func TestClinicDirectory_ConfigureClinicValidation(t *testing.T) {
	cases := []struct{ name, body string }{
		{"missing identity", `{"operatingHours":"h"}`},
		{"missing hours", `{"clinicIdentity":"c"}`},
		{"malformed", `{`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakes()
			f.dirs.seed(&adminmodel.ClinicDirectoryAggregate{ID: "dir-ccv"})
			router := NewRouter(f.deps())
			rec := doRequest(t, router, http.MethodPost, "/api/v1/clinic-directories/dir-ccv/clinics", tc.body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", rec.Code)
			}
		})
	}
}

func TestClinicDirectory_ConfigureClinicNotFound(t *testing.T) {
	f := newFakes()
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodPost, "/api/v1/clinic-directories/missing/clinics",
		`{"clinicIdentity":"c","operatingHours":"h"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

// --- audit trails ---

func TestAuditTrail_GetNotFound(t *testing.T) {
	f := newFakes()
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodGet, "/api/v1/audit-trails/missing", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestAuditTrail_ListEntriesNotFound(t *testing.T) {
	f := newFakes()
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodGet, "/api/v1/audit-trails/missing/entries", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestAuditTrail_ListEntriesFull(t *testing.T) {
	f := newFakes()
	f.trails.seed(seededTrail(t, "trail-le"))
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodGet, "/api/v1/audit-trails/trail-le/entries", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var out []auditEntryResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(out) != 3 {
		t.Fatalf("entries = %d, want 3", len(out))
	}
	if out[0].Hash == "" {
		t.Fatalf("entry hash should be exposed")
	}
}

func TestAuditTrail_PaginationClampAndOffsetBeyond(t *testing.T) {
	f := newFakes()
	f.trails.seed(seededTrail(t, "trail-pg"))
	router := NewRouter(f.deps())

	// A limit above the max is clamped, not rejected.
	if rec := doRequest(t, router, http.MethodGet, "/api/v1/audit-trails/trail-pg/entries?limit=5000", ""); rec.Code != http.StatusOK {
		t.Fatalf("clamp status = %d, want 200", rec.Code)
	}
	// An offset past the end yields an empty window, still 200.
	rec := doRequest(t, router, http.MethodGet, "/api/v1/audit-trails/trail-pg/entries?offset=100", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("offset status = %d, want 200", rec.Code)
	}
	var out []auditEntryResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if len(out) != 0 {
		t.Fatalf("expected empty window, got %d", len(out))
	}
}

func TestAuditTrail_InvalidOffset(t *testing.T) {
	f := newFakes()
	f.trails.seed(seededTrail(t, "trail-io"))
	router := NewRouter(f.deps())
	if rec := doRequest(t, router, http.MethodGet, "/api/v1/audit-trails/trail-io/entries?offset=-1", ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if rec := doRequest(t, router, http.MethodGet, "/api/v1/audit-trails/trail-io?offset=xyz", ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}
