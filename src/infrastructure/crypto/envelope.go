// Package crypto provides the field-level PHI crypto port used at the
// persistence boundary. It performs AES-256-GCM envelope encryption: each
// protected value is sealed under a freshly generated per-record data
// encryption key (DEK), and that DEK is itself wrapped (encrypted) by a master
// key exposed through the KeyEnvelope port. The wrapped DEK, nonce and
// ciphertext travel together as a CipherText so plaintext PHI is never written
// at rest.
package crypto

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
)

// KeySize is the required length, in bytes, of an AES-256 key (256 bits).
const KeySize = 32

var (
	// ErrInvalidKeySize is returned when a master or data key is not exactly
	// KeySize bytes long.
	ErrInvalidKeySize = errors.New("crypto: key must be 32 bytes (AES-256)")
	// ErrEmptyKeyRef is returned when a KeyEnvelope is constructed without a key
	// reference to identify the wrapping master key.
	ErrEmptyKeyRef = errors.New("crypto: key reference must not be empty")
)

// KeyEnvelope is the port that wraps and unwraps per-record data encryption
// keys under a master (key-encryption) key. It maps onto the audit-and-
// compliance CryptoKeyEnvelope aggregate's data keys: a production adapter
// would delegate to a KMS/HSM, while AESKeyEnvelope offers a self-contained
// software implementation for local development and tests.
type KeyEnvelope interface {
	// WrapDataKey encrypts a plaintext data key under the master key, returning
	// the wrapped bytes and a reference identifying the master key that wrapped
	// it (so the correct key can be selected at unwrap time).
	WrapDataKey(ctx context.Context, dek []byte) (wrapped []byte, keyRef string, err error)
	// UnwrapDataKey reverses WrapDataKey, recovering the plaintext data key. The
	// keyRef identifies which master key produced the wrapped bytes.
	UnwrapDataKey(ctx context.Context, wrapped []byte, keyRef string) (dek []byte, err error)
}

// AESKeyEnvelope wraps data keys with a single master key using AES-256-GCM.
// It is a software KeyEnvelope suitable for local development and tests; the
// master key material must be provisioned and rotated by the caller (in
// production, via the CryptoKeyEnvelope aggregate + a KMS-backed adapter).
type AESKeyEnvelope struct {
	keyRef string
	gcm    cipher.AEAD
}

// NewAESKeyEnvelope builds an AESKeyEnvelope from a 32-byte master key and the
// reference used to identify it. It fails fast on a wrong-sized key or an empty
// reference so misconfiguration surfaces at startup rather than at first write.
func NewAESKeyEnvelope(keyRef string, masterKey []byte) (*AESKeyEnvelope, error) {
	if keyRef == "" {
		return nil, ErrEmptyKeyRef
	}
	gcm, err := newGCM(masterKey)
	if err != nil {
		return nil, err
	}
	return &AESKeyEnvelope{keyRef: keyRef, gcm: gcm}, nil
}

// KeyRef reports the reference of the master key this envelope wraps under.
func (e *AESKeyEnvelope) KeyRef() string { return e.keyRef }

// WrapDataKey seals the DEK under the master key. The returned bytes are the
// GCM nonce followed by the ciphertext, so unwrap needs no side channel.
func (e *AESKeyEnvelope) WrapDataKey(_ context.Context, dek []byte) ([]byte, string, error) {
	if len(dek) != KeySize {
		return nil, "", ErrInvalidKeySize
	}
	wrapped, err := seal(e.gcm, dek)
	if err != nil {
		return nil, "", err
	}
	return wrapped, e.keyRef, nil
}

// UnwrapDataKey recovers the DEK, verifying that keyRef matches the master key
// held by this envelope before attempting decryption.
func (e *AESKeyEnvelope) UnwrapDataKey(_ context.Context, wrapped []byte, keyRef string) ([]byte, error) {
	if keyRef != e.keyRef {
		return nil, fmt.Errorf("crypto: unknown key reference %q", keyRef)
	}
	return open(e.gcm, wrapped)
}

// newGCM builds an AES-256-GCM AEAD from a 32-byte key.
func newGCM(key []byte) (cipher.AEAD, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKeySize
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// seal encrypts plaintext with a random nonce and returns nonce||ciphertext.
func seal(gcm cipher.AEAD, plaintext []byte) ([]byte, error) {
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// open reverses seal, splitting the leading nonce back off before decrypting.
func open(gcm cipher.AEAD, sealed []byte) ([]byte, error) {
	ns := gcm.NonceSize()
	if len(sealed) < ns {
		return nil, errors.New("crypto: ciphertext too short")
	}
	nonce, ciphertext := sealed[:ns], sealed[ns:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
