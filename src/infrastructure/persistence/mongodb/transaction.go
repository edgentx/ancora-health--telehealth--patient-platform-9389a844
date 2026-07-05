package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
)

// UnitOfWork is a caller-supplied function executed inside a transaction. The
// context it receives is session-scoped: every store operation performed with it
// participates in the surrounding transaction, so returning an error rolls back
// all of them atomically.
type UnitOfWork func(ctx context.Context) error

// TransactionRunner runs a UnitOfWork inside a single atomic transaction,
// committing on success and rolling back on error. The MongoDB adapter uses a
// server session + multi-document transaction; MemStore offers an in-memory
// implementation for tests.
type TransactionRunner interface {
	RunInTransaction(ctx context.Context, work UnitOfWork) error
}

// MongoTransactionRunner runs units of work inside a MongoDB multi-document
// transaction using a server session.
type MongoTransactionRunner struct {
	client *mongo.Client
}

// NewMongoTransactionRunner builds a TransactionRunner over a mongo client.
func NewMongoTransactionRunner(client *mongo.Client) *MongoTransactionRunner {
	return &MongoTransactionRunner{client: client}
}

// RunInTransaction opens a session, runs work inside a transaction, and lets the
// driver commit on success or abort (roll back) on error. Any error returned by
// work aborts the transaction and is propagated to the caller.
func (t *MongoTransactionRunner) RunInTransaction(ctx context.Context, work UnitOfWork) error {
	return t.client.UseSession(ctx, func(sc mongo.SessionContext) error {
		_, err := sc.WithTransaction(sc, func(txCtx mongo.SessionContext) (interface{}, error) {
			if err := work(txCtx); err != nil {
				return nil, err
			}
			return nil, nil
		})
		return err
	})
}
