package apiintegration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/integration/payment"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/interfaces/rest"
)

// idResponse captures the id/status every create endpoint returns.
type idResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// mustStatus fails the test unless rec carries the wanted status.
func mustStatus(t *testing.T, rec *httptest.ResponseRecorder, want int, what string) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("%s: status = %d, want %d (body=%q)", what, rec.Code, want, rec.Body.String())
	}
}

// createEncounter opens an encounter and returns its id, asserting success.
func createEncounter(t *testing.T, env *environment, subject string) idResponse {
	t.Helper()
	headers := map[string]string(nil)
	if subject != "" {
		headers = map[string]string{rest.HeaderSubject: subject}
	}
	rec := env.request(http.MethodPost, "/api/v1/encounters",
		`{"appointmentId":"appt-e2e","providerId":"prov-1","patientId":"pat-1"}`, headers)
	mustStatus(t, rec, http.StatusCreated, "open encounter")
	var enc idResponse
	if err := decodeInto(rec.Body.Bytes(), &enc); err != nil {
		t.Fatal(err)
	}
	if enc.Status != "open" {
		t.Fatalf("encounter status = %q, want open", enc.Status)
	}
	return enc
}

// TestEncounterFlow_CreatesAndAudits proves the clinical encounter-documentation
// flow end-to-end: opening provisions the encounter, it round-trips through GET,
// and the mutation produced an audit entry.
func TestEncounterFlow_CreatesAndAudits(t *testing.T) {
	env := newEnv(t)
	enc := createEncounter(t, env, "")

	get := env.request(http.MethodGet, "/api/v1/encounters/"+enc.ID, "", nil)
	mustStatus(t, get, http.StatusOK, "get encounter")

	count, found, status := env.auditEntryCount(enc.ID)
	if !found || count < 1 {
		t.Fatalf("encounter flow produced no audit entry: found=%v count=%d status=%d", found, count, status)
	}
}

// TestPrescriptionFlow_SubmitsViaPharmacyAndAudits proves prescription
// submission end-to-end through the pharmacy adapter: composing then
// transmitting drives the (stubbed-upstream) pharmacy gateway, which audits the
// outbound PHI access, so the prescription ends transmitted with an audit entry.
func TestPrescriptionFlow_SubmitsViaPharmacyAndAudits(t *testing.T) {
	env := newEnv(t)

	compose := env.request(http.MethodPost, "/api/v1/prescriptions",
		`{"patientId":"pat-1","providerId":"prov-1","medication":"amoxicillin","dosage":"500mg"}`, nil)
	mustStatus(t, compose, http.StatusCreated, "compose prescription")
	var rx idResponse
	if err := decodeInto(compose.Body.Bytes(), &rx); err != nil {
		t.Fatal(err)
	}

	transmit := env.request(http.MethodPost, "/api/v1/prescriptions/"+rx.ID+"/transmission",
		`{"pharmacyId":"pharmacy-42"}`, nil)
	mustStatus(t, transmit, http.StatusOK, "transmit prescription")
	var transmitted idResponse
	if err := decodeInto(transmit.Body.Bytes(), &transmitted); err != nil {
		t.Fatal(err)
	}
	if transmitted.Status != "transmitted" {
		t.Fatalf("prescription status = %q, want transmitted", transmitted.Status)
	}

	// The pharmacy adapter records the outbound submission against the
	// prescription's audit trail.
	count, found, status := env.auditEntryCount(rx.ID)
	if !found || count < 1 {
		t.Fatalf("pharmacy submission produced no audit entry: found=%v count=%d status=%d", found, count, status)
	}
}

// TestBillingFlow_InvoicePaymentAndWebhookReconcile proves the billing flow
// end-to-end: an invoice is generated, a tokenized payment initiated against it,
// and the payment advanced to reconciled by a signed gateway webhook. Each
// mutation, including the webhook-driven reconciliation, produces an audit entry.
func TestBillingFlow_InvoicePaymentAndWebhookReconcile(t *testing.T) {
	env := newEnv(t)

	invRec := env.request(http.MethodPost, "/api/v1/invoices",
		`{"encounterId":"enc-e2e","policyId":"pol-e2e","lineItems":[{"description":"telehealth visit","amountCents":15000}]}`, nil)
	mustStatus(t, invRec, http.StatusCreated, "generate invoice")
	var invoice idResponse
	if err := decodeInto(invRec.Body.Bytes(), &invoice); err != nil {
		t.Fatal(err)
	}

	payRec := env.request(http.MethodPost, "/api/v1/payments",
		fmt.Sprintf(`{"invoiceId":%q,"paymentToken":"tok_visa_e2e","amountCents":15000}`, invoice.ID), nil)
	mustStatus(t, payRec, http.StatusCreated, "initiate payment")
	var pay idResponse
	if err := decodeInto(payRec.Body.Bytes(), &pay); err != nil {
		t.Fatal(err)
	}
	if pay.Status != "initiated" {
		t.Fatalf("payment status = %q, want initiated", pay.Status)
	}

	// Play the gateway: deliver a webhook signed with the shared secret.
	payload := fmt.Sprintf(`{"id":"evt-e2e-1","type":"charge.succeeded","data":{"paymentId":%q,"status":"succeeded"}}`, pay.ID)
	signature := payment.NewVerifier(webhookSecret).Sign([]byte(payload))
	whRec := env.request(http.MethodPost, "/api/v1/payment-webhooks", payload,
		map[string]string{"X-Signature": signature})
	mustStatus(t, whRec, http.StatusOK, "payment webhook")

	// The payment is now reconciled.
	getPay := env.request(http.MethodGet, "/api/v1/payments/"+pay.ID, "", nil)
	mustStatus(t, getPay, http.StatusOK, "get payment")
	var reconciled idResponse
	if err := decodeInto(getPay.Body.Bytes(), &reconciled); err != nil {
		t.Fatal(err)
	}
	if reconciled.Status != "reconciled" {
		t.Fatalf("payment status after webhook = %q, want reconciled", reconciled.Status)
	}

	// An unsigned (or mis-signed) webhook must be rejected, never applied.
	bad := env.request(http.MethodPost, "/api/v1/payment-webhooks", payload,
		map[string]string{"X-Signature": "deadbeef"})
	if bad.Code == http.StatusOK {
		t.Fatalf("mis-signed webhook was accepted (status %d)", bad.Code)
	}

	// Invoice generation, payment initiation, and reconciliation all audited.
	if c, found, _ := env.auditEntryCount(invoice.ID); !found || c < 1 {
		t.Fatalf("invoice generation produced no audit entry (found=%v count=%d)", found, c)
	}
	if c, found, _ := env.auditEntryCount(pay.ID); !found || c < 2 {
		t.Fatalf("payment initiate+reconcile should produce >=2 audit entries (found=%v count=%d)", found, c)
	}
}

// TestTrustedHeaders_ThreadIdentityIntoAudit proves the trusted-edge headers are
// the identity source: an encounter opened as a specific subject records that
// subject as the audit actor, with no authentication logic in the path.
func TestTrustedHeaders_ThreadIdentityIntoAudit(t *testing.T) {
	env := newEnv(t)
	const actor = "dr-house-tenant-b"
	enc := createEncounter(t, env, actor)

	entriesRec := env.request(http.MethodGet, "/api/v1/audit-trails/audit_"+enc.ID+"/entries", "", nil)
	mustStatus(t, entriesRec, http.StatusOK, "read audit entries")
	var entries []struct {
		ActorContext string `json:"actorContext"`
		Action       string `json:"action"`
	}
	if err := decodeInto(entriesRec.Body.Bytes(), &entries); err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("no audit entries for opened encounter")
	}
	if entries[0].ActorContext != actor {
		t.Fatalf("audit actor = %q, want the trusted subject %q", entries[0].ActorContext, actor)
	}
}

// TestCrossContext_RepresentativeSurface boots the whole router and exercises a
// representative endpoint in each remaining bounded context, proving the surface
// wires together and every context answers end-to-end over real repositories.
func TestCrossContext_RepresentativeSurface(t *testing.T) {
	env := newEnv(t)

	// Scheduling: publish provider availability.
	sched := env.request(http.MethodPost, "/api/v1/provider-schedules",
		`{"providerId":"prov-1","windows":["2026-09-01T09:00Z/2026-09-01T12:00Z"]}`, nil)
	mustStatus(t, sched, http.StatusCreated, "publish availability")

	// Clinical records: place a lab order.
	lab := env.request(http.MethodPost, "/api/v1/lab-orders",
		`{"patientId":"pat-1","providerId":"prov-1","testCode":"CBC"}`, nil)
	mustStatus(t, lab, http.StatusCreated, "place lab order")

	// Billing/insurance: register a policy.
	pol := env.request(http.MethodPost, "/api/v1/insurance-policies",
		`{"patientId":"pat-1","payerIdentifier":"payer-1","effectiveDates":{"start":"2026-01-01","end":"2026-12-31"}}`, nil)
	mustStatus(t, pol, http.StatusCreated, "register policy")

	// Administration: create a clinic directory.
	dir := env.request(http.MethodPost, "/api/v1/clinic-directories", "", nil)
	mustStatus(t, dir, http.StatusCreated, "create clinic directory")

	// Operational endpoints answer too.
	if rec := env.request(http.MethodGet, "/health", "", nil); rec.Code != http.StatusOK {
		t.Fatalf("/health status = %d, want 200", rec.Code)
	}
}
