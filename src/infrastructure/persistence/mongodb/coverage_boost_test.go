package mongodb

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"
)

// TestDocVersionGetters exercises the trivial Version() accessor on every
// persistence document type. These getters are only reached on the update path
// (a stored aggregate being re-saved) or through the shared upsert helper, so a
// pure table over the mappers pins them down without a live database.
func TestDocVersionGetters(t *testing.T) {
	const v = 42
	// The audit trail entry has no version field; every other persistence doc is
	// a VersionedDocument. Enumerate them explicitly.
	docs := []VersionedDocument{
		&appointmentDoc{Ver: v},
		&analyticsDashboardDoc{Ver: v},
		&userAccountDoc{Ver: v},
		&sessionDoc{Ver: v},
		&encounterDoc{Ver: v},
		&authorizationPolicyDoc{Ver: v},
		&careRelationshipDoc{Ver: v},
		&clinicDirectoryDoc{Ver: v},
		&cryptoKeyEnvelopeDoc{Ver: v},
		&insurancePolicyDoc{Ver: v},
		&intakeFormDoc{Ver: v},
		&invoiceDoc{Ver: v},
		&labOrderDoc{Ver: v},
		&messageThreadDoc{Ver: v},
		&paymentDoc{Ver: v},
		&prescriptionDoc{Ver: v},
		&providerScheduleDoc{Ver: v},
	}

	for _, d := range docs {
		if got := d.Version(); got != v {
			t.Fatalf("%T.Version() = %d, want %d", d, got, v)
		}
		// SetVersion round-trips through Version so the getter reflects a write.
		d.SetVersion(7)
		if got := d.Version(); got != 7 {
			t.Fatalf("%T.Version() after SetVersion(7) = %d, want 7", d, got)
		}
	}
}

// TestOptimisticConcurrencyError_ErrorAndUnwrap covers the typed error's Error
// rendering and its bridge to the shared concurrency sentinel.
func TestOptimisticConcurrencyError_ErrorAndUnwrap(t *testing.T) {
	e := &OptimisticConcurrencyError{Collection: "widgets", ID: "w-9", ExpectedVersion: 3}

	msg := e.Error()
	for _, want := range []string{"widgets", "w-9", "version 3"} {
		if !containsSub(msg, want) {
			t.Fatalf("Error() = %q, missing %q", msg, want)
		}
	}
	if !errors.Is(e, shared.ErrConcurrencyConflict) {
		t.Fatal("OptimisticConcurrencyError must wrap shared.ErrConcurrencyConflict")
	}
	if got := e.Unwrap(); got != shared.ErrConcurrencyConflict {
		t.Fatalf("Unwrap() = %v, want shared.ErrConcurrencyConflict", got)
	}
}

func containsSub(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// TestNewRegistry_TimeRoundTrip exercises the UTC time encoder through the
// registry: a time carried in a non-UTC location marshals and round-trips back
// to the same instant.
func TestNewRegistry_TimeRoundTrip(t *testing.T) {
	reg := NewRegistry()
	if reg == nil {
		t.Fatal("NewRegistry returned nil")
	}

	type wrap struct {
		T time.Time `bson:"t"`
	}
	loc := time.FixedZone("UTC+5", 5*60*60)
	in := wrap{T: time.Date(2026, 7, 6, 12, 30, 0, 0, loc)}

	data, err := bson.MarshalWithRegistry(reg, in)
	if err != nil {
		t.Fatalf("MarshalWithRegistry: %v", err)
	}

	var out wrap
	if err := bson.Unmarshal(data, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !out.T.Equal(in.T) {
		t.Fatalf("time round trip mismatch: got %v, want instant %v", out.T, in.T)
	}
	if out.T.UnixMilli() != in.T.UTC().UnixMilli() {
		t.Fatalf("stored millis %d, want %d", out.T.UnixMilli(), in.T.UTC().UnixMilli())
	}
}

// TestUTCTimeEncoder_RejectsWrongValue covers the encoder's guard branches: an
// invalid reflect.Value and a value whose type is not time.Time both yield a
// ValueEncoderError before the writer is touched.
func TestUTCTimeEncoder_RejectsWrongValue(t *testing.T) {
	enc := utcTimeEncoder{}

	if err := enc.EncodeValue(bsoncodec.EncodeContext{}, nil, reflect.Value{}); err == nil {
		t.Fatal("expected error for an invalid reflect.Value")
	}
	if err := enc.EncodeValue(bsoncodec.EncodeContext{}, nil, reflect.ValueOf(1234)); err == nil {
		t.Fatal("expected error for a non-time value")
	}
}

// TestDateRange covers every branch of the [from, to) predicate builder.
func TestDateRange(t *testing.T) {
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	if got := dateRange(time.Time{}, time.Time{}); len(got) != 0 {
		t.Fatalf("open-ended range should be empty, got %v", got)
	}
	if got := dateRange(from, time.Time{}); len(got) != 1 || got["$gte"] != from {
		t.Fatalf("from-only range wrong: %v", got)
	}
	if got := dateRange(time.Time{}, to); len(got) != 1 || got["$lt"] != to {
		t.Fatalf("to-only range wrong: %v", got)
	}
	got := dateRange(from, to)
	if got["$gte"] != from || got["$lt"] != to {
		t.Fatalf("bounded range wrong: %v", got)
	}
}

// TestCoerceVersion covers every numeric form BSON may decode a version into,
// plus the unknown-type fallback.
func TestCoerceVersion(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want int
	}{
		{"int", int(5), 5},
		{"int32", int32(6), 6},
		{"int64", int64(7), 7},
		{"nil", nil, 0},
		{"string", "9", 0},
		{"float", 3.14, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := coerceVersion(tc.in); got != tc.want {
				t.Fatalf("coerceVersion(%v) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

// TestEpochMillisRoundTrip covers both the zero-time and populated branches of
// the epoch-millis timestamp codec.
func TestEpochMillisRoundTrip(t *testing.T) {
	if got := epochMillis(time.Time{}); got != 0 {
		t.Fatalf("epochMillis(zero) = %d, want 0", got)
	}
	if got := fromEpochMillis(0); !got.IsZero() {
		t.Fatalf("fromEpochMillis(0) = %v, want zero time", got)
	}

	instant := time.Date(2026, 7, 6, 8, 15, 30, 0, time.UTC)
	ms := epochMillis(instant)
	if ms != instant.UnixMilli() {
		t.Fatalf("epochMillis = %d, want %d", ms, instant.UnixMilli())
	}
	got := fromEpochMillis(ms)
	if !got.Equal(instant) {
		t.Fatalf("fromEpochMillis round trip = %v, want %v", got, instant)
	}
	if got.Location() != time.UTC {
		t.Fatalf("fromEpochMillis location = %v, want UTC", got.Location())
	}
}

// TestStoredVersion covers the happy path and the unmarshal-failure fallback of
// the MemStore's version reader.
func TestStoredVersion(t *testing.T) {
	data, err := bson.Marshal(struct {
		Version int `bson:"version"`
	}{Version: 11})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if got := storedVersion(data); got != 11 {
		t.Fatalf("storedVersion = %d, want 11", got)
	}
	if got := storedVersion([]byte("not-bson")); got != -1 {
		t.Fatalf("storedVersion(garbage) = %d, want -1", got)
	}
}

// TestInWindow covers each boundary branch of the [from, to) predicate.
func TestInWindow(t *testing.T) {
	from := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC)
	before := from.Add(-time.Hour)
	inside := from.Add(time.Hour)
	atTo := to

	cases := []struct {
		name        string
		t, from, to time.Time
		want        bool
	}{
		{"before from", before, from, to, false},
		{"inside", inside, from, to, true},
		{"at upper bound is excluded", atTo, from, to, false},
		{"open ended both", inside, time.Time{}, time.Time{}, true},
		{"open lower, before to", before, time.Time{}, to, true},
		{"open upper, after from", inside, from, time.Time{}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := inWindow(tc.t, tc.from, tc.to); got != tc.want {
				t.Fatalf("inWindow = %v, want %v", got, tc.want)
			}
		})
	}
}

// offlineClient builds a mongo client that is never actually connected to a
// server. mongo.Connect returns immediately (connection is established lazily),
// so the accessors and constructors under test never touch the network.
func offlineClient(t *testing.T) *mongo.Client {
	t.Helper()
	cli, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://127.0.0.1:27017"))
	if err != nil {
		t.Fatalf("mongo.Connect: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = cli.Disconnect(ctx)
	})
	return cli
}

// TestMongoAdapterConstructors covers the constructors that wrap a live driver
// handle. They only store references, so an offline client suffices.
func TestMongoAdapterConstructors(t *testing.T) {
	cli := offlineClient(t)
	db := cli.Database("ancora_test")
	coll := db.Collection("things")

	if NewMongoStore(coll) == nil {
		t.Fatal("NewMongoStore returned nil")
	}
	if NewMongoTransactionRunner(cli) == nil {
		t.Fatal("NewMongoTransactionRunner returned nil")
	}
	if NewMongoAuditEntryCollection(coll) == nil {
		t.Fatal("NewMongoAuditEntryCollection returned nil")
	}
	if NewMongoAuditTrailRepository(coll) == nil {
		t.Fatal("NewMongoAuditTrailRepository returned nil")
	}
	if NewMongoFactSource(db) == nil {
		t.Fatal("NewMongoFactSource returned nil")
	}
}

// TestClientAccessors covers the Client accessor methods and Disconnect using a
// hand-built Client over an offline driver handle.
func TestClientAccessors(t *testing.T) {
	cli := offlineClient(t)
	c := &Client{mongo: cli, db: cli.Database("ancora_test")}

	if c.Database() == nil {
		t.Fatal("Database() returned nil")
	}
	if c.Collection("widgets") == nil {
		t.Fatal("Collection() returned nil")
	}
	if c.Mongo() != cli {
		t.Fatal("Mongo() did not return the underlying client")
	}
	if err := c.Disconnect(context.Background()); err != nil {
		t.Fatalf("Disconnect: %v", err)
	}
}

// TestConfigFromEnv covers the environment parsing, including the pool-size
// helper's default, valid and malformed branches and the database default.
func TestConfigFromEnv(t *testing.T) {
	t.Run("defaults and overrides", func(t *testing.T) {
		t.Setenv(envURI, "mongodb://example.test:27017")
		t.Setenv(envDatabase, "")             // exercise the DefaultDatabase fallback
		t.Setenv(envMaxPoolSize, "250")       // valid override
		t.Setenv(envMinPoolSize, "not-a-num") // malformed -> default

		cfg, err := ConfigFromEnv()
		if err != nil {
			t.Fatalf("ConfigFromEnv: %v", err)
		}
		if cfg.URI != "mongodb://example.test:27017" {
			t.Fatalf("URI = %q", cfg.URI)
		}
		if cfg.Database != DefaultDatabase {
			t.Fatalf("Database = %q, want %q", cfg.Database, DefaultDatabase)
		}
		if cfg.MaxPoolSize != 250 {
			t.Fatalf("MaxPoolSize = %d, want 250", cfg.MaxPoolSize)
		}
		if cfg.MinPoolSize != defaultMinPoolSize {
			t.Fatalf("MinPoolSize = %d, want default %d", cfg.MinPoolSize, defaultMinPoolSize)
		}
		if cfg.ConnectTimeout != defaultConnectTimeout {
			t.Fatalf("ConnectTimeout = %v, want %v", cfg.ConnectTimeout, defaultConnectTimeout)
		}
	})

	t.Run("explicit database", func(t *testing.T) {
		t.Setenv(envURI, "mongodb://example.test:27017")
		t.Setenv(envDatabase, "custom_db")
		cfg, err := ConfigFromEnv()
		if err != nil {
			t.Fatalf("ConfigFromEnv: %v", err)
		}
		if cfg.Database != "custom_db" {
			t.Fatalf("Database = %q, want custom_db", cfg.Database)
		}
	})

	t.Run("missing uri", func(t *testing.T) {
		t.Setenv(envURI, "")
		if _, err := ConfigFromEnv(); !errors.Is(err, ErrMissingURI) {
			t.Fatalf("expected ErrMissingURI, got %v", err)
		}
	})
}

// TestNewClient_MissingURI covers the early guard on an empty URI.
func TestNewClient_MissingURI(t *testing.T) {
	if _, err := NewClient(context.Background(), Config{}); !errors.Is(err, ErrMissingURI) {
		t.Fatalf("expected ErrMissingURI, got %v", err)
	}
}

// TestNewClient_MalformedURI covers the mongo.Connect error branch: a URI whose
// scheme is invalid is rejected synchronously, before any network I/O.
func TestNewClient_MalformedURI(t *testing.T) {
	cfg := Config{URI: "://not-a-valid-uri", ConnectTimeout: time.Second}
	if _, err := NewClient(context.Background(), cfg); err == nil {
		t.Fatal("expected an error for a malformed URI")
	}
}

// TestNewClient_HealthCheckFails covers the success-of-Connect + failing-ping
// path: a well-formed but unreachable URI connects, then the health-check ping
// fails within the (short) connect timeout, so NewClient disconnects and
// returns the error.
func TestNewClient_HealthCheckFails(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping ping-timeout path in -short mode")
	}
	cfg := Config{
		URI:            "mongodb://127.0.0.1:1/?serverSelectionTimeoutMS=200",
		Database:       "ancora_test",
		ConnectTimeout: 500 * time.Millisecond,
	}
	if _, err := NewClient(context.Background(), cfg); err == nil {
		t.Fatal("expected a health-check failure against an unreachable server")
	}
}
