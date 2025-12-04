package tests

import (
	"crypto/sha256"
	"encoding/hex"
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

// RunCreateEvidenceSuite runs all create evidence tests for different subject types
func (r *EvidenceE2ETestsRunner) RunCreateEvidenceSuite(t *testing.T) {
	// Test different subject types
	t.Run("ForArtifact", func(t *testing.T) {
		r.RunCreateEvidenceForArtifact(t)
	})
	t.Run("ForBuild", func(t *testing.T) {
		r.RunCreateEvidenceForBuild(t)
	})
	t.Run("ForBuildWithProject", func(t *testing.T) {
		r.RunCreateEvidenceForBuildWithProject(t)
	})
	t.Run("ForPackage", func(t *testing.T) {
		r.RunCreateEvidenceForPackage(t)
	})
	t.Run("ForReleaseBundle", func(t *testing.T) {
		r.RunCreateEvidenceForReleaseBundle(t)
	})
	t.Run("ForApplicationVersion", func(t *testing.T) {
		r.RunCreateEvidenceForApplicationVersion(t)
	})
	t.Run("WithMarkdown", func(t *testing.T) {
		r.RunCreateEvidenceWithMarkdown(t)
	})
	t.Run("WithSubjectSha256", func(t *testing.T) {
		r.RunCreateEvidenceWithSubjectSha256(t)
	})
}

func (r *EvidenceE2ETestsRunner) RunCreateEvidenceForArtifact(t *testing.T) {
	t.Log("=== Create Evidence - Artifact/Repo Path Test ===")

	// Verify shared key pair is available
	if SharedPrivateKeyPath == "" || SharedPublicKeyPath == "" {
		t.Errorf("Shared key pair not initialized. Ensure PrepareTestsData() was called.")
	}
	t.Logf("Using shared key pair: %s (alias: %s)", SharedPrivateKeyPath, SharedKeyAlias)

	// Step 1: Create repository and upload artifact
	t.Log("Step 1: Creating repository and uploading artifact...")
	repoName := utils.CreateTestRepositoryWithName(t, r.ServicesManager, "generic")
	tempDir := t.TempDir()

	artifactContent := fmt.Sprintf("Test artifact for evidence - timestamp: %d", time.Now().Unix())
	artifactPath := utils.CreateTestArtifact(t, artifactContent)
	artifactFileName := filepath.Base(artifactPath)
	repoPath := fmt.Sprintf("%s/%s", repoName, artifactFileName)
	err := utils.UploadArtifact(r.ServicesManager, artifactPath, repoPath)
	require.NoError(t, err)
	t.Logf("✓ Artifact uploaded: %s", repoPath)

	// Step 2: Create predicate
	t.Log("Step 2: Creating predicate...")
	predicate := map[string]interface{}{
		"buildType":   "artifact-test",
		"timestamp":   time.Now().Unix(),
		"environment": "e2e-test",
		"subject":     repoPath,
	}
	predicateBytes, err := json.MarshalIndent(predicate, "", "  ")
	require.NoError(t, err)
	predicatePath := filepath.Join(tempDir, "predicate.json")
	err = os.WriteFile(predicatePath, predicateBytes, 0644)
	require.NoError(t, err)
	t.Log("✓ Predicate created")

	// Step 3: Create evidence using shared key
	t.Log("Step 3: Creating evidence for artifact...")
	createOutput := r.EvidenceUserCLI.RunCliCmdWithOutput(t,
		"create",
		"--predicate", predicatePath,
		"--predicate-type", "https://slsa.dev/provenance/v1",
		"--subject-repo-path", repoPath,
		"--key", SharedPrivateKeyPath,
		"--key-alias", SharedKeyAlias, // Tell CLI which key to use for auto-verification
	)
	t.Logf("Evidence creation output: %s", createOutput)
	require.NotContains(t, createOutput, "Error", "Evidence creation should not error")
	require.NotContains(t, createOutput, "Failed", "Evidence creation should not fail")
	t.Log("✓ Evidence created successfully")

	// Step 4: Get evidence to validate it was created correctly
	t.Log("Step 4: Getting evidence to validate creation...")
	getOutput := r.EvidenceAdminCLI.RunCliCmdWithOutput(t,
		"get",
		"--subject-repo-path", repoPath,
	)
	t.Logf("Evidence get output: %s", getOutput)
	require.NotContains(t, getOutput, "Error", "Should be able to get evidence")
	require.NotContains(t, getOutput, "Failed", "Get evidence should not fail")
	// Validate that evidence exists for the artifact
	require.Contains(t, getOutput, repoPath, "Should return evidence for the artifact")
	t.Log("✓ Evidence retrieved successfully")

	t.Log("=== ✅ Create Evidence for Artifact Test Completed Successfully! ===")
}

func (r *EvidenceE2ETestsRunner) RunCreateEvidenceForBuild(t *testing.T) {
	t.Log("=== Create Evidence - Build Test ===")

	// Verify shared key pair is available
	if SharedPrivateKeyPath == "" || SharedPublicKeyPath == "" {
		t.Errorf("Shared key pair not initialized. Ensure PrepareTestsData() was called.")
	}
	t.Logf("Using shared key pair: %s (alias: %s)", SharedPrivateKeyPath, SharedKeyAlias)

	tempDir := t.TempDir()

	// Step 1: Create build info
	t.Log("Step 1: Creating build info...")
	buildName, buildNumber := utils.CreateTestBuildInfo(t, r.ServicesManager, "")
	t.Logf("✓ Build info created: %s/%s", buildName, buildNumber)

	// Step 2: Create predicate
	t.Log("Step 2: Creating predicate...")
	predicate := map[string]interface{}{
		"buildType":   "build-test",
		"timestamp":   time.Now().Unix(),
		"environment": "e2e-test",
		"buildName":   buildName,
		"buildNumber": buildNumber,
	}
	predicateBytes, err := json.MarshalIndent(predicate, "", "  ")
	require.NoError(t, err)
	predicatePath := filepath.Join(tempDir, "predicate.json")
	err = os.WriteFile(predicatePath, predicateBytes, 0644)
	require.NoError(t, err)
	t.Log("✓ Predicate created")

	// Step 3: Create evidence for build using shared key
	t.Log("Step 3: Creating evidence for build...")
	createOutput := r.EvidenceUserCLI.RunCliCmdWithOutput(t,
		"create",
		"--predicate", predicatePath,
		"--predicate-type", "https://slsa.dev/provenance/v1",
		"--build-name", buildName,
		"--build-number", buildNumber,
		"--key", SharedPrivateKeyPath,
		"--key-alias", SharedKeyAlias, // Tell CLI which key to use for auto-verification
	)
	t.Logf("Evidence creation output: %s", createOutput)
	require.NotContains(t, createOutput, "Error", "Evidence creation should not error")
	require.NotContains(t, createOutput, "Failed", "Evidence creation should not fail")
	require.NotContains(t, createOutput, "failed to find buildName", "Build info should be found")
	t.Log("✓ Evidence created successfully")

	t.Log("Step 4: Verifying evidence using admin token")
	verifyOutput := r.EvidenceAdminCLI.RunCliCmdWithOutput(t,
		"verify",
		"--build-name", buildName,
		"--build-number", buildNumber,
		"--public-keys", SharedPublicKeyPath,
	)
	t.Logf("Verification output: %s", verifyOutput)
	require.Contains(t, verifyOutput, buildName, "Evidence should be verified")
	t.Log("✅ Evidence verified successfully!")

	t.Log("=== ✅ Create Evidence for Build Test Completed Successfully! ===")
}

// RunCreateEvidenceForReleaseBundle tests creating evidence for a release bundle
func (r *EvidenceE2ETestsRunner) RunCreateEvidenceForReleaseBundle(t *testing.T) {
	t.Log("=== Create Evidence - Release Bundle Test ===")

	// Lifecycle is embedded in Artifactory Pro/Enterprise
	t.Log("Using embedded Lifecycle service in Artifactory...")

	// Verify shared key pair is available
	if SharedPrivateKeyPath == "" || SharedPublicKeyPath == "" {
		t.Errorf("Shared key pair not initialized. Ensure PrepareTestsData() was called.")
	}
	t.Logf("Using shared key pair: %s (alias: %s)", SharedPrivateKeyPath, SharedKeyAlias)

	tempDir := t.TempDir()

	// Step 1: Create release bundle
	t.Log("Step 1: Creating release bundle...")
	rbName, rbVersion := utils.CreateTestReleaseBundle(t, r.ServicesManager, r.LifecycleManager, "")
	t.Logf("✓ Release bundle created: %s/%s", rbName, rbVersion)

	// Step 2: Create predicate
	t.Log("Step 2: Creating predicate...")
	predicate := map[string]interface{}{
		"buildType":            "release-bundle-test",
		"timestamp":            time.Now().Unix(),
		"environment":          "e2e-test",
		"releaseBundleName":    rbName,
		"releaseBundleVersion": rbVersion,
	}
	predicateBytes, err := json.MarshalIndent(predicate, "", "  ")
	require.NoError(t, err)
	predicatePath := filepath.Join(tempDir, "predicate.json")
	err = os.WriteFile(predicatePath, predicateBytes, 0644)
	require.NoError(t, err)
	t.Log("✓ Predicate created")

	// Step 3: Create evidence for release bundle using shared key
	t.Log("Step 3: Creating evidence for release bundle...")
	createOutput := r.EvidenceUserCLI.RunCliCmdWithOutput(t,
		"create",
		"--predicate", predicatePath,
		"--predicate-type", "https://slsa.dev/provenance/v1",
		"--release-bundle", rbName,
		"--release-bundle-version", rbVersion,
		"--key", SharedPrivateKeyPath,
		"--key-alias", SharedKeyAlias,
	)
	t.Logf("Evidence creation output: %s", createOutput)
	require.NotContains(t, createOutput, "Error", "Evidence creation should not error")
	require.NotContains(t, createOutput, "Failed", "Evidence creation should not fail")
	t.Log("✓ Evidence created successfully")

	// Step 4: Verify evidence using admin token
	t.Log("Step 4: Verifying evidence using admin token...")
	verifyOutput := r.EvidenceAdminCLI.RunCliCmdWithOutput(t,
		"verify",
		"--release-bundle", rbName,
		"--release-bundle-version", rbVersion,
		"--public-keys", SharedPublicKeyPath,
	)
	t.Logf("Verification output: %s", verifyOutput)
	require.Contains(t, verifyOutput, rbName, "Evidence should be verified")
	t.Log("✅ Evidence verified successfully!")

	t.Log("=== ✅ Create Evidence for Release Bundle Test Completed Successfully! ===")
}

// RunCreateEvidenceForPackage tests creating evidence for a package
func (r *EvidenceE2ETestsRunner) RunCreateEvidenceForPackage(t *testing.T) {
	t.Log("=== Create Evidence - Package Test ===")

	// Verify shared key pair is available
	if SharedPrivateKeyPath == "" || SharedPublicKeyPath == "" {
		t.Errorf("Shared key pair not initialized. Ensure PrepareTestsData() was called.")
	}
	t.Logf("Using shared key pair: %s (alias: %s)", SharedPrivateKeyPath, SharedKeyAlias)

	tempDir := t.TempDir()

	// Step 1: Create package
	t.Log("Step 1: Creating package...")
	packageName, packageVersion, repoName := utils.CreateTestPackage(t, r.ServicesManager, "generic")
	t.Logf("✓ Package created: %s/%s in repo %s", packageName, packageVersion, repoName)

	// Step 2: Create predicate
	t.Log("Step 2: Creating predicate...")
	predicate := map[string]interface{}{
		"buildType":      "package-test",
		"timestamp":      time.Now().Unix(),
		"environment":    "e2e-test",
		"packageName":    packageName,
		"packageVersion": packageVersion,
	}
	predicateBytes, err := json.MarshalIndent(predicate, "", "  ")
	require.NoError(t, err)
	predicatePath := filepath.Join(tempDir, "predicate.json")
	err = os.WriteFile(predicatePath, predicateBytes, 0644)
	require.NoError(t, err)
	t.Log("✓ Predicate created")

	// Step 3: Create evidence for package using shared key
	t.Log("Step 3: Creating evidence for package...")
	createOutput := r.EvidenceUserCLI.RunCliCmdWithOutput(t,
		"create",
		"--predicate", predicatePath,
		"--predicate-type", "https://slsa.dev/provenance/v1",
		"--package-name", packageName,
		"--package-version", packageVersion,
		"--package-repo-name", repoName,
		"--key", SharedPrivateKeyPath,
		"--key-alias", SharedKeyAlias,
	)
	t.Logf("Evidence creation output: %s", createOutput)
	require.NotContains(t, createOutput, "Error", "Evidence creation should not error")
	require.NotContains(t, createOutput, "Failed", "Evidence creation should not fail")
	t.Log("✓ Evidence created successfully")

	// Step 4: Verify evidence using admin token
	t.Log("Step 4: Verifying evidence using admin token...")
	verifyOutput := r.EvidenceAdminCLI.RunCliCmdWithOutput(t,
		"verify",
		"--package-name", packageName,
		"--package-version", packageVersion,
		"--package-repo-name", repoName,
		"--public-keys", SharedPublicKeyPath,
	)
	t.Logf("Verification output: %s", verifyOutput)
	require.Contains(t, verifyOutput, packageName, "Evidence should be verified")
	t.Log("✅ Evidence verified successfully!")

	t.Log("=== ✅ Create Evidence for Package Test Completed Successfully! ===")
}

func (r *EvidenceE2ETestsRunner) RunCreateEvidenceForBuildWithProject(t *testing.T) {
	t.Log("=== Create Evidence - Build With Project Test ===")

	// Verify shared key pair is available
	if SharedPrivateKeyPath == "" || SharedPublicKeyPath == "" {
		t.Errorf("Shared key pair not initialized. Ensure PrepareTestsData() was called.")
		return
	}
	t.Logf("Using shared key pair: %s (alias: %s)", SharedPrivateKeyPath, SharedKeyAlias)
	t.Logf("Using project: %s with role-based permissions (Developer)", e2e.ProjectKey)

	tempDir := t.TempDir()

	// Step 1: Create build info with project (using admin for build creation)
	t.Log("Step 1: Creating build info with project...")
	buildName, buildNumber := utils.CreateTestBuildInfo(t, r.ServicesManager, e2e.ProjectKey)
	t.Logf("✓ Build info created: %s/%s in project %s", buildName, buildNumber, e2e.ProjectKey)

	// Step 2: Create predicate
	t.Log("Step 2: Creating predicate...")
	predicate := map[string]interface{}{
		"buildType":   "build-project-test",
		"timestamp":   time.Now().Unix(),
		"environment": "e2e-test",
		"project":     e2e.ProjectKey,
		"role":        "Developer",
	}
	predicateBytes, err := json.MarshalIndent(predicate, "", "  ")
	require.NoError(t, err)
	predicatePath := filepath.Join(tempDir, "predicate.json")
	err = os.WriteFile(predicatePath, predicateBytes, 0644)
	require.NoError(t, err)
	t.Log("✓ Predicate created")

	// Step 3: Create evidence using project-scoped CLI (with Developer role permissions)
	t.Log("Step 3: Creating evidence using project-scoped token (Developer role)...")
	createOutput := r.EvidenceProjectCLI.RunCliCmdWithOutput(t,
		"create",
		"--predicate", predicatePath,
		"--predicate-type", "https://slsa.dev/provenance/v1",
		"--build-name", buildName,
		"--build-number", buildNumber,
		"--project", e2e.ProjectKey,
		"--key", SharedPrivateKeyPath,
		"--key-alias", SharedKeyAlias,
	)
	t.Logf("Evidence creation output: %s", createOutput)
	require.NotContains(t, createOutput, "Error", "Evidence creation should not error")
	require.NotContains(t, createOutput, "Failed", "Evidence creation should not fail")
	t.Log("✓ Evidence created successfully with project-scoped token (Developer role)")

	// Step 4: Verify evidence using admin token
	t.Log("Step 4: Verifying evidence using admin token...")
	verifyOutput := r.EvidenceAdminCLI.RunCliCmdWithOutput(t,
		"verify",
		"--build-name", buildName,
		"--build-number", buildNumber,
		"--project", e2e.ProjectKey,
		"--public-keys", SharedPublicKeyPath,
	)
	t.Logf("Verification output: %s", verifyOutput)
	require.Contains(t, verifyOutput, buildName, "Evidence should be verified")
	t.Log("✅ Evidence verified successfully!")

	t.Log("=== ✅ Create Evidence for Build with Project Test Completed Successfully! ===")
}

// RunCreateEvidenceWithMarkdown tests creating evidence with markdown file
func (r *EvidenceE2ETestsRunner) RunCreateEvidenceWithMarkdown(t *testing.T) {
	t.Log("=== Create Evidence - With Markdown Test ===")

	// Verify shared key pair is available
	if SharedPrivateKeyPath == "" || SharedPublicKeyPath == "" {
		t.Errorf("Shared key pair not initialized. Ensure PrepareTestsData() was called.")
		return
	}
	t.Logf("Using shared key pair: %s (alias: %s)", SharedPrivateKeyPath, SharedKeyAlias)

	tempDir := t.TempDir()

	// Step 1: Create repository and upload artifact
	t.Log("Step 1: Creating repository and uploading artifact...")
	repoName := utils.CreateTestRepositoryWithName(t, r.ServicesManager, "generic")
	artifactPath := utils.CreateTestArtifact(t, "Test artifact for markdown evidence")
	artifactFileName := filepath.Base(artifactPath)
	repoPath := fmt.Sprintf("%s/%s", repoName, artifactFileName)
	err := utils.UploadArtifact(r.ServicesManager, artifactPath, repoPath)
	require.NoError(t, err)
	t.Logf("✓ Artifact uploaded: %s", repoPath)

	// Step 2: Create predicate
	t.Log("Step 2: Creating predicate...")
	predicate := map[string]interface{}{
		"buildType":   "markdown-test",
		"timestamp":   time.Now().Unix(),
		"environment": "e2e-test",
		"description": "Evidence with markdown documentation",
	}
	predicateBytes, err := json.MarshalIndent(predicate, "", "  ")
	require.NoError(t, err)
	predicatePath := filepath.Join(tempDir, "predicate.json")
	err = os.WriteFile(predicatePath, predicateBytes, 0644)
	require.NoError(t, err)
	t.Log("✓ Predicate created")

	// Step 3: Create markdown file
	t.Log("Step 3: Creating markdown documentation...")
	timestamp := time.Now().Unix()
	markdownContent := fmt.Sprintf(
		"# Evidence Documentation\n\n"+
			"## Test Information\n"+
			"- **Created at**: %d\n"+
			"- **Subject**: `%s`\n"+
			"- **Repository**: `%s`\n"+
			"- **Test Type**: Markdown Evidence\n\n"+
			"## Description\n"+
			"This evidence includes markdown documentation to demonstrate the markdown feature.\n"+
			"The evidence was created as part of the E2E test suite to validate markdown support.\n\n"+
			"## Features Tested\n"+
			"- Markdown file attachment to evidence\n"+
			"- Integration with SLSA provenance predicate\n"+
			"- Evidence verification with public keys\n",
		timestamp, repoPath, repoName,
	)
	markdownPath := filepath.Join(tempDir, "evidence.md")
	err = os.WriteFile(markdownPath, []byte(markdownContent), 0644)
	require.NoError(t, err)
	t.Log("✓ Markdown documentation created")

	// Step 4: Create evidence with markdown using shared key
	t.Log("Step 4: Creating evidence with markdown...")
	createOutput := r.EvidenceUserCLI.RunCliCmdWithOutput(t,
		"create",
		"--predicate", predicatePath,
		"--predicate-type", "https://slsa.dev/provenance/v1",
		"--subject-repo-path", repoPath,
		"--markdown", markdownPath,
		"--key", SharedPrivateKeyPath,
		"--key-alias", SharedKeyAlias,
	)
	t.Logf("Evidence creation output: %s", createOutput)
	require.NotContains(t, createOutput, "Error", "Evidence creation should not error")
	require.NotContains(t, createOutput, "Failed", "Evidence creation should not fail")
	t.Log("✓ Evidence with markdown created successfully")

	// Step 5: Verify evidence using admin token
	t.Log("Step 5: Verifying evidence using admin token...")
	verifyOutput := r.EvidenceAdminCLI.RunCliCmdWithOutput(t,
		"verify",
		"--subject-repo-path", repoPath,
		"--public-keys", SharedPublicKeyPath,
	)
	t.Logf("Verification output: %s", verifyOutput)
	require.Contains(t, verifyOutput, repoPath, "Evidence should be verified")
	t.Log("✅ Evidence verified successfully!")

	t.Log("=== ✅ Create Evidence with Markdown Test Completed Successfully! ===")
}

// RunCreateEvidenceWithSubjectSha256 tests creating evidence with explicit SHA256
func (r *EvidenceE2ETestsRunner) RunCreateEvidenceWithSubjectSha256(t *testing.T) {
	t.Log("=== Create Evidence - With Subject SHA256 Test ===")

	// Verify shared key pair is available
	if SharedPrivateKeyPath == "" || SharedPublicKeyPath == "" {
		t.Errorf("Shared key pair not initialized. Ensure PrepareTestsData() was called.")
		return
	}
	t.Logf("Using shared key pair: %s (alias: %s)", SharedPrivateKeyPath, SharedKeyAlias)

	tempDir := t.TempDir()

	// Step 1: Create repository and upload artifact
	t.Log("Step 1: Creating repository and uploading artifact...")
	repoName := utils.CreateTestRepositoryWithName(t, r.ServicesManager, "generic")
	artifactContent := "Test artifact for SHA256 explicit specification"
	artifactPath := utils.CreateTestArtifact(t, artifactContent)
	artifactFileName := filepath.Base(artifactPath)
	repoPath := fmt.Sprintf("%s/%s", repoName, artifactFileName)
	err := utils.UploadArtifact(r.ServicesManager, artifactPath, repoPath)
	require.NoError(t, err)
	t.Logf("✓ Artifact uploaded: %s", repoPath)

	// Step 2: Calculate SHA256 of artifact
	t.Log("Step 2: Calculating artifact SHA256...")
	artifactData, err := os.ReadFile(artifactPath)
	require.NoError(t, err)
	hash := sha256.Sum256(artifactData)
	subjectSha256 := hex.EncodeToString(hash[:])
	t.Logf("✓ Artifact SHA256: %s", subjectSha256)

	// Step 3: Create predicate with SHA256
	t.Log("Step 3: Creating predicate with SHA256...")
	predicate := map[string]interface{}{
		"buildType":     "sha256-explicit-test",
		"timestamp":     time.Now().Unix(),
		"environment":   "e2e-test",
		"subjectSha256": subjectSha256,
		"description":   "Evidence created with explicit SHA256 specification",
	}
	predicateBytes, err := json.MarshalIndent(predicate, "", "  ")
	require.NoError(t, err)
	predicatePath := filepath.Join(tempDir, "predicate.json")
	err = os.WriteFile(predicatePath, predicateBytes, 0644)
	require.NoError(t, err)
	t.Log("✓ Predicate created with SHA256")

	// Step 4: Create evidence with explicit SHA256 using shared key
	t.Log("Step 4: Creating evidence with explicit SHA256...")
	createOutput := r.EvidenceUserCLI.RunCliCmdWithOutput(t,
		"create",
		"--predicate", predicatePath,
		"--predicate-type", "https://slsa.dev/provenance/v1",
		"--subject-repo-path", repoPath,
		"--subject-sha256", subjectSha256,
		"--key", SharedPrivateKeyPath,
		"--key-alias", SharedKeyAlias,
	)
	t.Logf("Evidence creation output: %s", createOutput)
	require.NotContains(t, createOutput, "Error", "Evidence creation should not error")
	require.NotContains(t, createOutput, "Failed", "Evidence creation should not fail")
	t.Log("✓ Evidence with explicit SHA256 created successfully")

	// Step 5: Verify evidence using admin token
	t.Log("Step 5: Verifying evidence using admin token...")
	verifyOutput := r.EvidenceAdminCLI.RunCliCmdWithOutput(t,
		"verify",
		"--subject-repo-path", repoPath,
		"--public-keys", SharedPublicKeyPath,
	)
	t.Logf("Verification output: %s", verifyOutput)
	require.Contains(t, verifyOutput, repoPath, "Evidence should be verified")
	require.Contains(t, verifyOutput, subjectSha256, "Verification should show the SHA256")
	t.Log("✅ Evidence verified successfully with SHA256!")

	t.Log("=== ✅ Create Evidence with Subject SHA256 Test Completed Successfully! ===")
}

// RunCreateEvidenceForApplicationVersion tests creating evidence for an application version
func (r *EvidenceE2ETestsRunner) RunCreateEvidenceForApplicationVersion(t *testing.T) {
	t.Log("=== Create Evidence - Application Version Test ===")

	// Verify shared key pair is available
	if SharedPrivateKeyPath == "" || SharedPublicKeyPath == "" {
		t.Errorf("Shared key pair not initialized. Ensure PrepareTestsData() was called.")
		return
	}
	t.Logf("Using shared key pair: %s (alias: %s)", SharedPrivateKeyPath, SharedKeyAlias)

	tempDir := t.TempDir()
	projectKey := "evidencee2e" // Use the project created by e2e-bootstrap.sh

	// Track resources for cleanup
	var applicationKey string
	var applicationVersion string

	// Register cleanup first - runs even if test fails
	t.Cleanup(func() {
		if applicationKey != "" {
			// Delete application (this also deletes all versions)
			utils.CleanupTestApplication(t, r.ServicesManager, applicationKey, projectKey)
		}
	})

	// Step 1: Create test application
	t.Log("Step 1: Creating test application...")
	var applicationName string
	applicationKey, applicationName = utils.CreateTestApplication(t, r.ServicesManager, projectKey)
	t.Logf("✓ Application created: %s (%s)", applicationKey, applicationName)

	// Step 2: Create test application version
	t.Log("Step 2: Creating test application version via AppTrust API...")
	applicationVersion = utils.CreateTestApplicationVersion(t, r.ServicesManager, r.LifecycleManager, applicationKey, projectKey)
	t.Logf("✓ Application version created: %s:%s", applicationKey, applicationVersion)

	t.Log("Step 3: Creating predicate...")
	predicate := map[string]interface{}{
		"buildType":          "application-version-test",
		"timestamp":          time.Now().Unix(),
		"environment":        "e2e-test",
		"applicationKey":     applicationKey,
		"applicationVersion": applicationVersion,
		"projectKey":         projectKey,
		"description":        "Testing evidence creation for application version",
	}
	predicateBytes, err := json.MarshalIndent(predicate, "", "  ")
	require.NoError(t, err)
	predicatePath := filepath.Join(tempDir, "predicate.json")
	err = os.WriteFile(predicatePath, predicateBytes, 0644)
	require.NoError(t, err)
	t.Log("✓ Predicate created")

	t.Log("Step 4: Creating evidence for application version...")
	createOutput := r.EvidenceUserCLI.RunCliCmdWithOutput(t,
		"create",
		"--predicate", predicatePath,
		"--predicate-type", "https://slsa.dev/provenance/v1",
		"--application-key", applicationKey,
		"--application-version", applicationVersion,
		"--key", SharedPrivateKeyPath,
		"--key-alias", SharedKeyAlias,
	)
	t.Logf("Evidence creation output: %s", createOutput)
	require.NotContains(t, createOutput, "Error", "Evidence creation should not error")
	require.NotContains(t, createOutput, "Failed", "Evidence creation should not fail")
	require.NotContains(t, createOutput, "does not exist", "Application version manifest should be found")

	t.Log("✅ Evidence created successfully!")

	t.Log("=== ✅ Create Evidence for Application Version Test Completed Successfully! ===")
}
