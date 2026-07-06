package rest

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	billingmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/billingandinsurance/model"
)

// --- insurance policies ---

func TestInsurancePolicy_RegisterValidation(t *testing.T) {
	cases := []struct{ name, body string }{
		{"missing patient", `{"payerIdentifier":"p","effectiveDates":{"start":"a","end":"b"}}`},
		{"missing payer", `{"patientId":"p","effectiveDates":{"start":"a","end":"b"}}`},
		{"missing start", `{"patientId":"p","payerIdentifier":"pay","effectiveDates":{"end":"b"}}`},
		{"missing end", `{"patientId":"p","payerIdentifier":"pay","effectiveDates":{"start":"a"}}`},
		{"malformed", `{`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakes()
			router := NewRouter(f.deps())
			rec := doRequest(t, router, http.MethodPost, "/api/v1/insurance-policies", tc.body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", rec.Code)
			}
		})
	}
}

func TestInsurancePolicy_GetSuccessAndNotFound(t *testing.T) {
	f := newFakes()
	f.pols.seed(&billingmodel.InsurancePolicyAggregate{ID: "pol-1", Status: billingmodel.PolicyStatusRegistered, PatientID: "pat-1"})
	router := NewRouter(f.deps())
	if rec := doRequest(t, router, http.MethodGet, "/api/v1/insurance-policies/pol-1", ""); rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if rec := doRequest(t, router, http.MethodGet, "/api/v1/insurance-policies/missing", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestInsurancePolicy_VerifyEligibilitySuccess(t *testing.T) {
	f := newFakes()
	f.pols.seed(&billingmodel.InsurancePolicyAggregate{ID: "pol-e", Status: billingmodel.PolicyStatusRegistered, PatientID: "pat-1"})
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodPost, "/api/v1/insurance-policies/pol-e/eligibility",
		`{"serviceDate":"2026-07-10"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%q)", rec.Code, rec.Body.String())
	}
	var resp insurancePolicyResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.VerifiedServiceDate != "2026-07-10" {
		t.Fatalf("verifiedServiceDate = %q", resp.VerifiedServiceDate)
	}
}

func TestInsurancePolicy_VerifyEligibilityMissingDate(t *testing.T) {
	f := newFakes()
	f.pols.seed(&billingmodel.InsurancePolicyAggregate{ID: "pol-m", Status: billingmodel.PolicyStatusRegistered})
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodPost, "/api/v1/insurance-policies/pol-m/eligibility", `{}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestInsurancePolicy_VerifyEligibilityNotFound(t *testing.T) {
	f := newFakes()
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodPost, "/api/v1/insurance-policies/missing/eligibility",
		`{"serviceDate":"2026-07-10"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestInsurancePolicy_VerifyEligibilityUnprocessable(t *testing.T) {
	f := newFakes()
	// A policy flagged with no verified eligibility refuses the check -> 422.
	f.pols.seed(&billingmodel.InsurancePolicyAggregate{ID: "pol-x", Status: billingmodel.PolicyStatusRegistered, EligibilityNotVerified: true})
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodPost, "/api/v1/insurance-policies/pol-x/eligibility",
		`{"serviceDate":"2026-07-10"}`)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422 (body=%q)", rec.Code, rec.Body.String())
	}
}

// --- invoices ---

func TestInvoice_GenerateSuccess(t *testing.T) {
	f := newExtFakes()
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodPost, "/api/v1/invoices",
		`{"encounterId":"enc-1","policyId":"pol-1","lineItems":[{"description":"visit","amountCents":15000}]}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body=%q)", rec.Code, rec.Body.String())
	}
	var resp invoiceResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID == "" || len(resp.LineItems) != 1 {
		t.Fatalf("unexpected: %+v", resp)
	}
	if f.audit.last.action != "invoice.generate" {
		t.Fatalf("audit action = %q", f.audit.last.action)
	}
	if get := doRequest(t, router, http.MethodGet, "/api/v1/invoices/"+resp.ID, ""); get.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want 200", get.Code)
	}
}

func TestInvoice_GenerateValidation(t *testing.T) {
	cases := []struct{ name, body string }{
		{"missing encounter", `{"policyId":"pol","lineItems":[{"description":"v","amountCents":1}]}`},
		{"missing policy", `{"encounterId":"enc","lineItems":[{"description":"v","amountCents":1}]}`},
		{"empty lineItems", `{"encounterId":"enc","policyId":"pol","lineItems":[]}`},
		{"malformed", `{`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newExtFakes()
			router := NewRouter(f.deps())
			rec := doRequest(t, router, http.MethodPost, "/api/v1/invoices", tc.body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", rec.Code)
			}
		})
	}
}

func TestInvoice_GenerateAuditFailure(t *testing.T) {
	f := newExtFakes()
	f.audit.err = errors.New("audit down")
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodPost, "/api/v1/invoices",
		`{"encounterId":"enc-1","policyId":"pol-1","lineItems":[{"description":"visit","amountCents":15000}]}`)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 (body=%q)", rec.Code, rec.Body.String())
	}
}

func TestInvoice_GetNotFound(t *testing.T) {
	f := newExtFakes()
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodGet, "/api/v1/invoices/missing", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestInvoice_ApplyAdjustmentSuccess(t *testing.T) {
	f := newExtFakes()
	f.invs.seed(&billingmodel.InvoiceAggregate{ID: "inv-1", Status: billingmodel.InvoiceStatusGenerated})
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodPost, "/api/v1/invoices/inv-1/adjustment",
		`{"verified":true,"coverageCents":10000,"copayCents":2000}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%q)", rec.Code, rec.Body.String())
	}
	var resp invoiceResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Status != string(billingmodel.InvoiceStatusAdjusted) || resp.CoverageCents != 10000 {
		t.Fatalf("unexpected: %+v", resp)
	}
	if f.audit.last.action != "invoice.adjust" {
		t.Fatalf("audit action = %q", f.audit.last.action)
	}
}

func TestInvoice_ApplyAdjustmentUnverifiedUnprocessable(t *testing.T) {
	f := newExtFakes()
	f.invs.seed(&billingmodel.InvoiceAggregate{ID: "inv-u", Status: billingmodel.InvoiceStatusGenerated})
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodPost, "/api/v1/invoices/inv-u/adjustment",
		`{"verified":false,"coverageCents":1,"copayCents":1}`)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422 (body=%q)", rec.Code, rec.Body.String())
	}
}

func TestInvoice_ApplyAdjustmentNotFound(t *testing.T) {
	f := newExtFakes()
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodPost, "/api/v1/invoices/missing/adjustment",
		`{"verified":true,"coverageCents":1,"copayCents":1}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestInvoice_ApplyAdjustmentMalformed(t *testing.T) {
	f := newExtFakes()
	f.invs.seed(&billingmodel.InvoiceAggregate{ID: "inv-m", Status: billingmodel.InvoiceStatusGenerated})
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodPost, "/api/v1/invoices/inv-m/adjustment", `{bad`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

// --- payments ---

func TestPayment_InitiateSuccess(t *testing.T) {
	f := newExtFakes()
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodPost, "/api/v1/payments",
		`{"invoiceId":"inv-1","paymentToken":"tok_abc","amountCents":12000}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body=%q)", rec.Code, rec.Body.String())
	}
	var resp paymentResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != string(billingmodel.PaymentStatusInitiated) || resp.AmountCents != 12000 {
		t.Fatalf("unexpected: %+v", resp)
	}
	if f.audit.last.action != "payment.initiate" {
		t.Fatalf("audit action = %q", f.audit.last.action)
	}
	if get := doRequest(t, router, http.MethodGet, "/api/v1/payments/"+resp.ID, ""); get.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want 200", get.Code)
	}
}

func TestPayment_InitiateValidation(t *testing.T) {
	cases := []struct{ name, body string }{
		{"missing invoice", `{"paymentToken":"tok","amountCents":100}`},
		{"missing token", `{"invoiceId":"inv","amountCents":100}`},
		{"non-positive amount", `{"invoiceId":"inv","paymentToken":"tok","amountCents":0}`},
		{"negative amount", `{"invoiceId":"inv","paymentToken":"tok","amountCents":-5}`},
		{"malformed", `{`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newExtFakes()
			router := NewRouter(f.deps())
			rec := doRequest(t, router, http.MethodPost, "/api/v1/payments", tc.body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", rec.Code)
			}
		})
	}
}

func TestPayment_GetNotFound(t *testing.T) {
	f := newExtFakes()
	router := NewRouter(f.deps())
	rec := doRequest(t, router, http.MethodGet, "/api/v1/payments/missing", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

// --- invoice-only wiring: payments routes absent ---

func TestBillingFlow_PaymentsUnmountedWhenNil(t *testing.T) {
	f := newFakes()
	d := f.deps()
	d.Invoices = newFakeRepo(func(i *billingmodel.InvoiceAggregate) string { return i.ID })
	// Payments left nil: the /payments routes are not mounted.
	router := NewRouter(d)
	if rec := doRequest(t, router, http.MethodGet, "/api/v1/payments/x", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (payments unmounted)", rec.Code)
	}
	// but invoices are mounted
	if rec := doRequest(t, router, http.MethodGet, "/api/v1/invoices/x", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("invoices route should be mounted (404 for missing), got %d", rec.Code)
	}
}
