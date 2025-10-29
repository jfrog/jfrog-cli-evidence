package tests

import (
	"testing"

	coreTests "github.com/jfrog/jfrog-cli-core/v2/utils/tests"
	"github.com/jfrog/jfrog-client-go/artifactory"
)

// EvidenceE2ETestsRunner contains the reusable test logic
// This runner is used in ALL environments: local, jfrog-cli-evidence CI, and jfrog-cli
type EvidenceE2ETestsRunner struct {
	EvidenceCLI     *coreTests.JfrogCli
	ServicesManager artifactory.ArtifactoryServicesManager
}

func NewEvidenceE2ETestsRunner(evidenceCli *coreTests.JfrogCli, servicesManager artifactory.ArtifactoryServicesManager) *EvidenceE2ETestsRunner {
	return &EvidenceE2ETestsRunner{
		EvidenceCLI:     evidenceCli,
		ServicesManager: servicesManager,
	}
}

func (r *EvidenceE2ETestsRunner) RunEvidenceCliTests(t *testing.T) {
	// Run all E2E test flows
	t.Run("CreateEvidenceHappyFlow", func(t *testing.T) {
		r.RunCreateEvidenceHappyFlow(t)
	})
	t.Run("CreateEvidenceHappyFlow2", func(t *testing.T) {
		r.RunCreateEvidenceHappyFlow2(t)
	})
	t.Run("CreateEvidenceHappyFlow3", func(t *testing.T) {
		r.RunCreateEvidenceHappyFlow3(t)
	})

	// Add more test flows here as they are implemented
	// t.Run("VerifyEvidence", func(t *testing.T) {
	//     r.RunVerifyEvidenceTest(t)
	// })
}
