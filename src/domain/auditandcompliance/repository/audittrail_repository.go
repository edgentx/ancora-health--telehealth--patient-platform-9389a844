// Package repository declares the persistence ports for the audit-and-compliance
// bounded context. Concrete adapters (in-memory, SQL, etc.) live elsewhere and
// implement these interfaces.
package repository

import (
	"context"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/auditandcompliance/model"
)

// AuditTrailRepository is the persistence port for AuditTrailAggregate.
// Save persists an aggregate; FindByID loads one by its identity.
type AuditTrailRepository interface {
	Save(ctx context.Context, a *model.AuditTrailAggregate) error
	FindByID(ctx context.Context, id string) (*model.AuditTrailAggregate, error)
}
