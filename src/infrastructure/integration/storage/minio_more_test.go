package storage

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/integration"
)

// failingAudit rejects every outbound-access record.
type failingAudit struct{ err error }

func (f failingAudit) RecordOutboundAccess(context.Context, integration.OutboundAccess) error {
	return f.err
}

func newStoreWithAudit(t *testing.T, h http.HandlerFunc, audit integration.AuditRecorder) (*Adapter, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(h)
	client := integration.NewClient(srv.Client(), integration.Config{MaxAttempts: 1})
	a, err := NewAdapter(client, Config{
		Endpoint:  srv.URL,
		Region:    "us-east-1",
		AccessKey: "AKIDEXAMPLE",
		SecretKey: "secretkey",
	}, audit)
	if err != nil {
		t.Fatalf("NewAdapter: %v", err)
	}
	return a, srv
}

func TestNewAdapter_RejectsUnparseableEndpoint(t *testing.T) {
	cases := []struct {
		name     string
		endpoint string
	}{
		{"no host", "not-a-url"},
		{"control character", "http://\x7f"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewAdapter(nil, Config{
				Endpoint:  tc.endpoint,
				Region:    "us-east-1",
				AccessKey: "AK",
				SecretKey: "sk",
			}, nil)
			if !errors.Is(err, ErrInvalidConfig) {
				t.Fatalf("err = %v, want ErrInvalidConfig", err)
			}
		})
	}
}

func TestMapStoreError(t *testing.T) {
	tests := []struct {
		name string
		in   error
		want error
	}{
		{"404 not found", &integration.StatusError{StatusCode: http.StatusNotFound}, ErrObjectNotFound},
		{"403 rejected", &integration.StatusError{StatusCode: http.StatusForbidden}, ErrStoreRejected},
		{"400 rejected", &integration.StatusError{StatusCode: http.StatusBadRequest}, ErrStoreRejected},
		{"500 unavailable", &integration.StatusError{StatusCode: http.StatusInternalServerError}, ErrStoreUnavailable},
		{"transport unavailable", errors.New("dial tcp: refused"), ErrStoreUnavailable},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := mapStoreError(tc.in); !errors.Is(got, tc.want) {
				t.Fatalf("mapStoreError(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestPut_ServerErrorMapsToUnavailable(t *testing.T) {
	a, srv := newStore(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}, "")
	defer srv.Close()

	err := a.Put(context.Background(), ObjectRef{Bucket: "b", Key: "k"}, Object{Data: []byte("x")})
	if !errors.Is(err, ErrStoreUnavailable) {
		t.Fatalf("err = %v, want ErrStoreUnavailable", err)
	}
}

func TestGet_RejectedMapsError(t *testing.T) {
	a, srv := newStore(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}, "")
	defer srv.Close()

	_, err := a.Get(context.Background(), ObjectRef{Bucket: "b", Key: "k"})
	if !errors.Is(err, ErrStoreRejected) {
		t.Fatalf("err = %v, want ErrStoreRejected", err)
	}
}

func TestGet_RejectsUnscopedRef(t *testing.T) {
	a, srv := newStore(t, func(http.ResponseWriter, *http.Request) {}, "")
	defer srv.Close()

	if _, err := a.Get(context.Background(), ObjectRef{Bucket: "b"}); !errors.Is(err, ErrInvalidRef) {
		t.Fatalf("err = %v, want ErrInvalidRef", err)
	}
}

func TestPut_AuditFailureAborts(t *testing.T) {
	var called bool
	a, srv := newStoreWithAudit(t, func(http.ResponseWriter, *http.Request) { called = true },
		failingAudit{err: errors.New("audit down")})
	defer srv.Close()

	if err := a.Put(context.Background(), ObjectRef{Bucket: "b", Key: "k"}, Object{Data: []byte("x")}); err == nil {
		t.Fatal("expected audit failure to abort Put")
	}
	if called {
		t.Fatal("store called despite audit failure")
	}
}

func TestPresignedGetURL_RejectsUnscopedRef(t *testing.T) {
	a, srv := newStore(t, func(http.ResponseWriter, *http.Request) {}, "")
	defer srv.Close()

	if _, err := a.PresignedGetURL(context.Background(), ObjectRef{Key: "k"}, time.Minute); !errors.Is(err, ErrInvalidRef) {
		t.Fatalf("err = %v, want ErrInvalidRef", err)
	}
}

func TestPresignedGetURL_NonPositiveTTLDefaultsTo15Min(t *testing.T) {
	a, srv := newStore(t, func(http.ResponseWriter, *http.Request) {}, "")
	defer srv.Close()

	url, err := a.PresignedGetURL(context.Background(), ObjectRef{Bucket: "b", Key: "k"}, 0)
	if err != nil {
		t.Fatalf("PresignedGetURL: %v", err)
	}
	// 15 minutes == 900 seconds.
	if !strings.Contains(url, "X-Amz-Expires=900") {
		t.Fatalf("presigned url %q missing default TTL of 900s", url)
	}
}

func TestPresignedGetURL_AuditFailureAborts(t *testing.T) {
	a, srv := newStoreWithAudit(t, func(http.ResponseWriter, *http.Request) {},
		failingAudit{err: errors.New("audit down")})
	defer srv.Close()

	if _, err := a.PresignedGetURL(context.Background(), ObjectRef{Bucket: "b", Key: "k"}, time.Minute); err == nil {
		t.Fatal("expected audit failure to abort presign")
	}
}
