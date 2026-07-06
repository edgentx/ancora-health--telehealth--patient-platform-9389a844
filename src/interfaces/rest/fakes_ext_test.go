package rest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	billingmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/billingandinsurance/model"
	clinicalmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/clinicalrecords/model"
	schedmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/scheduling/model"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/integration/payment"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/integration/pharmacy"
)

// extFakes augments the base fakes with the optional cross-context aggregates
// (encounters, invoices, payments) plus an audit spy, so the whole handler
// surface can be mounted and driven.
type extFakes struct {
	fakes
	encs  *fakeRepo[clinicalmodel.EncounterAggregate]
	invs  *fakeRepo[billingmodel.InvoiceAggregate]
	pays  *fakeRepo[billingmodel.PaymentAggregate]
	audit *fakeAudit
}

func newExtFakes() extFakes {
	return extFakes{
		fakes: newFakes(),
		encs:  newFakeRepo(func(e *clinicalmodel.EncounterAggregate) string { return e.ID }),
		invs:  newFakeRepo(func(i *billingmodel.InvoiceAggregate) string { return i.ID }),
		pays:  newFakeRepo(func(p *billingmodel.PaymentAggregate) string { return p.ID }),
		audit: &fakeAudit{},
	}
}

// deps wires the extended fakes onto the router Dependencies, mounting every
// optional flow (encounters, invoices+payments) and the audit sink.
func (f extFakes) deps() Dependencies {
	d := f.fakes.deps()
	d.Encounters = f.encs
	d.Invoices = f.invs
	d.Payments = f.pays
	d.Audit = f.audit
	return d
}

// fakeAudit is an AuditSink spy that records calls and can force a recording
// failure so the "broken audit chain -> 500" branch is reachable.
type fakeAudit struct {
	err   error
	calls int
	last  struct {
		actor, resourceRef, action string
	}
}

func (f *fakeAudit) Record(_ context.Context, actor, resourceRef, action string) error {
	f.calls++
	f.last.actor, f.last.resourceRef, f.last.action = actor, resourceRef, action
	return f.err
}

var _ AuditSink = (*fakeAudit)(nil)

// fakePharmacy is a PharmacyGateway double whose Submit outcome and error are
// programmable, so the transmit handler's gateway-accepted, gateway-rejected and
// transport-failure branches can each be driven.
type fakePharmacy struct {
	result pharmacy.TransmissionResult
	err    error
	calls  int
}

func (f *fakePharmacy) Submit(_ context.Context, _ pharmacy.ProviderContext, _ pharmacy.PrescriptionOrder) (pharmacy.TransmissionResult, error) {
	f.calls++
	return f.result, f.err
}

func (f *fakePharmacy) Cancel(_ context.Context, _ pharmacy.ProviderContext, _ pharmacy.CancelOrder) (pharmacy.TransmissionResult, error) {
	return f.result, f.err
}

var _ pharmacy.PharmacyGateway = (*fakePharmacy)(nil)

// bookingRepo wraps the in-memory appointment fake with the optional slotBooker
// capability, so the scheduling handler exercises the exclusive-hold path
// (reserveSlot's booker branch) rather than a plain Save.
type bookingRepo struct {
	*fakeRepo[schedmodel.AppointmentAggregate]
	bookErr error
}

func (b bookingRepo) Book(ctx context.Context, a *schedmodel.AppointmentAggregate) error {
	if b.bookErr != nil {
		return b.bookErr
	}
	return b.fakeRepo.Save(ctx, a)
}

var _ slotBooker = bookingRepo{}

// doReq issues a request with arbitrary headers (defaulting to the trusted
// identity headers), for cases doRequest's fixed header set cannot express.
func doReq(t *testing.T, h http.Handler, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set(HeaderSubject, "user-123")
	req.Header.Set(HeaderRoles, "provider,admin")
	req.Header.Set(HeaderTenant, "tenant-a")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// signedWebhookBody signs body with the webhook secret so the payment-webhook
// handler accepts it, returning the header map to attach.
func signedWebhookHeader(secret []byte, body string) map[string]string {
	sig := payment.NewVerifier(secret).Sign([]byte(body))
	return map[string]string{"X-Signature": sig}
}
