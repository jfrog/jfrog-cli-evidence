package tests

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jfrog/jfrog-cli-evidence/tests/e2e/utils"
	"github.com/stretchr/testify/require"
)

// RunCreateEvidenceHappyFlow runs the create evidence happy flow test
func (r *EvidenceE2ETestsRunner) RunCreateEvidenceHappyFlow(t *testing.T) {
	t.Log("=== Evidence Creation Happy Flow Test ===")

	// Step 0: Create unique test repository (automatically cleaned up after test)
	t.Log("Step 0: Creating test repository...")
	repoName := utils.CreateTestRepository(t, r.ServicesManager, "generic")
	t.Logf("✓ Test repository ready: %s", repoName)

	// Create temp directory for test files
	tempDir := t.TempDir()
	keyDir := filepath.Join(tempDir, "keys")
	err := os.MkdirAll(keyDir, 0755)
	require.NoError(t, err)

	// Step 1: Generate test key pair
	t.Log("Step 1: Generating test key pair...")
	keyAlias := fmt.Sprintf("e2e-test-key-%d", time.Now().Unix())
	keyFileName := "test-key"

	output := r.EvidenceCLI.RunCliCmdWithOutput(t,
		"generate-key-pair",
		"--key-file-path", keyDir,
		"--key-file-name", keyFileName,
		"--key-alias", keyAlias,
		"--upload-public-key=false",
	)
	t.Logf("Key generation output: %s", output)

	privateKeyPath := filepath.Join(keyDir, keyFileName+".key")
	publicKeyPath := filepath.Join(keyDir, keyFileName+".pub")

	// Verify keys were created
	require.FileExists(t, privateKeyPath, "Private key should be created")
	require.FileExists(t, publicKeyPath, "Public key should be created")
	t.Logf("✓ Keys generated: alias=%s", keyAlias)

	// Step 2: Create and upload test artifact
	t.Log("Step 2: Creating and uploading test artifact...")
	artifactContent := fmt.Sprintf("Test artifact content - timestamp: %d", time.Now().Unix())
	artifactPath := utils.CreateTestArtifact(t, artifactContent)

	// Upload to the test repository we created
	artifactFileName := filepath.Base(artifactPath)
	repoPath := fmt.Sprintf("%s/%s", repoName, artifactFileName)

	// Upload using Services Manager via utils function
	err = utils.UploadArtifact(r.ServicesManager, artifactPath, repoPath)
	require.NoError(t, err, "Failed to upload artifact")
	t.Logf("✓ Artifact uploaded to: %s", repoPath)

	// Step 3: Create predicate
	t.Log("Step 3: Creating predicate...")
	predicate := map[string]interface{}{
		"buildType":   "manual-test",
		"timestamp":   time.Now().Unix(),
		"environment": "e2e-test",
		"tester":      "evidence-e2e",
	}

	predicateBytes, err := json.MarshalIndent(predicate, "", "  ")
	require.NoError(t, err)

	predicatePath := filepath.Join(tempDir, "predicate.json")
	err = ioutil.WriteFile(predicatePath, predicateBytes, 0644)
	require.NoError(t, err)
	t.Logf("✓ Predicate created: %s", predicatePath)

	// Step 4: Create evidence
	t.Log("Step 4: Creating evidence...")
	evidenceOutput := r.EvidenceCLI.RunCliCmdWithOutput(t,
		"create",
		"--predicate", predicatePath,
		"--predicate-type", "https://slsa.dev/provenance/v1",
		"--subject-repo-path", repoPath,
		"--key", privateKeyPath,
		//"--key-alias", keyAlias,
	)
	t.Logf("Evidence creation output: %s", evidenceOutput)
	t.Log("✓ Evidence created successfully")
	t.Log("=== ✅ Happy Flow Test Completed Successfully! ===")
}

// RunCreateEvidenceHappyFlow runs the create evidence happy flow test
func (r *EvidenceE2ETestsRunner) RunCreateEvidenceHappyFlow2(t *testing.T) {
	t.Log("=== Evidence Creation Happy Flow Test ===")

	// Step 0: Create unique test repository (automatically cleaned up after test)
	t.Log("Step 0: Creating test repository...")
	repoName := utils.CreateTestRepository(t, r.ServicesManager, "generic")
	t.Logf("✓ Test repository ready: %s", repoName)

	// Create temp directory for test files
	tempDir := t.TempDir()
	keyDir := filepath.Join(tempDir, "keys")
	err := os.MkdirAll(keyDir, 0755)
	require.NoError(t, err)

	// Step 1: Generate test key pair
	t.Log("Step 1: Generating test key pair...")
	keyAlias := fmt.Sprintf("e2e-test-key-%d", time.Now().Unix())
	keyFileName := "test-key"

	output := r.EvidenceCLI.RunCliCmdWithOutput(t,
		"generate-key-pair",
		"--key-file-path", keyDir,
		"--key-file-name", keyFileName,
		"--key-alias", keyAlias,
		"--upload-public-key=false",
	)
	t.Logf("Key generation output: %s", output)

	privateKeyPath := filepath.Join(keyDir, keyFileName+".key")
	publicKeyPath := filepath.Join(keyDir, keyFileName+".pub")

	// Verify keys were created
	require.FileExists(t, privateKeyPath, "Private key should be created")
	require.FileExists(t, publicKeyPath, "Public key should be created")
	t.Logf("✓ Keys generated: alias=%s", keyAlias)

	// Step 2: Create and upload test artifact
	t.Log("Step 2: Creating and uploading test artifact...")
	artifactContent := fmt.Sprintf("Test artifact content - timestamp: %d", time.Now().Unix())
	artifactPath := utils.CreateTestArtifact(t, artifactContent)

	// Upload to the test repository we created
	artifactFileName := filepath.Base(artifactPath)
	repoPath := fmt.Sprintf("%s/%s", repoName, artifactFileName)

	// Upload using Services Manager via utils function
	err = utils.UploadArtifact(r.ServicesManager, artifactPath, repoPath)
	require.NoError(t, err, "Failed to upload artifact")
	t.Logf("✓ Artifact uploaded to: %s", repoPath)

	// Step 3: Create predicate
	t.Log("Step 3: Creating predicate...")
	predicate := map[string]interface{}{
		"buildType":   "manual-test",
		"timestamp":   time.Now().Unix(),
		"environment": "e2e-test",
		"tester":      "evidence-e2e",
	}

	predicateBytes, err := json.MarshalIndent(predicate, "", "  ")
	require.NoError(t, err)

	predicatePath := filepath.Join(tempDir, "predicate.json")
	err = ioutil.WriteFile(predicatePath, predicateBytes, 0644)
	require.NoError(t, err)
	t.Logf("✓ Predicate created: %s", predicatePath)

	// Step 4: Create evidence
	t.Log("Step 4: Creating evidence...")
	evidenceOutput := r.EvidenceCLI.RunCliCmdWithOutput(t,
		"create",
		"--predicate", predicatePath,
		"--predicate-type", "https://slsa.dev/provenance/v1",
		"--subject-repo-path", repoPath,
		"--key", privateKeyPath,
		//"--key-alias", keyAlias,
	)
	t.Logf("Evidence creation output: %s", evidenceOutput)
	t.Log("✓ Evidence created successfully")
	t.Log("=== ✅ Happy Flow Test Completed Successfully! ===")

	// Intentional failure to verify test framework
	assert.Equal(t, 1, 2, "❌ This test should fail intentionally - expected 1 to equal 2")
}

// RunCreateEvidenceHappyFlow runs the create evidence happy flow test
func (r *EvidenceE2ETestsRunner) RunCreateEvidenceHappyFlow3(t *testing.T) {
	t.Log("=== Evidence Creation Happy Flow Test ===")

	// Step 0: Create unique test repository (automatically cleaned up after test)
	t.Log("Step 0: Creating test repository...")
	repoName := utils.CreateTestRepository(t, r.ServicesManager, "generic")
	t.Logf("✓ Test repository ready: %s", repoName)

	// Create temp directory for test files
	tempDir := t.TempDir()
	keyDir := filepath.Join(tempDir, "keys")
	err := os.MkdirAll(keyDir, 0755)
	require.NoError(t, err)

	// Step 1: Generate test key pair
	t.Log("Step 1: Generating test key pair...")
	keyAlias := fmt.Sprintf("e2e-test-key-%d", time.Now().Unix())
	keyFileName := "test-key"

	output := r.EvidenceCLI.RunCliCmdWithOutput(t,
		"generate-key-pair",
		"--key-file-path", keyDir,
		"--key-file-name", keyFileName,
		"--key-alias", keyAlias,
		"--upload-public-key=false",
	)
	t.Logf("Key generation output: %s", output)

	privateKeyPath := filepath.Join(keyDir, keyFileName+".key")
	publicKeyPath := filepath.Join(keyDir, keyFileName+".pub")

	// Verify keys were created
	require.FileExists(t, privateKeyPath, "Private key should be created")
	require.FileExists(t, publicKeyPath, "Public key should be created")
	t.Logf("✓ Keys generated: alias=%s", keyAlias)

	// Step 2: Create and upload test artifact
	t.Log("Step 2: Creating and uploading test artifact...")
	artifactContent := fmt.Sprintf("Test artifact content - timestamp: %d", time.Now().Unix())
	artifactPath := utils.CreateTestArtifact(t, artifactContent)

	// Upload to the test repository we created
	artifactFileName := filepath.Base(artifactPath)
	repoPath := fmt.Sprintf("%s/%s", repoName, artifactFileName)

	// Upload using Services Manager via utils function
	err = utils.UploadArtifact(r.ServicesManager, artifactPath, repoPath)
	require.NoError(t, err, "Failed to upload artifact")
	t.Logf("✓ Artifact uploaded to: %s", repoPath)

	// Step 3: Create predicate
	t.Log("Step 3: Creating predicate...")
	predicate := map[string]interface{}{
		"buildType":   "manual-test",
		"timestamp":   time.Now().Unix(),
		"environment": "e2e-test",
		"tester":      "evidence-e2e",
	}

	predicateBytes, err := json.MarshalIndent(predicate, "", "  ")
	require.NoError(t, err)

	predicatePath := filepath.Join(tempDir, "predicate.json")
	err = ioutil.WriteFile(predicatePath, predicateBytes, 0644)
	require.NoError(t, err)
	t.Logf("✓ Predicate created: %s", predicatePath)

	// Step 4: Create evidence
	t.Log("Step 4: Creating evidence...")
	evidenceOutput := r.EvidenceCLI.RunCliCmdWithOutput(t,
		"create",
		"--predicate", predicatePath,
		"--predicate-type", "https://slsa.dev/provenance/v1",
		"--subject-repo-path", repoPath,
		"--key", privateKeyPath,
		//"--key-alias", keyAlias,
	)
	t.Logf("Evidence creation output: %s", evidenceOutput)
	t.Log("✓ Evidence created successfully")
	t.Log("=== ✅ Happy Flow Test Completed Successfully! ===")

	// Intentional failure to verify test framework
	assert.Equal(t, 1, 2, "❌ This test should fail intentionally - expected 1 to equal 2")
}
