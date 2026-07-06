package platform

import (
	"runtime"
	"strings"
	"testing"
)

func TestBuild_ReflectsPackageVars(t *testing.T) {
	// Save and restore the link-time vars so this test is self-contained.
	origV, origC, origD := Version, Commit, Date
	t.Cleanup(func() { Version, Commit, Date = origV, origC, origD })

	Version, Commit, Date = "1.2.3", "abc123", "2026-01-01T00:00:00Z"

	got := Build()
	if got.Version != "1.2.3" {
		t.Errorf("Version = %q, want 1.2.3", got.Version)
	}
	if got.Commit != "abc123" {
		t.Errorf("Commit = %q, want abc123", got.Commit)
	}
	if got.Date != "2026-01-01T00:00:00Z" {
		t.Errorf("Date = %q, want 2026-01-01T00:00:00Z", got.Date)
	}
	if got.Go != runtime.Version() {
		t.Errorf("Go = %q, want %q", got.Go, runtime.Version())
	}
}

func TestBuildInfo_String(t *testing.T) {
	b := BuildInfo{Version: "1.0.0", Commit: "deadbeef", Date: "2026-01-01", Go: "go1.99"}
	got := b.String()
	want := "version=1.0.0 commit=deadbeef built=2026-01-01 go=go1.99"
	if got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
	if !strings.Contains(got, "version=1.0.0") {
		t.Errorf("String() missing version field: %q", got)
	}
}
