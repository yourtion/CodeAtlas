package cli

import (
	"os"
	"testing"
)

const cliBinaryPath = "../../bin/cli"

// skipIfBinaryNotExists skips the test if the CLI binary doesn't exist
func skipIfBinaryNotExists(t *testing.T) {
	t.Helper()
	if _, err := os.Stat(cliBinaryPath); os.IsNotExist(err) {
		t.Skipf("CLI binary not found at %s. Run 'make build-cli' first.", cliBinaryPath)
	}
}
