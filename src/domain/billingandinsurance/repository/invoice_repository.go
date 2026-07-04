// Package repository declares the persistence contracts for the
// billing-and-insurance bounded context.
package repository

import (
	"context"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/billingandinsurance/model"
)

// InvoiceRepository persists and retrieves InvoiceAggregate instances. Concrete
// implementations (database-backed or in-memory) live outside the domain layer.
type InvoiceRepository interface {
	// Save persists the aggregate, honoring optimistic-concurrency semantics.
	Save(ctx context.Context, a *model.InvoiceAggregate) error
	// FindByID loads the aggregate identified by id.
	FindByID(ctx context.Context, id string) (*model.InvoiceAggregate, error)
}
