package apiintegration

import (
	"os"
	"testing"
)

// newEnv boots the API surface for a single test and registers its teardown. It
// reads MONGODB_URI and ANCO_REDIS_ADDR so the same test body runs against live
// infrastructure in CI and the in-memory equivalents locally; a failure to reach
// configured infrastructure fails the test rather than silently degrading, so a
// misconfigured CI job is loud.
func newEnv(t *testing.T) *environment {
	t.Helper()
	env, err := buildEnvironment(os.Getenv("MONGODB_URI"), os.Getenv("ANCO_REDIS_ADDR"))
	if err != nil {
		t.Fatalf("build environment: %v", err)
	}
	t.Cleanup(env.cleanup)
	if env.usingMongo || env.usingRedis {
		t.Logf("integration infra: mongo=%v redis=%v", env.usingMongo, env.usingRedis)
	}
	return env
}
