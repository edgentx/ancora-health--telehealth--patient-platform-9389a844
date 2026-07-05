// Command worker is the deployable background-processing entrypoint for the
// Ancora Health platform. It hosts the durable Temporal workflows and activities
// for appointment reminders, results-ready notifications, the billing/eligibility
// saga, and the scheduled analytics rollups (S-74), and it is one of the three
// selectable backend entrypoints the container image builds — api, realtime,
// worker (S-76).
//
// Alongside the Temporal worker it serves the standard operational surface
// (/health, /ready, /metrics, /version) so the container has a readiness probe
// and a metrics scrape regardless of which entrypoint the image runs, and it
// accepts the `-healthcheck` self-probe the distroless HEALTHCHECK invokes.
//
// It is a separate process from cmd/server: the API server serves business HTTP
// while this process polls Temporal. Both share the same repositories and
// adapters, so wiring mirrors the server's graceful-degradation style — it binds
// MongoDB and the external gateways when configured and falls back to in-memory
// doubles for local runs. The ops surface comes up even when no Temporal frontend
// is reachable, which is what lets the container start and pass its readiness
// probe locally.
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	temporalapp "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/application/temporal"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/crypto"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/integration"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/integration/eligibility"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/integration/payment"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/locking"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/persistence/mongodb"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/pubsub"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/platform"
)

func main() {
	// -healthcheck is the container HEALTHCHECK entrypoint: probe /ready on the
	// local listen address and exit non-zero if the service is not serving, so
	// the distroless runtime image needs no shell or curl to be health-checked.
	healthcheck := flag.Bool("healthcheck", false, "probe /ready on the local listen address and exit")
	flag.Parse()
	if *healthcheck {
		if err := platform.SelfCheck(); err != nil {
			log.Fatalf("healthcheck failed: %v", err)
		}
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	activities := buildActivities(ctx)
	cfg := temporalapp.ConfigFromEnv()

	// Host the Temporal worker off the request path, in a goroutine, so the ops
	// server can serve /metrics and /ready in parallel. Sharing the process this
	// way means the container reports health and metrics whether or not a Temporal
	// frontend is reachable: a dial failure (e.g. no Temporal locally) is logged
	// and the ops surface stays up rather than crashing the container.
	go func() {
		log.Printf("temporal worker: task queue %q, namespace %q, host %q",
			cfg.TaskQueue, cfg.Namespace, cfg.HostPort)
		if err := temporalapp.Run(cfg, activities); err != nil {
			log.Printf("temporal worker: not running (%v); ops surface still served", err)
		}
	}()

	// Dependency-free readiness: the worker has no request-serving backing store,
	// so /ready reports 200 once the process is up (mirroring the realtime gateway).
	mux := platform.NewOpsMux(platform.NewMetricsRegistry(), nil)
	addr := platform.ListenAddr()
	platform.LogStartup("ancora-worker", addr)
	if err := platform.Serve(ctx, addr, mux); err != nil {
		log.Fatalf("worker error: %v", err)
	}
}

// buildActivities wires the activity dependencies. It binds MongoDB-backed
// repositories and the external gateways when their deploy-time config is
// present, and falls back to in-memory doubles otherwise so the process runs
// locally without external services.
func buildActivities(ctx context.Context) *temporalapp.Activities {
	client := connectMongo(ctx)
	cipher := devCipher()
	locker := locking.NewMemorySlotLocker()

	// Fact source: production reads Mongo read models; local uses the in-memory
	// fact source.
	var facts mongodb.FactSource
	if client != nil {
		facts = mongodb.NewMongoFactSource(client.Database())
	} else {
		facts = &mongodb.MemFactSource{}
	}

	// Notifications route through the realtime pub/sub broker (S-73). Without a
	// configured Redis broker the process uses the in-process broker; delivery
	// still fans out to any gateway sharing it.
	notifier := temporalapp.NewBrokerNotifier(pubsub.NewMemoryBroker(0))

	acts := &temporalapp.Activities{
		Appointments: mongodb.NewAppointmentRepository(store(client, "appointments"), txRunner(client), locker, ""),
		Invoices:     mongodb.NewInvoiceRepository(store(client, "invoices")),
		Payments:     mongodb.NewPaymentRepository(store(client, "payments"), cipher),
		Dashboards:   mongodb.NewAnalyticsDashboardRepository(store(client, "analytics_dashboards"), facts),
		Facts:        facts,
		Rollups:      &temporalapp.MemRollupStore{},
		Notifier:     notifier,
	}

	if gw := eligibilityGateway(); gw != nil {
		acts.Eligibility = gw
	}
	if gw := paymentGateway(); gw != nil {
		acts.PaymentGateway = gw
	}
	return acts
}

// connectMongo binds the shared MongoDB instance when MONGODB_URI is configured,
// returning nil (and logging) otherwise so the worker degrades to in-memory
// stores.
func connectMongo(ctx context.Context) *mongodb.Client {
	cfg, err := mongodb.ConfigFromEnv()
	if err != nil {
		log.Printf("mongodb: not configured (%v); using in-memory stores", err)
		return nil
	}
	client, err := mongodb.NewClient(ctx, cfg)
	if err != nil {
		log.Printf("mongodb: connection failed (%v); using in-memory stores", err)
		return nil
	}
	log.Printf("mongodb: connected to database %q", cfg.Database)
	return client
}

// store returns a document store for a named collection, backed by Mongo when a
// client is present and by an in-memory store otherwise.
func store(client *mongodb.Client, collection string) mongodb.DocumentStore {
	if client == nil {
		return mongodb.NewMemStore()
	}
	return mongodb.NewMongoStore(client.Collection(collection))
}

// txRunner returns the transaction runner the appointment repository commits its
// slot-hold writes through: a Mongo session runner in production, the in-memory
// store's runner locally.
func txRunner(client *mongodb.Client) mongodb.TransactionRunner {
	if client == nil {
		return mongodb.NewMemStore()
	}
	return mongodb.NewMongoTransactionRunner(client.Mongo())
}

// devCipher builds a field cipher for local runs. Production must supply managed
// key material via the crypto envelope; this fixed dev key exists only so the
// payment repository is usable without external key management.
func devCipher() *crypto.FieldCipher {
	env, err := crypto.NewAESKeyEnvelope("worker-dev", make([]byte, crypto.KeySize))
	if err != nil {
		log.Fatalf("crypto: build dev cipher: %v", err)
	}
	return crypto.NewFieldCipher(env)
}

// eligibilityGateway wires the payer-eligibility adapter when ELIGIBILITY_BASE_URL
// is configured; otherwise the saga's eligibility step is inactive.
func eligibilityGateway() eligibility.Gateway {
	base := os.Getenv("ELIGIBILITY_BASE_URL")
	if base == "" {
		return nil
	}
	client := integration.NewClient(&http.Client{Timeout: 10 * time.Second}, integration.Config{BaseURL: base})
	return eligibility.NewAdapter(client, nil)
}

// paymentGateway wires the payment adapter when PAYMENT_BASE_URL is configured;
// otherwise the saga's payment step is inactive.
func paymentGateway() payment.PaymentGateway {
	base := os.Getenv("PAYMENT_BASE_URL")
	if base == "" {
		return nil
	}
	client := integration.NewClient(&http.Client{Timeout: 10 * time.Second}, integration.Config{BaseURL: base})
	return payment.NewAdapter(client)
}
