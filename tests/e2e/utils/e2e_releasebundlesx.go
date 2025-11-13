package utils

import (
	"fmt"
	"testing"
	"time"

	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/lifecycle"
	"github.com/jfrog/jfrog-client-go/lifecycle/services"
)

// CreateTestReleaseBundle creates a test release bundle from a build
// Returns release bundle name and version
func CreateTestReleaseBundle(t *testing.T, servicesManager artifactory.ArtifactoryServicesManager, lifecycleManager *lifecycle.LifecycleServicesManager, project string) (string, string) {
	t.Helper()

	// Generate unique names
	releaseBundleName := fmt.Sprintf("test-rb-%d", time.Now().UnixNano())
	releaseBundleVersion := fmt.Sprintf("1.0.%d", time.Now().Unix())

	t.Logf("Creating test release bundle: %s/%s", releaseBundleName, releaseBundleVersion)

	// First, create a build to use as source for the release bundle
	buildName, buildNumber := CreateTestBuildInfo(t, servicesManager, project)

	// Wait for build info to propagate (it's stored in Artifactory and needs time to be indexed)
	t.Log("Waiting 2 seconds for build info to propagate...")
	time.Sleep(2 * time.Second)

	// Create release bundle from the build
	rbDetails := services.ReleaseBundleDetails{
		ReleaseBundleName:    releaseBundleName,
		ReleaseBundleVersion: releaseBundleVersion,
	}

	// Get the correct build repository name based on project
	// For default/empty project, it's "artifactory-build-info"
	buildRepoName := "artifactory-build-info"
	if project != "" && project != "default" {
		buildRepoName = fmt.Sprintf("%s-build-info", project)
	}
	t.Logf("Using build repository: %s", buildRepoName)

	// Build sources
	buildSources := []services.BuildSource{
		{
			BuildRepository:     buildRepoName,
			BuildName:           buildName,
			BuildNumber:         buildNumber,
			IncludeDependencies: false,
		},
	}

	t.Logf("Creating release bundle from build: %s/%s (repo: %s)", buildName, buildNumber, buildRepoName)

	sources := []services.RbSource{
		{
			SourceType: "builds",
			Builds:     buildSources,
		},
	}

	queryParams := services.CommonOptionalQueryParams{
		Async:      false, // Wait for completion
		ProjectKey: project,
	}

	// Create the release bundle (without signing key as it's optional in newer versions)
	t.Logf("Calling CreateReleaseBundlesFromMultipleSources with project: %s", project)
	_, err := lifecycleManager.CreateReleaseBundlesFromMultipleSources(rbDetails, queryParams, "", sources)
	if err != nil {
		t.Errorf("Failed to create release bundle: %v", err)
	}

	t.Logf("✓ Release bundle created: %s/%s", releaseBundleName, releaseBundleVersion)

	// Register cleanup - delete both release bundle and build
	t.Cleanup(func() {
		t.Logf("Cleaning up release bundle: %s/%s", releaseBundleName, releaseBundleVersion)
		deleteErr := lifecycleManager.DeleteReleaseBundleVersion(rbDetails, services.CommonOptionalQueryParams{
			Async:      false,
			ProjectKey: project,
		})
		if deleteErr != nil {
			t.Logf("Warning: Failed to delete release bundle: %v", deleteErr)
		} else {
			t.Logf("✓ Release bundle deleted: %s/%s", releaseBundleName, releaseBundleVersion)
		}

		// Also delete the build
		DeleteTestBuildInfo(t, buildName)
	})

	return releaseBundleName, releaseBundleVersion
}
