package crypto

import (
	"context"
	"crypto/rand"
	"io"
)

// cipherVersion identifies the on-disk CipherText layout so the format can
// evolve (e.g. new AEAD, added associated data) without breaking stored data.
const cipherVersion = 1

// CipherText is the at-rest envelope written in place of a plaintext PHI value.
// It carries everything needed to decrypt except the master key: the wrapped
// per-record data key, the reference to the master key that wrapped it, the GCM
// nonce and the ciphertext. Only these fields are ever serialized, so a stored
// document never contains PHI plaintext — verifiable by inspecting the BSON.
type CipherText struct {
	Version    int    `bson:"v" json:"v"`
	KeyRef     string `bson:"kref" json:"kref"`
	WrappedDEK []byte `bson:"wdek" json:"wdek"`
	Nonce      []byte `bson:"nonce" json:"nonce"`
	Ciphertext []byte `bson:"ct" json:"ct"`
}

// FieldCipher encrypts and decrypts individual PHI field values using envelope
// encryption. Each Encrypt call generates a fresh data key, so two identical
// plaintexts never yield identical ciphertext and a single leaked DEK exposes
// only one record.
type FieldCipher struct {
	envelope KeyEnvelope
}

// NewFieldCipher builds a FieldCipher backed by the given key envelope.
func NewFieldCipher(envelope KeyEnvelope) *FieldCipher {
	return &FieldCipher{envelope: envelope}
}

// Encrypt seals plaintext under a freshly generated per-record data key and
// wraps that data key with the envelope's master key.
func (c *FieldCipher) Encrypt(ctx context.Context, plaintext []byte) (CipherText, error) {
	dek := make([]byte, KeySize)
	if _, err := io.ReadFull(rand.Reader, dek); err != nil {
		return CipherText{}, err
	}

	gcm, err := newGCM(dek)
	if err != nil {
		return CipherText{}, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return CipherText{}, err
	}
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	wrapped, keyRef, err := c.envelope.WrapDataKey(ctx, dek)
	if err != nil {
		return CipherText{}, err
	}

	return CipherText{
		Version:    cipherVersion,
		KeyRef:     keyRef,
		WrappedDEK: wrapped,
		Nonce:      nonce,
		Ciphertext: ciphertext,
	}, nil
}

// Decrypt reverses Encrypt: it unwraps the per-record data key via the envelope
// and uses it to open the ciphertext. GCM authentication fails (returning an
// error) if any part of the CipherText has been tampered with.
func (c *FieldCipher) Decrypt(ctx context.Context, ct CipherText) ([]byte, error) {
	dek, err := c.envelope.UnwrapDataKey(ctx, ct.WrappedDEK, ct.KeyRef)
	if err != nil {
		return nil, err
	}
	gcm, err := newGCM(dek)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, ct.Nonce, ct.Ciphertext, nil)
}
