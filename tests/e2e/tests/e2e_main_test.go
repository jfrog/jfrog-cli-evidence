package tests

import (
	"os"
	"testing"

	"github.com/jfrog/jfrog-cli-evidence/tests/e2e"
)

// TestMain is the entry point for tests subpackage
// It calls the parent e2e package's setup to initialize shared resources
func TestMain(m *testing.M) {
	e2e.SetupE2ETests()
	result := m.Run()
	os.Exit(result)
}

