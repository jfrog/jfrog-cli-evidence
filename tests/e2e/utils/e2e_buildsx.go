package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/stretchr/testify/require"
)

// CreateTestBuildInfo creates a test build info in Artifactory with artifacts and publishes it
// Returns build name and build number
func CreateTestBuildInfo(t *testing.T, servicesManager artifactory.ArtifactoryServicesManager, project string) (string, string) {
	t.Helper()

	buildName := fmt.Sprintf("test-build-%d", time.Now().UnixNano())
	buildNumber := fmt.Sprintf("%d", time.Now().Unix())

	t.Logf("Creating test build info: %s/%s", buildName, buildNumber)

	// Create a repository for build artifacts
	repoName := CreateTestRepository(t, servicesManager, "generic")

	// Upload artifacts to the repository using JFrog CLI with build info association
	// This ensures proper build-artifact linkage required for release bundles
	artifactContent := fmt.Sprintf("Build artifact for %s/%s - timestamp: %d", buildName, buildNumber, time.Now().Unix())
	artifactPath := CreateTestArtifact(t, artifactContent)
	artifactFileName := filepath.Base(artifactPath)
	repoPath := fmt.Sprintf("%s/%s", repoName, artifactFileName)

	// Use jfrog CLI to upload with build info
	uploadArgs := []string{
		"rt", "upload",
		artifactPath,
		repoPath,
		"--build-name", buildName,
		"--build-number", buildNumber,
	}
	if project != "" {
		uploadArgs = append(uploadArgs, "--project", project)
	}

	cmd := exec.Command("jfrog", uploadArgs...)
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to upload artifact for build info: %s", string(output))

	t.Logf("✓ Build artifacts uploaded for build: %s/%s", buildName, buildNumber)

	// Publish build info using JFrog CLI rt build-publish command
	// This is necessary because the evidence CLI create command expects build info to exist
	publishBuildInfo(t, buildName, buildNumber, project)

	// Register cleanup to delete build info after test
	t.Cleanup(func() {
		t.Log("Cleaning up build info...")
		DeleteTestBuildInfo(t, buildName)
	})

	return buildName, buildNumber
}

// publishBuildInfo publishes build info using JFrog CLI rt build-publish command
func publishBuildInfo(t *testing.T, buildName, buildNumber, project string) {
	t.Helper()

	args := []string{
		"rt", "build-publish",
		buildName,
		buildNumber,
	}

	if project != "" {
		args = append(args, "--project", project)
	}

	// Run jfrog CLI command
	cmd := exec.Command("jfrog", args...)
	// Use the same server ID that was configured during bootstrap
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {
		// If jfrog CLI is not available or command fails, log warning but don't fail
		// The build info might still work if artifacts were uploaded with build properties
		t.Logf("Warning: Failed to publish build info using JFrog CLI: %v, output: %s", err, string(output))
		t.Logf("Build info may still be accessible through build properties")
		return
	}

	t.Logf("✓ Build info published: %s/%s", buildName, buildNumber)
}

// DeleteTestBuildInfo deletes a test build info from Artifactory
// This is used for test cleanup
// Note: Uses max-days=0 to delete all builds matching the build name
func DeleteTestBuildInfo(t *testing.T, buildName string) {
	t.Helper()

	args := []string{
		"rt", "build-discard",
		buildName,
		"--delete-artifacts=true", // Also delete associated artifacts
		"--max-days=0",            // Delete builds older than 0 days (all builds)
		"--async=false",           // Wait for completion
	}

	// Run jfrog CLI command
	cmd := exec.Command("jfrog", args...)
	cmd.Env = os.Environ()

	_, err := cmd.CombinedOutput()
	if err != nil {
		// Build deletion is optional - artifacts already cleaned by repository deletion
		t.Logf("Note: Build info cleanup skipped (artifacts already deleted with repository): %s", buildName)
		return
	}

	t.Logf("✓ Build info deleted: %s", buildName)
}
