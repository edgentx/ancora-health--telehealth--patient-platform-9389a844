package storage

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/integration"
)

func newStore(t *testing.T, h http.HandlerFunc, prefix string) (*Adapter, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(h)
	client := integration.NewClient(srv.Client(), integration.Config{MaxAttempts: 2})
	a, err := NewAdapter(client, Config{
		Endpoint:  srv.URL,
		Region:    "us-east-1",
		AccessKey: "AKIDEXAMPLE",
		SecretKey: "secretkey",
		KeyPrefix: prefix,
	}, nil)
	if err != nil {
		t.Fatalf("NewAdapter: %v", err)
	}
	return a, srv
}

func TestNewAdapter_RejectsIncompleteConfig(t *testing.T) {
	_, err := NewAdapter(nil, Config{Endpoint: "http://x", Region: "us-east-1"}, nil)
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("err = %v, want ErrInvalidConfig", err)
	}
}

func TestPut_SignsAndScopesBucketKey(t *testing.T) {
	var gotPath, gotAuth, gotAmz string
	var gotBody []byte
	a, srv := newStore(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotAmz = r.Header.Get("X-Amz-Content-Sha256")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}, "docs")
	defer srv.Close()

	err := a.Put(context.Background(), ObjectRef{Bucket: "attachments", Key: "patient/report.pdf"},
		Object{Data: []byte("PDFDATA"), ContentType: "application/pdf"})
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	// Key is scoped under the configured prefix, within the bucket.
	if gotPath != "/attachments/docs/patient/report.pdf" {
		t.Fatalf("path = %q, want /attachments/docs/patient/report.pdf", gotPath)
	}
	if !strings.HasPrefix(gotAuth, "AWS4-HMAC-SHA256 ") {
		t.Fatalf("Authorization not SigV4: %q", gotAuth)
	}
	if gotAmz == "" {
		t.Fatal("X-Amz-Content-Sha256 not set")
	}
	if string(gotBody) != "PDFDATA" {
		t.Fatalf("body = %q", gotBody)
	}
}

func TestGet_ReturnsObject(t *testing.T) {
	a, srv := newStore(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write([]byte("BYTES"))
	}, "")
	defer srv.Close()

	obj, err := a.Get(context.Background(), ObjectRef{Bucket: "attachments", Key: "a/b.pdf"})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(obj.Data) != "BYTES" || obj.ContentType != "application/pdf" {
		t.Fatalf("obj = %+v", obj)
	}
}

func TestGet_NotFoundMapsError(t *testing.T) {
	a, srv := newStore(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}, "")
	defer srv.Close()

	_, err := a.Get(context.Background(), ObjectRef{Bucket: "attachments", Key: "missing"})
	if !errors.Is(err, ErrObjectNotFound) {
		t.Fatalf("err = %v, want ErrObjectNotFound", err)
	}
}

func TestPut_RejectsUnscopedRef(t *testing.T) {
	a, srv := newStore(t, func(http.ResponseWriter, *http.Request) {}, "")
	defer srv.Close()

	err := a.Put(context.Background(), ObjectRef{Bucket: "", Key: "k"}, Object{Data: []byte("x")})
	if !errors.Is(err, ErrInvalidRef) {
		t.Fatalf("err = %v, want ErrInvalidRef", err)
	}
}

func TestPresignedGetURL_CarriesSignature(t *testing.T) {
	a, srv := newStore(t, func(http.ResponseWriter, *http.Request) {}, "docs")
	defer srv.Close()

	url, err := a.PresignedGetURL(context.Background(), ObjectRef{Bucket: "attachments", Key: "r.pdf"}, 5*time.Minute)
	if err != nil {
		t.Fatalf("PresignedGetURL: %v", err)
	}
	for _, want := range []string{
		srv.URL + "/attachments/docs/r.pdf",
		"X-Amz-Algorithm=AWS4-HMAC-SHA256",
		"X-Amz-Expires=300",
		"X-Amz-Signature=",
	} {
		if !strings.Contains(url, want) {
			t.Fatalf("presigned url %q missing %q", url, want)
		}
	}
}
