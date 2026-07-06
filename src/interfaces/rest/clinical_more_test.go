package rest

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	clinicalmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/clinicalrecords/model"
	engagementmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/patientengagement/model"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/integration/pharmacy"
)

// --- lab orders ---

func TestLabOrder_PlaceSuccess(t *testing.T) {
	f := newFakes()
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodPost, "/api/v1/lab-orders",
		`{"patientId":"pat-1","providerId":"prov-1","testCode":"CBC"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body=%q)", rec.Code, rec.Body.String())
	}
	var resp labOrderResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID == "" || resp.TestCode != "CBC" {
		t.Fatalf("unexpected: %+v", resp)
	}
	if get := doRequest(t, router, http.MethodGet, "/api/v1/lab-orders/"+resp.ID, ""); get.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want 200", get.Code)
	}
}

func TestLabOrder_PlaceValidation(t *testing.T) {
	cases := []struct{ name, body string }{
		{"missing patient", `{"providerId":"pr","testCode":"CBC"}`},
		{"missing provider", `{"patientId":"p","testCode":"CBC"}`},
		{"missing testCode", `{"patientId":"p","providerId":"pr"}`},
		{"malformed", `{`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakes()
			router := NewRouter(f.deps())
			rec := doRequest(t, router, http.MethodPost, "/api/v1/lab-orders", tc.body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", rec.Code)
			}
		})
	}
}

func TestLabOrder_GetNotFound(t *testing.T) {
	f := newFakes()
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodGet, "/api/v1/lab-orders/missing", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

// --- prescriptions ---

func TestPrescription_ComposeValidation(t *testing.T) {
	cases := []struct{ name, body string }{
		{"missing patient", `{"providerId":"pr","medication":"m","dosage":"d"}`},
		{"missing provider", `{"patientId":"p","medication":"m","dosage":"d"}`},
		{"missing medication", `{"patientId":"p","providerId":"pr","dosage":"d"}`},
		{"missing dosage", `{"patientId":"p","providerId":"pr","medication":"m"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakes()
			router := NewRouter(f.deps())
			rec := doRequest(t, router, http.MethodPost, "/api/v1/prescriptions", tc.body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", rec.Code)
			}
		})
	}
}

func composedRx(id string) *engagementmodel.PrescriptionAggregate {
	return &engagementmodel.PrescriptionAggregate{
		ID:               id,
		Status:           engagementmodel.PrescriptionStatusComposed,
		ScopedPatientID:  "pat-1",
		ScopedProviderID: "prov-1",
		Medication:       "amoxicillin",
		Dosage:           "500mg",
	}
}

func TestPrescription_TransmitNoGateway(t *testing.T) {
	f := newFakes()
	f.rx.seed(composedRx("rx-1"))
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodPost, "/api/v1/prescriptions/rx-1/transmission",
		`{"pharmacyId":"pharm-9"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%q)", rec.Code, rec.Body.String())
	}
	var resp prescriptionResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Status != string(engagementmodel.PrescriptionStatusTransmitted) {
		t.Fatalf("status = %q, want transmitted", resp.Status)
	}
}

func TestPrescription_TransmitMissingPharmacy(t *testing.T) {
	f := newFakes()
	f.rx.seed(composedRx("rx-2"))
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodPost, "/api/v1/prescriptions/rx-2/transmission", `{}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestPrescription_TransmitNotFound(t *testing.T) {
	f := newFakes()
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodPost, "/api/v1/prescriptions/missing/transmission",
		`{"pharmacyId":"pharm-9"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestPrescription_TransmitGateway(t *testing.T) {
	t.Run("accepted -> 200", func(t *testing.T) {
		f := newFakes()
		f.rx.seed(composedRx("rx-a"))
		d := f.deps()
		gw := &fakePharmacy{result: pharmacy.TransmissionResult{Status: pharmacy.StatusAccepted}}
		d.Pharmacy = gw
		router := NewRouter(d)
		rec := doRequest(t, router, http.MethodPost, "/api/v1/prescriptions/rx-a/transmission",
			`{"pharmacyId":"pharm-9"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body=%q)", rec.Code, rec.Body.String())
		}
		if gw.calls != 1 {
			t.Fatalf("gateway calls = %d, want 1", gw.calls)
		}
	})

	t.Run("rejected -> 422", func(t *testing.T) {
		f := newFakes()
		f.rx.seed(composedRx("rx-r"))
		d := f.deps()
		d.Pharmacy = &fakePharmacy{result: pharmacy.TransmissionResult{Status: pharmacy.StatusRejected}}
		router := NewRouter(d)
		rec := doRequest(t, router, http.MethodPost, "/api/v1/prescriptions/rx-r/transmission",
			`{"pharmacyId":"pharm-9"}`)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("status = %d, want 422 (body=%q)", rec.Code, rec.Body.String())
		}
	})

	t.Run("transport error -> 500", func(t *testing.T) {
		f := newFakes()
		f.rx.seed(composedRx("rx-e"))
		d := f.deps()
		d.Pharmacy = &fakePharmacy{err: pharmacy.ErrGatewayUnavailable}
		router := NewRouter(d)
		rec := doRequest(t, router, http.MethodPost, "/api/v1/prescriptions/rx-e/transmission",
			`{"pharmacyId":"pharm-9"}`)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500 (body=%q)", rec.Code, rec.Body.String())
		}
	})
}

func TestPrescription_SafetyCheck(t *testing.T) {
	f := newFakes()
	f.rx.seed(composedRx("rx-s"))
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodPost, "/api/v1/prescriptions/rx-s/safety-check", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%q)", rec.Code, rec.Body.String())
	}
	var resp prescriptionResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if !resp.SafetyChecked {
		t.Fatalf("safetyChecked = false, want true")
	}
}

func TestPrescription_SafetyCheckNotFound(t *testing.T) {
	f := newFakes()
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodPost, "/api/v1/prescriptions/missing/safety-check", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestPrescription_TransmitInfraSaveError(t *testing.T) {
	f := newFakes()
	f.rx.seed(composedRx("rx-i"))
	f.rx.saveErr = errors.New("mongo down")
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodPost, "/api/v1/prescriptions/rx-i/transmission",
		`{"pharmacyId":"pharm-9"}`)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 (body=%q)", rec.Code, rec.Body.String())
	}
}

// --- encounters ---

func TestEncounter_OpenSuccess(t *testing.T) {
	f := newExtFakes()
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodPost, "/api/v1/encounters",
		`{"appointmentId":"appt-1","providerId":"prov-1","patientId":"pat-1"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body=%q)", rec.Code, rec.Body.String())
	}
	var resp encounterResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != string(clinicalmodel.EncounterStatusOpen) {
		t.Fatalf("status = %q, want open", resp.Status)
	}
	if f.audit.calls != 1 || f.audit.last.action != "encounter.open" {
		t.Fatalf("audit not recorded: %+v", f.audit)
	}
	if f.audit.last.actor != "user-123" {
		t.Fatalf("audit actor = %q, want user-123", f.audit.last.actor)
	}
	// round trip
	if get := doRequest(t, router, http.MethodGet, "/api/v1/encounters/"+resp.ID, ""); get.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want 200", get.Code)
	}
}

func TestEncounter_OpenValidation(t *testing.T) {
	cases := []struct{ name, body string }{
		{"missing appointment", `{"providerId":"pr","patientId":"p"}`},
		{"missing provider", `{"appointmentId":"a","patientId":"p"}`},
		{"missing patient", `{"appointmentId":"a","providerId":"pr"}`},
		{"unknown field", `{"appointmentId":"a","providerId":"pr","patientId":"p","x":1}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newExtFakes()
			router := NewRouter(f.deps())
			rec := doRequest(t, router, http.MethodPost, "/api/v1/encounters", tc.body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", rec.Code)
			}
		})
	}
}

func TestEncounter_OpenAuditFailure(t *testing.T) {
	f := newExtFakes()
	f.audit.err = errors.New("audit chain broken")
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodPost, "/api/v1/encounters",
		`{"appointmentId":"appt-1","providerId":"prov-1","patientId":"pat-1"}`)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 (body=%q)", rec.Code, rec.Body.String())
	}
}

func TestEncounter_GetNotFound(t *testing.T) {
	f := newExtFakes()
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodGet, "/api/v1/encounters/missing", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestEncounter_SignNoteSuccess(t *testing.T) {
	f := newExtFakes()
	f.encs.seed(&clinicalmodel.EncounterAggregate{
		ID:               "enc-1",
		Status:           clinicalmodel.EncounterStatusOpen,
		ScopedProviderID: "prov-1",
		ScopedPatientID:  "pat-1",
	})
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodPost, "/api/v1/encounters/enc-1/soap-note",
		`{"providerId":"prov-1","soapNote":"S/O/A/P","diagnoses":[{"code":"J06.9","description":"URI"}]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%q)", rec.Code, rec.Body.String())
	}
	var resp encounterResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if !resp.NoteSigned {
		t.Fatalf("noteSigned = false, want true")
	}
}

func TestEncounter_SignNoteValidation(t *testing.T) {
	cases := []struct{ name, body string }{
		{"missing soapNote", `{"providerId":"prov-1","diagnoses":[{"code":"J06.9"}]}`},
		{"empty diagnoses", `{"providerId":"prov-1","soapNote":"x","diagnoses":[]}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newExtFakes()
			f.encs.seed(&clinicalmodel.EncounterAggregate{ID: "enc-v", Status: clinicalmodel.EncounterStatusOpen, ScopedProviderID: "prov-1"})
			router := NewRouter(f.deps())
			rec := doRequest(t, router, http.MethodPost, "/api/v1/encounters/enc-v/soap-note", tc.body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", rec.Code)
			}
		})
	}
}

func TestEncounter_SignNoteUncodedDiagnosisUnprocessable(t *testing.T) {
	f := newExtFakes()
	f.encs.seed(&clinicalmodel.EncounterAggregate{ID: "enc-u", Status: clinicalmodel.EncounterStatusOpen, ScopedProviderID: "prov-1"})
	router := NewRouter(f.deps())
	// A diagnosis with an empty code is a domain rule violation -> 422.
	rec := doRequest(t, router, http.MethodPost, "/api/v1/encounters/enc-u/soap-note",
		`{"providerId":"prov-1","soapNote":"x","diagnoses":[{"code":"","description":"d"}]}`)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422 (body=%q)", rec.Code, rec.Body.String())
	}
}

func TestEncounter_SignNoteNotFound(t *testing.T) {
	f := newExtFakes()
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodPost, "/api/v1/encounters/missing/soap-note",
		`{"providerId":"prov-1","soapNote":"x","diagnoses":[{"code":"J06.9"}]}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestEncounter_CompleteSuccess(t *testing.T) {
	f := newExtFakes()
	f.encs.seed(&clinicalmodel.EncounterAggregate{
		ID:               "enc-c",
		Status:           clinicalmodel.EncounterStatusOpen,
		ScopedProviderID: "prov-1",
		Note:             &clinicalmodel.ClinicalNote{Content: "note", Signed: true},
		Diagnoses:        []clinicalmodel.Diagnosis{{Code: "J06.9", Description: "URI"}},
	})
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodPost, "/api/v1/encounters/enc-c/completion",
		`{"providerId":"prov-1"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%q)", rec.Code, rec.Body.String())
	}
	var resp encounterResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Status != string(clinicalmodel.EncounterStatusCompleted) {
		t.Fatalf("status = %q, want completed", resp.Status)
	}
}

func TestEncounter_CompleteWithoutSignedNoteUnprocessable(t *testing.T) {
	f := newExtFakes()
	f.encs.seed(&clinicalmodel.EncounterAggregate{ID: "enc-cn", Status: clinicalmodel.EncounterStatusOpen, ScopedProviderID: "prov-1"})
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodPost, "/api/v1/encounters/enc-cn/completion",
		`{"providerId":"prov-1"}`)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422 (body=%q)", rec.Code, rec.Body.String())
	}
}

func TestEncounter_CompleteMalformed(t *testing.T) {
	f := newExtFakes()
	f.encs.seed(&clinicalmodel.EncounterAggregate{ID: "enc-cm", Status: clinicalmodel.EncounterStatusOpen, ScopedProviderID: "prov-1"})
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodPost, "/api/v1/encounters/enc-cm/completion", `{bad`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestEncounter_NotMountedWhenUnwired(t *testing.T) {
	// Base fakes leave Encounters nil, so the route is not mounted at all.
	f := newFakes()
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodGet, "/api/v1/encounters/anything", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (unmounted)", rec.Code)
	}
}
