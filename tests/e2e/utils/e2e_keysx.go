package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jfrog/jfrog-client-go/artifactory"
	clientutils "github.com/jfrog/jfrog-client-go/utils"
)

// TrustedKey represents a trusted key from Artifactory
type TrustedKey struct {
	Kid   string `json:"kid"`
	Alias string `json:"alias"`
}

// TrustedKeysResponse represents the response from listing trusted keys
type TrustedKeysResponse struct {
	Keys []TrustedKey `json:"keys"`
}

// Polling settings for WaitForTrustedKey.
const (
	trustedKeyPollInterval = time.Second
	// trustedKeyStableFor is how long a new key must stay listed before it is used. One CI run
	// found a new key and then lost it again a few seconds later, so a single sighting is not enough.
	trustedKeyStableFor = 5 * time.Second
)

// DeleteTrustedKey deletes a trusted key from Artifactory using the REST API
// keyAlias: the alias of the key to delete (e.g., "e2e-shared-key-1762361018")
func DeleteTrustedKey(servicesManager artifactory.ArtifactoryServicesManager, keyAlias string) error {
	client := servicesManager.Client()
	artifactoryDetails := servicesManager.GetConfig().GetServiceDetails()
	httpClientDetails := artifactoryDetails.CreateHttpClientDetails()
	baseURL := clientutils.AddTrailingSlashIfNeeded(artifactoryDetails.GetUrl())

	// Step 1: Find the kid for this alias
	kidToDelete, err := findTrustedKeyKid(servicesManager, keyAlias)
	if err != nil {
		return err
	}

	if kidToDelete == "" {
		// Key not found - might have been already deleted or never existed
		return nil
	}

	// Step 2: Delete the key using its kid
	deleteURL := baseURL + "api/security/keys/trusted/" + kidToDelete
	resp, body, err := client.SendDelete(deleteURL, nil, &httpClientDetails)
	if err != nil {
		return fmt.Errorf("failed to send delete request: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil // Key already deleted
	}

	if resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("insufficient permissions to delete trusted key (403 Forbidden)")
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// WaitForTrustedKey waits until Artifactory has listed the trusted key with keyAlias on every poll for
// trustedKeyStableFor. Evidence created right after the key is uploaded can fail with
// "signing key ... not found", because the key is not yet visible to the evidence service.
func WaitForTrustedKey(servicesManager artifactory.ArtifactoryServicesManager, keyAlias string, timeout time.Duration) error {
	err := waitUntilStable(func() (bool, error) {
		kid, err := findTrustedKeyKid(servicesManager, keyAlias)
		return kid != "", err
	}, trustedKeyStableFor, trustedKeyPollInterval, timeout)
	if err != nil {
		return fmt.Errorf("trusted key %s did not become available: %w", keyAlias, err)
	}
	return nil
}

// waitUntilStable calls check every interval until it has returned true on every call for at least stableFor.
// A false result or an error starts the wait again. It fails if that does not happen within timeout.
func waitUntilStable(check func() (bool, error), stableFor, interval, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var stableSince time.Time
	var lastErr error
	for {
		found, err := check()
		now := time.Now()
		switch {
		case err != nil:
			stableSince = time.Time{}
			lastErr = err
		case !found:
			stableSince = time.Time{}
		case stableSince.IsZero():
			stableSince = now
		}
		if !stableSince.IsZero() && now.Sub(stableSince) >= stableFor {
			return nil
		}
		if now.Add(interval).After(deadline) {
			if lastErr != nil {
				return fmt.Errorf("timed out after %s, last error: %w", timeout, lastErr)
			}
			return fmt.Errorf("timed out after %s", timeout)
		}
		time.Sleep(interval)
	}
}

// findTrustedKeyKid returns the kid of the trusted key with keyAlias, or "" if no such key is listed.
func findTrustedKeyKid(servicesManager artifactory.ArtifactoryServicesManager, keyAlias string) (string, error) {
	client := servicesManager.Client()
	artifactoryDetails := servicesManager.GetConfig().GetServiceDetails()
	httpClientDetails := artifactoryDetails.CreateHttpClientDetails()
	listURL := clientutils.AddTrailingSlashIfNeeded(artifactoryDetails.GetUrl()) + "api/security/keys/trusted"

	resp, body, _, err := client.SendGet(listURL, true, &httpClientDetails)
	if err != nil {
		return "", fmt.Errorf("failed to list trusted keys: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to list trusted keys, status %d: %s", resp.StatusCode, string(body))
	}

	var keysResponse TrustedKeysResponse
	if err := json.Unmarshal(body, &keysResponse); err != nil {
		return "", fmt.Errorf("failed to parse trusted keys response: %w", err)
	}
	for _, key := range keysResponse.Keys {
		if key.Alias == keyAlias {
			return key.Kid, nil
		}
	}
	return "", nil
}
