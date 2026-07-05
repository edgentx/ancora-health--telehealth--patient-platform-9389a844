package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// versionField is the BSON field every persisted document uses to carry its
// optimistic-concurrency version. The store filters and increments on this key,
// so VersionedDocument implementations must tag their version field `bson:"version"`.
const versionField = "version"

// DocumentStore is the narrow persistence port the BaseRepository is written
// against. Isolating these four operations keeps the optimistic-concurrency and
// transaction logic in one place and lets tests substitute an in-memory store
// (MemStore) for the MongoDB-backed adapter without a live database.
type DocumentStore interface {
	// InsertOne stores a new document under id. Implementations should surface a
	// duplicate-key condition as an error.
	InsertOne(ctx context.Context, id string, doc any) error
	// FindOne decodes the document with the given id into dest, returning
	// ErrDocumentNotFound if no such document exists.
	FindOne(ctx context.Context, id string, dest any) error
	// ReplaceVersioned atomically replaces the document with id ONLY if its
	// stored version equals expectedVersion. matched is false (with a nil error)
	// when the version guard fails — the signal the repository turns into an
	// OptimisticConcurrencyError.
	ReplaceVersioned(ctx context.Context, id string, expectedVersion int, doc any) (matched bool, err error)
	// DeleteOne removes the document with id, reporting whether a document was
	// actually deleted.
	DeleteOne(ctx context.Context, id string) (deleted bool, err error)
}

// mongoStore adapts a *mongo.Collection to the DocumentStore port.
type mongoStore struct {
	collection *mongo.Collection
}

// NewMongoStore builds a DocumentStore backed by a MongoDB collection.
func NewMongoStore(collection *mongo.Collection) DocumentStore {
	return &mongoStore{collection: collection}
}

func (s *mongoStore) InsertOne(ctx context.Context, _ string, doc any) error {
	_, err := s.collection.InsertOne(ctx, doc)
	return err
}

func (s *mongoStore) FindOne(ctx context.Context, id string, dest any) error {
	err := s.collection.FindOne(ctx, bson.M{"_id": id}).Decode(dest)
	if err == mongo.ErrNoDocuments {
		return ErrDocumentNotFound
	}
	return err
}

func (s *mongoStore) ReplaceVersioned(ctx context.Context, id string, expectedVersion int, doc any) (bool, error) {
	filter := bson.M{"_id": id, versionField: expectedVersion}
	res, err := s.collection.ReplaceOne(ctx, filter, doc)
	if err != nil {
		return false, err
	}
	return res.MatchedCount == 1, nil
}

func (s *mongoStore) DeleteOne(ctx context.Context, id string) (bool, error) {
	res, err := s.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return false, err
	}
	return res.DeletedCount == 1, nil
}
