package rest

import (
	"errors"
	"net/http"
	"testing"

	adminmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/administrationandanalytics/model"
	billingmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/billingandinsurance/model"
	clinicalmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/clinicalrecords/model"
	engagementmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/patientengagement/model"
	schedmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/scheduling/model"
)

var errSave = errors.New("mongo: write failed")

// Every mutating handler persists through Save; a Save failure that is neither
// not-found nor a conflict must surface as 500. These drive that branch across
// the handler surface.

func TestSaveError_Scheduling(t *testing.T) {
	t.Run("book", func(t *testing.T) {
		f := newFakes()
		f.appts.seed(&schedmodel.AppointmentAggregate{ID: "a1", Status: schedmodel.AppointmentStatusHeld})
		f.appts.saveErr = errSave
		router := NewRouter(f.deps())
		rec := doRequest(t, router, http.MethodPost, "/api/v1/appointments/a1/booking",
			`{"holdToken":"t","patientId":"p","reason":"r"}`)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}
	})
	t.Run("cancel", func(t *testing.T) {
		f := newFakes()
		f.appts.seed(&schedmodel.AppointmentAggregate{ID: "a2", Status: schedmodel.AppointmentStatusHeld})
		f.appts.saveErr = errSave
		router := NewRouter(f.deps())
		rec := doRequest(t, router, http.MethodPost, "/api/v1/appointments/a2/cancellation", `{"reason":"r"}`)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}
	})
	t.Run("reschedule", func(t *testing.T) {
		f := newFakes()
		f.appts.seed(&schedmodel.AppointmentAggregate{ID: "a3", Status: schedmodel.AppointmentStatusHeld})
		f.appts.saveErr = errSave
		router := NewRouter(f.deps())
		rec := doRequest(t, router, http.MethodPost, "/api/v1/appointments/a3/reschedule", `{"newTimeSlot":"s"}`)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}
	})
	t.Run("publishAvailability", func(t *testing.T) {
		f := newFakes()
		f.scheds.saveErr = errSave
		router := NewRouter(f.deps())
		rec := doRequest(t, router, http.MethodPost, "/api/v1/provider-schedules",
			`{"providerId":"p","windows":["w"]}`)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}
	})
}

func TestMalformedJSON_SchedulingSubresources(t *testing.T) {
	f := newFakes()
	f.appts.seed(&schedmodel.AppointmentAggregate{ID: "m1", Status: schedmodel.AppointmentStatusHeld})
	router := NewRouter(f.deps())
	for _, sub := range []string{"booking", "reschedule"} {
		rec := doRequest(t, router, http.MethodPost, "/api/v1/appointments/m1/"+sub, `{bad`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s status = %d, want 400", sub, rec.Code)
		}
	}
}

func TestSaveError_Clinical(t *testing.T) {
	t.Run("labOrder place", func(t *testing.T) {
		f := newFakes()
		f.labs.saveErr = errSave
		router := NewRouter(f.deps())
		rec := doRequest(t, router, http.MethodPost, "/api/v1/lab-orders",
			`{"patientId":"p","providerId":"pr","testCode":"CBC"}`)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}
	})
	t.Run("compose", func(t *testing.T) {
		f := newFakes()
		f.rx.saveErr = errSave
		router := NewRouter(f.deps())
		rec := doRequest(t, router, http.MethodPost, "/api/v1/prescriptions",
			`{"patientId":"p","providerId":"pr","medication":"m","dosage":"d"}`)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}
	})
	t.Run("safetyCheck", func(t *testing.T) {
		f := newFakes()
		f.rx.seed(composedRx("rx-se"))
		f.rx.saveErr = errSave
		router := NewRouter(f.deps())
		rec := doRequest(t, router, http.MethodPost, "/api/v1/prescriptions/rx-se/safety-check", "")
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}
	})
}

func TestSaveError_Encounter(t *testing.T) {
	t.Run("open", func(t *testing.T) {
		f := newExtFakes()
		f.encs.saveErr = errSave
		router := NewRouter(f.deps())
		rec := doRequest(t, router, http.MethodPost, "/api/v1/encounters",
			`{"appointmentId":"a","providerId":"pr","patientId":"pt"}`)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}
	})
	t.Run("signNote save + malformed", func(t *testing.T) {
		f := newExtFakes()
		f.encs.seed(&clinicalmodel.EncounterAggregate{ID: "e-se", Status: clinicalmodel.EncounterStatusOpen, ScopedProviderID: "prov-1"})
		f.encs.saveErr = errSave
		router := NewRouter(f.deps())
		rec := doRequest(t, router, http.MethodPost, "/api/v1/encounters/e-se/soap-note",
			`{"providerId":"prov-1","soapNote":"x","diagnoses":[{"code":"J06.9"}]}`)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}
		// malformed sign-note body
		f.encs.saveErr = nil
		mrec := doRequest(t, router, http.MethodPost, "/api/v1/encounters/e-se/soap-note", `{bad`)
		if mrec.Code != http.StatusBadRequest {
			t.Fatalf("malformed status = %d, want 400", mrec.Code)
		}
	})
	t.Run("complete save + not found", func(t *testing.T) {
		f := newExtFakes()
		f.encs.seed(&clinicalmodel.EncounterAggregate{
			ID: "e-cs", Status: clinicalmodel.EncounterStatusOpen, ScopedProviderID: "prov-1",
			Note:      &clinicalmodel.ClinicalNote{Signed: true},
			Diagnoses: []clinicalmodel.Diagnosis{{Code: "J06.9"}},
		})
		f.encs.saveErr = errSave
		router := NewRouter(f.deps())
		rec := doRequest(t, router, http.MethodPost, "/api/v1/encounters/e-cs/completion", `{"providerId":"prov-1"}`)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}
		nf := doRequest(t, router, http.MethodPost, "/api/v1/encounters/nope/completion", `{"providerId":"prov-1"}`)
		if nf.Code != http.StatusNotFound {
			t.Fatalf("not found status = %d, want 404", nf.Code)
		}
	})
	t.Run("signNote audit failure", func(t *testing.T) {
		f := newExtFakes()
		f.encs.seed(&clinicalmodel.EncounterAggregate{ID: "e-af", Status: clinicalmodel.EncounterStatusOpen, ScopedProviderID: "prov-1"})
		f.audit.err = errors.New("audit down")
		router := NewRouter(f.deps())
		rec := doRequest(t, router, http.MethodPost, "/api/v1/encounters/e-af/soap-note",
			`{"providerId":"prov-1","soapNote":"x","diagnoses":[{"code":"J06.9"}]}`)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}
	})
}

func TestSaveError_Billing(t *testing.T) {
	t.Run("register policy", func(t *testing.T) {
		f := newFakes()
		f.pols.saveErr = errSave
		router := NewRouter(f.deps())
		rec := doRequest(t, router, http.MethodPost, "/api/v1/insurance-policies",
			`{"patientId":"p","payerIdentifier":"pay","effectiveDates":{"start":"a","end":"b"}}`)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}
	})
	t.Run("verifyEligibility save + malformed", func(t *testing.T) {
		f := newFakes()
		f.pols.seed(&billingmodel.InsurancePolicyAggregate{ID: "pol-se", Status: billingmodel.PolicyStatusRegistered})
		f.pols.saveErr = errSave
		router := NewRouter(f.deps())
		rec := doRequest(t, router, http.MethodPost, "/api/v1/insurance-policies/pol-se/eligibility",
			`{"serviceDate":"2026-07-10"}`)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}
		f.pols.saveErr = nil
		mrec := doRequest(t, router, http.MethodPost, "/api/v1/insurance-policies/pol-se/eligibility", `{bad`)
		if mrec.Code != http.StatusBadRequest {
			t.Fatalf("malformed status = %d, want 400", mrec.Code)
		}
	})
	t.Run("generate invoice", func(t *testing.T) {
		f := newExtFakes()
		f.invs.saveErr = errSave
		router := NewRouter(f.deps())
		rec := doRequest(t, router, http.MethodPost, "/api/v1/invoices",
			`{"encounterId":"e","policyId":"p","lineItems":[{"description":"v","amountCents":1}]}`)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}
	})
	t.Run("apply adjustment save", func(t *testing.T) {
		f := newExtFakes()
		f.invs.seed(&billingmodel.InvoiceAggregate{ID: "inv-se", Status: billingmodel.InvoiceStatusGenerated})
		f.invs.saveErr = errSave
		router := NewRouter(f.deps())
		rec := doRequest(t, router, http.MethodPost, "/api/v1/invoices/inv-se/adjustment",
			`{"verified":true,"coverageCents":1,"copayCents":1}`)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}
	})
	t.Run("initiate payment save", func(t *testing.T) {
		f := newExtFakes()
		f.pays.saveErr = errSave
		router := NewRouter(f.deps())
		rec := doRequest(t, router, http.MethodPost, "/api/v1/payments",
			`{"invoiceId":"i","paymentToken":"t","amountCents":100}`)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}
	})
	t.Run("invoice audit failure on adjustment", func(t *testing.T) {
		f := newExtFakes()
		f.invs.seed(&billingmodel.InvoiceAggregate{ID: "inv-af", Status: billingmodel.InvoiceStatusGenerated})
		f.audit.err = errors.New("audit down")
		router := NewRouter(f.deps())
		rec := doRequest(t, router, http.MethodPost, "/api/v1/invoices/inv-af/adjustment",
			`{"verified":true,"coverageCents":1,"copayCents":1}`)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}
	})
	t.Run("payment audit failure on initiate", func(t *testing.T) {
		f := newExtFakes()
		f.audit.err = errors.New("audit down")
		router := NewRouter(f.deps())
		rec := doRequest(t, router, http.MethodPost, "/api/v1/payments",
			`{"invoiceId":"i","paymentToken":"t","amountCents":100}`)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}
	})
}

func TestSaveError_Administration(t *testing.T) {
	t.Run("create", func(t *testing.T) {
		f := newFakes()
		f.dirs.saveErr = errSave
		router := NewRouter(f.deps())
		rec := doRequest(t, router, http.MethodPost, "/api/v1/clinic-directories", "")
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}
	})
	t.Run("registerProvider", func(t *testing.T) {
		f := newFakes()
		f.dirs.seed(&adminmodel.ClinicDirectoryAggregate{ID: "d-rp"})
		f.dirs.saveErr = errSave
		router := NewRouter(f.deps())
		rec := doRequest(t, router, http.MethodPost, "/api/v1/clinic-directories/d-rp/providers",
			`{"providerId":"p","specialties":["c"],"clinicIds":["cl"]}`)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}
	})
	t.Run("manageSpecialty save + malformed", func(t *testing.T) {
		f := newFakes()
		f.dirs.seed(&adminmodel.ClinicDirectoryAggregate{ID: "d-ms"})
		f.dirs.saveErr = errSave
		router := NewRouter(f.deps())
		rec := doRequest(t, router, http.MethodPost, "/api/v1/clinic-directories/d-ms/specialties",
			`{"specialtyCode":"c","displayName":"n"}`)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}
		f.dirs.saveErr = nil
		mrec := doRequest(t, router, http.MethodPost, "/api/v1/clinic-directories/d-ms/specialties", `{bad`)
		if mrec.Code != http.StatusBadRequest {
			t.Fatalf("malformed status = %d, want 400", mrec.Code)
		}
	})
	t.Run("configureClinic save", func(t *testing.T) {
		f := newFakes()
		f.dirs.seed(&adminmodel.ClinicDirectoryAggregate{ID: "d-cc"})
		f.dirs.saveErr = errSave
		router := NewRouter(f.deps())
		rec := doRequest(t, router, http.MethodPost, "/api/v1/clinic-directories/d-cc/clinics",
			`{"clinicIdentity":"c","operatingHours":"h"}`)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}
	})
}

// Prescription transmit conflict when already transmitted (immutable) -> 422.
func TestPrescription_TransmitAlreadyTransmitted(t *testing.T) {
	f := newFakes()
	rx := composedRx("rx-imm")
	rx.Status = engagementmodel.PrescriptionStatusTransmitted
	f.rx.seed(rx)
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodPost, "/api/v1/prescriptions/rx-imm/transmission",
		`{"pharmacyId":"pharm-9"}`)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422 (body=%q)", rec.Code, rec.Body.String())
	}
}
