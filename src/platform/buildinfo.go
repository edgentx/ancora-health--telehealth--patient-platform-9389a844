// Package platform holds process-level runtime helpers shared by every backend
// entrypoint (the API, realtime and worker commands under cmd/): build metadata
// stamped in at link time, the operational HTTP surface (/health, /ready,
// /metrics, /version), and the in-process readiness self-probe the container
// HEALTHCHECK invokes. Keeping this here lets each cmd/* main stay a thin wiring
// shim while the container contract (how it reports health and identity) is
// defined in exactly one place.
package platform

import (
	"fmt"
	"runtime"
)

// Build metadata. These are deliberately package-level vars, not consts, so the
// container build can stamp real values in at link time with
// -ldflags "-X .../src/platform.Version=… -X .../src/platform.Commit=… -X .../src/platform.Date=…".
// The defaults are what an un-stamped `go build`/`go run` reports locally.
var (
	// Version is the release version (git tag or semver) of the build.
	Version = "dev"
	// Commit is the git SHA the build was produced from.
	Commit = "unknown"
	// Date is the RFC3339 UTC timestamp the build was produced at.
	Date = "unknown"
)

// BuildInfo is the structured build identity served from /version and logged at
// startup, so an operator can tie a running container back to an exact commit.
type BuildInfo struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
	Go      string `json:"go"`
}

// Build returns the current build identity, resolving the Go toolchain version
// from the runtime so it never drifts from the compiler that produced the binary.
func Build() BuildInfo {
	return BuildInfo{
		Version: Version,
		Commit:  Commit,
		Date:    Date,
		Go:      runtime.Version(),
	}
}

// String renders the build identity as a single startup log line.
func (b BuildInfo) String() string {
	return fmt.Sprintf("version=%s commit=%s built=%s go=%s", b.Version, b.Commit, b.Date, b.Go)
}
