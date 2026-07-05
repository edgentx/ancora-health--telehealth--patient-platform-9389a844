package crypto

import (
	"bytes"
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
)

// newTestCipher builds a FieldCipher backed by a deterministic 32-byte master
// key so tests are self-contained.
func newTestCipher(t *testing.T) *FieldCipher {
	t.Helper()
	master := bytes.Repeat([]byte{0x2a}, KeySize)
	env, err := NewAESKeyEnvelope("test-master-v1", master)
	if err != nil {
		t.Fatalf("NewAESKeyEnvelope: %v", err)
	}
	return NewFieldCipher(env)
}

func TestFieldCipher_RoundTrip(t *testing.T) {
	cipher := newTestCipher(t)
	ctx := context.Background()

	plaintext := []byte("123-45-6789") // e.g. an SSN

	ct, err := cipher.Encrypt(ctx, plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// The stored ciphertext must never contain the plaintext.
	if bytes.Contains(ct.Ciphertext, plaintext) {
		t.Fatal("ciphertext contains the plaintext — encryption at rest is broken")
	}
	if len(ct.WrappedDEK) == 0 || len(ct.Nonce) == 0 {
		t.Fatal("expected a wrapped data key and nonce in the envelope")
	}

	got, err := cipher.Decrypt(ctx, ct)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("round trip mismatch: got %q want %q", got, plaintext)
	}
}

func TestFieldCipher_FreshDataKeyPerRecord(t *testing.T) {
	cipher := newTestCipher(t)
	ctx := context.Background()

	a, err := cipher.Encrypt(ctx, []byte("same"))
	if err != nil {
		t.Fatalf("Encrypt a: %v", err)
	}
	b, err := cipher.Encrypt(ctx, []byte("same"))
	if err != nil {
		t.Fatalf("Encrypt b: %v", err)
	}
	if bytes.Equal(a.Ciphertext, b.Ciphertext) {
		t.Fatal("identical plaintexts produced identical ciphertext — DEK/nonce reuse")
	}
}

func TestFieldCipher_TamperDetection(t *testing.T) {
	cipher := newTestCipher(t)
	ctx := context.Background()

	ct, err := cipher.Encrypt(ctx, []byte("diagnosis: confidential"))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	// Flip a bit in the ciphertext; GCM authentication must reject it.
	ct.Ciphertext[0] ^= 0xff
	if _, err := cipher.Decrypt(ctx, ct); err == nil {
		t.Fatal("expected authentication failure on tampered ciphertext")
	}
}

// phiRecord is a persistence document with a PHI-tagged field, used to exercise
// the struct-level Codec.
type phiRecord struct {
	ID        string `bson:"_id"`
	Version   int    `bson:"version"`
	FirstName string `bson:"first_name" phi:"true"`
	Status    string `bson:"status"`
}

func TestCodec_DocumentRoundTrip(t *testing.T) {
	codec := NewCodec(newTestCipher(t))
	ctx := context.Background()

	in := phiRecord{ID: "p-1", Version: 3, FirstName: "Alice", Status: "active"}

	doc, err := codec.EncryptDocument(ctx, in)
	if err != nil {
		t.Fatalf("EncryptDocument: %v", err)
	}

	// At rest, the PHI field must be a CipherText, not the plaintext name.
	if doc["first_name"] == "Alice" {
		t.Fatal("PHI field stored in plaintext")
	}
	if _, ok := doc["first_name"].(CipherText); !ok {
		t.Fatalf("expected first_name to be a CipherText, got %T", doc["first_name"])
	}
	// Non-PHI fields pass through untouched.
	if doc["status"] != "active" {
		t.Fatalf("non-PHI field altered: %v", doc["status"])
	}

	// The plaintext name must not appear anywhere in the serialized document.
	raw, err := bson.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal doc: %v", err)
	}
	if bytes.Contains(raw, []byte("Alice")) {
		t.Fatal("plaintext PHI is present in the stored BSON")
	}

	var out phiRecord
	if err := codec.DecryptDocument(ctx, doc, &out); err != nil {
		t.Fatalf("DecryptDocument: %v", err)
	}
	if out != in {
		t.Fatalf("round trip mismatch: got %+v want %+v", out, in)
	}
}

func TestCodec_RejectsNonStringPHIField(t *testing.T) {
	codec := NewCodec(newTestCipher(t))
	type bad struct {
		Secret int `bson:"secret" phi:"true"`
	}
	if _, err := codec.EncryptDocument(context.Background(), bad{Secret: 42}); err == nil {
		t.Fatal("expected an error for a non-string PHI field")
	}
}
