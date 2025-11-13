package utils

import (
	"fmt"
	"testing"
	"time"

	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/stretchr/testify/require"
)

// CreateTestPackage creates a test package in Artifactory
// Returns package name, version, and repo name
func CreateTestPackage(t *testing.T, servicesManager artifactory.ArtifactoryServicesManager, packageType string) (string, string, string) {
	t.Helper()

	packageName := fmt.Sprintf("test-package-%d", time.Now().UnixNano())
	packageVersion := fmt.Sprintf("1.0.%d", time.Now().Unix())
	repoName := CreateTestRepository(t, servicesManager, packageType)

	t.Logf("Creating test package: %s/%s in repo %s", packageName, packageVersion, repoName)

	// Create and upload a package artifact
	artifactContent := fmt.Sprintf("Package: %s\nVersion: %s\nTimestamp: %d", packageName, packageVersion, time.Now().Unix())
	artifactPath := CreateTestArtifact(t, artifactContent)

	// Upload to repo path: repo-name/package-name/package-version/file.ext
	packagePath := fmt.Sprintf("%s/%s/%s/%s-%s.txt", repoName, packageName, packageVersion, packageName, packageVersion)
	err := UploadArtifact(servicesManager, artifactPath, packagePath)
	require.NoError(t, err, "Failed to upload package artifact")

	t.Logf("✓ Package created: %s/%s in repo %s", packageName, packageVersion, repoName)

	return packageName, packageVersion, repoName
}
