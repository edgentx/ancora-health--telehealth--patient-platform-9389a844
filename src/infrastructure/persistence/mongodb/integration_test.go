package mongodb

import (
	"context"
	"errors"
	"os"
	"strconv"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

// TestIntegration_MongoRoundTrip exercises the full stack against a real MongoDB
// instance. It is skipped unless MONGODB_URI is set, so the default (CI) test run
// stays hermetic — no Docker or database required. Point MONGODB_URI at an
// ephemeral instance (testcontainers, mongodb-memory-server, or a throwaway
// container) to run it:
//
//	MONGODB_URI=mongodb://localhost:27017 go test ./src/infrastructure/persistence/mongodb/...
func TestIntegration_MongoRoundTrip(t *testing.T) {
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		t.Skip("MONGODB_URI not set; skipping live MongoDB integration test")
	}

	ctx := context.Background()
	cfg, err := ConfigFromEnv()
	if err != nil {
		t.Fatalf("ConfigFromEnv: %v", err)
	}

	client, err := NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Disconnect(context.Background())
	})

	if err := client.HealthCheck(ctx); err != nil {
		t.Fatalf("HealthCheck: %v", err)
	}

	// Use a unique, self-cleaning collection so repeated runs don't collide.
	collName := "s68_it_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	coll := client.Collection(collName)
	t.Cleanup(func() {
		_ = coll.Drop(context.Background())
	})

	repo := NewBaseRepository(NewMongoStore(coll), collName)

	if err := repo.Insert(ctx, &widget{WID: "w-1", Name: "live"}); err != nil {
		t.Fatalf("Insert: %v", err)
	}

	var winner, loser widget
	if err := repo.FindByID(ctx, "w-1", &winner); err != nil {
		t.Fatalf("FindByID winner: %v", err)
	}
	if err := repo.FindByID(ctx, "w-1", &loser); err != nil {
		t.Fatalf("FindByID loser: %v", err)
	}

	winner.Name = "winner"
	if err := repo.Update(ctx, &winner); err != nil {
		t.Fatalf("winner Update: %v", err)
	}

	loser.Name = "loser"
	if err := repo.Update(ctx, &loser); err == nil {
		t.Fatal("expected optimistic concurrency conflict against live MongoDB")
	} else {
		var occ *OptimisticConcurrencyError
		if !errors.As(err, &occ) {
			t.Fatalf("expected *OptimisticConcurrencyError, got %T: %v", err, err)
		}
	}

	// Confirm at rest the stored version advanced exactly once.
	var raw bson.M
	if err := coll.FindOne(ctx, bson.M{"_id": "w-1"}).Decode(&raw); err != nil {
		t.Fatalf("raw FindOne: %v", err)
	}
	if v, _ := raw["version"].(int32); v != 2 {
		t.Fatalf("expected stored version 2, got %v", raw["version"])
	}
}
