package rest

import (
	"net/http"
	"strconv"
)

// Pagination defaults and bounds. Limit is clamped so a client cannot ask for an
// unbounded page; the default keeps first-page responses small.
const (
	defaultPageLimit = 25
	maxPageLimit     = 200
)

// Page describes a validated pagination + filtering request parsed from the
// query string. Offset/Limit drive the window; Filter carries an optional
// caller-supplied filter token the read side interprets.
type Page struct {
	Offset int
	Limit  int
	Filter string
}

// parsePage reads limit, offset and filter from the query string, applying
// defaults and clamping to safe bounds. A non-numeric or negative limit/offset
// is a client mistake and returns a validation error rather than being silently
// coerced, so pagination bugs surface loudly.
func parsePage(r *http.Request) (Page, error) {
	q := r.URL.Query()

	limit := defaultPageLimit
	if raw := q.Get("limit"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 0 {
			return Page{}, badRequest("limit must be a non-negative integer")
		}
		limit = v
	}
	if limit == 0 || limit > maxPageLimit {
		limit = maxPageLimit
	}

	offset := 0
	if raw := q.Get("offset"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 0 {
			return Page{}, badRequest("offset must be a non-negative integer")
		}
		offset = v
	}

	return Page{Offset: offset, Limit: limit, Filter: q.Get("filter")}, nil
}

// window returns the [start:end] sub-slice bounds of the page against a
// collection of length n, clamped so neither bound escapes the slice. It lets a
// read handler apply the parsed Page to an in-memory result set safely.
func (p Page) window(n int) (start, end int) {
	start = p.Offset
	if start > n {
		start = n
	}
	end = start + p.Limit
	if end > n {
		end = n
	}
	return start, end
}
