// Command server is the HTTP entrypoint for the Ancora Health telehealth
// platform. It builds the dependency-injection container, wires the S-69/S-70
// MongoDB repositories into the REST handler layer, and serves the versioned
// chi router (business API under /api/v1, plus /health, /ready and /metrics).
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"os"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/audit"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/crypto"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/locking"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/persistence/mongodb"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/interfaces/rest"
)

// envMasterKey names the hex-encoded 32-byte master key that seals PHI at rest.
// When unset the server mints an ephemeral key so a local run still boots; that
// key does not survive a restart and must never be used against durable storage.
const envMasterKey = "PHI_MASTER_KEY"

// buildContainer assembles the REST dependencies. Document storage is backed by
// the in-memory store (the same store the S-70 repository suite runs against),
// which keeps the service bootable without a live MongoDB while wiring the real
// repository types. When MONGODB_URI is configured its client backs the /ready
// probe, so readiness still reflects true database reachability.
func buildContainer(ctx context.Context) rest.Dependencies {
	store := mongodb.NewMemStore()
	locker := locking.NewMemorySlotLocker()
	cipher := loadFieldCipher()

	auditTrails := mongodb.NewAuditTrailRepository(mongodb.NewMemAuditEntryCollection())

	return rest.Dependencies{
		Health:            resolveHealth(ctx),
		Appointments:      mongodb.NewAppointmentRepository(store, store, locker, ""),
		ProviderSchedules: mongodb.NewProviderScheduleRepository(store),
		LabOrders:         mongodb.NewLabOrderRepository(store, cipher),
		Prescriptions:     mongodb.NewPrescriptionRepository(store, cipher),
		InsurancePolicies: mongodb.NewInsurancePolicyRepository(store, cipher),
		ClinicDirectories: mongodb.NewClinicDirectoryRepository(store),
		AuditTrails:       auditTrails,
		// Cross-context flows added in S-75. The pharmacy gateway and payment
		// webhook are left unset in a local run (they require a live upstream and
		// a shared signing secret); the integration suite wires stubs for them.
		Encounters: mongodb.NewEncounterRepository(store, cipher),
		Invoices:   mongodb.NewInvoiceRepository(store),
		Payments:   mongodb.NewPaymentRepository(store, cipher),
		Audit:      audit.NewTrailRecorder(auditTrails),
	}
}

// resolveHealth returns the readiness probe. It connects to MongoDB when
// MONGODB_URI is set; otherwise it returns nil and /ready reports "not
// configured", which is the correct signal for a database-less local run.
func resolveHealth(ctx context.Context) rest.HealthChecker {
	cfg, err := mongodb.ConfigFromEnv()
	if err != nil {
		log.Printf("mongodb: not configured (%v); readiness probe will report unavailable", err)
		return nil
	}
	client, err := mongodb.NewClient(ctx, cfg)
	if err != nil {
		log.Printf("mongodb: initial connection failed: %v; readiness probe will report unavailable", err)
		return nil
	}
	log.Printf("mongodb: connected to database %q", cfg.Database)
	return client
}

// loadFieldCipher builds the AES-256 envelope cipher that the PHI-bearing
// repositories seal fields with. It prefers the configured master key and falls
// back to an ephemeral one so local runs work out of the box.
func loadFieldCipher() *crypto.FieldCipher {
	env, err := crypto.NewAESKeyEnvelope("phi-master", masterKey())
	if err != nil {
		// A bad key length is a configuration error the process cannot serve
		// through, so fail fast rather than boot with broken encryption.
		log.Fatalf("crypto: invalid master key: %v", err)
	}
	return crypto.NewFieldCipher(env)
}

// masterKey resolves the 32-byte master key from the environment, or mints an
// ephemeral one when none is configured.
func masterKey() []byte {
	if raw := os.Getenv(envMasterKey); raw != "" {
		key, err := hex.DecodeString(raw)
		if err != nil {
			log.Fatalf("crypto: %s must be hex-encoded: %v", envMasterKey, err)
		}
		return key
	}
	log.Printf("crypto: %s not set; minting an ephemeral master key (not persisted)", envMasterKey)
	key := make([]byte, crypto.KeySize)
	if _, err := rand.Read(key); err != nil {
		log.Fatalf("crypto: unable to mint ephemeral key: %v", err)
	}
	return key
}

func main() {
	deps := buildContainer(context.Background())

	const addr = ":8000"
	log.Printf("Ancora Health — Telehealth & Patient Platform listening on %s", addr)
	if err := http.ListenAndServe(addr, rest.NewRouter(deps)); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
