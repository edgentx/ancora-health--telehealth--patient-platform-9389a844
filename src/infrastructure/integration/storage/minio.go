// Package storage is the outbound object-storage adapter for documents and
// attachments. It targets a MinIO (S3-compatible) endpoint, signing every
// request with AWS Signature Version 4, scoping objects to a bucket and key, and
// issuing time-limited presigned URLs for direct client download. Every outbound
// access to a stored object — which may be PHI — is audited.
package storage

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/integration"
)

// ObjectRef identifies a stored object by its bucket and key. Both are required;
// the adapter refuses an empty bucket or key so an object can never be written to
// or read from an unscoped location.
type ObjectRef struct {
	Bucket string
	Key    string
}

// Object is the payload and metadata of a stored object.
type Object struct {
	Data        []byte
	ContentType string
}

// ObjectStore is the outbound port for object storage. Put uploads an object;
// Get downloads one; PresignedGetURL mints a time-limited download URL that
// needs no credentials, for handing a document directly to a client.
type ObjectStore interface {
	Put(ctx context.Context, ref ObjectRef, obj Object) error
	Get(ctx context.Context, ref ObjectRef) (Object, error)
	PresignedGetURL(ctx context.Context, ref ObjectRef, ttl time.Duration) (string, error)
}

// Sentinel errors surfaced to callers.
var (
	// ErrInvalidRef is returned when an object reference omits its bucket or key.
	ErrInvalidRef = errors.New("storage: object reference requires a bucket and key")

	// ErrInvalidConfig is returned when the adapter is built without an endpoint,
	// region, or credentials.
	ErrInvalidConfig = errors.New("storage: endpoint, region, and credentials are required")

	// ErrObjectNotFound is returned when a Get targets a key that does not exist
	// (the store answered 404).
	ErrObjectNotFound = errors.New("storage: object not found")

	// ErrStoreRejected is returned when the store answers with a client (4xx)
	// status other than 404.
	ErrStoreRejected = errors.New("storage: object store rejected the request")

	// ErrStoreUnavailable is returned when the store could not be reached or
	// answered with a server error after retries.
	ErrStoreUnavailable = errors.New("storage: object store unavailable")
)

// Config captures the S3-compatible endpoint and credentials.
type Config struct {
	// Endpoint is the store root, e.g. "http://minio:9000". Its scheme selects
	// HTTP vs HTTPS.
	Endpoint string
	// Region is the signing region, e.g. "us-east-1". MinIO accepts any region as
	// long as signing and request agree.
	Region string
	// AccessKey and SecretKey are the S3 credentials.
	AccessKey string
	SecretKey string
	// KeyPrefix optionally namespaces every key under a fixed prefix so a shared
	// bucket can isolate this platform's objects. It may be empty.
	KeyPrefix string
}

// Adapter is the MinIO/S3 ObjectStore. It signs requests with SigV4 and sends
// them through the shared transport, gaining its retry and timeout envelope.
type Adapter struct {
	client *integration.Client
	cfg    Config
	creds  credentials
	host   string
	audit  integration.AuditRecorder
	now    func() time.Time
}

// NewAdapter builds the object-storage adapter over an integration transport and
// the S3-compatible endpoint in cfg. audit may be nil, in which case
// outbound-access recording is skipped.
func NewAdapter(client *integration.Client, cfg Config, audit integration.AuditRecorder) (*Adapter, error) {
	if cfg.Endpoint == "" || cfg.Region == "" || cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, ErrInvalidConfig
	}
	u, err := url.Parse(cfg.Endpoint)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("%w: bad endpoint %q", ErrInvalidConfig, cfg.Endpoint)
	}
	return &Adapter{
		client: client,
		cfg:    cfg,
		creds:  credentials{AccessKey: cfg.AccessKey, SecretKey: cfg.SecretKey, Region: cfg.Region},
		host:   u.Host,
		audit:  audit,
		now:    time.Now,
	}, nil
}

// Put uploads obj to the bucket/key in ref. It scopes the key under any
// configured prefix, signs the request over the body's hash, records the
// outbound access, and maps the store's response to the error vocabulary.
func (a *Adapter) Put(ctx context.Context, ref ObjectRef, obj Object) error {
	key, err := a.scopedKey(ref)
	if err != nil {
		return err
	}
	if err := a.record(ctx, "storage.put", ref); err != nil {
		return err
	}

	canonicalURI := canonicalURIPath(ref.Bucket, key)
	header := http.Header{}
	if obj.ContentType != "" {
		header.Set("Content-Type", obj.ContentType)
	}
	signHeaders(a.creds, http.MethodPut, canonicalURI, "", a.host, hexSHA256(obj.Data), header, a.now())

	_, err = a.client.Send(ctx, &integration.Request{
		Method: http.MethodPut,
		URL:    a.cfg.Endpoint + canonicalURI,
		Header: header,
		Body:   obj.Data,
	})
	if err != nil {
		return mapStoreError(err)
	}
	return nil
}

// Get downloads the object at ref. It signs the request over the empty-body
// hash, records the outbound access, and maps a 404 to ErrObjectNotFound.
func (a *Adapter) Get(ctx context.Context, ref ObjectRef) (Object, error) {
	key, err := a.scopedKey(ref)
	if err != nil {
		return Object{}, err
	}
	if err := a.record(ctx, "storage.get", ref); err != nil {
		return Object{}, err
	}

	canonicalURI := canonicalURIPath(ref.Bucket, key)
	header := http.Header{}
	signHeaders(a.creds, http.MethodGet, canonicalURI, "", a.host, hexSHA256(nil), header, a.now())

	resp, err := a.client.Send(ctx, &integration.Request{
		Method: http.MethodGet,
		URL:    a.cfg.Endpoint + canonicalURI,
		Header: header,
	})
	if err != nil {
		return Object{}, mapStoreError(err)
	}
	return Object{
		Data:        resp.Body,
		ContentType: resp.Header.Get("Content-Type"),
	}, nil
}

// PresignedGetURL mints a credential-free URL that grants GET access to ref for
// ttl. It is used to hand a document to a client (e.g. a browser) without
// proxying the bytes through the platform. Minting a URL is a local signing
// operation — no request is made — but it is still audited as an outbound PHI
// access because the URL grants access to the object.
func (a *Adapter) PresignedGetURL(ctx context.Context, ref ObjectRef, ttl time.Duration) (string, error) {
	key, err := a.scopedKey(ref)
	if err != nil {
		return "", err
	}
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	if err := a.record(ctx, "storage.presign", ref); err != nil {
		return "", err
	}

	canonicalURI := canonicalURIPath(ref.Bucket, key)
	return presign(a.creds, http.MethodGet, a.cfg.Endpoint, canonicalURI, a.host, ttl, a.now()), nil
}

// scopedKey validates the reference and applies the configured key prefix, so
// every object is confined to its bucket and the platform's key namespace.
func (a *Adapter) scopedKey(ref ObjectRef) (string, error) {
	if ref.Bucket == "" || ref.Key == "" {
		return "", ErrInvalidRef
	}
	if a.cfg.KeyPrefix == "" {
		return ref.Key, nil
	}
	prefix := strings.TrimSuffix(a.cfg.KeyPrefix, "/")
	return prefix + "/" + strings.TrimPrefix(ref.Key, "/"), nil
}

// record audits an outbound access referencing the bucket/key, never the object
// contents.
func (a *Adapter) record(ctx context.Context, action string, ref ObjectRef) error {
	if err := integration.RecordIfSet(ctx, a.audit, integration.OutboundAccess{
		ActorContext: "object-store-client",
		ResourceRef:  ref.Bucket + "/" + ref.Key,
		Action:       action,
		Destination:  a.host,
		OccurredAt:   a.now(),
	}); err != nil {
		return fmt.Errorf("storage: audit outbound access: %w", err)
	}
	return nil
}

// mapStoreError translates a transport error into the storage error vocabulary:
// 404 is ErrObjectNotFound, another 4xx is ErrStoreRejected, and anything else
// (5xx or transport) is ErrStoreUnavailable.
func mapStoreError(err error) error {
	var statusErr *integration.StatusError
	if errors.As(err, &statusErr) {
		switch {
		case statusErr.StatusCode == http.StatusNotFound:
			return ErrObjectNotFound
		case statusErr.StatusCode >= 400 && statusErr.StatusCode < 500:
			return fmt.Errorf("%w: status %d", ErrStoreRejected, statusErr.StatusCode)
		default:
			return fmt.Errorf("%w: status %d", ErrStoreUnavailable, statusErr.StatusCode)
		}
	}
	return fmt.Errorf("%w: %v", ErrStoreUnavailable, err)
}
