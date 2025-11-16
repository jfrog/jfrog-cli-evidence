package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jfrog/jfrog-cli-evidence/tests/e2e/utils"
	"github.com/stretchr/testify/require"
)

// RunGenerateKeyPairSuite runs all generate key pair tests
func (r *EvidenceE2ETestsRunner) RunGenerateKeyPairSuite(t *testing.T) {
	t.Run("GenerateKeyPairBasic", func(t *testing.T) {
		r.RunGenerateKeyPairBasic(t)
	})
	t.Run("GenerateKeyPairWithCustomPath", func(t *testing.T) {
		r.RunGenerateKeyPairWithCustomPath(t)
	})
	t.Run("GenerateKeyPairWithUpload", func(t *testing.T) {
		r.RunGenerateKeyPairWithUpload(t)
	})
}

// RunGenerateKeyPairBasic tests basic key pair generation
func (r *EvidenceE2ETestsRunner) RunGenerateKeyPairBasic(t *testing.T) {
	t.Log("=== Generate Key Pair - Basic Test ===")

	tempDir := t.TempDir()

	// Step 1: Prepare key directory
	t.Log("Step 1: Preparing key directory...")
	keyDir := filepath.Join(tempDir, "keys")
	err := os.MkdirAll(keyDir, 0755)
	require.NoError(t, err)
	t.Logf("✓ Key directory created: %s", keyDir)

	// Step 2: Generate key pair using User CLI (no upload)
	t.Log("Step 2: Generating key pair using User CLI...")
	keyAlias := fmt.Sprintf("e2e-basic-key-%d", time.Now().Unix())
	keyFileName := "basic-key"

	generateOutput := r.EvidenceUserCLI.RunCliCmdWithOutput(t,
		"generate-key-pair",
		"--key-file-path", keyDir,
		"--key-file-name", keyFileName,
		"--key-alias", keyAlias,
		"--upload-public-key=false",
	)
	t.Logf("Key generation output: %s", generateOutput)
	require.NotContains(t, generateOutput, "Error", "Key generation should not error")
	t.Log("✓ Key pair generated successfully")

	// Step 3: Verify key files exist
	t.Log("Step 3: Verifying key files...")
	privateKeyPath := filepath.Join(keyDir, keyFileName+".key")
	publicKeyPath := filepath.Join(keyDir, keyFileName+".pub")

	require.FileExists(t, privateKeyPath, "Private key should be created")
	require.FileExists(t, publicKeyPath, "Public key should be created")
	t.Logf("✓ Key files exist: %s.key and %s.pub", keyFileName, keyFileName)

	t.Log("=== ✅ Basic Key Pair Generation Test Completed Successfully! ===")
}

// RunGenerateKeyPairWithCustomPath tests key generation with custom nested path
func (r *EvidenceE2ETestsRunner) RunGenerateKeyPairWithCustomPath(t *testing.T) {
	t.Log("=== Generate Key Pair - Custom Path Test ===")

	tempDir := t.TempDir()

	// Step 1: Define custom nested path (directory doesn't exist yet)
	t.Log("Step 1: Defining custom nested path...")
	customKeyDir := filepath.Join(tempDir, "custom", "nested", "path")
	t.Logf("✓ Custom path: %s", customKeyDir)

	// Step 2: Generate key pair in custom path using User CLI
	t.Log("Step 2: Generating key pair in custom nested path using User CLI...")
	keyAlias := fmt.Sprintf("e2e-custom-path-key-%d", time.Now().Unix())
	keyFileName := "custom-key"

	generateOutput := r.EvidenceUserCLI.RunCliCmdWithOutput(t,
		"generate-key-pair",
		"--key-file-path", customKeyDir,
		"--key-file-name", keyFileName,
		"--key-alias", keyAlias,
		"--upload-public-key=false",
	)
	t.Logf("Key generation output: %s", generateOutput)
	require.NotContains(t, generateOutput, "Error", "Key generation should not error")
	t.Log("✓ Key pair generated successfully in nested path")

	// Step 3: Verify key files exist in custom path
	t.Log("Step 3: Verifying key files in custom path...")
	privateKeyPath := filepath.Join(customKeyDir, keyFileName+".key")
	publicKeyPath := filepath.Join(customKeyDir, keyFileName+".pub")

	require.FileExists(t, privateKeyPath, "Private key should be created in custom path")
	require.FileExists(t, publicKeyPath, "Public key should be created in custom path")
	t.Logf("✓ Key files exist in custom path: %s", customKeyDir)

	// Step 4: Verify directory was created
	t.Log("Step 4: Verifying directory structure was created...")
	dirInfo, err := os.Stat(customKeyDir)
	require.NoError(t, err)
	require.True(t, dirInfo.IsDir(), "Custom path should be a directory")
	t.Log("✓ Nested directory structure created successfully")

	t.Log("=== ✅ Custom Path Key Pair Generation Test Completed Successfully! ===")
}

// RunGenerateKeyPairWithUpload tests key generation with upload to Artifactory Trusted Keys Store
func (r *EvidenceE2ETestsRunner) RunGenerateKeyPairWithUpload(t *testing.T) {
	t.Log("=== Generate Key Pair - With Upload Test ===")

	tempDir := t.TempDir()

	// Step 1: Prepare key directory
	t.Log("Step 1: Preparing key directory...")
	keyDir := filepath.Join(tempDir, "keys")
	err := os.MkdirAll(keyDir, 0755)
	require.NoError(t, err)
	t.Logf("✓ Key directory created: %s", keyDir)

	// Step 2: Generate key pair with upload using Admin CLI (requires MANAGE_TRUSTED_KEYS permission)
	t.Log("Step 2: Generating key pair with upload using Admin CLI...")
	keyAlias := fmt.Sprintf("e2e-upload-key-%d", time.Now().Unix())
	keyFileName := "upload-key"

	generateOutput := r.EvidenceAdminCLI.RunCliCmdWithOutput(t,
		"generate-key-pair",
		"--key-file-path", keyDir,
		"--key-file-name", keyFileName,
		"--key-alias", keyAlias,
		"--upload-public-key=true",
	)
	t.Logf("Key generation output: %s", generateOutput)
	require.NotContains(t, generateOutput, "Error", "Key generation should not error")
	require.NotContains(t, generateOutput, "Failed", "Key generation should not fail")
	t.Log("✓ Key pair generated and uploaded successfully")

	// Register cleanup to delete the uploaded key
	t.Cleanup(func() {
		t.Logf("Cleaning up uploaded test key: %s", keyAlias)
		if err := utils.DeleteTrustedKey(r.ServicesManager, keyAlias); err != nil {
			t.Logf("Warning: Failed to delete test key %s: %v", keyAlias, err)
		} else {
			t.Logf("✓ Test key deleted: %s", keyAlias)
		}
	})

	// Step 3: Verify key files exist locally
	t.Log("Step 3: Verifying key files exist locally...")
	privateKeyPath := filepath.Join(keyDir, keyFileName+".key")
	publicKeyPath := filepath.Join(keyDir, keyFileName+".pub")

	require.FileExists(t, privateKeyPath, "Private key should be created")
	require.FileExists(t, publicKeyPath, "Public key should be created")
	t.Logf("✓ Key files exist: %s.key and %s.pub", keyFileName, keyFileName)

	// Step 4: Verify upload succeeded (validation happens in step 2)
	t.Log("Step 4: Verifying key pair generation completed successfully...")
	require.NotContains(t, generateOutput, "403", "Should not have permission error")
	require.NotContains(t, generateOutput, "failed to upload", "Should not have upload failure")
	t.Log("✓ Key pair generation with upload completed successfully")

	t.Log("=== ✅ Key Pair Generation with Upload Test Completed Successfully! ===")
}
