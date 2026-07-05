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
)

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

	// Business surface — versioned, traced, and identity-aware.
	r.Group(func(api chi.Router) {
		api.Use(obs.telemetryMiddleware)
		api.Use(IdentityMiddleware)

		api.Route("/api/"+APIVersion, func(v1 chi.Router) {
			schedulingAPI{appointments: deps.Appointments, schedules: deps.ProviderSchedules}.mount(v1)
			clinicalAPI{labOrders: deps.LabOrders}.mount(v1)
			engagementAPI{prescriptions: deps.Prescriptions}.mount(v1)
			billingAPI{policies: deps.InsurancePolicies}.mount(v1)
			adminAPI{directories: deps.ClinicDirectories}.mount(v1)
			auditAPI{trails: deps.AuditTrails}.mount(v1)
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
