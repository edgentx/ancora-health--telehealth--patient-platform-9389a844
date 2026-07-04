package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// container is the dependency-injection root for the service. Per-aggregate
// scaffold stories wire their repositories and handlers into this struct as the
// platform grows; for the shared base it holds no dependencies yet.
type container struct{}

func newContainer() *container {
	return &container{}
}

// routes builds the chi router and mounts the shared endpoints. Aggregate-level
// routes are registered by later stories that depend on this shared base.
func (c *container) routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	return r
}

func main() {
	c := newContainer()

	const addr = ":8000"
	log.Printf("Ancora Health — Telehealth & Patient Platform listening on %s", addr)
	if err := http.ListenAndServe(addr, c.routes()); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
