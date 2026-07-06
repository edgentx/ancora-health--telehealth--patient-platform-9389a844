package main

import (
	"context"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/crypto"
)

// TestBuildContainerWiresDependencies verifies buildContainer assembles a fully
// wired REST dependency set from the in-memory stores, without requiring any
// external service.
func TestBuildContainerWiresDependencies(t *testing.T) {
	t.Setenv(envMasterKey, "")

	deps := buildContainer(context.Background())

	if deps.Appointments == nil {
		t.Error("Appointments repository not wired")
	}
	if deps.ProviderSchedules == nil {
		t.Error("ProviderSchedules repository not wired")
	}
	if deps.LabOrders == nil {
		t.Error("LabOrders repository not wired")
	}
	if deps.Prescriptions == nil {
		t.Error("Prescriptions repository not wired")
	}
	if deps.InsurancePolicies == nil {
		t.Error("InsurancePolicies repository not wired")
	}
	if deps.ClinicDirectories == nil {
		t.Error("ClinicDirectories repository not wired")
	}
	if deps.AuditTrails == nil {
		t.Error("AuditTrails repository not wired")
	}
	if deps.Encounters == nil {
		t.Error("Encounters repository not wired")
	}
	if deps.Invoices == nil {
		t.Error("Invoices repository not wired")
	}
	if deps.Payments == nil {
		t.Error("Payments repository not wired")
	}
	if deps.Audit == nil {
		t.Error("Audit recorder not wired")
	}
}

// TestResolveHealthNotConfigured returns nil (readiness reports unavailable)
// when MONGODB_URI is not set.
func TestResolveHealthNotConfigured(t *testing.T) {
	t.Setenv("MONGODB_URI", "")

	if hc := resolveHealth(context.Background()); hc != nil {
		t.Errorf("resolveHealth with no MONGODB_URI = %v, want nil", hc)
	}
}

// TestResolveHealthConnectFailure returns nil when MONGODB_URI is set but the
// client cannot be constructed (invalid URI), exercising the connect-failure
// branch.
func TestResolveHealthConnectFailure(t *testing.T) {
	t.Setenv("MONGODB_URI", "not-a-valid-scheme://bad-host")

	if hc := resolveHealth(context.Background()); hc != nil {
		t.Errorf("resolveHealth with bad MONGODB_URI = %v, want nil", hc)
	}
}

// TestLoadFieldCipherEphemeral builds a cipher from an ephemeral key when no
// master key is configured.
func TestLoadFieldCipherEphemeral(t *testing.T) {
	t.Setenv(envMasterKey, "")

	if c := loadFieldCipher(); c == nil {
		t.Fatal("loadFieldCipher returned nil")
	}
}

// TestLoadFieldCipherConfiguredKey builds a cipher from a valid hex-encoded
// master key supplied in the environment.
func TestLoadFieldCipherConfiguredKey(t *testing.T) {
	key := strings.Repeat("2a", crypto.KeySize)
	t.Setenv(envMasterKey, key)

	if c := loadFieldCipher(); c == nil {
		t.Fatal("loadFieldCipher returned nil")
	}
}

// TestMasterKeyFromEnv decodes the configured hex master key.
func TestMasterKeyFromEnv(t *testing.T) {
	want := make([]byte, crypto.KeySize)
	for i := range want {
		want[i] = byte(i)
	}
	t.Setenv(envMasterKey, hex.EncodeToString(want))

	got := masterKey()
	if len(got) != crypto.KeySize {
		t.Fatalf("masterKey length = %d, want %d", len(got), crypto.KeySize)
	}
	if string(got) != string(want) {
		t.Errorf("masterKey = %x, want %x", got, want)
	}
}

// TestMasterKeyEphemeral mints a fresh 32-byte key when none is configured.
func TestMasterKeyEphemeral(t *testing.T) {
	t.Setenv(envMasterKey, "")

	got := masterKey()
	if len(got) != crypto.KeySize {
		t.Fatalf("ephemeral masterKey length = %d, want %d", len(got), crypto.KeySize)
	}
}
