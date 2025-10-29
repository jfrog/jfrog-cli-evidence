package tests

import (
	"github.com/jfrog/jfrog-cli-evidence/tests/e2e"
	"testing"
)

func TestAllEvidenceCLIFlows(t *testing.T) {
	if e2e.EvidenceCli == nil || e2e.ArtifactoryClient == nil {
		t.Fatal("Evidence CLI or Artifactory services manager not initialized")
	}
	runner := NewEvidenceE2ETestsRunner(e2e.EvidenceCli, e2e.ArtifactoryClient)
	// Run All E2E Tests
	runner.RunEvidenceCliTests(t)
}
