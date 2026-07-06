package model

import (
	"errors"
	"testing"
	"time"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"
)

// activeEnvelope returns an envelope in the only state that lets a command
// succeed: an active master key, unrevoked, and not expired.
func activeEnvelope() *CryptoKeyEnvelopeAggregate {
	return &CryptoKeyEnvelopeAggregate{
		ID:              "envelope-1",
		MasterKeyActive: true,
	}
}

func TestExecuteIssueDataKeyEmitsEvent(t *testing.T) {
	agg := activeEnvelope()

	events, err := agg.Execute(IssueDataKeyCmd{TenantId: "tenant-1", FieldClass: "ssn"})
	if err != nil {
		t.Fatalf("Execute(IssueDataKeyCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	evt, ok := events[0].(DataKeyIssuedEvent)
	if !ok {
		t.Fatalf("event type = %T, want DataKeyIssuedEvent", events[0])
	}
	if evt.Type() != "crypto.datakey.issued" {
		t.Fatalf("event type = %q, want crypto.datakey.issued", evt.Type())
	}
	if evt.AggregateID() != "envelope-1" || evt.EnvelopeID != "envelope-1" {
		t.Fatalf("event aggregate id = %q / envelope id = %q, want envelope-1", evt.AggregateID(), evt.EnvelopeID)
	}
	if evt.TenantID != "tenant-1" || evt.FieldClass != "ssn" {
		t.Fatalf("event payload not copied from command: %+v", evt)
	}
	if buffered := agg.Events(); len(buffered) != 1 {
		t.Fatalf("buffered %d events, want 1", len(buffered))
	}
}

func TestExecuteIssueDataKeyRejects(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*CryptoKeyEnvelopeAggregate)
		cmd     IssueDataKeyCmd
		wantErr error
	}{
		{
			name:    "missing tenant",
			cmd:     IssueDataKeyCmd{FieldClass: "ssn"},
			wantErr: ErrMissingTenant,
		},
		{
			name:    "missing field class",
			cmd:     IssueDataKeyCmd{TenantId: "tenant-1"},
			wantErr: ErrMissingFieldClass,
		},
		{
			name:    "revoked",
			mutate:  func(a *CryptoKeyEnvelopeAggregate) { a.Revoked = true },
			cmd:     IssueDataKeyCmd{TenantId: "tenant-1", FieldClass: "ssn"},
			wantErr: ErrEnvelopeRevoked,
		},
		{
			name:    "expired",
			mutate:  func(a *CryptoKeyEnvelopeAggregate) { a.ExpiresAt = time.Now().Add(-time.Hour) },
			cmd:     IssueDataKeyCmd{TenantId: "tenant-1", FieldClass: "ssn"},
			wantErr: ErrEnvelopeExpired,
		},
		{
			name:    "master key inactive",
			mutate:  func(a *CryptoKeyEnvelopeAggregate) { a.MasterKeyActive = false },
			cmd:     IssueDataKeyCmd{TenantId: "tenant-1", FieldClass: "ssn"},
			wantErr: ErrMasterKeyInactive,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := activeEnvelope()
			if tt.mutate != nil {
				tt.mutate(agg)
			}
			events, err := agg.Execute(tt.cmd)
			assertCryptoRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestExecuteRotateMasterKeyEmitsEvent(t *testing.T) {
	agg := activeEnvelope()

	events, err := agg.Execute(RotateMasterKeyCmd{NewMasterKeyId: "master-2"})
	if err != nil {
		t.Fatalf("Execute(RotateMasterKeyCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	evt, ok := events[0].(MasterKeyRotatedEvent)
	if !ok {
		t.Fatalf("event type = %T, want MasterKeyRotatedEvent", events[0])
	}
	if evt.Type() != "crypto.masterkey.rotated" {
		t.Fatalf("event type = %q, want crypto.masterkey.rotated", evt.Type())
	}
	if evt.AggregateID() != "envelope-1" || evt.EnvelopeID != "envelope-1" {
		t.Fatalf("event aggregate id = %q / envelope id = %q, want envelope-1", evt.AggregateID(), evt.EnvelopeID)
	}
	if evt.NewMasterKeyID != "master-2" {
		t.Fatalf("event new master key id = %q, want master-2", evt.NewMasterKeyID)
	}
	if buffered := agg.Events(); len(buffered) != 1 {
		t.Fatalf("buffered %d events, want 1", len(buffered))
	}
}

func TestExecuteRotateMasterKeyRejects(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*CryptoKeyEnvelopeAggregate)
		cmd     RotateMasterKeyCmd
		wantErr error
	}{
		{
			name:    "missing master key id",
			cmd:     RotateMasterKeyCmd{},
			wantErr: ErrMissingMasterKeyId,
		},
		{
			name:    "revoked",
			mutate:  func(a *CryptoKeyEnvelopeAggregate) { a.Revoked = true },
			cmd:     RotateMasterKeyCmd{NewMasterKeyId: "master-2"},
			wantErr: ErrEnvelopeRevoked,
		},
		{
			name:    "expired",
			mutate:  func(a *CryptoKeyEnvelopeAggregate) { a.ExpiresAt = time.Now().Add(-time.Hour) },
			cmd:     RotateMasterKeyCmd{NewMasterKeyId: "master-2"},
			wantErr: ErrEnvelopeExpired,
		},
		{
			name:    "master key inactive",
			mutate:  func(a *CryptoKeyEnvelopeAggregate) { a.MasterKeyActive = false },
			cmd:     RotateMasterKeyCmd{NewMasterKeyId: "master-2"},
			wantErr: ErrMasterKeyInactive,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := activeEnvelope()
			if tt.mutate != nil {
				tt.mutate(agg)
			}
			events, err := agg.Execute(tt.cmd)
			assertCryptoRejected(t, agg, events, err, tt.wantErr)
		})
	}
}

func TestCryptoKeyEnvelopeExecuteUnknownCommand(t *testing.T) {
	agg := activeEnvelope()

	type bogusCmd struct{}
	events, err := agg.Execute(bogusCmd{})
	if !errors.Is(err, shared.ErrUnknownCommand) {
		t.Fatalf("error = %v, want %v", err, shared.ErrUnknownCommand)
	}
	if events != nil {
		t.Fatalf("expected nil events, got %v", events)
	}
	if agg.Version != 0 {
		t.Fatalf("version = %d, want 0", agg.Version)
	}
	if len(agg.Events()) != 0 {
		t.Fatalf("expected no buffered events, got %d", len(agg.Events()))
	}
}

// assertCryptoRejected checks that a command produced the expected sentinel
// error, emitted no events and buffered nothing.
func assertCryptoRejected(t *testing.T, agg *CryptoKeyEnvelopeAggregate, events []shared.DomainEvent, err, wantErr error) {
	t.Helper()
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
	if len(events) != 0 {
		t.Fatalf("expected no events on rejection, got %d", len(events))
	}
	if len(agg.Events()) != 0 {
		t.Fatalf("expected no buffered events on rejection, got %d", len(agg.Events()))
	}
}
