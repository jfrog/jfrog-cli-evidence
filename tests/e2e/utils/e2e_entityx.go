package utils

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/artifactory/services"
	"github.com/stretchr/testify/require"
)

const (
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

type entityGetOutput struct {
	Result struct {
		Evidence []struct {
			Subject map[string]any `json:"subject"`
		} `json:"evidence"`
	} `json:"result"`
}

// CleanupEntityEvidenceFromGetOutput deletes the entity subjects reported by a successful
// entity get, identified by subject.fullPath. Artifactory and the Evidence service remove the
// evidence attached to a subject along with it. Warnings only — cleanup must not fail the test.
func CleanupEntityEvidenceFromGetOutput(t *testing.T, servicesManager artifactory.ArtifactoryServicesManager, getOutput string) {
	t.Helper()

	if getOutput == "" {
		t.Log("Warning: no get output available for entity cleanup")
		return
	}

	var parsed entityGetOutput
	if err := json.Unmarshal([]byte(getOutput), &parsed); err != nil {
		t.Logf("Warning: Failed to parse get output for entity cleanup: %v", err)
		return
	}
	if len(parsed.Result.Evidence) == 0 {
		t.Log("Warning: get output contained no evidence entries to clean up")
		return
	}

	seenSubjects := map[string]bool{}
	for _, entry := range parsed.Result.Evidence {
		subjectPath := subjectFullPath(entry.Subject)
		if subjectPath == "" || seenSubjects[subjectPath] {
			continue
		}
		seenSubjects[subjectPath] = true
		t.Logf("Cleaning up entity subject under %s", subjectPath)
		deleteArtifactoryPath(t, servicesManager, subjectPath)
	}
}

func subjectFullPath(subject map[string]any) string {
	if subject == nil {
		return ""
	}
	if fullPath, ok := subject["fullPath"].(string); ok {
		return fullPath
	}
	return ""
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
