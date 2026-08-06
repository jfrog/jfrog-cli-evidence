package client

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	evidenceutils "github.com/jfrog/jfrog-cli-evidence/evidence/utils"
	clientutils "github.com/jfrog/jfrog-client-go/utils"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"github.com/jfrog/jfrog-client-go/utils/log"
)

const evidenceVersionAPI = "api/v1/system/version"

// GetVersion returns the Evidence service version from GET /api/v1/system/version.
func (c *EvidenceClient) GetVersion() (string, error) {
	httpClientDetails := c.details.CreateHttpClientDetails()
	requestURL := c.details.GetUrl() + evidenceVersionAPI
	resp, body, _, err := c.httpClient.SendGet(requestURL, true, &httpClientDetails)
	if err != nil {
		return "", err
	}
	if err = errorutils.CheckResponseStatusWithBody(resp, body, http.StatusOK); err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}

// EnsureEntityAPISupported fails when the Evidence service is a release older than
// the minimum that exposes /entity APIs. Non-release / development version strings
// are allowed through so local and snapshot builds are not blocked.
func EnsureEntityAPISupported(serverDetails *config.ServerDetails) error {
	client, err := NewEvidenceClient(serverDetails)
	if err != nil {
		return fmt.Errorf("entity evidence requires JFrog Evidence version %s or higher; failed to create Evidence client: %w",
			evidenceutils.MinEvidenceVersionForEntityAPI, err)
	}
	version, err := client.GetVersion()
	if err != nil {
		return fmt.Errorf("entity evidence requires JFrog Evidence version %s or higher; failed to determine Evidence version: %w",
			evidenceutils.MinEvidenceVersionForEntityAPI, err)
	}
	if evidenceutils.IsNonReleaseEvidenceVersion(version) {
		log.Debug("Skipping Evidence entity API version check for non-release version:", version)
		return nil
	}
	if err = clientutils.ValidateMinimumVersion("JFrog Evidence", version, evidenceutils.MinEvidenceVersionForEntityAPI); err != nil {
		return err
	}
	return nil
}
