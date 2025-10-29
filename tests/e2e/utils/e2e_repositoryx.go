package utils

import (
	"fmt"
	"testing"
	"time"

	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/artifactory/services"
	"github.com/stretchr/testify/require"
)

func CreateTestRepository(t *testing.T, servicesManager artifactory.ArtifactoryServicesManager, packageType string) string {
	t.Helper()

	// Generate unique repository name using timestamp
	repoName := fmt.Sprintf("test-%s-%d", packageType, time.Now().UnixNano())

	t.Logf("Creating test repository: %s", repoName)

	params := services.GenericLocalRepositoryParams{
		LocalRepositoryBaseParams: services.LocalRepositoryBaseParams{
			RepositoryBaseParams: services.RepositoryBaseParams{
				Key:         repoName,
				PackageType: packageType,
				Description: "Temporary test repository - auto-cleanup",
				Rclass:      "local",
			},
		},
	}

	// Create the repository using services manager
	err := servicesManager.CreateLocalRepository().Generic(params)
	require.NoError(t, err, "Failed to create repository %s", repoName)

	t.Logf("✓ Repository created: %s", repoName)

	// Register cleanup to delete repository after test
	t.Cleanup(func() {
		t.Logf("Cleaning up repository: %s", repoName)
		err := servicesManager.DeleteRepository(repoName)
		if err != nil {
			t.Logf("Warning: Failed to delete repository %s: %v", repoName, err)
		} else {
			t.Logf("✓ Repository deleted: %s", repoName)
		}
	})

	return repoName
}
