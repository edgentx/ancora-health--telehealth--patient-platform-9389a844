// Package apiintegration is the API-level integration test suite (S-75). It
// boots the real chi router over the real repository types and exercises
// representative flows across every bounded context end-to-end through HTTP —
// appointment booking with double-book prevention, clinical encounter
// documentation, prescription submission through the pharmacy adapter, and
// invoice/payment reconciled by a signed gateway webhook — asserting each
// mutating flow produces an audit entry.
//
// # Infrastructure
//
// The suite runs against real MongoDB and real Redis when the environment names
// them (MONGODB_URI and ANCO_REDIS_ADDR), which is how CI runs it: the Makefile
// `integration` target and the CI workflow spin up ephemeral Mongo and Redis
// containers and point those variables at them. With neither set the suite
// still runs fully in-process against the in-memory store and slot locker — the
// same real repository and locking *types* the service wires in production, so
// every developer's `go test ./...` exercises the flows hermetically, with no
// Docker required, while CI additionally proves them against live infrastructure.
//
// External services are stubbed at their transport seam, not their adapter: the
// real pharmacy adapter runs over a stubbed HTTP upstream, so the adapter's own
// audit and error-mapping logic is exercised, and the payment webhook is driven
// by the test signing a payload with the shared secret exactly as the gateway
// would. No authentication logic is under test — identity, role and tenant are
// injected via the trusted edge headers the Kong+OPA gateway would stamp.
package apiintegration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/audit"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/crypto"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/integration"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/integration/payment"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/integration/pharmacy"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/locking"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/persistence/mongodb"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/interfaces/rest"
)

// Trusted-edge identity the suite injects on every request unless a test
// overrides it, standing in for the Kong+OPA gateway's stamped headers.
const (
	defaultSubject = "provider-it-1"
	defaultRoles   = "provider,clinician"
	defaultTenant  = "tenant-anco-it"
)

// webhookSecret is the shared HMAC secret the payment webhook is verified
// against. The test signs payloads with it exactly as the gateway would.
var webhookSecret = []byte("s75-integration-webhook-secret")

// environment is a booted API surface plus everything a test needs to drive and
// tear it down.
type environment struct {
	router     http.Handler
	usingMongo bool
	usingRedis bool
	cleanup    func()
}

// buildEnvironment wires the router over real repositories and stubbed external
// transports, selecting live MongoDB/Redis when the environment names them and
// falling back to the in-memory equivalents otherwise. The returned cleanup must
// be called to drop any live collections and close connections.
func buildEnvironment(mongoURI, redisAddr string) (*environment, error) {
	ctx := context.Background()
	cipher, err := testCipher()
	if err != nil {
		return nil, err
	}

	var (
		store      mongodb.DocumentStore
		apptTx     mongodb.TransactionRunner
		auditRepo  = mongodb.NewAuditTrailRepository(mongodb.NewMemAuditEntryCollection())
		usingMongo bool
		cleanups   []func()
	)

	if mongoURI != "" {
		client, err := mongodb.NewClient(ctx, mongodb.Config{URI: mongoURI, Database: mongodb.DefaultDatabase, ConnectTimeout: 10 * time.Second})
		if err != nil {
			return nil, fmt.Errorf("connect mongo: %w", err)
		}
		suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
		aggColl := client.Collection("s75_agg_" + suffix)
		auditColl := client.Collection("s75_audit_" + suffix)
		store = mongodb.NewMongoStore(aggColl)
		// A standalone Mongo container has no replica set, so a real multi-document
		// transaction is unavailable; the double-booking guard is the Redis slot
		// lock, not the transaction, so a direct runner preserves the guarantee
		// while a single-document Save stays atomic on its own.
		apptTx = directRunner{}
		auditRepo = mongodb.NewMongoAuditTrailRepository(auditColl)
		usingMongo = true
		cleanups = append(cleanups, func() {
			_ = aggColl.Drop(context.Background())
			_ = auditColl.Drop(context.Background())
			_ = client.Disconnect(context.Background())
		})
	} else {
		mem := mongodb.NewMemStore()
		store = mem
		apptTx = mem
	}

	locker, usingRedis, redisCleanup, err := buildLocker(redisAddr)
	if err != nil {
		runAll(cleanups)
		return nil, err
	}
	if redisCleanup != nil {
		cleanups = append(cleanups, redisCleanup)
	}

	recorder := audit.NewTrailRecorder(auditRepo)
	pharmacyGateway := pharmacy.NewAdapter(
		integration.NewClient(stubUpstream{}, integration.Config{BaseURL: "http://pharmacy.stub"}),
		recorder,
	)

	deps := rest.Dependencies{
		Appointments:      mongodb.NewAppointmentRepository(store, apptTx, locker, ""),
		ProviderSchedules: mongodb.NewProviderScheduleRepository(store),
		LabOrders:         mongodb.NewLabOrderRepository(store, cipher),
		Prescriptions:     mongodb.NewPrescriptionRepository(store, cipher),
		InsurancePolicies: mongodb.NewInsurancePolicyRepository(store, cipher),
		ClinicDirectories: mongodb.NewClinicDirectoryRepository(store),
		AuditTrails:       auditRepo,

		Encounters: mongodb.NewEncounterRepository(store, cipher),
		Invoices:   mongodb.NewInvoiceRepository(store),
		Payments:   mongodb.NewPaymentRepository(store, cipher),

		Pharmacy:             pharmacyGateway,
		PaymentWebhookSecret: webhookSecret,
		PaymentIdempotency:   payment.NewMemoryIdempotencyStore(),
		Audit:                recorder,
	}

	return &environment{
		router:     rest.NewRouter(deps),
		usingMongo: usingMongo,
		usingRedis: usingRedis,
		cleanup:    func() { runAll(cleanups) },
	}, nil
}

// buildLocker returns the slot locker: a Redis-backed one over a live Redis when
// redisAddr is set, otherwise the in-process locker, which enforces the same
// one-holder-at-a-time semantics.
func buildLocker(redisAddr string) (locking.SlotLocker, bool, func(), error) {
	if redisAddr == "" {
		return locking.NewMemorySlotLocker(), false, nil, nil
	}
	conn, err := dialRedis(redisAddr)
	if err != nil {
		return nil, false, nil, fmt.Errorf("connect redis: %w", err)
	}
	return locking.NewRedisSlotLocker(conn, ""), true, func() { _ = conn.Close() }, nil
}

// testCipher builds the AES-256 field cipher the PHI-bearing repositories seal
// with, under a fixed 32-byte key so a test run is deterministic.
func testCipher() (*crypto.FieldCipher, error) {
	key := make([]byte, crypto.KeySize)
	for i := range key {
		key[i] = byte(i + 1)
	}
	env, err := crypto.NewAESKeyEnvelope("s75-it", key)
	if err != nil {
		return nil, err
	}
	return crypto.NewFieldCipher(env), nil
}

func runAll(fns []func()) {
	for i := len(fns) - 1; i >= 0; i-- {
		fns[i]()
	}
}

// directRunner runs a unit of work without an enclosing transaction. It is the
// TransactionRunner used against a standalone Mongo (no replica set): the slot
// lock provides mutual exclusion and a single-document Save is atomic, so no
// server-side transaction is required for the flows the suite drives.
type directRunner struct{}

func (directRunner) RunInTransaction(ctx context.Context, work mongodb.UnitOfWork) error {
	return work(ctx)
}

// stubUpstream is the stubbed pharmacy gateway HTTP upstream. It answers every
// SCRIPT message with an accepted acknowledgement, so the real pharmacy adapter
// (auth gate, audit, response mapping) runs end-to-end without a live gateway.
type stubUpstream struct{}

func (stubUpstream) Do(req *http.Request) (*http.Response, error) {
	body := `{"status":"Accepted","messageRef":"stub-msg-1"}`
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       newStringBody(body),
	}, nil
}

// --- response decoding helpers shared by the flow tests ---

// decodeInto unmarshals a recorded JSON body into dst, returning a descriptive
// error so a test failure names the offending payload.
func decodeInto(bodyBytes []byte, dst any) error {
	if err := json.Unmarshal(bodyBytes, dst); err != nil {
		return fmt.Errorf("decode %T: %w (body=%q)", dst, err, string(bodyBytes))
	}
	return nil
}
