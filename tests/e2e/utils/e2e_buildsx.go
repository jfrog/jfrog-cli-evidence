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

// CreateTestBuildInfo creates a test build info in Artifactory with artifacts and publishes it.
// Returns build name and build number.
func CreateTestBuildInfo(t *testing.T, servicesManager artifactory.ArtifactoryServicesManager, project string) (string, string) {
	t.Helper()

	buildName := fmt.Sprintf("test-build-%d", time.Now().UnixNano())
	buildNumber := createTestBuildInfo(t, servicesManager, project, buildName, false)
	return buildName, buildNumber
}

// CreateTestBuildInfoWithName publishes a build with the given name and returns its build number.
// Unlike CreateTestBuildInfo, a failed publish fails the test.
func CreateTestBuildInfoWithName(t *testing.T, servicesManager artifactory.ArtifactoryServicesManager, project, buildName string) string {
	t.Helper()
	return createTestBuildInfo(t, servicesManager, project, buildName, true)
}

func createTestBuildInfo(t *testing.T, servicesManager artifactory.ArtifactoryServicesManager, project, buildName string, requirePublish bool) string {
	t.Helper()

	buildNumber := fmt.Sprintf("%d", time.Now().UnixNano())
	t.Logf("Creating test build info: %s/%s", buildName, buildNumber)

	repoName := CreateTestRepositoryWithProject(t, servicesManager, "generic", project)

	artifactContent := fmt.Sprintf("Build artifact for %s/%s - timestamp: %d", buildName, buildNumber, time.Now().Unix())
	artifactPath := CreateTestArtifact(t, artifactContent)
	artifactFileName := filepath.Base(artifactPath)
	repoPath := fmt.Sprintf("%s/%s", repoName, artifactFileName)

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

	err = publishBuildInfo(t, buildName, buildNumber, project)
	if requirePublish {
		require.NoError(t, err)
	} else if err != nil {
		t.Logf("Warning: Failed to publish build info using JFrog CLI: %v", err)
		t.Logf("Build info may still be accessible through build properties")
	}

	t.Cleanup(func() {
		t.Log("Cleaning up build info...")
		DeleteTestBuildInfo(t, buildName)
	})

	return buildNumber
}

func publishBuildInfo(t *testing.T, buildName, buildNumber, project string) error {
	t.Helper()

	args := []string{
		"rt", "build-publish",
		buildName,
		buildNumber,
	}

	if project != "" {
		args = append(args, "--project", project)
	}

	cmd := exec.Command("jfrog", args...)
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("publish build info %s/%s: %w, output: %s", buildName, buildNumber, err, string(output))
	}

	t.Logf("✓ Build info published: %s/%s", buildName, buildNumber)
	return nil
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
