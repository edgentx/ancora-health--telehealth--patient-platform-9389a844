package platform

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// DefaultAddr is the listen address every entrypoint falls back to. A single
// port across services keeps the container contract uniform: the readiness
// probe, the metrics scrape and the HEALTHCHECK all target the same place
// regardless of which entrypoint the image runs.
const DefaultAddr = ":8000"

// ReadyFunc reports whether the process is ready to serve. Returning a non-nil
// error makes /ready answer 503; nil makes it answer 200. A service with no
// backing dependency can pass nil to NewOpsMux and always report ready.
type ReadyFunc func(ctx context.Context) error

// ListenAddr resolves the listen address from the environment: PORT (a bare
// number, the convention most orchestrators inject) takes precedence, then ADDR
// (a full host:port), then DefaultAddr. This lets the same image bind wherever
// the platform schedules it without a rebuild.
func ListenAddr() string {
	if p := os.Getenv("PORT"); p != "" {
		return ":" + p
	}
	if a := os.Getenv("ADDR"); a != "" {
		return a
	}
	return DefaultAddr
}

// NewMetricsRegistry builds a private Prometheus registry pre-loaded with the Go
// runtime and process collectors. Each entrypoint owns one so /metrics exposes
// exactly the series that process produced. (The REST API keeps its own registry
// wired through its observability middleware; this serves the realtime and
// worker entrypoints, which have no such middleware.)
func NewMetricsRegistry() *prometheus.Registry {
	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	return reg
}

// NewOpsMux returns an http.ServeMux exposing the operational surface every
// container is probed on: /health (liveness — is the process up), /ready
// (readiness — is it able to serve, delegated to ready), /metrics (the given
// registry in Prometheus exposition format) and /version (build identity). A nil
// ready func means "always ready", the correct signal for a dependency-free
// process. Callers may mount additional routes on the returned mux.
func NewOpsMux(reg *prometheus.Registry, ready ReadyFunc) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if ready != nil {
			if err := ready(r.Context()); err != nil {
				writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable", "error": err.Error()})
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})

	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	mux.HandleFunc("/version", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, Build())
	})

	return mux
}

// Serve runs handler on addr and blocks until ctx is cancelled (a SIGINT/SIGTERM
// from the orchestrator), then drains in-flight requests within a bounded grace
// period before returning. Returning cleanly on shutdown — rather than being
// SIGKILLed — is what lets a rolling deploy retire a replica without dropping
// live requests.
func Serve(ctx context.Context, addr string, handler http.Handler) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

// SelfCheck is the in-process readiness probe the container HEALTHCHECK runs as
// `<binary> -healthcheck`. It GETs /ready on the local listen address and
// returns an error unless the response is 200. Running the probe as the image's
// own binary means the runtime stage needs no shell, curl or wget — which is
// exactly why a distroless/scratch image can still declare a HEALTHCHECK.
func SelfCheck() error {
	addr := ListenAddr()
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("healthcheck: invalid listen address %q: %w", addr, err)
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}

	url := fmt.Sprintf("http://%s/ready", net.JoinHostPort(host, port))
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("healthcheck: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("healthcheck: /ready returned %d", resp.StatusCode)
	}
	return nil
}

// LogStartup emits a single structured startup line naming the service and the
// build it came from, so container logs pin every running replica to an exact
// commit.
func LogStartup(service, addr string) {
	log.Printf("%s starting — %s addr=%s", service, Build(), addr)
}

// writeJSON writes v as an indent-free JSON response with the given status.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
