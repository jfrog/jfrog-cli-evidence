package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

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

	repositoryWaitAttempts = 10
	repositoryWaitInterval = time.Second
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
//
// Entity repo keys are fixed, so two suites running against the same platform compete over
// them. Artifactory rejects the loser of a concurrent creation with 409 Conflict, which is a
// success for our purposes as long as the repository ends up existing.
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
	if err = servicesManager.CreateLocalRepository().Generic(params); err != nil {
		t.Logf("Creation of entity repository %s failed, checking whether another run created it: %v", repoKey, err)
		require.Truef(t, waitForRepository(t, servicesManager, repoKey),
			"Failed to create entity repository %s and it does not exist: %v", repoKey, err)
		t.Logf("✓ Entity repository exists, created concurrently: %s", repoKey)
		return
	}
	t.Logf("✓ Entity repository created: %s", repoKey)
}

// waitForRepository reports whether the repository exists, giving a concurrent creation that
// is still in flight a chance to complete.
func waitForRepository(t *testing.T, servicesManager artifactory.ArtifactoryServicesManager, repoKey string) bool {
	t.Helper()

	for attempt := 0; attempt < repositoryWaitAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(repositoryWaitInterval)
		}
		exists, err := servicesManager.IsRepoExists(repoKey)
		if err != nil {
			t.Logf("Warning: Failed to check whether repository %s exists: %v", repoKey, err)
			continue
		}
		if exists {
			return true
		}
	}
	return false
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
