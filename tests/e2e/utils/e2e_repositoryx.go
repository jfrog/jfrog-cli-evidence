package utils

import (
	"fmt"
	"testing"
	"time"

	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/artifactory/services"
	"github.com/stretchr/testify/require"
)

type RepositoryOption func(*repositoryConfig)

type repositoryConfig struct {
	key string
}

func WithRepoKey(key string) RepositoryOption {
	return func(config *repositoryConfig) {
		config.key = key
	}
}

func CreateTestRepositoryWithProject(t *testing.T, servicesManager artifactory.ArtifactoryServicesManager, packageType, project string, opts ...RepositoryOption) string {
	t.Helper()

	config := &repositoryConfig{}
	for _, opt := range opts {
		opt(config)
	}

	repoName := config.key
	if repoName == "" {
		repoName = fmt.Sprintf("test-%s-%d", packageType, time.Now().UnixNano())
	}

	if project != "" {
		repoName = fmt.Sprintf("%s-%s", project, repoName)
	}

	t.Logf("Creating test repository: %s", repoName)

	params := services.GenericLocalRepositoryParams{
		LocalRepositoryBaseParams: services.LocalRepositoryBaseParams{
			RepositoryBaseParams: services.RepositoryBaseParams{
				Key:         repoName,
				PackageType: packageType,
				ProjectKey:  project,
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

// CreateTestRepositoryWithName is a convenience function that returns the repository name
func CreateTestRepositoryWithName(t *testing.T, servicesManager artifactory.ArtifactoryServicesManager, packageType string, opts ...RepositoryOption) string {
	t.Helper()
	return CreateTestRepositoryWithProject(t, servicesManager, packageType, "", opts...)
}

func CreateTestRepository(t *testing.T, servicesManager artifactory.ArtifactoryServicesManager, packageType string) string {
	return CreateTestRepositoryWithProject(t, servicesManager, packageType, "")
}
