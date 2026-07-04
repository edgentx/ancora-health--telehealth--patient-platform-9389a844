// Package repository declares the persistence ports for the billing-and-insurance
// bounded context. Concrete adapters (in-memory, SQL, etc.) live elsewhere and
// implement these interfaces.
package repository

import (
	"context"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/billingandinsurance/model"
)

// InsurancePolicyRepository is the persistence port for InsurancePolicyAggregate.
// Save persists an aggregate; FindByID loads one by its identity.
type InsurancePolicyRepository interface {
	Save(ctx context.Context, a *model.InsurancePolicyAggregate) error
	FindByID(ctx context.Context, id string) (*model.InsurancePolicyAggregate, error)
}
