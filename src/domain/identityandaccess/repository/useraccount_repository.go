// Package repository declares the persistence ports for the identity-and-access
// bounded context. Concrete adapters (in-memory, SQL, etc.) live elsewhere and
// implement these interfaces.
package repository

import (
	"context"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/identityandaccess/model"
)

// UserAccountRepository is the persistence port for UserAccountAggregate.
// Save persists an aggregate; FindByID loads one by its identity.
type UserAccountRepository interface {
	Save(ctx context.Context, a *model.UserAccountAggregate) error
	FindByID(ctx context.Context, id string) (*model.UserAccountAggregate, error)
}
