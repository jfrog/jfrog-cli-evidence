package tests

import (
	"github.com/jfrog/jfrog-cli-evidence/tests/e2e"
	"testing"
)

// TestEvidenceCLICommands is the main test entry point for jfrog-cli-evidence E2E tests.
//
// This test is called ONLY in local or in standalone jfrog-cli-evidence tests, NOT in jfrog-cli integration.
// For jfrog-cli integration, see jfrog-cli/evidence_test.go which uses the same test runner.
//
// # How it works:
//
// 1. Validation:
//   - Checks that SetupE2ETests() successfully initialized all required components
//   - Fails fast if any critical component (CLI, service managers) is missing
//
// 2. Test Runner Initialization:
//   - Creates EvidenceE2ETestsRunner with all required CLI instances and service managers
//   - The runner encapsulates all test logic and is shared between jfrog-cli-evidence and jfrog-cli
//
// 3. Test Data Preparation:
//   - Calls runner.PrepareTestsData() to set up shared resources (key pairs, etc.)
//   - Registers cleanup via t.Cleanup() to ensure resources are deleted even if tests fail
//
// 4. Test Execution:
//   - Calls runner.RunEvidenceCliTests() which runs ALL evidence command test suites:
//   - CreateEvidence suite (artifact, build, release bundle, package, markdown, etc.)
//   - GenerateKeyPair suite
//   - VerifyEvidence suite
//   - GetEvidence suite
//   - Each suite contains multiple subtests for different scenarios
//
// # Test Structure:
//
// TestEvidenceCLICommands (this test)
//
//	└── runner.RunEvidenceCliTests()
//	    ├── CreateEvidenceSuite
//	    ├── GenerateKeyPairSuite
//	    ├── VerifyEvidenceSuite
//	    └── GetEvidenceSuite
//
// # Running the tests:
//
// Local Docker:
//
//	make start-e2e-env  # Bootstrap environment
//	go test ./tests/e2e/tests/...
//
// SaaS Environment (https://ecosysjfrog.jfrog.io/):
//
// Note: Tests run against the JFrog ecosystem SaaS environment at https://ecosysjfrog.jfrog.io/
// This is the designated test environment with proper permissions and projects configured.
func TestEvidenceCLICommands(t *testing.T) {
	// Validate that all required components are initialized
	if err := e2e.ValidateSetup(); err != nil {
		t.Errorf("Setup validation failed: %v", err)
	}

	// Initialize the E2E tests runner with required clients
	runner := NewEvidenceE2ETestsRunner(e2e.EvidenceUserCli,
		e2e.EvidenceAdminCli,
		e2e.EvidenceProjectCli,
		e2e.ArtifactoryClient,
		e2e.AccessClient,
		e2e.LifecycleClient)

	// Setup: Prepare shared test data (key pair, etc.)
	t.Log("=== Setting up shared E2E test data ===")
	err := runner.PrepareTestsData()
	if err != nil {
		t.Errorf("Failed to prepare test data: %v", err)
	}

	// Register cleanup to run after all tests (even if tests fail)
	t.Cleanup(func() {
		t.Log("=== Running cleanup for shared E2E test data ===")
		runner.CleanupTestsData()
	})

	// Run all E2E tests
	runner.RunEvidenceCliTests(t)
}
