package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/artifactory/services"
	"github.com/stretchr/testify/require"
)

const (
	// entityRoot is the Artifactory path prefix under which entity subjects are stored.
	entityRoot = ".entities"
	// emptySubjectSha256 is the sha256 of an empty payload used as the subject checksum
	// for entity evidence storage paths.
	emptySubjectSha256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
)

// DefaultEntityRepoKey returns the default-scope entity repository key: {entityType}-entity.
func DefaultEntityRepoKey(entityType string) string {
	return entityType + "-entity"
}

// ProjectEntityRepoKey returns the project-scope entity repository key:
// {projectKey}-{entityType}-entity.
func ProjectEntityRepoKey(projectKey, entityType string) string {
	return projectKey + "-" + entityType + "-entity"
}

// EnsureEntityRepository creates a generic local repository for entity evidence if it does
// not already exist. Fixed-name entity repos are left in place across runs; they are not
// cleaned up.
func EnsureEntityRepository(t *testing.T, servicesManager artifactory.ArtifactoryServicesManager, repoKey, projectKey string) {
	t.Helper()

	exists, err := servicesManager.IsRepoExists(repoKey)
	require.NoError(t, err, "Failed to check whether repository %s exists", repoKey)
	if exists {
		t.Logf("✓ Entity repository already exists: %s", repoKey)
		return
	}

	t.Logf("Creating entity repository: %s", repoKey)
	params := services.GenericLocalRepositoryParams{
		LocalRepositoryBaseParams: services.LocalRepositoryBaseParams{
			RepositoryBaseParams: services.RepositoryBaseParams{
				Key:         repoKey,
				PackageType: "generic",
				ProjectKey:  projectKey,
				Description: "Entity evidence repository for E2E tests",
				Rclass:      "local",
			},
		},
	}
	err = servicesManager.CreateLocalRepository().Generic(params)
	require.NoError(t, err, "Failed to create entity repository %s", repoKey)
	t.Logf("✓ Entity repository created: %s", repoKey)
}

// CleanupEntityEvidence deletes the Artifactory subject placeholder and evidence files
// created for the given entity. Warnings only — cleanup must not fail the test.
func CleanupEntityEvidence(t *testing.T, servicesManager artifactory.ArtifactoryServicesManager, repoKey, entityType, entityID string) {
	t.Helper()

	shard := entityID
	if len(entityID) >= 4 {
		shard = entityID[:4]
	}
	relativeSubjectPath := fmt.Sprintf("%s/%s/%s/%s", entityRoot, entityType, shard, entityID)
	subjectDigest := sha256Hex(relativeSubjectPath)
	evidenceDir := fmt.Sprintf("%s/.evidence/%s/%s", repoKey, subjectDigest, emptySubjectSha256)
	subjectPath := fmt.Sprintf("%s/%s", repoKey, relativeSubjectPath)

	t.Logf("Cleaning up entity evidence under %s and %s", subjectPath, evidenceDir)
	deleteArtifactoryPath(t, servicesManager, evidenceDir)
	deleteArtifactoryPath(t, servicesManager, subjectPath)
}

func deleteArtifactoryPath(t *testing.T, servicesManager artifactory.ArtifactoryServicesManager, pattern string) {
	t.Helper()

	deleteParams := services.NewDeleteParams()
	deleteParams.Pattern = pattern
	deleteParams.Recursive = true
	reader, err := servicesManager.GetPathsToDelete(deleteParams)
	if err != nil {
		t.Logf("Warning: Failed to plan cleanup for %s: %v", pattern, err)
		return
	}
	if reader == nil {
		return
	}
	defer func() {
		_ = reader.Close()
	}()
	if _, err = servicesManager.DeleteFiles(reader); err != nil {
		t.Logf("Warning: Failed to delete %s: %v", pattern, err)
		return
	}
	t.Logf("✓ Cleaned up: %s", pattern)
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
