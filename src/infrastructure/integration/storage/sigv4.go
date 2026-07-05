package storage

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// AWS Signature Version 4 constants. MinIO speaks the same S3 signing protocol,
// so the adapter signs requests exactly as it would for AWS S3.
const (
	sigV4Algorithm = "AWS4-HMAC-SHA256"
	sigV4Service   = "s3"
	sigV4Request   = "aws4_request"
	// unsignedPayload is used when the body is not hashed into the signature —
	// notably for presigned URLs, where the eventual body is unknown at sign time.
	unsignedPayload = "UNSIGNED-PAYLOAD"

	amzDateFormat  = "20060102T150405Z"
	shortDateFmt   = "20060102"
	headerAmzDate  = "X-Amz-Date"
	headerAmzSHA   = "X-Amz-Content-Sha256"
	headerAuthz    = "Authorization"
	signedHeadersH = "host;x-amz-content-sha256;x-amz-date"
)

// credentials are the access key pair and region a request is signed under.
type credentials struct {
	AccessKey string
	SecretKey string
	Region    string
}

// signHeaders applies header-based SigV4 to a request. It sets the x-amz-date,
// x-amz-content-sha256, and Authorization headers on header so the request Send
// transmits is signed. host is the request Host (as it will appear on the wire),
// and payloadHash is the hex SHA-256 of the body (or unsignedPayload).
func signHeaders(creds credentials, method, canonicalURI, canonicalQuery, host, payloadHash string, header http.Header, t time.Time) {
	amzDate := t.UTC().Format(amzDateFormat)
	shortDate := t.UTC().Format(shortDateFmt)

	header.Set(headerAmzDate, amzDate)
	header.Set(headerAmzSHA, payloadHash)

	canonicalHeaders := "host:" + host + "\n" +
		"x-amz-content-sha256:" + payloadHash + "\n" +
		"x-amz-date:" + amzDate + "\n"

	canonicalRequest := strings.Join([]string{
		method,
		canonicalURI,
		canonicalQuery,
		canonicalHeaders,
		signedHeadersH,
		payloadHash,
	}, "\n")

	scope := strings.Join([]string{shortDate, creds.Region, sigV4Service, sigV4Request}, "/")
	stringToSign := strings.Join([]string{
		sigV4Algorithm,
		amzDate,
		scope,
		hexSHA256([]byte(canonicalRequest)),
	}, "\n")

	signature := hex.EncodeToString(hmacSHA256(signingKey(creds, shortDate), stringToSign))

	authz := sigV4Algorithm +
		" Credential=" + creds.AccessKey + "/" + scope +
		", SignedHeaders=" + signedHeadersH +
		", Signature=" + signature
	header.Set(headerAuthz, authz)
}

// presign builds a query-signed (presigned) URL for method against the given
// path, valid for expiry. The returned URL carries the signature in its query so
// it can be handed to a client with no credentials — used for time-limited
// document downloads.
func presign(creds credentials, method, endpoint, canonicalURI, host string, expiry time.Duration, t time.Time) string {
	amzDate := t.UTC().Format(amzDateFormat)
	shortDate := t.UTC().Format(shortDateFmt)
	scope := strings.Join([]string{shortDate, creds.Region, sigV4Service, sigV4Request}, "/")

	q := url.Values{}
	q.Set("X-Amz-Algorithm", sigV4Algorithm)
	q.Set("X-Amz-Credential", creds.AccessKey+"/"+scope)
	q.Set("X-Amz-Date", amzDate)
	q.Set("X-Amz-Expires", strconv.Itoa(int(expiry.Seconds())))
	q.Set("X-Amz-SignedHeaders", "host")

	canonicalQuery := encodeCanonicalQuery(q)
	canonicalHeaders := "host:" + host + "\n"

	canonicalRequest := strings.Join([]string{
		method,
		canonicalURI,
		canonicalQuery,
		canonicalHeaders,
		"host",
		unsignedPayload,
	}, "\n")

	stringToSign := strings.Join([]string{
		sigV4Algorithm,
		amzDate,
		scope,
		hexSHA256([]byte(canonicalRequest)),
	}, "\n")

	signature := hex.EncodeToString(hmacSHA256(signingKey(creds, shortDate), stringToSign))
	return endpoint + canonicalURI + "?" + canonicalQuery + "&X-Amz-Signature=" + signature
}

// signingKey derives the SigV4 signing key by chaining HMACs over the scope
// components, starting from the secret.
func signingKey(creds credentials, shortDate string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+creds.SecretKey), shortDate)
	kRegion := hmacSHA256(kDate, creds.Region)
	kService := hmacSHA256(kRegion, sigV4Service)
	return hmacSHA256(kService, sigV4Request)
}

// encodeCanonicalQuery renders query parameters in the canonical form SigV4
// requires: keys sorted, each key and value RFC 3986 encoded, joined by '&'.
func encodeCanonicalQuery(q url.Values) string {
	keys := make([]string, 0, len(q))
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		for _, v := range q[k] {
			parts = append(parts, rfc3986Escape(k)+"="+rfc3986Escape(v))
		}
	}
	return strings.Join(parts, "&")
}

// canonicalURIPath URI-encodes an object key into a canonical path, preserving
// the '/' separators between segments so nested keys sign correctly.
func canonicalURIPath(bucket, key string) string {
	segments := strings.Split(key, "/")
	for i, s := range segments {
		segments[i] = rfc3986Escape(s)
	}
	return "/" + rfc3986Escape(bucket) + "/" + strings.Join(segments, "/")
}

// rfc3986Escape percent-encodes s per RFC 3986, leaving the unreserved set
// (A-Z a-z 0-9 - _ . ~) untouched — the encoding SigV4 canonicalisation
// mandates, which is stricter than url.QueryEscape.
func rfc3986Escape(s string) string {
	const unreserved = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_.~"
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if strings.IndexByte(unreserved, c) >= 0 {
			b.WriteByte(c)
		} else {
			b.WriteByte('%')
			b.WriteByte(upperHex(c >> 4))
			b.WriteByte(upperHex(c & 0xf))
		}
	}
	return b.String()
}

func upperHex(n byte) byte {
	if n < 10 {
		return '0' + n
	}
	return 'A' + (n - 10)
}

func hmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

func hexSHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
