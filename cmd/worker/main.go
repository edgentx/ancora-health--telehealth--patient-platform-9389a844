// Command worker is the deployable Temporal worker process. It hosts the
// durable workflows and activities for appointment reminders, results-ready
// notifications, the billing/eligibility saga, and the scheduled analytics
// rollups, registering them on the configured task queue and connecting with
// deploy-time Temporal config.
//
// It is a separate process from cmd/server: the API server serves HTTP while
// this process polls Temporal. Both share the same repositories and adapters, so
// wiring mirrors the server's graceful-degradation style — it binds MongoDB and
// the external gateways when configured and falls back to in-memory doubles for
// local runs, so the worker is always launchable.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	temporalapp "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/application/temporal"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/crypto"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/integration"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/integration/eligibility"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/integration/payment"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/locking"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/persistence/mongodb"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/pubsub"
)

func main() {
	ctx := context.Background()

	activities := buildActivities(ctx)
	cfg := temporalapp.ConfigFromEnv()

	log.Printf("temporal worker: task queue %q, namespace %q, host %q",
		cfg.TaskQueue, cfg.Namespace, cfg.HostPort)

	if err := temporalapp.Run(cfg, activities); err != nil {
		log.Fatalf("temporal worker: %v", err)
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
