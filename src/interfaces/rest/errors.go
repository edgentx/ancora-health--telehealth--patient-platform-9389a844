package rest

import (
	"errors"
	"net/http"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/persistence/mongodb"
)

// ErrorResponse is the single, consistent envelope every failed request returns.
// It is an RFC-7807-style problem document trimmed to the fields this platform
// needs: a stable machine-readable Code, a human-readable Message, and the HTTP
// Status echoed into the body for clients that only inspect the payload.
//
// Message is always drawn from a domain sentinel or a fixed validation string —
// never from request data — so no PHI can leak into an error body.
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"status"`
}

// Machine-readable error codes carried in ErrorResponse.Code. They are stable
// contract identifiers clients can branch on without parsing prose.
const (
	codeValidation    = "validation_error"
	codeNotFound      = "not_found"
	codeConflict      = "conflict"
	codeUnprocessable = "unprocessable_entity"
	codeInternal      = "internal_error"
)

// validationError is a typed transport-level validation failure. Handlers raise
// it when a request DTO is malformed or missing a required field, before any
// domain command is built. Its message names the offending field but never
// echoes its value, keeping request data out of the response.
type validationError struct {
	msg string
}

func (e validationError) Error() string { return e.msg }

// badRequest builds a validationError with a fixed, PHI-free message.
func badRequest(msg string) error { return validationError{msg: msg} }

// conflictError marks a domain error that must surface as 409 Conflict even
// though it is an ordinary domain rule violation (which otherwise maps to 422).
// Handlers wrap the specific domain sentinels that represent a genuine resource
// conflict — a double-booked slot, a duplicate unique key — in this type so the
// central classifier can tell them apart from invariant violations.
type conflictError struct {
	err error
}

func (e conflictError) Error() string { return e.err.Error() }
func (e conflictError) Unwrap() error { return e.err }

// asConflict tags err so statusForError classifies it as 409 Conflict.
func asConflict(err error) error { return conflictError{err: err} }

// domainError tags an aggregate rule violation so the classifier maps it to 422
// Unprocessable Entity. Tagging is what lets the classifier distinguish a
// well-formed request the domain refused (422) from an unexpected infrastructure
// failure on a repository call (500): only errors that flowed through execErr
// carry this tag.
type domainError struct {
	err error
}

func (e domainError) Error() string { return e.err.Error() }
func (e domainError) Unwrap() error { return e.err }

// execErr classifies the error an aggregate's Execute returned. A sentinel in
// conflicts (a double-booked slot, a duplicate unique code) becomes a 409;
// an unknown command stays itself (a 400); every other rule violation is tagged
// as a 422 domain error. Handlers wrap every Execute result with it, so domain
// refusals never fall through to the 500 bucket reserved for infrastructure.
func execErr(err error, conflicts ...error) error {
	if err == nil {
		return nil
	}
	for _, c := range conflicts {
		if errors.Is(err, c) {
			return asConflict(err)
		}
	}
	if errors.Is(err, shared.ErrUnknownCommand) {
		return err
	}
	return domainError{err: err}
}

// statusForError maps an error to its HTTP status and response envelope. It is
// the one place transport status is decided, so every handler returns
// consistent codes:
//
//   - transport validation failures      -> 400 Bad Request
//   - record not found                   -> 404 Not Found
//   - optimistic-concurrency / tagged    -> 409 Conflict
//   - unknown command                    -> 400 Bad Request
//   - tagged domain rule violation       -> 422 Unprocessable Entity
//   - anything else (infrastructure)     -> 500 Internal Server Error
//
// Domain-rule and infrastructure errors are told apart by provenance, not by
// inspecting their text: an Execute error reaches here wrapped by execErr (so it
// is a validation/conflict/domain tag), whereas a raw repository failure that is
// neither not-found nor a conflict is an unexpected infrastructure fault and
// correctly falls through to 500.
//
// The Message comes from the error's own text for the client-correctable classes
// (a curated sentinel or a fixed field message, always PHI-free) and from a
// fixed string for 404/500, where echoing the cause could leak an identifier or
// an infrastructure detail.
func statusForError(err error) (int, ErrorResponse) {
	var vErr validationError
	var dErr domainError
	switch {
	case errors.As(err, &vErr):
		return resp(http.StatusBadRequest, codeValidation, err)
	case errors.Is(err, mongodb.ErrDocumentNotFound):
		return resp(http.StatusNotFound, codeNotFound, notFoundMessage)
	case isConflict(err):
		return resp(http.StatusConflict, codeConflict, err)
	case errors.Is(err, shared.ErrUnknownCommand):
		return resp(http.StatusBadRequest, codeValidation, err)
	case errors.As(err, &dErr):
		return resp(http.StatusUnprocessableEntity, codeUnprocessable, err)
	default:
		return resp(http.StatusInternalServerError, codeInternal, internalMessage)
	}
}

// notFoundMessage and internalMessage are the fixed, PHI-free bodies used where
// echoing the underlying error could leak an identifier or infrastructure
// detail.
var (
	notFoundMessage = errors.New("resource not found")
	internalMessage = errors.New("internal server error")
)

// isConflict reports whether err represents a resource conflict: either an
// optimistic-concurrency clash from the persistence layer or a domain sentinel a
// handler explicitly tagged with asConflict.
func isConflict(err error) bool {
	var cErr conflictError
	if errors.As(err, &cErr) {
		return true
	}
	return errors.Is(err, shared.ErrConcurrencyConflict) || errors.Is(err, mongodb.ErrDuplicateKey)
}

// resp assembles the status/envelope pair for a status, code and error message.
func resp(status int, code string, err error) (int, ErrorResponse) {
	return status, ErrorResponse{Code: code, Message: err.Error(), Status: status}
}

// writeError classifies err and writes the structured JSON error envelope with
// the matching HTTP status.
func writeError(w http.ResponseWriter, err error) {
	status, body := statusForError(err)
	writeJSON(w, status, body)
}
