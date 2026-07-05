package rest

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// maxRequestBody caps how many bytes a request body may carry. It bounds memory
// per request and makes an oversized payload a clean 400 rather than an
// unbounded read.
const maxRequestBody = 1 << 20 // 1 MiB

// writeJSON serializes body as JSON with the given status. Encoding failures are
// swallowed after the header is written, since the status line is already
// committed to the wire and nothing more can be signaled to the client.
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if body == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(body)
}

// decodeJSON reads and strictly decodes the request body into dst. It rejects
// unknown fields and trailing content, and maps every failure mode to a fixed,
// PHI-free validation error so a malformed payload never surfaces request data
// in the response.
func decodeJSON(r *http.Request, dst any) error {
	if r.Body == nil {
		return badRequest("request body is required")
	}
	limited := http.MaxBytesReader(nil, r.Body, maxRequestBody)
	dec := json.NewDecoder(limited)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return badRequest("request body is required")
		}
		return badRequest("request body is not valid JSON")
	}
	// A well-formed request carries exactly one JSON value; anything trailing is a
	// malformed payload.
	if dec.More() {
		return badRequest("request body must contain a single JSON object")
	}
	return nil
}
