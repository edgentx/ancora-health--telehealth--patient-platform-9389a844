package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/persistence/mongodb"
)

// healthChecker is the readiness dependency: anything that can confirm its
// backing store is reachable. *mongodb.Client satisfies it via Ping.
type healthChecker interface {
	HealthCheck(ctx context.Context) error
}

// container is the dependency-injection root for the service. Per-aggregate
// scaffold stories wire their repositories and handlers into this struct as the
// platform grows; the shared base wires the MongoDB connection and its health
// probe.
type container struct {
	// health is the MongoDB readiness probe. It is nil when MONGODB_URI is not
	// configured (e.g. local runs without a database), in which case readiness
	// reports "not configured".
	health healthChecker
}

func newContainer(ctx context.Context) *container {
	c := &container{}

	cfg, err := mongodb.ConfigFromEnv()
	if err != nil {
		log.Printf("mongodb: not configured (%v); readiness probe will report unavailable", err)
		return c
	}

	client, err := mongodb.NewClient(ctx, cfg)
	if err != nil {
		log.Printf("mongodb: initial connection failed: %v; readiness probe will report unavailable", err)
		return c
	}
	log.Printf("mongodb: connected to database %q", cfg.Database)
	c.health = client
	return c
}

// routes builds the chi router and mounts the shared endpoints. Aggregate-level
// routes are registered by later stories that depend on this shared base.
func (c *container) routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Liveness: the process is up and serving.
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Readiness: the process can reach its MongoDB backing store.
	r.Get("/ready", c.handleReady)

	return r
}

// handleReady is the readiness probe. It pings MongoDB via the health check and
// returns 503 when the database is unconfigured or unreachable.
func (c *container) handleReady(w http.ResponseWriter, req *http.Request) {
	if c.health == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "unavailable",
			"reason": "database not configured",
		})
		return
	}

	ctx, cancel := context.WithTimeout(req.Context(), 2*time.Second)
	defer cancel()

	if err := c.health.HealthCheck(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "unavailable",
			"reason": "database unreachable",
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func main() {
	c := newContainer(context.Background())

	const addr = ":8000"
	log.Printf("Ancora Health — Telehealth & Patient Platform listening on %s", addr)
	if err := http.ListenAndServe(addr, c.routes()); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
