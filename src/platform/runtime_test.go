package platform

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestListenAddr(t *testing.T) {
	t.Run("PORT takes precedence", func(t *testing.T) {
		t.Setenv("PORT", "9090")
		t.Setenv("ADDR", "10.0.0.1:1234")
		if got := ListenAddr(); got != ":9090" {
			t.Fatalf("ListenAddr() = %q, want :9090", got)
		}
	})
	t.Run("ADDR when no PORT", func(t *testing.T) {
		t.Setenv("PORT", "")
		t.Setenv("ADDR", "10.0.0.1:1234")
		if got := ListenAddr(); got != "10.0.0.1:1234" {
			t.Fatalf("ListenAddr() = %q, want 10.0.0.1:1234", got)
		}
	})
	t.Run("default when neither set", func(t *testing.T) {
		t.Setenv("PORT", "")
		t.Setenv("ADDR", "")
		if got := ListenAddr(); got != DefaultAddr {
			t.Fatalf("ListenAddr() = %q, want %q", got, DefaultAddr)
		}
	})
}

func TestNewMetricsRegistry(t *testing.T) {
	reg := NewMetricsRegistry()
	if reg == nil {
		t.Fatal("NewMetricsRegistry() returned nil")
	}
	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather() error = %v", err)
	}
	if len(mfs) == 0 {
		t.Fatal("expected Go/process collectors to produce metrics")
	}
}

func doRequest(t *testing.T, mux *http.ServeMux, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	return rr
}

func TestNewOpsMux_HealthVersionMetrics(t *testing.T) {
	mux := NewOpsMux(NewMetricsRegistry(), nil)

	health := doRequest(t, mux, "/health")
	if health.Code != http.StatusOK {
		t.Fatalf("/health = %d, want 200", health.Code)
	}
	if ct := health.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("/health content-type = %q, want application/json", ct)
	}
	var hbody map[string]string
	if err := json.Unmarshal(health.Body.Bytes(), &hbody); err != nil {
		t.Fatalf("/health body not JSON: %v", err)
	}
	if hbody["status"] != "ok" {
		t.Fatalf("/health status = %q, want ok", hbody["status"])
	}

	version := doRequest(t, mux, "/version")
	if version.Code != http.StatusOK {
		t.Fatalf("/version = %d, want 200", version.Code)
	}
	var info BuildInfo
	if err := json.Unmarshal(version.Body.Bytes(), &info); err != nil {
		t.Fatalf("/version body not BuildInfo JSON: %v", err)
	}
	if info.Go == "" {
		t.Fatal("/version missing go version")
	}

	metrics := doRequest(t, mux, "/metrics")
	if metrics.Code != http.StatusOK {
		t.Fatalf("/metrics = %d, want 200", metrics.Code)
	}
}

func TestNewOpsMux_ReadyNilAlwaysReady(t *testing.T) {
	mux := NewOpsMux(NewMetricsRegistry(), nil)
	rr := doRequest(t, mux, "/ready")
	if rr.Code != http.StatusOK {
		t.Fatalf("/ready = %d, want 200", rr.Code)
	}
	var body map[string]string
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body["status"] != "ready" {
		t.Fatalf("/ready status = %q, want ready", body["status"])
	}
}

func TestNewOpsMux_ReadyFuncOK(t *testing.T) {
	mux := NewOpsMux(NewMetricsRegistry(), func(context.Context) error { return nil })
	rr := doRequest(t, mux, "/ready")
	if rr.Code != http.StatusOK {
		t.Fatalf("/ready = %d, want 200", rr.Code)
	}
}

func TestNewOpsMux_ReadyFuncUnavailable(t *testing.T) {
	mux := NewOpsMux(NewMetricsRegistry(), func(context.Context) error {
		return errors.New("db down")
	})
	rr := doRequest(t, mux, "/ready")
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("/ready = %d, want 503", rr.Code)
	}
	var body map[string]string
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body["status"] != "unavailable" || body["error"] != "db down" {
		t.Fatalf("/ready body = %+v, want unavailable/db down", body)
	}
}

func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	addr := l.Addr().String()
	_ = l.Close()
	return addr
}

func TestServe_GracefulShutdownOnContextCancel(t *testing.T) {
	addr := freeAddr(t)
	mux := NewOpsMux(NewMetricsRegistry(), nil)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Serve(ctx, addr, mux) }()

	// Wait until the server accepts connections, then hit an endpoint.
	waitForServer(t, addr)
	resp, err := http.Get("http://" + addr + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	_ = resp.Body.Close()

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Serve() returned %v on graceful shutdown, want nil", err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("Serve() did not return after context cancel")
	}
}

func TestServe_ListenError(t *testing.T) {
	// An unparseable address makes ListenAndServe fail immediately, driving the
	// error branch of Serve.
	err := Serve(context.Background(), "invalid-address-without-port", NewOpsMux(NewMetricsRegistry(), nil))
	if err == nil {
		t.Fatal("Serve() = nil, want listen error")
	}
}

func waitForServer(t *testing.T, addr string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		c, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			_ = c.Close()
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("server at %s never came up", addr)
}

func TestSelfCheck_OK(t *testing.T) {
	addr := freeAddr(t)
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("split addr: %v", err)
	}

	srv := &http.Server{Addr: addr, Handler: NewOpsMux(NewMetricsRegistry(), nil)}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })
	waitForServer(t, addr)

	t.Setenv("PORT", port)
	t.Setenv("ADDR", "")
	if err := SelfCheck(); err != nil {
		t.Fatalf("SelfCheck() = %v, want nil", err)
	}
}

func TestSelfCheck_Non200(t *testing.T) {
	addr := freeAddr(t)
	_, port, _ := net.SplitHostPort(addr)

	mux := http.NewServeMux()
	mux.HandleFunc("/ready", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })
	waitForServer(t, addr)

	t.Setenv("PORT", port)
	t.Setenv("ADDR", "")
	err = SelfCheck()
	if err == nil || !strings.Contains(err.Error(), "returned 503") {
		t.Fatalf("SelfCheck() = %v, want error mentioning 503", err)
	}
}

func TestSelfCheck_ConnectionRefused(t *testing.T) {
	addr := freeAddr(t) // reserved then freed: nothing is listening
	_, port, _ := net.SplitHostPort(addr)
	t.Setenv("PORT", port)
	t.Setenv("ADDR", "")
	err := SelfCheck()
	if err == nil || !strings.Contains(err.Error(), "healthcheck:") {
		t.Fatalf("SelfCheck() = %v, want healthcheck error", err)
	}
}

func TestSelfCheck_InvalidListenAddr(t *testing.T) {
	// A bare host with no port makes net.SplitHostPort fail.
	t.Setenv("PORT", "")
	t.Setenv("ADDR", "not-a-valid-host-port")
	err := SelfCheck()
	if err == nil || !strings.Contains(err.Error(), "invalid listen address") {
		t.Fatalf("SelfCheck() = %v, want invalid listen address error", err)
	}
}

func TestLogStartup(t *testing.T) {
	// Smoke test: LogStartup only writes to the standard logger; assert it does
	// not panic and executes fully.
	LogStartup("api", ":8000")
}

func TestWriteJSON(t *testing.T) {
	rr := httptest.NewRecorder()
	writeJSON(rr, http.StatusTeapot, map[string]int{"n": 42})
	if rr.Code != http.StatusTeapot {
		t.Fatalf("status = %d, want 418", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type = %q, want application/json", ct)
	}
	var body map[string]int
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("body not JSON: %v", err)
	}
	if body["n"] != 42 {
		t.Fatalf("body = %+v, want n=42", body)
	}
}
