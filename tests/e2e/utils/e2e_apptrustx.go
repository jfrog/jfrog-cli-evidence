package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/http/jfroghttpclient"
	"github.com/jfrog/jfrog-client-go/lifecycle"
	clientutils "github.com/jfrog/jfrog-client-go/utils"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"github.com/stretchr/testify/require"
)

// CreateApplicationRequest represents the request to create an application
type CreateApplicationRequest struct {
	ApplicationName string `json:"application_name"`
	ApplicationKey  string `json:"application_key"`
	ProjectKey      string `json:"project_key"`
	Description     string `json:"description,omitempty"`
}

// ApplicationResponse represents the response when creating an application
type ApplicationResponse struct {
	ApplicationName string `json:"application_name"`
	ApplicationKey  string `json:"application_key"`
	ProjectKey      string `json:"project_key"`
}

// CreateApplicationVersionRequest represents the request to create an application version
type CreateApplicationVersionRequest struct {
	Version string                     `json:"version"`
	Sources *ApplicationVersionSources `json:"sources"`
}

// ApplicationVersionSources represents the sources for an application version
type ApplicationVersionSources struct {
	Artifacts []ApplicationVersionArtifact `json:"artifacts,omitempty"`
}

// ApplicationVersionArtifact represents an artifact source
type ApplicationVersionArtifact struct {
	Path   string `json:"path"`
	Sha256 string `json:"sha256"`
}

// ApplicationVersionResponse represents the response when creating an application version
type ApplicationVersionResponse struct {
	ApplicationKey string `json:"application_key"`
	Version        string `json:"version"`
}

// CreateTestApplication creates a test application in AppTrust using the AppTrust API
// If the application already exists (409 Conflict), it will delete it and retry
func CreateTestApplication(t *testing.T, artifactoryManager artifactory.ArtifactoryServicesManager, projectKey string) (string, string) {
	// Generate unique application name with timestamp
	timestamp := time.Now().Unix()
	applicationKey := fmt.Sprintf("test-app-%d", timestamp)
	applicationName := fmt.Sprintf("Test Application %d", timestamp)

	t.Logf("Creating test application via AppTrust API: %s in project: %s", applicationKey, projectKey)

	// Get the base JFrog platform URL (strip /artifactory/ if present)
	config := artifactoryManager.GetConfig()
	serviceDetails := config.GetServiceDetails()
	url := serviceDetails.GetUrl()
	// AppTrust API is at platform level, not under /artifactory/
	baseURL := strings.TrimSuffix(url, "artifactory/")
	baseURL = clientutils.AddTrailingSlashIfNeeded(baseURL)

	// Create application using AppTrust API
	request := CreateApplicationRequest{
		ApplicationName: applicationName,
		ApplicationKey:  applicationKey,
		ProjectKey:      projectKey,
		Description:     "Test application for E2E evidence testing",
	}

	requestBody, err := json.Marshal(request)
	require.NoError(t, err)

	// Create HTTP client
	client := serviceDetails.GetClient()
	if client == nil {
		client, err = jfroghttpclient.JfrogClientBuilder().Build()
		require.NoError(t, err)
	}

	// POST /apptrust/api/v1/applications
	appURL := baseURL + "apptrust/api/v1/applications"
	httpDetails := serviceDetails.CreateHttpClientDetails()
	httpDetails.Headers["Content-Type"] = "application/json"
	t.Logf("Creating test application via AppTrust API: %s", appURL)

	resp, body, err := client.SendPost(appURL, requestBody, &httpDetails)
	require.NoError(t, err)

	// If application already exists (409 Conflict), delete it and retry
	if resp.StatusCode == http.StatusConflict {
		t.Logf("Application %s already exists, deleting and retrying...", applicationKey)
		deleteApplicationSilent(artifactoryManager, applicationKey)
		// Retry creation
		resp, body, err = client.SendPost(appURL, requestBody, &httpDetails)
		require.NoError(t, err)
	}

	err = errorutils.CheckResponseStatusWithBody(resp, body, http.StatusCreated)
	require.NoError(t, err, "Failed to create application: %s", string(body))

	var appResp ApplicationResponse
	err = json.Unmarshal(body, &appResp)
	require.NoError(t, err)

	t.Logf("✓ Application created via API: %s (%s)", applicationKey, applicationName)
	return applicationKey, applicationName
}

// CreateTestApplicationVersion creates a test application version using AppTrust API
func CreateTestApplicationVersion(t *testing.T, artifactoryManager artifactory.ArtifactoryServicesManager, lifecycleManager *lifecycle.LifecycleServicesManager, applicationKey, projectKey string) string {
	// Generate unique version with timestamp
	timestamp := time.Now().Unix()
	version := fmt.Sprintf("1.0.%d", timestamp%10000)

	t.Logf("Creating test application version via AppTrust API: %s:%s", applicationKey, version)

	// Create an artifact to include in the version
	repoName := CreateTestRepositoryWithName(t, artifactoryManager, "generic")
	artifactContent := fmt.Sprintf("Test artifact for app version - timestamp: %d", timestamp)
	artifactPath := CreateTestArtifact(t, artifactContent)
	artifactFileName := "test-app-artifact.txt"
	repoPath := fmt.Sprintf("%s/%s", repoName, artifactFileName)

	err := UploadArtifact(artifactoryManager, artifactPath, repoPath)
	require.NoError(t, err)

	// Get artifact checksum
	fileInfo, err := artifactoryManager.FileInfo(repoPath)
	require.NoError(t, err)

	// Create application version using AppTrust API
	config := artifactoryManager.GetConfig()
	serviceDetails := config.GetServiceDetails()
	rawURL := serviceDetails.GetUrl()
	// AppTrust API is at platform level, not under /artifactory/
	baseURL := clientutils.AddTrailingSlashIfNeeded(strings.TrimSuffix(rawURL, "artifactory/"))

	request := CreateApplicationVersionRequest{
		Version: version,
		Sources: &ApplicationVersionSources{
			Artifacts: []ApplicationVersionArtifact{
				{
					Path:   repoPath,
					Sha256: fileInfo.Checksums.Sha256,
				},
			},
		},
	}

	requestBody, err := json.Marshal(request)
	require.NoError(t, err)

	// Create HTTP client
	client := serviceDetails.GetClient()
	if client == nil {
		client, err = jfroghttpclient.JfrogClientBuilder().Build()
		require.NoError(t, err)
	}

	// POST /apptrust/api/v1/applications/{applicationKey}/versions?async=false
	url := fmt.Sprintf("%sapptrust/api/v1/applications/%s/versions?async=false", baseURL, applicationKey)
	httpDetails := serviceDetails.CreateHttpClientDetails()
	httpDetails.Headers["Content-Type"] = "application/json"

	resp, body, err := client.SendPost(url, requestBody, &httpDetails)
	require.NoError(t, err)

	err = errorutils.CheckResponseStatusWithBody(resp, body, http.StatusCreated)
	require.NoError(t, err, "Failed to create application version: %s", string(body))

	var versionResp ApplicationVersionResponse
	err = json.Unmarshal(body, &versionResp)
	require.NoError(t, err)

	t.Logf("✓ Application version created via API: %s:%s", applicationKey, version)
	return version
}

// PromoteApplicationVersionRequest represents the request to promote an application version
type PromoteApplicationVersionRequest struct {
	TargetStage string `json:"target_stage"`
	Comment     string `json:"comment,omitempty"`
}

// PromoteApplicationVersion promotes an application version using AppTrust API
func PromoteApplicationVersion(t *testing.T, artifactoryManager artifactory.ArtifactoryServicesManager, applicationKey, version, projectKey, targetStage string) error {
	t.Logf("Promoting application version %s:%s to stage: %s via AppTrust API", applicationKey, version, targetStage)

	// Get the base JFrog platform URL (strip /artifactory/ if present)
	config := artifactoryManager.GetConfig()
	serviceDetails := config.GetServiceDetails()
	rawURL := serviceDetails.GetUrl()
	// AppTrust API is at platform level, not under /artifactory/
	baseURL := clientutils.AddTrailingSlashIfNeeded(strings.TrimSuffix(rawURL, "artifactory/"))

	// Create promotion request
	request := PromoteApplicationVersionRequest{
		TargetStage: targetStage,
		Comment:     fmt.Sprintf("E2E test promotion to %s", targetStage),
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal promotion request: %w", err)
	}

	// Create HTTP client
	client := serviceDetails.GetClient()
	if client == nil {
		client, err = jfroghttpclient.JfrogClientBuilder().Build()
		if err != nil {
			return fmt.Errorf("failed to create HTTP client: %w", err)
		}
	}

	// POST /apptrust/api/v1/applications/{applicationKey}/versions/{version}/promote?async=false
	url := fmt.Sprintf("%sapptrust/api/v1/applications/%s/versions/%s/promote?async=false", baseURL, applicationKey, version)
	httpDetails := serviceDetails.CreateHttpClientDetails()
	httpDetails.Headers["Content-Type"] = "application/json"

	resp, body, err := client.SendPost(url, requestBody, &httpDetails)
	if err != nil {
		return fmt.Errorf("failed to send promote request: %w", err)
	}

	err = errorutils.CheckResponseStatusWithBody(resp, body, http.StatusOK)
	if err != nil {
		return fmt.Errorf("failed to promote application version: %s", string(body))
	}

	t.Logf("✓ Application version %s:%s promoted to %s", applicationKey, version, targetStage)
	return nil
}

// CleanupTestApplicationVersion deletes an application version using AppTrust API
// This should be called before deleting the application
func CleanupTestApplicationVersion(t *testing.T, artifactoryManager artifactory.ArtifactoryServicesManager, applicationKey, version string) {
	t.Logf("Deleting application version %s:%s via AppTrust API", applicationKey, version)

	// Get the base JFrog platform URL (strip /artifactory/ if present)
	config := artifactoryManager.GetConfig()
	serviceDetails := config.GetServiceDetails()
	rawURL := serviceDetails.GetUrl()
	baseURL := clientutils.AddTrailingSlashIfNeeded(strings.TrimSuffix(rawURL, "artifactory/"))

	// Create HTTP client
	client := serviceDetails.GetClient()
	if client == nil {
		var err error
		client, err = jfroghttpclient.JfrogClientBuilder().Build()
		if err != nil {
			t.Logf("Warning: Failed to create HTTP client for cleanup: %v", err)
			return
		}
	}

	// DELETE /apptrust/api/v1/applications/{applicationKey}/versions/{version}?async=false
	url := fmt.Sprintf("%sapptrust/api/v1/applications/%s/versions/%s?async=false", baseURL, applicationKey, version)
	httpDetails := serviceDetails.CreateHttpClientDetails()

	resp, body, err := client.SendDelete(url, nil, &httpDetails)
	if err != nil {
		t.Logf("Warning: Failed to delete application version %s:%s: %v", applicationKey, version, err)
		return
	}

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusAccepted {
		t.Logf("Warning: Failed to delete application version %s:%s: %s", applicationKey, version, string(body))
		return
	}

	t.Logf("✓ Application version deleted: %s:%s", applicationKey, version)
}

// CleanupTestApplication deletes the application using AppTrust API
// Note: This also deletes all associated versions
func CleanupTestApplication(t *testing.T, artifactoryManager artifactory.ArtifactoryServicesManager, applicationKey, projectKey string) {
	t.Logf("Deleting application %s via AppTrust API", applicationKey)

	// Get the base JFrog platform URL (strip /artifactory/ if present)
	config := artifactoryManager.GetConfig()
	serviceDetails := config.GetServiceDetails()
	rawURL := serviceDetails.GetUrl()
	// AppTrust API is at platform level, not under /artifactory/
	baseURL := clientutils.AddTrailingSlashIfNeeded(strings.TrimSuffix(rawURL, "artifactory/"))

	// Create HTTP client
	client := serviceDetails.GetClient()
	if client == nil {
		var err error
		client, err = jfroghttpclient.JfrogClientBuilder().Build()
		if err != nil {
			t.Logf("Warning: Failed to create HTTP client for cleanup: %v", err)
			return
		}
	}

	// DELETE /apptrust/api/v1/applications/{applicationKey}?async=false
	url := fmt.Sprintf("%sapptrust/api/v1/applications/%s?async=false", baseURL, applicationKey)
	httpDetails := serviceDetails.CreateHttpClientDetails()

	resp, body, err := client.SendDelete(url, nil, &httpDetails)
	if err != nil {
		t.Logf("Warning: Failed to delete application %s: %v", applicationKey, err)
		return
	}

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		t.Logf("Warning: Failed to delete application %s: %s", applicationKey, string(body))
		return
	}

	t.Logf("✓ Application deleted: %s", applicationKey)
}

// deleteApplicationSilent deletes an application without logging (for cleanup before create)
func deleteApplicationSilent(artifactoryManager artifactory.ArtifactoryServicesManager, applicationKey string) {
	config := artifactoryManager.GetConfig()
	serviceDetails := config.GetServiceDetails()
	rawURL := serviceDetails.GetUrl()
	baseURL := clientutils.AddTrailingSlashIfNeeded(strings.TrimSuffix(rawURL, "artifactory/"))

	client := serviceDetails.GetClient()
	if client == nil {
		client, _ = jfroghttpclient.JfrogClientBuilder().Build()
		if client == nil {
			return
		}
	}

	url := fmt.Sprintf("%sapptrust/api/v1/applications/%s?async=false", baseURL, applicationKey)
	httpDetails := serviceDetails.CreateHttpClientDetails()
	_, _, _ = client.SendDelete(url, nil, &httpDetails)
}

// ApplicationVersionExists checks if an application version exists using AppTrust API
func ApplicationVersionExists(t *testing.T, artifactoryManager artifactory.ArtifactoryServicesManager, applicationKey, version, projectKey string) bool {
	// Get the base JFrog platform URL (strip /artifactory/ if present)
	config := artifactoryManager.GetConfig()
	serviceDetails := config.GetServiceDetails()
	rawURL := serviceDetails.GetUrl()
	// AppTrust API is at platform level, not under /artifactory/
	baseURL := clientutils.AddTrailingSlashIfNeeded(strings.TrimSuffix(rawURL, "artifactory/"))

	// Create HTTP client
	client := serviceDetails.GetClient()
	if client == nil {
		var err error
		client, err = jfroghttpclient.JfrogClientBuilder().Build()
		if err != nil {
			t.Logf("Warning: Failed to create HTTP client: %v", err)
			return false
		}
	}

	// GET /apptrust/api/v1/applications/{applicationKey}/versions/{version}
	url := fmt.Sprintf("%sapptrust/api/v1/applications/%s/versions/%s", baseURL, applicationKey, version)
	httpDetails := serviceDetails.CreateHttpClientDetails()

	resp, _, _, err := client.SendGet(url, true, &httpDetails)
	if err != nil {
		return false
	}

	return resp.StatusCode == http.StatusOK
}
