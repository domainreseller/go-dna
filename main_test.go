package dna

import (
	"os"
	"testing"
)

// TestMain keeps the whole test run away from the real Sentry project:
// clients built without WithSentryDSN get an empty DSN (telemetry no-op).
func TestMain(m *testing.M) {
	defaultSentryDSN = ""
	os.Exit(m.Run())
}
