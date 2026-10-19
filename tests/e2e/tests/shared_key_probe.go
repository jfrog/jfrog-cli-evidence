package tests

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli-evidence/evidence/create"
	"github.com/jfrog/jfrog-cli-evidence/tests/e2e/utils"
	"github.com/jfrog/jfrog-client-go/artifactory/services"
	clientutils "github.com/jfrog/jfrog-client-go/utils"
)

// The Evidence service does not see a newly uploaded trusted key for a few seconds, and evidence signed
// with it is rejected with "Key alias ... is not found in Artifactory trusted keys". Artifactory already
// lists the key by then, so the shared key is probed by creating evidence with it. One CI run saw a key
// work and then fail again a few seconds later, so several probes in a row must succeed.
const (
	sharedKeyProbeTimeout   = 2 * time.Minute
	sharedKeyProbeInterval  = 2 * time.Second
	sharedKeyProbeSuccesses = 3
)

// waitForSharedKeyInEvidenceService creates evidence signed with the shared key on a throwaway artifact
// until that succeeds sharedKeyProbeSuccesses times in a row. The evidence commands exit the process on
// failure when run through the CLI, so the probe runs the create command directly.
func (r *EvidenceE2ETestsRunner) waitForSharedKeyInEvidenceService() error {
	serverDetails, err := r.evidenceServerDetails()
	if err != nil {
		fmt.Printf("Warning: skipping the shared key probe: %v\n", err)
		return nil
	}

	repoKey := fmt.Sprintf("test-generic-key-probe-%d", time.Now().UnixNano())
	err = r.ServicesManager.CreateLocalRepository().Generic(services.GenericLocalRepositoryParams{
		LocalRepositoryBaseParams: services.LocalRepositoryBaseParams{
			RepositoryBaseParams: services.RepositoryBaseParams{
				Key:         repoKey,
				PackageType: "generic",
				Description: "Temporary repository for the E2E signing key probe - auto-cleanup",
				Rclass:      "local",
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create key probe repository %s: %w", repoKey, err)
	}
	defer func() {
		if err := r.ServicesManager.DeleteRepository(repoKey); err != nil {
			fmt.Printf("Warning: failed to delete key probe repository %s: %v\n", repoKey, err)
		}
	}()

	artifactPath := filepath.Join(SharedKeyDir, "key-probe.txt")
	if err := os.WriteFile(artifactPath, []byte("E2E signing key probe"), 0600); err != nil {
		return fmt.Errorf("failed to write key probe artifact: %w", err)
	}
	subjectRepoPath := repoKey + "/key-probe.txt"
	if err := utils.UploadArtifact(r.ServicesManager, artifactPath, subjectRepoPath); err != nil {
		return fmt.Errorf("failed to upload key probe artifact: %w", err)
	}
	predicatePath := filepath.Join(SharedKeyDir, "key-probe-predicate.json")
	if err := os.WriteFile(predicatePath, []byte(`{"probe": "e2e shared signing key"}`), 0600); err != nil {
		return fmt.Errorf("failed to write key probe predicate: %w", err)
	}

	fmt.Printf("Probing the Evidence service with shared key %s...\n", SharedKeyAlias)
	deadline := time.Now().Add(sharedKeyProbeTimeout)
	successes := 0
	var lastErr error
	for attempt := 1; ; attempt++ {
		createCmd := create.NewCreateEvidenceCustom(serverDetails, predicatePath, "https://slsa.dev/provenance/v1", "",
			SharedPrivateKeyPath, SharedKeyAlias, subjectRepoPath, "", "", "", "", "", "", "")
		if err := createCmd.Run(); err != nil {
			successes = 0
			lastErr = err
			fmt.Printf("Key probe %d failed: %v\n", attempt, err)
		} else {
			successes++
			if successes == sharedKeyProbeSuccesses {
				fmt.Printf("✓ Evidence service accepted shared key %s (%d probes)\n", SharedKeyAlias, attempt)
				return nil
			}
		}
		if time.Now().Add(sharedKeyProbeInterval).After(deadline) {
			if lastErr == nil {
				return fmt.Errorf("shared key %s did not pass %d probes in a row within %s", SharedKeyAlias, sharedKeyProbeSuccesses, sharedKeyProbeTimeout)
			}
			return fmt.Errorf("shared key %s did not pass %d probes in a row within %s, last error: %w", SharedKeyAlias, sharedKeyProbeSuccesses, sharedKeyProbeTimeout, lastErr)
		}
		time.Sleep(sharedKeyProbeInterval)
	}
}

// evidenceServerDetails returns platform server details for the evidence commands, derived from the
// Artifactory services manager the same way the CLI derives them from --url (platformToEvidenceUrls).
func (r *EvidenceE2ETestsRunner) evidenceServerDetails() (*config.ServerDetails, error) {
	rtDetails := r.ServicesManager.GetConfig().GetServiceDetails()
	if rtDetails.GetAccessToken() == "" {
		return nil, errors.New("the Artifactory services manager has no access token")
	}
	artifactoryUrl := clientutils.AddTrailingSlashIfNeeded(rtDetails.GetUrl())
	if !strings.HasSuffix(artifactoryUrl, "/artifactory/") {
		return nil, fmt.Errorf("cannot derive the platform URL from Artifactory URL %s", artifactoryUrl)
	}
	platformUrl := strings.TrimSuffix(artifactoryUrl, "artifactory/")
	return &config.ServerDetails{
		Url:            platformUrl,
		ArtifactoryUrl: artifactoryUrl,
		EvidenceUrl:    platformUrl + "evidence/",
		MetadataUrl:    platformUrl + "metadata/",
		OnemodelUrl:    platformUrl + "onemodel/",
		LifecycleUrl:   platformUrl + "lifecycle/",
		ApptrustUrl:    platformUrl + "apptrust/",
		AccessToken:    rtDetails.GetAccessToken(),
	}, nil
}
