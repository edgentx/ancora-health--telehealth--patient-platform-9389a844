// Package repository declares the persistence contracts for the
// administration-and-analytics bounded context.
package repository

import (
	"context"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/administrationandanalytics/model"
)

// AnalyticsDashboardRepository persists and retrieves AnalyticsDashboardAggregate
// instances. Concrete implementations (database-backed or in-memory) live outside
// the domain layer.
type AnalyticsDashboardRepository interface {
	// Save persists the aggregate, honoring optimistic-concurrency semantics.
	Save(ctx context.Context, a *model.AnalyticsDashboardAggregate) error
	// FindByID loads the aggregate identified by id.
	FindByID(ctx context.Context, id string) (*model.AnalyticsDashboardAggregate, error)
}
