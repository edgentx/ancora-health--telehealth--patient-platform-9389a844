package rest

import (
	"net/http"
	"testing"

	engagementmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/patientengagement/model"
)

// A whitespace-only path segment trims to empty, so every {id} GET route
// rejects it with a 400 through pathID before touching its repository.
func TestBlankPathID_RejectedAcrossGetRoutes(t *testing.T) {
	f := newExtFakes()
	d := f.deps()
	d.PaymentWebhookSecret = webhookSecret
	router := NewRouter(d)

	paths := []string{
		"/api/v1/lab-orders/%20",
		"/api/v1/prescriptions/%20",
		"/api/v1/insurance-policies/%20",
		"/api/v1/encounters/%20",
		"/api/v1/invoices/%20",
		"/api/v1/payments/%20",
		"/api/v1/provider-schedules/%20",
		"/api/v1/clinic-directories/%20",
		"/api/v1/audit-trails/%20",
		"/api/v1/audit-trails/%20/entries",
	}
	for _, p := range paths {
		t.Run(p, func(t *testing.T) {
			rec := doRequest(t, router, http.MethodGet, p, "")
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (body=%q)", rec.Code, rec.Body.String())
			}
			if er := decodeErr(t, rec); er.Code != codeValidation {
				t.Fatalf("code = %q, want %q", er.Code, codeValidation)
			}
		})
	}
}

func TestPrescription_ComposeMalformed(t *testing.T) {
	f := newFakes()
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodPost, "/api/v1/prescriptions", `{bad`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestPrescription_SafetyCheckImmutableUnprocessable(t *testing.T) {
	f := newFakes()
	rx := composedRx("rx-sci")
	rx.Status = engagementmodel.PrescriptionStatusTransmitted
	f.rx.seed(rx)
	router := NewRouter(f.deps())
	// A transmitted prescription is sealed; re-checking is a rule violation -> 422.
	rec := doRequest(t, router, http.MethodPost, "/api/v1/prescriptions/rx-sci/safety-check", "")
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422 (body=%q)", rec.Code, rec.Body.String())
	}
}
