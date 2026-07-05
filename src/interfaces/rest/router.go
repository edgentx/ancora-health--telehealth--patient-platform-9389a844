package rest

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	adminrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/administrationandanalytics/repository"
	auditrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/auditandcompliance/repository"
	billingrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/billingandinsurance/repository"
	clinicalrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/clinicalrecords/repository"
	engagementrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/patientengagement/repository"
	schedulingrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/scheduling/repository"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/integration/payment"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/integration/pharmacy"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/platform"
)

// AuditSink is the narrow recording seam mutating handlers append a compliance
// entry through after a successful command. It is deliberately minimal — one
// method taking references, never PHI — so any sink (the audit hash chain, a
// SIEM shipper, a test spy) can satisfy it. A nil sink makes recording a no-op,
// which is what keeps the handler unit suite free of an audit dependency.
type AuditSink interface {
	Record(ctx context.Context, actor, resourceRef, action string) error
}

// APIVersion is the single version segment every business route is mounted
// under. Bumping it (and mounting a second tree) is how a breaking API revision
// is introduced without disturbing existing clients.
const APIVersion = "v1"

// HealthChecker is the readiness dependency: anything that can confirm its
// backing store is reachable. It mirrors the port the persistence layer already
// satisfies, so main can hand the MongoDB client straight in.
type HealthChecker interface {
	HealthCheck(ctx context.Context) error
}

// Dependencies is the wiring contract for the router: one repository port per
// exposed aggregate, plus an optional readiness probe. main supplies the
// concrete MongoDB repositories from S-69/S-70; tests supply in-memory fakes.
// Handlers depend on these ports (never on concrete infrastructure), which is
// what keeps the layer unit-testable with httptest.
type Dependencies struct {
	// Health backs /ready. When nil, readiness reports "not configured" so a
	// database-less local run still boots.
	Health HealthChecker

	Appointments      schedulingrepo.AppointmentRepository
	ProviderSchedules schedulingrepo.ProviderScheduleRepository
	LabOrders         clinicalrepo.LabOrderRepository
	Prescriptions     engagementrepo.PrescriptionRepository
	InsurancePolicies billingrepo.InsurancePolicyRepository
	ClinicDirectories adminrepo.ClinicDirectoryRepository
	AuditTrails       auditrepo.AuditTrailRepository

	// The fields below extend the surface with the cross-context flows the S-72
	// adapters and the remaining aggregates back. Each is optional: a nil
	// repository leaves its routes unmounted, so a caller (or a focused test)
	// wires only the contexts it exercises. main and the integration suite wire
	// them all.

	// Encounters backs the clinical encounter-documentation flow.
	Encounters clinicalrepo.EncounterRepository
	// Invoices and Payments back the billing invoice+payment flow. Payment
	// reconciliation additionally flows through the gateway webhook below.
	Invoices billingrepo.InvoiceRepository
	Payments billingrepo.PaymentRepository

	// Pharmacy is the outbound e-prescribing gateway a prescription is
	// transmitted through. When set, POST …/transmission submits to it (which
	// audits the outbound PHI access); when nil, transmission is a domain-only
	// state advance.
	Pharmacy pharmacy.PharmacyGateway

	// PaymentWebhookSecret is the shared HMAC secret the gateway signs webhooks
	// with. When set together with Payments, POST /payment-webhooks is mounted:
	// it verifies the signature and reconciles the payment. PaymentIdempotency
	// makes a redelivered webhook a no-op; a nil store falls back to an in-memory
	// one.
	PaymentWebhookSecret []byte
	PaymentIdempotency   payment.IdempotencyStore

	// Audit is the compliance sink mutating flows append an entry through. When
	// nil, recording is skipped.
	Audit AuditSink
}

// NewRouter builds the versioned chi router: shared middleware, the operational
// endpoints (/health, /ready, /metrics), and every bounded context mounted under
// /api/<version>. Business traffic passes through the telemetry and identity
// middleware; the operational endpoints deliberately do not, so scrapes and
// probes neither emit business spans nor pollute the request metrics.
func NewRouter(deps Dependencies) http.Handler {
	obs := newObservability()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	// Operational endpoints — outside the telemetry/identity chain.
	r.Handle("/metrics", obs.metricsHandler())
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Get("/ready", deps.handleReady)
	r.Get("/version", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, platform.Build())
	})

	// Business surface — versioned, traced, and identity-aware.
	r.Group(func(api chi.Router) {
		api.Use(obs.telemetryMiddleware)
		api.Use(IdentityMiddleware)

		api.Route("/api/"+APIVersion, func(v1 chi.Router) {
			schedulingAPI{appointments: deps.Appointments, schedules: deps.ProviderSchedules}.mount(v1)
			clinicalAPI{labOrders: deps.LabOrders}.mount(v1)
			engagementAPI{prescriptions: deps.Prescriptions, pharmacy: deps.Pharmacy}.mount(v1)
			billingAPI{policies: deps.InsurancePolicies}.mount(v1)
			adminAPI{directories: deps.ClinicDirectories}.mount(v1)
			auditAPI{trails: deps.AuditTrails}.mount(v1)

			// Optional cross-context flows: mounted only when their backing
			// aggregate repository is wired, so a partial deployment (or a focused
			// test) exposes exactly the contexts it configured.
			if deps.Encounters != nil {
				encounterAPI{encounters: deps.Encounters, audit: deps.Audit}.mount(v1)
			}
			if deps.Invoices != nil || deps.Payments != nil {
				billingFlowAPI{
					invoices:       deps.Invoices,
					payments:       deps.Payments,
					audit:          deps.Audit,
					webhookSecret:  deps.PaymentWebhookSecret,
					webhookIdempot: deps.PaymentIdempotency,
				}.mount(v1)
			}
		})
	})

	return r
}

// handleReady is the readiness probe: it pings the backing store and reports 503
// when the database is unconfigured or unreachable, 200 once it answers.
func (deps Dependencies) handleReady(w http.ResponseWriter, req *http.Request) {
	if deps.Health == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "unavailable",
			"reason": "database not configured",
		})
		return
	}
	ctx, cancel := context.WithTimeout(req.Context(), 2*time.Second)
	defer cancel()
	if err := deps.Health.HealthCheck(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "unavailable",
			"reason": "database unreachable",
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
