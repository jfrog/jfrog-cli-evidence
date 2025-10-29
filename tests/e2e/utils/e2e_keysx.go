package utils

import (
	"encoding/json"
	"fmt"
	"net/http"

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

// DeleteTrustedKey deletes a trusted key from Artifactory using the REST API
// keyAlias: the alias of the key to delete (e.g., "e2e-shared-key-1762361018")
func DeleteTrustedKey(servicesManager artifactory.ArtifactoryServicesManager, keyAlias string) error {
	client := servicesManager.Client()
	artifactoryDetails := servicesManager.GetConfig().GetServiceDetails()
	httpClientDetails := artifactoryDetails.CreateHttpClientDetails()
	baseURL := clientutils.AddTrailingSlashIfNeeded(artifactoryDetails.GetUrl())

	// Step 1: Get all trusted keys to find the kid for this alias
	listURL := baseURL + "api/security/keys/trusted"
	resp, body, _, err := client.SendGet(listURL, true, &httpClientDetails)
	if err != nil {
		return fmt.Errorf("failed to list trusted keys: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to list trusted keys, status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response to find the kid
	var keysResponse TrustedKeysResponse
	if err := json.Unmarshal(body, &keysResponse); err != nil {
		return fmt.Errorf("failed to parse trusted keys response: %w", err)
	}

	// Find the key with matching alias
	var kidToDelete string
	for _, key := range keysResponse.Keys {
		if key.Alias == keyAlias {
			kidToDelete = key.Kid
			break
		}
	}

	if kidToDelete == "" {
		// Key not found - might have been already deleted or never existed
		return nil
	}

	// Step 2: Delete the key using its kid
	deleteURL := baseURL + "api/security/keys/trusted/" + kidToDelete
	resp, body, err = client.SendDelete(deleteURL, nil, &httpClientDetails)
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
