package mongodb

import (
	"context"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/crypto"
)

// encryptField seals a PHI/PCI string value into a CipherText for storage,
// reusing the S-68 envelope cipher so a fresh data key protects every value.
func encryptField(ctx context.Context, cipher *crypto.FieldCipher, plain string) (crypto.CipherText, error) {
	return cipher.Encrypt(ctx, []byte(plain))
}

// decryptField reverses encryptField. A zero-value CipherText (version 0) marks
// a field that was never encrypted — an absent or empty value — so it maps back
// to the empty string rather than failing decryption.
func decryptField(ctx context.Context, cipher *crypto.FieldCipher, ct crypto.CipherText) (string, error) {
	if ct.Version == 0 {
		return "", nil
	}
	plain, err := cipher.Decrypt(ctx, ct)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
