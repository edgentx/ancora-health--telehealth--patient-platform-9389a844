package mongodb

import (
	"context"
	"errors"
	"testing"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"
)

// widget is a minimal VersionedDocument used to exercise the base repository.
type widget struct {
	WID  string `bson:"_id"`
	Ver  int    `bson:"version"`
	Name string `bson:"name"`
}

func (w *widget) ID() string       { return w.WID }
func (w *widget) Version() int     { return w.Ver }
func (w *widget) SetVersion(v int) { w.Ver = v }

func newRepo() *BaseRepository {
	return NewBaseRepository(NewMemStore(), "widgets")
}

func TestBaseRepository_InsertFindByID(t *testing.T) {
	repo := newRepo()
	ctx := context.Background()

	if err := repo.Insert(ctx, &widget{WID: "w-1", Name: "gizmo"}); err != nil {
		t.Fatalf("Insert: %v", err)
	}

	var got widget
	if err := repo.FindByID(ctx, "w-1", &got); err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.Name != "gizmo" {
		t.Fatalf("name mismatch: %q", got.Name)
	}
	if got.Ver != 1 {
		t.Fatalf("insert should stamp version 1, got %d", got.Ver)
	}
}

func TestBaseRepository_FindByID_NotFound(t *testing.T) {
	repo := newRepo()
	var got widget
	if err := repo.FindByID(context.Background(), "missing", &got); !errors.Is(err, ErrDocumentNotFound) {
		t.Fatalf("expected ErrDocumentNotFound, got %v", err)
	}
}

func TestBaseRepository_Update_IncrementsVersion(t *testing.T) {
	repo := newRepo()
	ctx := context.Background()

	w := &widget{WID: "w-1", Name: "v1"}
	if err := repo.Insert(ctx, w); err != nil {
		t.Fatalf("Insert: %v", err)
	}

	w.Name = "v2"
	if err := repo.Update(ctx, w); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if w.Ver != 2 {
		t.Fatalf("expected version 2 after update, got %d", w.Ver)
	}

	var got widget
	if err := repo.FindByID(ctx, "w-1", &got); err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.Name != "v2" || got.Ver != 2 {
		t.Fatalf("stored state mismatch: %+v", got)
	}
}

// TestBaseRepository_Update_StaleVersionConflict is the acceptance scenario:
// two writers load the same version, both try to update, and the loser gets a
// typed OptimisticConcurrencyError.
func TestBaseRepository_Update_StaleVersionConflict(t *testing.T) {
	repo := newRepo()
	ctx := context.Background()

	if err := repo.Insert(ctx, &widget{WID: "w-1", Name: "base"}); err != nil {
		t.Fatalf("Insert: %v", err)
	}

	// Two independent reads observe the same version (1).
	var winner, loser widget
	if err := repo.FindByID(ctx, "w-1", &winner); err != nil {
		t.Fatalf("FindByID winner: %v", err)
	}
	if err := repo.FindByID(ctx, "w-1", &loser); err != nil {
		t.Fatalf("FindByID loser: %v", err)
	}

	// Winner commits first, advancing the stored version to 2.
	winner.Name = "winner"
	if err := repo.Update(ctx, &winner); err != nil {
		t.Fatalf("winner Update: %v", err)
	}

	// Loser updates from the now-stale version 1 and must be rejected.
	loser.Name = "loser"
	err := repo.Update(ctx, &loser)
	if err == nil {
		t.Fatal("expected an optimistic concurrency conflict, got nil")
	}

	var occ *OptimisticConcurrencyError
	if !errors.As(err, &occ) {
		t.Fatalf("expected *OptimisticConcurrencyError, got %T: %v", err, err)
	}
	if occ.ID != "w-1" || occ.ExpectedVersion != 1 {
		t.Fatalf("unexpected conflict details: %+v", occ)
	}
	// It must also satisfy the shared sentinel for cross-cutting handling.
	if !errors.Is(err, shared.ErrConcurrencyConflict) {
		t.Fatal("OptimisticConcurrencyError should wrap shared.ErrConcurrencyConflict")
	}

	// The stored document is unchanged by the failed update.
	var got widget
	if err := repo.FindByID(ctx, "w-1", &got); err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.Name != "winner" {
		t.Fatalf("failed update leaked into storage: %q", got.Name)
	}
}

func TestBaseRepository_Delete(t *testing.T) {
	repo := newRepo()
	ctx := context.Background()

	if err := repo.Insert(ctx, &widget{WID: "w-1"}); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	if err := repo.Delete(ctx, "w-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := repo.Delete(ctx, "w-1"); !errors.Is(err, ErrDocumentNotFound) {
		t.Fatalf("expected ErrDocumentNotFound on second delete, got %v", err)
	}
}
