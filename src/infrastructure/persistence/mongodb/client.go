package mongodb

import (
	"context"
	"errors"
	"os"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// Environment variables read by ConfigFromEnv.
const (
	envURI         = "MONGODB_URI"
	envDatabase    = "MONGODB_DATABASE"
	envMaxPoolSize = "MONGODB_MAX_POOL_SIZE"
	envMinPoolSize = "MONGODB_MIN_POOL_SIZE"
)

// DefaultDatabase is the project-dedicated database on the shared VForce360
// MongoDB instance used when MONGODB_DATABASE is not set.
const DefaultDatabase = "ancora_health"

const (
	defaultMaxPoolSize    uint64 = 100
	defaultMinPoolSize    uint64 = 5
	defaultConnectTimeout        = 10 * time.Second
)

// ErrMissingURI is returned by ConfigFromEnv when MONGODB_URI is not set.
var ErrMissingURI = errors.New("mongodb: MONGODB_URI is not set")

// Config captures the connection settings for the Mongo client factory.
type Config struct {
	URI            string
	Database       string
	MaxPoolSize    uint64
	MinPoolSize    uint64
	ConnectTimeout time.Duration
}

// ConfigFromEnv reads the client configuration from the environment. MONGODB_URI
// is required (it names the shared instance and credentials); the database name
// and pool bounds fall back to project defaults.
func ConfigFromEnv() (Config, error) {
	uri := os.Getenv(envURI)
	if uri == "" {
		return Config{}, ErrMissingURI
	}
	db := os.Getenv(envDatabase)
	if db == "" {
		db = DefaultDatabase
	}
	return Config{
		URI:            uri,
		Database:       db,
		MaxPoolSize:    envUint(envMaxPoolSize, defaultMaxPoolSize),
		MinPoolSize:    envUint(envMinPoolSize, defaultMinPoolSize),
		ConnectTimeout: defaultConnectTimeout,
	}, nil
}

// Client wraps a connected mongo client bound to the project database and
// exposes the collection accessor and health check the rest of the platform
// depends on.
type Client struct {
	mongo *mongo.Client
	db    *mongo.Database
}

// NewClient connects to MongoDB using cfg, applies the connection pool, verifies
// connectivity with a Ping, and binds the project-dedicated database. The
// returned Client must be closed with Disconnect.
func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	if cfg.URI == "" {
		return nil, ErrMissingURI
	}
	opts := options.Client().
		ApplyURI(cfg.URI).
		SetMaxPoolSize(cfg.MaxPoolSize).
		SetMinPoolSize(cfg.MinPoolSize).
		SetRegistry(NewRegistry())

	connectCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()

	m, err := mongo.Connect(connectCtx, opts)
	if err != nil {
		return nil, err
	}

	c := &Client{mongo: m, db: m.Database(cfg.Database)}
	if err := c.HealthCheck(connectCtx); err != nil {
		_ = m.Disconnect(context.Background())
		return nil, err
	}
	return c, nil
}

// Database returns the bound project database.
func (c *Client) Database() *mongo.Database { return c.db }

// Collection returns a handle to a named collection in the project database.
func (c *Client) Collection(name string) *mongo.Collection { return c.db.Collection(name) }

// Mongo exposes the underlying driver client, e.g. to build a transaction
// runner with NewMongoTransactionRunner.
func (c *Client) Mongo() *mongo.Client { return c.mongo }

// HealthCheck pings the primary to confirm the connection is live. It is the
// probe consumed by the service's readiness endpoint.
func (c *Client) HealthCheck(ctx context.Context) error {
	return c.mongo.Ping(ctx, readpref.Primary())
}

// Disconnect closes the client's connection pool.
func (c *Client) Disconnect(ctx context.Context) error {
	return c.mongo.Disconnect(ctx)
}

// envUint reads an unsigned integer environment variable, falling back to def on
// an unset or malformed value.
func envUint(key string, def uint64) uint64 {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	parsed, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return def
	}
	return parsed
}
