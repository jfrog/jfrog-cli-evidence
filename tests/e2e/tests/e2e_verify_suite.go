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

// RunVerifyEvidenceSuite runs all verify evidence tests
func (r *EvidenceE2ETestsRunner) RunVerifyEvidenceSuite(t *testing.T) {
	t.Run("ForArtifact", func(t *testing.T) {
		r.RunVerifyEvidenceForArtifact(t)
	})
	t.Run("ForArtifactWithPublicKey", func(t *testing.T) {
		r.RunVerifyEvidenceForArtifactWithPublicKey(t)
	})
	t.Run("ForArtifactWithProject", func(t *testing.T) {
		r.RunVerifyEvidenceForArtifactWithProject(t)
	})
	t.Run("ForArtifactWithUseArtifactoryKeys", func(t *testing.T) {
		r.RunVerifyEvidenceWithUseArtifactoryKeys(t)
	})
	t.Run("ForReleaseBundle", func(t *testing.T) {
		r.RunVerifyEvidenceForReleaseBundle(t)
	})
	t.Run("ForBuild", func(t *testing.T) {
		r.RunVerifyEvidenceForBuild(t)
	})
	t.Run("ForBuildWithProject", func(t *testing.T) {
		r.RunVerifyEvidenceForBuildWithProject(t)
	})
	t.Run("ForPackage", func(t *testing.T) {
		r.RunVerifyEvidenceForPackage(t)
	})
	t.Run("WithJsonFormat", func(t *testing.T) {
		r.RunVerifyEvidenceWithJsonFormat(t)
	})
}

// RunVerifyEvidenceForArtifact tests verifying evidence for an artifact
func (r *EvidenceE2ETestsRunner) RunVerifyEvidenceForArtifact(t *testing.T) {
	t.Log("=== Verify Evidence - Artifact Test ===")

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
	artifactContent := fmt.Sprintf("Test artifact for verify evidence - timestamp: %d", time.Now().Unix())
	artifactPath := utils.CreateTestArtifact(t, artifactContent)
	artifactFileName := filepath.Base(artifactPath)
	repoPath := fmt.Sprintf("%s/%s", repoName, artifactFileName)
	err := utils.UploadArtifact(r.ServicesManager, artifactPath, repoPath)
	require.NoError(t, err)
	t.Logf("✓ Artifact uploaded: %s", repoPath)

	// Step 2: Create predicate (Admin)
	t.Log("Step 2: Creating predicate...")
	predicate := map[string]interface{}{
		"buildType":   "verify-artifact-test",
		"timestamp":   time.Now().Unix(),
		"environment": "e2e-verify-test",
	}
	predicateBytes, err := json.MarshalIndent(predicate, "", "  ")
	require.NoError(t, err)
	predicatePath := filepath.Join(tempDir, "predicate.json")
	err = os.WriteFile(predicatePath, predicateBytes, 0644)
	require.NoError(t, err)
	t.Log("✓ Predicate created")

	// Step 3: Create evidence using Admin CLI
	t.Log("Step 3: Creating evidence using Admin CLI...")
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

	// Step 4: Verify evidence using User CLI
	t.Log("Step 4: Verifying evidence using User CLI...")
	verifyOutput := r.EvidenceUserCLI.RunCliCmdWithOutput(t,
		"verify",
		"--subject-repo-path", repoPath,
		"--public-keys", SharedPublicKeyPath,
	)
	t.Logf("Verification output: %s", verifyOutput)
	require.Contains(t, verifyOutput, repoPath, "Evidence should be verified")
	require.Contains(t, verifyOutput, "Verification passed", "Verification should pass")
	t.Log("✅ User successfully verified evidence!")

	t.Log("=== ✅ Verify Evidence for Artifact Test Completed Successfully! ===")
}

// RunVerifyEvidenceForArtifactWithPublicKey tests verification with explicit public key
func (r *EvidenceE2ETestsRunner) RunVerifyEvidenceForArtifactWithPublicKey(t *testing.T) {
	t.Log("=== Verify Evidence - With Public Key Test ===")

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
	artifactPath := utils.CreateTestArtifact(t, "Test for public key verification")
	artifactFileName := filepath.Base(artifactPath)
	repoPath := fmt.Sprintf("%s/%s", repoName, artifactFileName)
	err := utils.UploadArtifact(r.ServicesManager, artifactPath, repoPath)
	require.NoError(t, err)
	t.Logf("✓ Artifact uploaded: %s", repoPath)

	// Step 2: Create predicate (Admin)
	t.Log("Step 2: Creating predicate...")
	predicate := map[string]interface{}{
		"buildType":   "verify-public-key-test",
		"timestamp":   time.Now().Unix(),
		"environment": "e2e-verify-test",
		"description": "Testing explicit public key verification",
	}
	predicateBytes, err := json.MarshalIndent(predicate, "", "  ")
	require.NoError(t, err)
	predicatePath := filepath.Join(tempDir, "predicate.json")
	err = os.WriteFile(predicatePath, predicateBytes, 0644)
	require.NoError(t, err)
	t.Log("✓ Predicate created")

	// Step 3: Create evidence using Admin CLI
	t.Log("Step 3: Creating evidence using Admin CLI...")
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

	// Step 4: Verify with explicit public key using User CLI (THIS IS WHAT WE'RE TESTING)
	t.Log("Step 4: Verifying evidence with explicit public key using User CLI...")
	verifyOutput := r.EvidenceUserCLI.RunCliCmdWithOutput(t,
		"verify",
		"--subject-repo-path", repoPath,
		"--public-keys", SharedPublicKeyPath,
		"--format", "json",
	)
	t.Logf("Verification output: %s", verifyOutput)

	// Verify output is valid JSON
	var jsonData map[string]interface{}
	err = json.Unmarshal([]byte(verifyOutput), &jsonData)
	require.NoError(t, err, "Output should be valid JSON")

	// Check for success in JSON format
	status, ok := jsonData["overallVerificationStatus"]
	require.True(t, ok, "JSON should contain overallVerificationStatus field")
	require.Equal(t, "success", status, "Overall verification status should be success")
	t.Log("✅ User successfully verified evidence with explicit public key (JSON format)!")

	t.Log("=== ✅ Verify Evidence with Public Key Test Completed Successfully! ===")
}

// RunVerifyEvidenceForArtifactWithProject tests verifying evidence for an artifact with project
func (r *EvidenceE2ETestsRunner) RunVerifyEvidenceForArtifactWithProject(t *testing.T) {
	t.Skip("Skipping project-based test - GraphQL authentication context issue pending fix")
	t.Log("=== Verify Evidence - Artifact With Project Test ===")

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
	repoName := utils.CreateTestRepositoryWithProject(t, r.ServicesManager, "generic", e2e.ProjectKey)
	artifactContent := fmt.Sprintf("Test artifact for project verify - timestamp: %d", time.Now().Unix())
	artifactPath := utils.CreateTestArtifact(t, artifactContent)
	artifactFileName := filepath.Base(artifactPath)
	repoPath := fmt.Sprintf("%s/%s", repoName, artifactFileName)
	err := utils.UploadArtifact(r.ServicesManager, artifactPath, repoPath)
	require.NoError(t, err)
	t.Logf("✓ Artifact uploaded: %s", repoPath)

	// Step 2: Create predicate (Admin)
	t.Log("Step 2: Creating predicate...")
	predicate := map[string]interface{}{
		"buildType":   "verify-artifact-project-test",
		"timestamp":   time.Now().Unix(),
		"environment": "e2e-verify-test",
		"project":     e2e.ProjectKey,
		"description": "Testing artifact verification with project scope",
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

	// Step 4: Verify evidence for artifact with project using Project CLI
	t.Log("Step 4: Verifying evidence for artifact with project using Project CLI...")
	verifyOutput := r.EvidenceProjectCLI.RunCliCmdWithOutput(t,
		"verify",
		"--subject-repo-path", repoPath,
		"--project", e2e.ProjectKey,
		"--public-keys", SharedPublicKeyPath,
	)
	t.Logf("Verification output: %s", verifyOutput)
	require.Contains(t, verifyOutput, repoPath, "Evidence should be verified")
	require.Contains(t, verifyOutput, "Verification passed", "Verification should pass")
	t.Log("✅ Project user successfully verified artifact evidence with project scope!")

	t.Log("=== ✅ Verify Evidence for Artifact with Project Test Completed Successfully! ===")
}

// RunVerifyEvidenceForBuild tests verifying evidence for a build
func (r *EvidenceE2ETestsRunner) RunVerifyEvidenceForBuild(t *testing.T) {
	t.Log("=== Verify Evidence - Build Test ===")

	// Verify shared key pair is available
	if SharedPrivateKeyPath == "" || SharedPublicKeyPath == "" {
		t.Errorf("Shared key pair not initialized. Ensure PrepareTestsData() was called.")
		return
	}
	t.Logf("Using shared key pair: %s (alias: %s)", SharedPrivateKeyPath, SharedKeyAlias)

	tempDir := t.TempDir()

	// Step 1: Create build info (Admin)
	t.Log("Step 1: Creating build info...")
	buildName, buildNumber := utils.CreateTestBuildInfo(t, r.ServicesManager, "")
	t.Logf("✓ Build info created: %s/%s", buildName, buildNumber)

	// Step 2: Create predicate (Admin)
	t.Log("Step 2: Creating predicate...")
	predicate := map[string]interface{}{
		"buildType":   "verify-build-test",
		"timestamp":   time.Now().Unix(),
		"environment": "e2e-verify-test",
		"buildName":   buildName,
		"buildNumber": buildNumber,
	}
	predicateBytes, err := json.MarshalIndent(predicate, "", "  ")
	require.NoError(t, err)
	predicatePath := filepath.Join(tempDir, "predicate.json")
	err = os.WriteFile(predicatePath, predicateBytes, 0644)
	require.NoError(t, err)
	t.Log("✓ Predicate created")

	// Step 3: Create evidence for build using Admin CLI
	t.Log("Step 3: Creating evidence for build using Admin CLI...")
	createOutput := r.EvidenceAdminCLI.RunCliCmdWithOutput(t,
		"create",
		"--predicate", predicatePath,
		"--predicate-type", "https://slsa.dev/provenance/v1",
		"--build-name", buildName,
		"--build-number", buildNumber,
		"--key", SharedPrivateKeyPath,
		"--key-alias", SharedKeyAlias,
	)
	t.Logf("Evidence creation output: %s", createOutput)
	require.NotContains(t, createOutput, "Error", "Evidence creation should not error")
	t.Log("✓ Evidence created successfully")

	// Step 4: Verify evidence for build using User CLI (THIS IS WHAT WE'RE TESTING)
	t.Log("Step 4: Verifying evidence for build using User CLI...")
	verifyOutput := r.EvidenceUserCLI.RunCliCmdWithOutput(t,
		"verify",
		"--build-name", buildName,
		"--build-number", buildNumber,
		"--public-keys", SharedPublicKeyPath,
	)
	t.Logf("Verification output: %s", verifyOutput)
	require.Contains(t, verifyOutput, buildName, "Evidence should contain build name")
	require.Contains(t, verifyOutput, "Verification passed", "Verification should pass")
	t.Log("✅ User successfully verified build evidence!")

	t.Log("=== ✅ Verify Evidence for Build Test Completed Successfully! ===")
}

// RunVerifyEvidenceForPackage tests verifying evidence for a package
func (r *EvidenceE2ETestsRunner) RunVerifyEvidenceForPackage(t *testing.T) {
	t.Log("=== Verify Evidence - Package Test ===")

	// Verify shared key pair is available
	if SharedPrivateKeyPath == "" || SharedPublicKeyPath == "" {
		t.Errorf("Shared key pair not initialized. Ensure PrepareTestsData() was called.")
		return
	}
	t.Logf("Using shared key pair: %s (alias: %s)", SharedPrivateKeyPath, SharedKeyAlias)

	tempDir := t.TempDir()

	// Step 1: Create package (Admin)
	t.Log("Step 1: Creating package...")
	packageName, packageVersion, repoName := utils.CreateTestPackage(t, r.ServicesManager, "generic")
	t.Logf("✓ Package created: %s@%s in %s", packageName, packageVersion, repoName)

	// Step 2: Create predicate (Admin)
	t.Log("Step 2: Creating predicate...")
	predicate := map[string]interface{}{
		"buildType":      "verify-package-test",
		"timestamp":      time.Now().Unix(),
		"environment":    "e2e-verify-test",
		"packageName":    packageName,
		"packageVersion": packageVersion,
		"repository":     repoName,
	}
	predicateBytes, err := json.MarshalIndent(predicate, "", "  ")
	require.NoError(t, err)
	predicatePath := filepath.Join(tempDir, "predicate.json")
	err = os.WriteFile(predicatePath, predicateBytes, 0644)
	require.NoError(t, err)
	t.Log("✓ Predicate created")

	// Step 3: Create evidence for package using Admin CLI
	t.Log("Step 3: Creating evidence for package using Admin CLI...")
	createOutput := r.EvidenceAdminCLI.RunCliCmdWithOutput(t,
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
	t.Log("✓ Evidence created successfully")

	// Step 4: Verify evidence for package using User CLI (THIS IS WHAT WE'RE TESTING)
	t.Log("Step 4: Verifying evidence for package using User CLI...")
	verifyOutput := r.EvidenceUserCLI.RunCliCmdWithOutput(t,
		"verify",
		"--package-name", packageName,
		"--package-version", packageVersion,
		"--package-repo-name", repoName,
		"--public-keys", SharedPublicKeyPath,
	)
	t.Logf("Verification output: %s", verifyOutput)
	require.Contains(t, verifyOutput, packageName, "Evidence should contain package name")
	require.Contains(t, verifyOutput, "Verification passed", "Verification should pass")
	t.Log("✅ User successfully verified package evidence!")

	t.Log("=== ✅ Verify Evidence for Package Test Completed Successfully! ===")
}

// RunVerifyEvidenceWithUseArtifactoryKeys tests verification using Artifactory keys
func (r *EvidenceE2ETestsRunner) RunVerifyEvidenceWithUseArtifactoryKeys(t *testing.T) {
	t.Log("=== Verify Evidence - With Use Artifactory Keys Test ===")

	// Verify shared key pair is available (already uploaded to Artifactory)
	if SharedPrivateKeyPath == "" || SharedPublicKeyPath == "" {
		t.Errorf("Shared key pair not initialized. Ensure PrepareTestsData() was called.")
		return
	}
	t.Logf("Using shared key pair: %s (alias: %s)", SharedPrivateKeyPath, SharedKeyAlias)

	tempDir := t.TempDir()

	// Step 1: Create repository and upload artifact (Admin)
	t.Log("Step 1: Creating repository and uploading artifact...")
	repoName := utils.CreateTestRepository(t, r.ServicesManager, "generic")
	artifactPath := utils.CreateTestArtifact(t, "Test for use Artifactory keys")
	artifactFileName := filepath.Base(artifactPath)
	repoPath := fmt.Sprintf("%s/%s", repoName, artifactFileName)
	err := utils.UploadArtifact(r.ServicesManager, artifactPath, repoPath)
	require.NoError(t, err)
	t.Logf("✓ Artifact uploaded: %s", repoPath)

	// Step 2: Create predicate (Admin)
	t.Log("Step 2: Creating predicate...")
	predicate := map[string]interface{}{
		"buildType":   "verify-artifactory-keys-test",
		"timestamp":   time.Now().Unix(),
		"environment": "e2e-verify-test",
		"description": "Testing verification with Artifactory Trusted Keys Store",
	}
	predicateBytes, err := json.MarshalIndent(predicate, "", "  ")
	require.NoError(t, err)
	predicatePath := filepath.Join(tempDir, "predicate.json")
	err = os.WriteFile(predicatePath, predicateBytes, 0644)
	require.NoError(t, err)
	t.Log("✓ Predicate created")

	// Step 3: Create evidence using Admin CLI (key already uploaded)
	t.Log("Step 3: Creating evidence using Admin CLI...")
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

	// Step 4: Verify using Artifactory keys with User CLI
	t.Log("Step 4: Verifying evidence using --use-artifactory-keys with User CLI...")

	verifyOutput := r.EvidenceUserCLI.RunCliCmdWithOutput(t,
		"verify",
		"--subject-repo-path", repoPath,
		"--use-artifactory-keys",
	)
	t.Logf("Verification output: %s", verifyOutput)
	require.Contains(t, verifyOutput, repoPath, "Evidence should be verified")
	require.Contains(t, verifyOutput, "Verification passed", "Verification should pass")
	t.Log("✅ User successfully verified evidence using Artifactory Trusted Keys Store!")

	t.Log("=== ✅ Verify Evidence with Use Artifactory Keys Test Completed Successfully! ===")
}

// RunVerifyEvidenceForReleaseBundle tests verifying evidence for a Release Bundle v2
func (r *EvidenceE2ETestsRunner) RunVerifyEvidenceForReleaseBundle(t *testing.T) {
	t.Log("=== Verify Evidence - Release Bundle Test ===")

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
		"buildType":            "verify-release-bundle-test",
		"timestamp":            time.Now().Unix(),
		"environment":          "e2e-verify-test",
		"releaseBundleName":    releaseBundleName,
		"releaseBundleVersion": releaseBundleVersion,
		"description":          "Testing verification of release bundle evidence",
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

	// Step 4: Verify evidence for release bundle using User CLI
	t.Log("Step 4: Verifying evidence for release bundle using User CLI...")
	verifyOutput := r.EvidenceUserCLI.RunCliCmdWithOutput(t,
		"verify",
		"--release-bundle", releaseBundleName,
		"--release-bundle-version", releaseBundleVersion,
		"--public-keys", SharedPublicKeyPath,
	)
	t.Logf("Verification output: %s", verifyOutput)
	require.Contains(t, verifyOutput, releaseBundleName, "Verification output should contain release bundle name")
	require.Contains(t, verifyOutput, "Verification passed", "Verification should pass")
	t.Log("✅ User successfully verified release bundle evidence!")

	t.Log("=== ✅ Verify Evidence for Release Bundle Test Completed Successfully! ===")
}

// RunVerifyEvidenceForBuildWithProject tests verifying evidence for a build with project
func (r *EvidenceE2ETestsRunner) RunVerifyEvidenceForBuildWithProject(t *testing.T) {
	t.Skip("Skipping project-based test - GraphQL authentication context issue pending fix")
	t.Log("=== Verify Evidence - Build With Project Test ===")

	// Verify shared key pair is available
	if SharedPrivateKeyPath == "" || SharedPublicKeyPath == "" {
		t.Errorf("Shared key pair not initialized. Ensure PrepareTestsData() was called.")
		return
	}
	t.Logf("Using shared key pair: %s (alias: %s)", SharedPrivateKeyPath, SharedKeyAlias)
	t.Logf("Using project: %s", e2e.ProjectKey)

	tempDir := t.TempDir()

	// Step 1: Create build info with project (Admin)
	t.Log("Step 1: Creating build info with project...")
	buildName, buildNumber := utils.CreateTestBuildInfo(t, r.ServicesManager, e2e.ProjectKey)
	t.Logf("✓ Build info created: %s/%s (project: %s)", buildName, buildNumber, e2e.ProjectKey)

	// Step 2: Create predicate (Admin)
	t.Log("Step 2: Creating predicate...")
	predicate := map[string]interface{}{
		"buildType":   "verify-build-project-test",
		"timestamp":   time.Now().Unix(),
		"environment": "e2e-verify-test",
		"buildName":   buildName,
		"buildNumber": buildNumber,
		"project":     e2e.ProjectKey,
	}
	predicateBytes, err := json.MarshalIndent(predicate, "", "  ")
	require.NoError(t, err)
	predicatePath := filepath.Join(tempDir, "predicate.json")
	err = os.WriteFile(predicatePath, predicateBytes, 0644)
	require.NoError(t, err)
	t.Log("✓ Predicate created")

	// Step 3: Create evidence for build with project using Admin CLI
	t.Log("Step 3: Creating evidence for build with project using Admin CLI...")
	createOutput := r.EvidenceAdminCLI.RunCliCmdWithOutput(t,
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
	t.Log("✓ Evidence created successfully")

	// Step 4: Verify evidence for build with project using Project CLI (THIS IS WHAT WE'RE TESTING)
	t.Log("Step 4: Verifying evidence for build with project using Project CLI...")
	verifyOutput := r.EvidenceProjectCLI.RunCliCmdWithOutput(t,
		"verify",
		"--build-name", buildName,
		"--build-number", buildNumber,
		"--project", e2e.ProjectKey,
		"--public-keys", SharedPublicKeyPath,
	)
	t.Logf("Verification output: %s", verifyOutput)
	require.Contains(t, verifyOutput, buildName, "Evidence should contain build name")
	require.Contains(t, verifyOutput, "Verification passed", "Verification should pass")
	t.Log("✅ Project user successfully verified build evidence with project!")

	t.Log("=== ✅ Verify Evidence for Build with Project Test Completed Successfully! ===")
}

// RunVerifyEvidenceWithJsonFormat tests verification with JSON format output
func (r *EvidenceE2ETestsRunner) RunVerifyEvidenceWithJsonFormat(t *testing.T) {
	t.Log("=== Verify Evidence - With JSON Format Test ===")

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
	artifactPath := utils.CreateTestArtifact(t, "Test for JSON format verification")
	artifactFileName := filepath.Base(artifactPath)
	repoPath := fmt.Sprintf("%s/%s", repoName, artifactFileName)
	err := utils.UploadArtifact(r.ServicesManager, artifactPath, repoPath)
	require.NoError(t, err)
	t.Logf("✓ Artifact uploaded: %s", repoPath)

	// Step 2: Create predicate (Admin)
	t.Log("Step 2: Creating predicate...")
	predicate := map[string]interface{}{
		"buildType":   "verify-json-format-test",
		"timestamp":   time.Now().Unix(),
		"environment": "e2e-verify-test",
		"description": "Testing JSON format output for verification",
	}
	predicateBytes, err := json.MarshalIndent(predicate, "", "  ")
	require.NoError(t, err)
	predicatePath := filepath.Join(tempDir, "predicate.json")
	err = os.WriteFile(predicatePath, predicateBytes, 0644)
	require.NoError(t, err)
	t.Log("✓ Predicate created")

	// Step 3: Create evidence using Admin CLI
	t.Log("Step 3: Creating evidence using Admin CLI...")
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

	// Step 4: Verify evidence with JSON format using User CLI (THIS IS WHAT WE'RE TESTING)
	t.Log("Step 4: Verifying evidence with JSON format using User CLI...")
	verifyOutput := r.EvidenceUserCLI.RunCliCmdWithOutput(t,
		"verify",
		"--subject-repo-path", repoPath,
		"--public-keys", SharedPublicKeyPath,
		"--format", "json",
	)
	t.Logf("Verification output: %s", verifyOutput)

	// Verify output is valid JSON
	var jsonData map[string]interface{}
	err = json.Unmarshal([]byte(verifyOutput), &jsonData)
	require.NoError(t, err, "Output should be valid JSON")

	// Check for success in JSON format
	status, ok := jsonData["overallVerificationStatus"]
	require.True(t, ok, "JSON should contain overallVerificationStatus field")
	require.Equal(t, "success", status, "Overall verification status should be success")
	t.Log("✅ User successfully verified evidence with JSON format output!")

	t.Log("=== ✅ Verify Evidence with JSON Format Test Completed Successfully! ===")
}
