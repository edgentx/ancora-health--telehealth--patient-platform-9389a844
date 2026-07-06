package main

import (
	"context"
	"testing"
)

// clearWorkerEnv resets every environment variable the worker wiring reads so
// each test starts from the in-memory/degraded defaults.
func clearWorkerEnv(t *testing.T) {
	t.Helper()
	t.Setenv("MONGODB_URI", "")
	t.Setenv("ELIGIBILITY_BASE_URL", "")
	t.Setenv("PAYMENT_BASE_URL", "")
}

// TestBuildActivitiesInMemory wires the full activity set from in-memory doubles
// when no external service is configured.
func TestBuildActivitiesInMemory(t *testing.T) {
	clearWorkerEnv(t)

	acts := buildActivities(context.Background())
	if acts == nil {
		t.Fatal("buildActivities returned nil")
	}
	if acts.Appointments == nil {
		t.Error("Appointments not wired")
	}
	if acts.Invoices == nil {
		t.Error("Invoices not wired")
	}
	if acts.Payments == nil {
		t.Error("Payments not wired")
	}
	if acts.Dashboards == nil {
		t.Error("Dashboards not wired")
	}
	if acts.Facts == nil {
		t.Error("Facts not wired")
	}
	if acts.Rollups == nil {
		t.Error("Rollups not wired")
	}
	if acts.Notifier == nil {
		t.Error("Notifier not wired")
	}
	// Optional gateways are inactive without their base URLs.
	if acts.Eligibility != nil {
		t.Error("Eligibility should be nil without ELIGIBILITY_BASE_URL")
	}
	if acts.PaymentGateway != nil {
		t.Error("PaymentGateway should be nil without PAYMENT_BASE_URL")
	}
}

// TestBuildActivitiesWithGateways wires the optional eligibility and payment
// gateways when their base URLs are configured.
func TestBuildActivitiesWithGateways(t *testing.T) {
	clearWorkerEnv(t)
	t.Setenv("ELIGIBILITY_BASE_URL", "https://eligibility.example.test")
	t.Setenv("PAYMENT_BASE_URL", "https://payment.example.test")

	acts := buildActivities(context.Background())
	if acts.Eligibility == nil {
		t.Error("Eligibility gateway not wired despite ELIGIBILITY_BASE_URL")
	}
	if acts.PaymentGateway == nil {
		t.Error("PaymentGateway not wired despite PAYMENT_BASE_URL")
	}
}

// TestConnectMongoNotConfigured returns nil when MONGODB_URI is unset.
func TestConnectMongoNotConfigured(t *testing.T) {
	t.Setenv("MONGODB_URI", "")

	if c := connectMongo(context.Background()); c != nil {
		t.Errorf("connectMongo with no MONGODB_URI = %v, want nil", c)
	}
}

// TestConnectMongoConnectFailure returns nil when MONGODB_URI is set but invalid,
// exercising the connect-failure branch.
func TestConnectMongoConnectFailure(t *testing.T) {
	t.Setenv("MONGODB_URI", "not-a-valid-scheme://bad-host")

	if c := connectMongo(context.Background()); c != nil {
		t.Errorf("connectMongo with bad MONGODB_URI = %v, want nil", c)
	}
}

// TestStoreNilClient falls back to an in-memory document store when no client
// is present.
func TestStoreNilClient(t *testing.T) {
	if s := store(nil, "appointments"); s == nil {
		t.Error("store(nil, ...) returned nil, want in-memory store")
	}
}

// TestTxRunnerNilClient falls back to the in-memory transaction runner when no
// client is present.
func TestTxRunnerNilClient(t *testing.T) {
	if r := txRunner(nil); r == nil {
		t.Error("txRunner(nil) returned nil, want in-memory runner")
	}
}

// TestDevCipher builds the local dev field cipher.
func TestDevCipher(t *testing.T) {
	if c := devCipher(); c == nil {
		t.Error("devCipher returned nil")
	}
}

// TestEligibilityGateway covers both the unset (inactive) and configured paths.
func TestEligibilityGateway(t *testing.T) {
	t.Setenv("ELIGIBILITY_BASE_URL", "")
	if gw := eligibilityGateway(); gw != nil {
		t.Error("eligibilityGateway with no base URL should be nil")
	}

	t.Setenv("ELIGIBILITY_BASE_URL", "https://eligibility.example.test")
	if gw := eligibilityGateway(); gw == nil {
		t.Error("eligibilityGateway with base URL should be non-nil")
	}
}

// TestPaymentGateway covers both the unset (inactive) and configured paths.
func TestPaymentGateway(t *testing.T) {
	t.Setenv("PAYMENT_BASE_URL", "")
	if gw := paymentGateway(); gw != nil {
		t.Error("paymentGateway with no base URL should be nil")
	}

	t.Setenv("PAYMENT_BASE_URL", "https://payment.example.test")
	if gw := paymentGateway(); gw == nil {
		t.Error("paymentGateway with base URL should be non-nil")
	}
}
