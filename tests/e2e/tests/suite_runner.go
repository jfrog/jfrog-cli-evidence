package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	coreTests "github.com/jfrog/jfrog-cli-core/v2/utils/tests"
	"github.com/jfrog/jfrog-cli-evidence/tests/e2e/utils"
	"github.com/jfrog/jfrog-client-go/access"
	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/lifecycle"
)

// Shared key pair for all E2E tests
// Generated once in PrepareTestsData() and cleaned up after all tests
var (
	SharedPrivateKeyPath string
	SharedPublicKeyPath  string
	SharedKeyAlias       string
	SharedKeyDir         string
)

// EvidenceE2ETestsRunner contains the reusable test logic
// This runner is used in ALL environments: local, jfrog-cli-evidence CI, and jfrog-cli
type EvidenceE2ETestsRunner struct {
	EvidenceUserCLI    *coreTests.JfrogCli
	EvidenceAdminCLI   *coreTests.JfrogCli
	EvidenceProjectCLI *coreTests.JfrogCli
	ServicesManager    artifactory.ArtifactoryServicesManager
	AccessManager      *access.AccessServicesManager
	LifecycleManager   *lifecycle.LifecycleServicesManager
}

func NewEvidenceE2ETestsRunner(evidenceUserCli *coreTests.JfrogCli,
	evidenceAdminCli *coreTests.JfrogCli,
	evidenceProjectCli *coreTests.JfrogCli,
	servicesManager artifactory.ArtifactoryServicesManager,
	accessManager *access.AccessServicesManager,
	lifecycleManager *lifecycle.LifecycleServicesManager) *EvidenceE2ETestsRunner {
	return &EvidenceE2ETestsRunner{
		EvidenceUserCLI:    evidenceUserCli,
		EvidenceAdminCLI:   evidenceAdminCli,
		EvidenceProjectCLI: evidenceProjectCli,
		ServicesManager:    servicesManager,
		AccessManager:      accessManager,
		LifecycleManager:   lifecycleManager,
	}
}

// RunEvidenceCliTests runs all Evidence CLI command tests
// Each command has its own test group that runs its suite of tests
func (r *EvidenceE2ETestsRunner) RunEvidenceCliTests(t *testing.T) {
	// Create Evidence command tests suite
	t.Run("CreateEvidence", func(t *testing.T) {
		r.RunCreateEvidenceSuite(t)
	})

	// Generate Key Pair command tests suite
	t.Run("GenerateKeyPair", func(t *testing.T) {
		r.RunGenerateKeyPairSuite(t)
	})

	// Verify Evidence command tests suite
	t.Run("VerifyEvidence", func(t *testing.T) {
		r.RunVerifyEvidenceSuite(t)
	})

	// Get Evidence command tests suite
	t.Run("GetEvidence", func(t *testing.T) {
		r.RunGetEvidenceSuite(t)
	})
}

// PrepareTestsData generates shared resources needed for all E2E tests
func (r *EvidenceE2ETestsRunner) PrepareTestsData() error {
	fmt.Println("=== Preparing E2E Test Data ===")

	err := generateCommonPrivatePublicKey(r)
	if err != nil {
		return err
	}

	return nil
}

// CleanupTestsData removes shared resources created during PrepareTestsData
// This should be called after all tests complete
func (r *EvidenceE2ETestsRunner) CleanupTestsData() {
	fmt.Println("=== Cleaning Up E2E Test Data ===")

	// Delete the shared key from Artifactory Trusted Keys Store
	if SharedKeyAlias != "" {
		fmt.Printf("Deleting shared key from Artifactory: %s\n", SharedKeyAlias)
		if err := utils.DeleteTrustedKey(r.ServicesManager, SharedKeyAlias); err != nil {
			fmt.Printf("Failed to delete key from Artifactory: %v\n", err)
		} else {
			fmt.Printf("Shared key deleted from Artifactory: %s\n", SharedKeyAlias)
		}
	}

	// Delete the temporary key directory
	if SharedKeyDir != "" {
		fmt.Printf("Deleting shared key directory: %s\n", SharedKeyDir)
		if err := os.RemoveAll(SharedKeyDir); err != nil {
			fmt.Printf("Warning: Failed to delete key directory: %v\n", err)
		} else {
			fmt.Printf("Shared key directory deleted\n")
		}
	}

	fmt.Println("=== Test Data Cleanup Complete ===")
}

func generateCommonPrivatePublicKey(r *EvidenceE2ETestsRunner) error {
	// Create a temporary directory for the shared key pair
	tempDir, err := os.MkdirTemp("", "evidence-e2e-keys-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory for keys: %w", err)
	}
	SharedKeyDir = tempDir
	fmt.Printf("✓ Created temp directory for keys: %s\n", SharedKeyDir)

	// Generate unique key alias with timestamp to avoid conflicts
	SharedKeyAlias = fmt.Sprintf("e2e-shared-key-%d", time.Now().Unix())
	keyFileName := "e2e-shared-key"

	fmt.Println("Generating shared key pair (with upload to Artifactory)...")

	// Generate key pair and upload public key to Artifactory
	// Uses EvidenceAdminCLI which is configured with admin token for key upload permissions
	tmpT := &testing.T{}
	output := r.EvidenceAdminCLI.RunCliCmdWithOutput(tmpT,
		"generate-key-pair",
		"--key-file-path", SharedKeyDir,
		"--key-file-name", keyFileName,
		"--key-alias", SharedKeyAlias,
		"--upload-public-key=true", // Upload to Artifactory (requires admin permissions)
	)

	fmt.Printf("Key generation output: %s\n", output)

	// Set global paths
	SharedPrivateKeyPath = filepath.Join(SharedKeyDir, keyFileName+".key")
	SharedPublicKeyPath = filepath.Join(SharedKeyDir, keyFileName+".pub")

	// Verify keys were created
	if _, err := os.Stat(SharedPrivateKeyPath); os.IsNotExist(err) {
		return fmt.Errorf("private key not found at %s", SharedPrivateKeyPath)
	}
	if _, err := os.Stat(SharedPublicKeyPath); os.IsNotExist(err) {
		return fmt.Errorf("public key not found at %s", SharedPublicKeyPath)
	}

	fmt.Printf("✓ Shared key pair generated:\n")
	fmt.Printf("  - Alias: %s\n", SharedKeyAlias)
	fmt.Printf("  - Public Key: %s\n", SharedPublicKeyPath)
	fmt.Printf("  - Public key uploaded to Artifactory: yes\n")
	fmt.Println("=== Test Data Preparation Complete ===")
	return nil
}
