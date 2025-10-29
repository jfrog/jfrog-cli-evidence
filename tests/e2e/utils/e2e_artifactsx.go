package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/artifactory/services"
	"github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	"github.com/stretchr/testify/require"
)

func CreateTestArtifact(t *testing.T, content string) string {
	t.Helper()

	tempDir := t.TempDir() // Automatically cleaned up
	artifactPath := filepath.Join(tempDir, "test-artifact.txt")

	err := os.WriteFile(artifactPath, []byte(content), 0644)
	require.NoError(t, err)

	t.Logf("✓ Test artifact created: %s", artifactPath)
	return artifactPath
}

// UploadArtifact uploads a file to Artifactory using the services manager
// sourcePath: local file path
// targetPath: full repository path (e.g., "repo-name/path/to/file.txt")
func UploadArtifact(servicesManager artifactory.ArtifactoryServicesManager, sourcePath, targetPath string) error {
	// Create upload parameters
	uploadParams := services.UploadParams{}
	uploadParams.CommonParams = &utils.CommonParams{
		Pattern: sourcePath,
		Target:  targetPath,
	}

	// Create upload service options (empty for basic upload)
	uploadServiceOptions := artifactory.UploadServiceOptions{}

	// Upload using services manager
	totalUploaded, totalFailed, err := servicesManager.UploadFiles(uploadServiceOptions, uploadParams)
	if err != nil {
		return fmt.Errorf("failed to upload file to %s: %w", targetPath, err)
	}

	if totalFailed > 0 {
		return fmt.Errorf("upload failed: %d files failed to upload", totalFailed)
	}

	if totalUploaded == 0 {
		return fmt.Errorf("no files were uploaded")
	}

	return nil
}
