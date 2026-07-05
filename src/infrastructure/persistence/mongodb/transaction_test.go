package mongodb

import (
	"context"
	"errors"
	"testing"
)

// TestRunInTransaction_RollsBackOnError verifies the transaction helper contract:
// a unit of work that writes and then fails leaves no partial state behind. The
// MemStore stands in for MongoDB's session/transaction so the test is hermetic.
func TestRunInTransaction_RollsBackOnError(t *testing.T) {
	store := NewMemStore()
	repo := NewBaseRepository(store, "widgets")
	ctx := context.Background()

	// Seed a committed document that must survive the aborted transaction.
	if err := repo.Insert(ctx, &widget{WID: "keep", Name: "original"}); err != nil {
		t.Fatalf("seed Insert: %v", err)
	}

	var runner TransactionRunner = store
	boom := errors.New("business rule violated")

	err := runner.RunInTransaction(ctx, func(txCtx context.Context) error {
		// Write inside the transaction...
		if err := repo.Insert(txCtx, &widget{WID: "temp", Name: "should vanish"}); err != nil {
			return err
		}
		if err := repo.Update(txCtx, &widget{WID: "keep", Ver: 1, Name: "mutated"}); err != nil {
			return err
		}
		// ...then fail, forcing a rollback of everything above.
		return boom
	})

	if !errors.Is(err, boom) {
		t.Fatalf("expected the work error to propagate, got %v", err)
	}

	// The inserted document must have been rolled back.
	var temp widget
	if err := repo.FindByID(ctx, "temp", &temp); !errors.Is(err, ErrDocumentNotFound) {
		t.Fatalf("expected temp document to be rolled back, got %v", err)
	}

	// The pre-existing document must be untouched.
	var keep widget
	if err := repo.FindByID(ctx, "keep", &keep); err != nil {
		t.Fatalf("FindByID keep: %v", err)
	}
	if keep.Name != "original" || keep.Ver != 1 {
		t.Fatalf("committed document was mutated by a rolled-back transaction: %+v", keep)
	}
}

func TestRunInTransaction_CommitsOnSuccess(t *testing.T) {
	store := NewMemStore()
	repo := NewBaseRepository(store, "widgets")
	ctx := context.Background()

	err := store.RunInTransaction(ctx, func(txCtx context.Context) error {
		return repo.Insert(txCtx, &widget{WID: "committed", Name: "kept"})
	})
	if err != nil {
		t.Fatalf("RunInTransaction: %v", err)
	}

	var got widget
	if err := repo.FindByID(ctx, "committed", &got); err != nil {
		t.Fatalf("expected committed document to persist, got %v", err)
	}
}
