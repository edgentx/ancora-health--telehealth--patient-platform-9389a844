// Package repository declares the persistence ports for the audit-and-compliance
// bounded context. Concrete adapters (in-memory, SQL, etc.) live elsewhere and
// implement these interfaces.
package repository

import (
	"context"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/auditandcompliance/model"
)

// CryptoKeyEnvelopeRepository is the persistence port for CryptoKeyEnvelopeAggregate.
// Save persists an aggregate; FindByID loads one by its identity.
type CryptoKeyEnvelopeRepository interface {
	Save(ctx context.Context, a *model.CryptoKeyEnvelopeAggregate) error
	FindByID(ctx context.Context, id string) (*model.CryptoKeyEnvelopeAggregate, error)
}
