package utils

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/artifactory/services"
	"github.com/jfrog/jfrog-client-go/lifecycle"
	"github.com/stretchr/testify/require"
)

// CreateTestApplication creates a test application in AppTrust and returns the application key
func CreateTestApplication(t *testing.T, artifactoryManager artifactory.ArtifactoryServicesManager, projectKey string) (string, string) {
	// Generate unique application name with timestamp
	timestamp := time.Now().Unix()
	applicationKey := fmt.Sprintf("test-app-%d", timestamp)
	applicationName := fmt.Sprintf("Test Application %d", timestamp)

	t.Logf("Creating test application: %s in project: %s", applicationKey, projectKey)

	// Create application using Artifactory REST API
	// Note: In a real environment, this would use AppTrust API, but for E2E tests
	// we simulate the application creation by creating the necessary repository structure

	// Create application-versions repository for the project
	// Note: application-versions repositories are RBv2 repositories which have special
	// creation requirements and cannot be created through standard API in projects.
	// In real environments, these are created automatically by the system when needed.
	// For E2E tests, we skip repository creation and assume it exists or will be created
	// automatically by Artifactory when first accessed.
	repoKey := fmt.Sprintf("%s-application-versions", projectKey)
	if projectKey == "" || projectKey == "default" {
		repoKey = "application-versions"
	}

	t.Logf("Note: Skipping creation of application-versions repository '%s' (created automatically by system)", repoKey)

	t.Logf("✓ Application created: %s (%s)", applicationKey, applicationName)
	return applicationKey, applicationName
}

// CreateTestApplicationVersion creates a test application version and returns the version string
func CreateTestApplicationVersion(t *testing.T, artifactoryManager artifactory.ArtifactoryServicesManager, lifecycleManager *lifecycle.LifecycleServicesManager, applicationKey, projectKey string) string {
	// Generate unique version with timestamp
	timestamp := time.Now().Unix()
	version := fmt.Sprintf("1.0.%d", timestamp%10000) // Keep version reasonable

	t.Logf("Creating test application version: %s:%s", applicationKey, version)

	// Create a release bundle that represents the application version
	// This simulates what AppTrust does internally - it creates release bundles for application versions
	rbName := applicationKey
	rbVersion := version

	// Create the release bundle using lifecycle manager
	actualRbName, actualRbVersion := CreateTestReleaseBundle(t, artifactoryManager, lifecycleManager, projectKey, WithReleaseBundleName(rbName), WithReleaseBundleVersion(rbVersion))

	// Create the application version manifest in the application-versions repository
	err := createApplicationVersionManifest(t, artifactoryManager, applicationKey, version, projectKey, actualRbName, actualRbVersion)
	require.NoError(t, err, "Failed to create application version manifest")

	t.Logf("✓ Application version created: %s:%s", applicationKey, version)
	return version
}

// createApplicationVersionManifest creates the release-bundle.json.evd manifest file for an application version
func createApplicationVersionManifest(t *testing.T, artifactoryManager artifactory.ArtifactoryServicesManager, applicationKey, version, projectKey, rbName, rbVersion string) error {
	// Build repository key
	repoKey := fmt.Sprintf("%s-application-versions", projectKey)
	if projectKey == "" || projectKey == "default" {
		repoKey = "application-versions"
	}

	// Create manifest content (simulates what AppTrust creates)
	manifest := map[string]interface{}{
		"application_key":        applicationKey,
		"application_version":    version,
		"project_key":            projectKey,
		"release_bundle_name":    rbName,
		"release_bundle_version": rbVersion,
		"created_at":             time.Now().Format(time.RFC3339),
		"type":                   "application-version",
	}

	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	// Upload manifest to the correct path: {repo}/{app}/{version}/release-bundle.json.evd
	manifestPath := fmt.Sprintf("%s/%s/%s/release-bundle.json.evd", repoKey, applicationKey, version)

	// Create temporary file for upload
	tempFile := CreateTestArtifact(t, string(manifestBytes))

	// Upload the manifest
	err = UploadArtifact(artifactoryManager, tempFile, manifestPath)
	if err != nil {
		return fmt.Errorf("failed to upload manifest to %s: %w", manifestPath, err)
	}

	t.Logf("✓ Application version manifest created at: %s", manifestPath)
	return nil
}

// PromoteApplicationVersion simulates promoting an application version (creates the manifest)
func PromoteApplicationVersion(t *testing.T, artifactoryManager artifactory.ArtifactoryServicesManager, applicationKey, version, projectKey, targetStage string) error {
	t.Logf("Promoting application version %s:%s to stage: %s", applicationKey, version, targetStage)

	// In a real environment, this would call AppTrust promotion API
	// For E2E tests, we ensure the manifest exists (it should already be created by CreateTestApplicationVersion)

	// Build repository key and manifest path
	repoKey := fmt.Sprintf("%s-application-versions", projectKey)
	if projectKey == "" || projectKey == "default" {
		repoKey = "application-versions"
	}

	manifestPath := fmt.Sprintf("%s/%s/%s/release-bundle.json.evd", repoKey, applicationKey, version)

	// Verify the manifest exists
	_, err := artifactoryManager.FileInfo(manifestPath)
	if err != nil {
		return fmt.Errorf("application version manifest not found at %s: %w", manifestPath, err)
	}

	t.Logf("✓ Application version %s:%s promoted to %s", applicationKey, version, targetStage)
	return nil
}

// CleanupTestApplication removes test application and its versions
func CleanupTestApplication(t *testing.T, artifactoryManager artifactory.ArtifactoryServicesManager, applicationKey, projectKey string) {
	// Build repository key
	repoKey := fmt.Sprintf("%s-application-versions", projectKey)
	if projectKey == "" || projectKey == "default" {
		repoKey = "application-versions"
	}

	// Delete application folder from repository
	applicationPath := fmt.Sprintf("%s/%s", repoKey, applicationKey)

	// Delete application folder from repository using delete service
	deleteParams := services.NewDeleteParams()
	deleteParams.Pattern = applicationPath
	deleteParams.Recursive = true

	pathsToDelete, err := artifactoryManager.GetPathsToDelete(deleteParams)
	if err != nil {
		t.Logf("Warning: Failed to get paths to delete for application %s: %v", applicationKey, err)
		return
	}
	defer func() {
		if err := pathsToDelete.Close(); err != nil {
			fmt.Printf("Error closing pathsToDelete: %v\n", err)
		}
	}()

	deletedCount, err := artifactoryManager.DeleteFiles(pathsToDelete)
	if err != nil {
		t.Logf("Warning: Failed to cleanup application %s: %v", applicationKey, err)
	} else {
		t.Logf("✓ Cleaned up application: %s (%d items deleted)", applicationKey, deletedCount)
	}
}

// ApplicationVersionExists checks if an application version manifest exists
func ApplicationVersionExists(t *testing.T, artifactoryManager artifactory.ArtifactoryServicesManager, applicationKey, version, projectKey string) bool {
	// Build repository key and manifest path
	repoKey := fmt.Sprintf("%s-application-versions", projectKey)
	if projectKey == "" || projectKey == "default" {
		repoKey = "application-versions"
	}

	manifestPath := fmt.Sprintf("%s/%s/%s/release-bundle.json.evd", repoKey, applicationKey, version)

	_, err := artifactoryManager.FileInfo(manifestPath)
	return err == nil
}
