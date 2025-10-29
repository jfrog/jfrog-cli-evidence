package tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jfrog/jfrog-cli-evidence/tests/e2e"
	"github.com/jfrog/jfrog-cli-evidence/tests/e2e/utils"
	"github.com/stretchr/testify/require"
)

// RunGetEvidenceSuite runs all get evidence tests
func (r *EvidenceE2ETestsRunner) RunGetEvidenceSuite(t *testing.T) {
	t.Run("ForArtifact", func(t *testing.T) {
		r.RunGetEvidenceForArtifact(t)
	})
	t.Run("ForArtifactWithFormat", func(t *testing.T) {
		r.RunGetEvidenceForArtifactWithFormat(t)
	})
	t.Run("ForArtifactWithIncludePredicate", func(t *testing.T) {
		r.RunGetEvidenceForArtifactWithIncludePredicate(t)
	})
	t.Run("ForArtifactWithOutputFile", func(t *testing.T) {
		r.RunGetEvidenceWithOutputFile(t)
	})
	t.Run("ForArtifactWithArtifactsLimit", func(t *testing.T) {
		r.RunGetEvidenceWithArtifactsLimit(t)
	})
	t.Run("ForArtifactWithProject", func(t *testing.T) {
		r.RunGetEvidenceForArtifactWithProject(t)
	})
	t.Run("ForReleaseBundle", func(t *testing.T) {
		r.RunGetEvidenceForReleaseBundle(t)
	})
}

// RunGetEvidenceForArtifact tests getting evidence for an artifact
func (r *EvidenceE2ETestsRunner) RunGetEvidenceForArtifact(t *testing.T) {
	t.Log("=== Get Evidence - Artifact Test ===")

	// Verify shared key pair is available
	if SharedPrivateKeyPath == "" || SharedPublicKeyPath == "" {
		t.Errorf("Shared key pair not initialized. Ensure PrepareTestsData() was called.")
		return
	}
	t.Logf("Using shared key pair: %s (alias: %s)", SharedPrivateKeyPath, SharedKeyAlias)

	tempDir := t.TempDir()

	// Step 1: Create repository and upload artifact (Admin)
	t.Log("Step 1: Creating repository and uploading artifact...")
	repoName := utils.CreateTestRepository(t, r.ServicesManager, "generic")
	artifactContent := fmt.Sprintf("Test artifact for get evidence - timestamp: %d", time.Now().Unix())
	artifactPath := utils.CreateTestArtifact(t, artifactContent)
	artifactFileName := filepath.Base(artifactPath)
	repoPath := fmt.Sprintf("%s/%s", repoName, artifactFileName)
	err := utils.UploadArtifact(r.ServicesManager, artifactPath, repoPath)
	require.NoError(t, err)
	t.Logf("✓ Artifact uploaded: %s", repoPath)

	// Step 2: Create predicate (Admin)
	t.Log("Step 2: Creating predicate...")
	predicate := map[string]interface{}{
		"buildType":   "get-artifact-test",
		"timestamp":   time.Now().Unix(),
		"environment": "e2e-get-test",
		"description": "Testing get evidence for artifact",
	}
	predicateBytes, err := json.MarshalIndent(predicate, "", "  ")
	require.NoError(t, err)
	predicatePath := filepath.Join(tempDir, "predicate.json")
	err = os.WriteFile(predicatePath, predicateBytes, 0644)
	require.NoError(t, err)
	t.Log("✓ Predicate created")

	// Step 3: Create evidence for artifact using Admin CLI
	t.Log("Step 3: Creating evidence for artifact using Admin CLI...")
	createOutput := r.EvidenceAdminCLI.RunCliCmdWithOutput(t,
		"create",
		"--predicate", predicatePath,
		"--predicate-type", "https://slsa.dev/provenance/v1",
		"--subject-repo-path", repoPath,
		"--key", SharedPrivateKeyPath,
		"--key-alias", SharedKeyAlias,
	)
	t.Logf("Evidence creation output: %s", createOutput)
	require.NotContains(t, createOutput, "Error", "Evidence creation should not error")
	t.Log("✓ Evidence created successfully")

	// Step 4: Get evidence for artifact using User CLI
	t.Log("Step 4: Getting evidence for artifact using User CLI...")
	getOutput := r.EvidenceUserCLI.RunCliCmdWithOutput(t,
		"get",
		"--subject-repo-path", repoPath,
	)
	t.Logf("Get evidence output: %s", getOutput)
	require.Contains(t, getOutput, repoPath, "Output should contain the subject path")
	t.Log("✅ User successfully retrieved artifact evidence!")

	t.Log("=== ✅ Get Evidence for Artifact Test Completed Successfully! ===")
}

// RunGetEvidenceForArtifactWithFormat tests getting evidence with specific format (JSON)
func (r *EvidenceE2ETestsRunner) RunGetEvidenceForArtifactWithFormat(t *testing.T) {
	t.Log("=== Get Evidence - With Format Test ===")

	// Verify shared key pair is available
	if SharedPrivateKeyPath == "" || SharedPublicKeyPath == "" {
		t.Errorf("Shared key pair not initialized. Ensure PrepareTestsData() was called.")
		return
	}
	t.Logf("Using shared key pair: %s (alias: %s)", SharedPrivateKeyPath, SharedKeyAlias)

	tempDir := t.TempDir()

	// Step 1: Create repository and upload artifact (Admin)
	t.Log("Step 1: Creating repository and uploading artifact...")
	repoName := utils.CreateTestRepository(t, r.ServicesManager, "generic")
	artifactContent := fmt.Sprintf("Test artifact for format - timestamp: %d", time.Now().Unix())
	artifactPath := utils.CreateTestArtifact(t, artifactContent)
	artifactFileName := filepath.Base(artifactPath)
	repoPath := fmt.Sprintf("%s/%s", repoName, artifactFileName)
	err := utils.UploadArtifact(r.ServicesManager, artifactPath, repoPath)
	require.NoError(t, err)
	t.Logf("✓ Artifact uploaded: %s", repoPath)

	// Step 2: Create predicate (Admin)
	t.Log("Step 2: Creating predicate...")
	predicate := map[string]interface{}{
		"buildType":   "get-format-test",
		"timestamp":   time.Now().Unix(),
		"environment": "e2e-get-test",
		"description": "Testing get evidence with JSON format",
	}
	predicateBytes, err := json.MarshalIndent(predicate, "", "  ")
	require.NoError(t, err)
	predicatePath := filepath.Join(tempDir, "predicate.json")
	err = os.WriteFile(predicatePath, predicateBytes, 0644)
	require.NoError(t, err)
	t.Log("✓ Predicate created")

	// Step 3: Create evidence for artifact using Admin CLI
	t.Log("Step 3: Creating evidence for artifact using Admin CLI...")
	createOutput := r.EvidenceAdminCLI.RunCliCmdWithOutput(t,
		"create",
		"--predicate", predicatePath,
		"--predicate-type", "https://slsa.dev/provenance/v1",
		"--subject-repo-path", repoPath,
		"--key", SharedPrivateKeyPath,
		"--key-alias", SharedKeyAlias,
	)
	t.Logf("Evidence creation output: %s", createOutput)
	require.NotContains(t, createOutput, "Error", "Evidence creation should not error")
	t.Log("✓ Evidence created successfully")

	// Step 4: Get evidence with JSON format using User CLI
	t.Log("Step 4: Getting evidence with JSON format using User CLI...")
	getOutput := r.EvidenceUserCLI.RunCliCmdWithOutput(t,
		"get",
		"--subject-repo-path", repoPath,
		"--format", "json",
	)
	t.Logf("Get evidence with format output: %s", getOutput)
	require.Contains(t, getOutput, repoPath, "Output should contain the subject")
	// Validate that output is valid JSON by attempting to parse it
	var jsonData interface{}
	err = json.Unmarshal([]byte(getOutput), &jsonData)
	require.NoError(t, err, "Output should be valid JSON")
	t.Log("✅ User successfully retrieved artifact evidence with JSON format!")

	t.Log("=== ✅ Get Evidence with Format Test Completed Successfully! ===")
}

// RunGetEvidenceForArtifactWithIncludePredicate tests getting evidence with predicate included
func (r *EvidenceE2ETestsRunner) RunGetEvidenceForArtifactWithIncludePredicate(t *testing.T) {
	t.Log("=== Get Evidence - With Include Predicate Test ===")

	// Verify shared key pair is available
	if SharedPrivateKeyPath == "" || SharedPublicKeyPath == "" {
		t.Errorf("Shared key pair not initialized. Ensure PrepareTestsData() was called.")
		return
	}
	t.Logf("Using shared key pair: %s (alias: %s)", SharedPrivateKeyPath, SharedKeyAlias)

	tempDir := t.TempDir()

	// Step 1: Create repository and upload artifact (Admin)
	t.Log("Step 1: Creating repository and uploading artifact...")
	repoName := utils.CreateTestRepository(t, r.ServicesManager, "generic")
	artifactContent := fmt.Sprintf("Test artifact for include predicate - timestamp: %d", time.Now().Unix())
	artifactPath := utils.CreateTestArtifact(t, artifactContent)
	artifactFileName := filepath.Base(artifactPath)
	repoPath := fmt.Sprintf("%s/%s", repoName, artifactFileName)
	err := utils.UploadArtifact(r.ServicesManager, artifactPath, repoPath)
	require.NoError(t, err)
	t.Logf("✓ Artifact uploaded: %s", repoPath)

	// Step 2: Create predicate with custom data (Admin)
	t.Log("Step 2: Creating predicate with custom data...")
	predicate := map[string]interface{}{
		"buildType":   "get-include-predicate-test",
		"timestamp":   time.Now().Unix(),
		"environment": "e2e-get-test",
		"testData":    "include-predicate-test",
		"description": "Testing get evidence with include-predicate flag",
	}
	predicateBytes, err := json.MarshalIndent(predicate, "", "  ")
	require.NoError(t, err)
	predicatePath := filepath.Join(tempDir, "predicate.json")
	err = os.WriteFile(predicatePath, predicateBytes, 0644)
	require.NoError(t, err)
	t.Log("✓ Predicate created with custom testData field")

	// Step 3: Create evidence for artifact using Admin CLI
	t.Log("Step 3: Creating evidence for artifact using Admin CLI...")
	createOutput := r.EvidenceAdminCLI.RunCliCmdWithOutput(t,
		"create",
		"--predicate", predicatePath,
		"--predicate-type", "https://slsa.dev/provenance/v1",
		"--subject-repo-path", repoPath,
		"--key", SharedPrivateKeyPath,
		"--key-alias", SharedKeyAlias,
	)
	t.Logf("Evidence creation output: %s", createOutput)
	require.NotContains(t, createOutput, "Error", "Evidence creation should not error")
	t.Log("✓ Evidence created successfully")

	// Step 4: Get evidence with include-predicate flag using User CLI
	t.Log("Step 4: Getting evidence with include-predicate flag using User CLI...")
	getOutput := r.EvidenceUserCLI.RunCliCmdWithOutput(t,
		"get",
		"--subject-repo-path", repoPath,
		"--include-predicate",
	)
	t.Logf("Get evidence with include predicate output: %s", getOutput)
	require.Contains(t, getOutput, repoPath, "Output should contain the subject")
	require.Contains(t, getOutput, "testData", "Output should include predicate data field")
	require.Contains(t, getOutput, "include-predicate-test", "Output should include predicate test value")
	t.Log("✅ User successfully retrieved artifact evidence with predicate included!")

	t.Log("=== ✅ Get Evidence with Include Predicate Test Completed Successfully! ===")
}

// RunGetEvidenceWithOutputFile tests getting evidence and saving to output file
func (r *EvidenceE2ETestsRunner) RunGetEvidenceWithOutputFile(t *testing.T) {
	t.Log("=== Get Evidence - With Output File Test ===")

	// Verify shared key pair is available
	if SharedPrivateKeyPath == "" || SharedPublicKeyPath == "" {
		t.Errorf("Shared key pair not initialized. Ensure PrepareTestsData() was called.")
		return
	}
	t.Logf("Using shared key pair: %s (alias: %s)", SharedPrivateKeyPath, SharedKeyAlias)

	tempDir := t.TempDir()

	// Step 1: Create repository and upload artifact (Admin)
	t.Log("Step 1: Creating repository and uploading artifact...")
	repoName := utils.CreateTestRepository(t, r.ServicesManager, "generic")
	artifactContent := fmt.Sprintf("Test artifact for output file - timestamp: %d", time.Now().Unix())
	artifactPath := utils.CreateTestArtifact(t, artifactContent)
	artifactFileName := filepath.Base(artifactPath)
	repoPath := fmt.Sprintf("%s/%s", repoName, artifactFileName)
	err := utils.UploadArtifact(r.ServicesManager, artifactPath, repoPath)
	require.NoError(t, err)
	t.Logf("✓ Artifact uploaded: %s", repoPath)

	// Step 2: Create predicate (Admin)
	t.Log("Step 2: Creating predicate...")
	predicate := map[string]interface{}{
		"buildType":   "get-output-file-test",
		"timestamp":   time.Now().Unix(),
		"environment": "e2e-get-test",
		"description": "Testing get evidence with output file",
	}
	predicateBytes, err := json.MarshalIndent(predicate, "", "  ")
	require.NoError(t, err)
	predicatePath := filepath.Join(tempDir, "predicate.json")
	err = os.WriteFile(predicatePath, predicateBytes, 0644)
	require.NoError(t, err)
	t.Log("✓ Predicate created")

	// Step 3: Create evidence for artifact using Admin CLI
	t.Log("Step 3: Creating evidence for artifact using Admin CLI...")
	createOutput := r.EvidenceAdminCLI.RunCliCmdWithOutput(t,
		"create",
		"--predicate", predicatePath,
		"--predicate-type", "https://slsa.dev/provenance/v1",
		"--subject-repo-path", repoPath,
		"--key", SharedPrivateKeyPath,
		"--key-alias", SharedKeyAlias,
	)
	t.Logf("Evidence creation output: %s", createOutput)
	require.NotContains(t, createOutput, "Error", "Evidence creation should not error")
	t.Log("✓ Evidence created successfully")

	// Step 4: Get evidence and save to output file using User CLI
	t.Log("Step 4: Getting evidence and saving to output file using User CLI...")
	outputFile := filepath.Join(tempDir, "evidence-output.json")
	getOutput := r.EvidenceUserCLI.RunCliCmdWithOutput(t,
		"get",
		"--subject-repo-path", repoPath,
		"--output", outputFile,
	)
	t.Logf("Get evidence output: %s", getOutput)

	// Verify output file was created and contains data
	require.FileExists(t, outputFile, "Output file should be created")
	outputData, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	require.NotEmpty(t, outputData, "Output file should contain data")
	require.Contains(t, string(outputData), repoPath, "Output file should contain subject path")
	t.Logf("✓ Output file created: %s", outputFile)
	t.Log("✅ User successfully retrieved artifact evidence and saved to file!")

	t.Log("=== ✅ Get Evidence with Output File Test Completed Successfully! ===")
}

// RunGetEvidenceWithArtifactsLimit tests getting evidence with artifacts limit
func (r *EvidenceE2ETestsRunner) RunGetEvidenceWithArtifactsLimit(t *testing.T) {
	t.Log("=== Get Evidence - With Artifacts Limit Test ===")

	// Verify shared key pair is available
	if SharedPrivateKeyPath == "" || SharedPublicKeyPath == "" {
		t.Errorf("Shared key pair not initialized. Ensure PrepareTestsData() was called.")
		return
	}
	t.Logf("Using shared key pair: %s (alias: %s)", SharedPrivateKeyPath, SharedKeyAlias)

	tempDir := t.TempDir()

	// Step 1: Create repository and upload artifact (Admin)
	t.Log("Step 1: Creating repository and uploading artifact...")
	repoName := utils.CreateTestRepository(t, r.ServicesManager, "generic")
	artifactContent := fmt.Sprintf("Test artifact for artifacts limit - timestamp: %d", time.Now().Unix())
	artifactPath := utils.CreateTestArtifact(t, artifactContent)
	artifactFileName := filepath.Base(artifactPath)
	repoPath := fmt.Sprintf("%s/%s", repoName, artifactFileName)
	err := utils.UploadArtifact(r.ServicesManager, artifactPath, repoPath)
	require.NoError(t, err)
	t.Logf("✓ Artifact uploaded: %s", repoPath)

	// Step 2: Create predicate (Admin)
	t.Log("Step 2: Creating predicate...")
	predicate := map[string]interface{}{
		"buildType":   "get-artifacts-limit-test",
		"timestamp":   time.Now().Unix(),
		"environment": "e2e-get-test",
		"description": "Testing get evidence with artifacts limit",
	}
	predicateBytes, err := json.MarshalIndent(predicate, "", "  ")
	require.NoError(t, err)
	predicatePath := filepath.Join(tempDir, "predicate.json")
	err = os.WriteFile(predicatePath, predicateBytes, 0644)
	require.NoError(t, err)
	t.Log("✓ Predicate created")

	// Step 3: Create evidence for artifact using Admin CLI
	t.Log("Step 3: Creating evidence for artifact using Admin CLI...")
	createOutput := r.EvidenceAdminCLI.RunCliCmdWithOutput(t,
		"create",
		"--predicate", predicatePath,
		"--predicate-type", "https://slsa.dev/provenance/v1",
		"--subject-repo-path", repoPath,
		"--key", SharedPrivateKeyPath,
		"--key-alias", SharedKeyAlias,
	)
	t.Logf("Evidence creation output: %s", createOutput)
	require.NotContains(t, createOutput, "Error", "Evidence creation should not error")
	t.Log("✓ Evidence created successfully")

	// Step 4: Get evidence with artifacts limit using User CLI
	t.Log("Step 4: Getting evidence with artifacts limit using User CLI...")
	getOutput := r.EvidenceUserCLI.RunCliCmdWithOutput(t,
		"get",
		"--subject-repo-path", repoPath,
		"--artifacts-limit", "10",
	)
	t.Logf("Get evidence with limit output: %s", getOutput)
	require.Contains(t, getOutput, repoPath, "Output should contain the subject")
	t.Log("✅ User successfully retrieved artifact evidence with artifacts limit!")

	t.Log("=== ✅ Get Evidence with Artifacts Limit Test Completed Successfully! ===")
}

// RunGetEvidenceForArtifactWithProject tests getting evidence for an artifact with project
func (r *EvidenceE2ETestsRunner) RunGetEvidenceForArtifactWithProject(t *testing.T) {
	t.Log("=== Get Evidence - Artifact With Project Test ===")

	// Verify shared key pair is available
	if SharedPrivateKeyPath == "" || SharedPublicKeyPath == "" {
		t.Errorf("Shared key pair not initialized. Ensure PrepareTestsData() was called.")
		return
	}
	t.Logf("Using shared key pair: %s (alias: %s)", SharedPrivateKeyPath, SharedKeyAlias)
	t.Logf("Using project: %s", e2e.ProjectKey)

	tempDir := t.TempDir()

	// Step 1: Create repository and upload artifact (Admin)
	t.Log("Step 1: Creating repository and uploading artifact...")
	repoName := utils.CreateTestRepository(t, r.ServicesManager, "generic")
	artifactContent := fmt.Sprintf("Test artifact for project get - timestamp: %d", time.Now().Unix())
	artifactPath := utils.CreateTestArtifact(t, artifactContent)
	artifactFileName := filepath.Base(artifactPath)
	repoPath := fmt.Sprintf("%s/%s", repoName, artifactFileName)
	err := utils.UploadArtifact(r.ServicesManager, artifactPath, repoPath)
	require.NoError(t, err)
	t.Logf("✓ Artifact uploaded: %s", repoPath)

	// Step 2: Create predicate (Admin)
	t.Log("Step 2: Creating predicate...")
	predicate := map[string]interface{}{
		"buildType":   "get-artifact-project-test",
		"timestamp":   time.Now().Unix(),
		"environment": "e2e-get-test",
		"project":     e2e.ProjectKey,
		"description": "Testing get evidence for artifact with project scope",
	}
	predicateBytes, err := json.MarshalIndent(predicate, "", "  ")
	require.NoError(t, err)
	predicatePath := filepath.Join(tempDir, "predicate.json")
	err = os.WriteFile(predicatePath, predicateBytes, 0644)
	require.NoError(t, err)
	t.Log("✓ Predicate created")

	// Step 3: Create evidence for artifact with project using Admin CLI
	t.Log("Step 3: Creating evidence for artifact with project using Admin CLI...")
	createOutput := r.EvidenceAdminCLI.RunCliCmdWithOutput(t,
		"create",
		"--predicate", predicatePath,
		"--predicate-type", "https://slsa.dev/provenance/v1",
		"--subject-repo-path", repoPath,
		"--project", e2e.ProjectKey,
		"--key", SharedPrivateKeyPath,
		"--key-alias", SharedKeyAlias,
	)
	t.Logf("Evidence creation output: %s", createOutput)
	require.NotContains(t, createOutput, "Error", "Evidence creation should not error")
	t.Log("✓ Evidence created successfully")

	// Step 4: Get evidence for artifact with project using Project CLI
	t.Log("Step 4: Getting evidence for artifact with project using Project CLI...")
	getOutput := r.EvidenceProjectCLI.RunCliCmdWithOutput(t,
		"get",
		"--subject-repo-path", repoPath,
		"--project", e2e.ProjectKey,
	)
	t.Logf("Get evidence output: %s", getOutput)
	require.Contains(t, getOutput, repoPath, "Output should contain the subject path")
	t.Log("✅ Project user successfully retrieved artifact evidence with project scope!")

	t.Log("=== ✅ Get Evidence for Artifact with Project Test Completed Successfully! ===")
}

// RunGetEvidenceForReleaseBundle tests getting evidence for a Release Bundle v2
func (r *EvidenceE2ETestsRunner) RunGetEvidenceForReleaseBundle(t *testing.T) {
	t.Log("=== Get Evidence - Release Bundle Test ===")

	// Verify shared key pair is available
	if SharedPrivateKeyPath == "" || SharedPublicKeyPath == "" {
		t.Errorf("Shared key pair not initialized. Ensure PrepareTestsData() was called.")
		return
	}
	t.Logf("Using shared key pair: %s (alias: %s)", SharedPrivateKeyPath, SharedKeyAlias)

	tempDir := t.TempDir()

	// Step 1: Create release bundle using Lifecycle Manager (Admin)
	t.Log("Step 1: Creating release bundle using Lifecycle Manager...")

	releaseBundleName, releaseBundleVersion := utils.CreateTestReleaseBundle(t, r.ServicesManager, r.LifecycleManager, "")
	t.Logf("✓ Release bundle created: %s/%s", releaseBundleName, releaseBundleVersion)

	// Step 2: Create predicate (Admin)
	t.Log("Step 2: Creating predicate...")
	predicate := map[string]interface{}{
		"buildType":            "get-release-bundle-test",
		"timestamp":            time.Now().Unix(),
		"environment":          "e2e-get-test",
		"releaseBundleName":    releaseBundleName,
		"releaseBundleVersion": releaseBundleVersion,
		"description":          "Testing get evidence for release bundle",
	}
	predicateBytes, err := json.MarshalIndent(predicate, "", "  ")
	require.NoError(t, err)
	predicatePath := filepath.Join(tempDir, "predicate.json")
	err = os.WriteFile(predicatePath, predicateBytes, 0644)
	require.NoError(t, err)
	t.Log("✓ Predicate created")

	// Step 3: Create evidence for release bundle using Admin CLI
	t.Log("Step 3: Creating evidence for release bundle using Admin CLI...")
	createOutput := r.EvidenceAdminCLI.RunCliCmdWithOutput(t,
		"create",
		"--predicate", predicatePath,
		"--predicate-type", "https://slsa.dev/provenance/v1",
		"--release-bundle", releaseBundleName,
		"--release-bundle-version", releaseBundleVersion,
		"--key", SharedPrivateKeyPath,
		"--key-alias", SharedKeyAlias,
	)
	t.Logf("Evidence creation output: %s", createOutput)
	require.NotContains(t, createOutput, "Error", "Evidence creation should not error")
	t.Log("✓ Evidence created successfully")

	// Step 4: Get evidence for release bundle using User CLI
	t.Log("Step 4: Getting evidence for release bundle using User CLI...")
	getOutput := r.EvidenceUserCLI.RunCliCmdWithOutput(t,
		"get",
		"--release-bundle", releaseBundleName,
		"--release-bundle-version", releaseBundleVersion,
	)
	t.Logf("Get evidence output: %s", getOutput)
	require.Contains(t, getOutput, releaseBundleName, "Output should contain release bundle name")
	require.Contains(t, getOutput, releaseBundleVersion, "Output should contain release bundle version")
	t.Log("✅ User successfully retrieved release bundle evidence!")

	t.Log("=== ✅ Get Evidence for Release Bundle Test Completed Successfully! ===")
}
