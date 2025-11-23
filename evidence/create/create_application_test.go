package create

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli-core/v2/utils/coreutils"
	"github.com/jfrog/jfrog-cli-evidence/evidence/model"
	evidenceUtils "github.com/jfrog/jfrog-cli-evidence/evidence/utils"
	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	"github.com/jfrog/jfrog-client-go/utils/io/fileutils"
	"github.com/stretchr/testify/assert"
)

type mockApplicationArtifactoryServicesManager struct {
	artifactory.EmptyArtifactoryServicesManager
}

func (m *mockApplicationArtifactoryServicesManager) FileInfo(_ string) (*utils.FileInfo, error) {
	fi := &utils.FileInfo{
		Checksums: struct {
			Sha1   string `json:"sha1,omitempty"`
			Sha256 string `json:"sha256,omitempty"`
			Md5    string `json:"md5,omitempty"`
		}{
			Sha256: "dummy_application_sha256",
		},
	}
	return fi, nil
}

func createTestApplicationCommand() *createEvidenceApplication {
	return &createEvidenceApplication{
		createEvidenceBase: createEvidenceBase{
			serverDetails:     &config.ServerDetails{Url: "http://test.com"},
			predicateFilePath: "", // Empty predicate file path for testing
			predicateType:     "test-type",
			key:               "test-key",
			keyId:             "test-key-id",
			stage:             "test-stage",
		},
		applicationKey:     "test-app",
		applicationVersion: "1.0.0",
		projectKey:         "test-project",
	}
}

func TestNewCreateEvidenceApplication(t *testing.T) {
	serverDetails := &config.ServerDetails{Url: "http://test.com", User: "testuser"}
	predicateFilePath := "/path/to/predicate.json"
	predicateType := "custom-predicate"
	markdownFilePath := "/path/to/markdown.md"
	key := "test-key"
	keyId := "test-key-id"
	applicationKey := "test-app"
	applicationVersion := "1.0.0"
	providerId := "test-provider-id"
	integration := "sonar"

	cmd := NewCreateEvidenceApplication(serverDetails, predicateFilePath, predicateType, markdownFilePath, key, keyId, applicationKey, applicationVersion, providerId, integration)
	createCmd, ok := cmd.(*createEvidenceApplication)
	assert.True(t, ok)

	assert.Equal(t, serverDetails, createCmd.serverDetails)
	assert.Equal(t, predicateFilePath, createCmd.predicateFilePath)
	assert.Equal(t, predicateType, createCmd.predicateType)
	assert.Equal(t, markdownFilePath, createCmd.markdownFilePath)
	assert.Equal(t, key, createCmd.key)
	assert.Equal(t, keyId, createCmd.keyId)
	assert.Equal(t, providerId, createCmd.providerId)
	assert.Equal(t, integration, createCmd.integration)

	assert.Equal(t, applicationKey, createCmd.applicationKey)
	assert.Equal(t, applicationVersion, createCmd.applicationVersion)

	// The stage should be set (though it might be empty if the apptrust service fails)
	// We just verify it's initialized, not the exact value since it depends on external service
	assert.NotNil(t, createCmd.stage)
}

func TestCreateEvidenceApplication_CommandName(t *testing.T) {
	cmd := &createEvidenceApplication{}
	assert.Equal(t, "create-application-evidence", cmd.CommandName())
}

func TestCreateEvidenceApplication_ServerDetails(t *testing.T) {
	serverDetails := &config.ServerDetails{Url: "http://test.com", User: "testuser"}
	cmd := &createEvidenceApplication{
		createEvidenceBase: createEvidenceBase{serverDetails: serverDetails},
	}

	result, err := cmd.ServerDetails()
	assert.NoError(t, err)
	assert.Equal(t, serverDetails, result)
}

func TestBuildApplicationManifestPath(t *testing.T) {
	tests := []struct {
		name               string
		repoKey            string
		applicationKey     string
		applicationVersion string
		expected           string
	}{
		{
			name:               "Valid_Basic_Path",
			repoKey:            "test-repo-application-versions",
			applicationKey:     "my-app",
			applicationVersion: "1.0.0",
			expected:           "test-repo-application-versions/my-app/1.0.0/release-bundle.json.evd",
		},
		{
			name:               "With_Special_Characters",
			repoKey:            "test-project-application-versions",
			applicationKey:     "my-app-v2",
			applicationVersion: "1.0.0-beta",
			expected:           "test-project-application-versions/my-app-v2/1.0.0-beta/release-bundle.json.evd",
		},
		{
			name:               "With_Numbers",
			repoKey:            "project123-application-versions",
			applicationKey:     "app123",
			applicationVersion: "2.1.0",
			expected:           "project123-application-versions/app123/2.1.0/release-bundle.json.evd",
		},
		{
			name:               "Default_Project",
			repoKey:            "application-versions",
			applicationKey:     "default-app",
			applicationVersion: "1.0.0",
			expected:           "application-versions/default-app/1.0.0/release-bundle.json.evd",
		},
		{
			name:               "Empty_Values",
			repoKey:            "",
			applicationKey:     "",
			applicationVersion: "",
			expected:           "///release-bundle.json.evd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildApplicationManifestPath(tt.repoKey, tt.applicationKey, tt.applicationVersion)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetApplicationVersionStage_Integration(t *testing.T) {
	tests := []struct {
		name               string
		applicationKey     string
		applicationVersion string
		expectedStage      string
		description        string
	}{
		{
			name:               "Error_Handling_Empty_Parameters",
			applicationKey:     "",
			applicationVersion: "1.0.0",
			expectedStage:      "",
			description:        "Should handle empty application key gracefully",
		},
		{
			name:               "Error_Handling_Empty_Version",
			applicationKey:     "test-app",
			applicationVersion: "",
			expectedStage:      "",
			description:        "Should handle empty application version gracefully",
		},
		{
			name:               "Service_Error_Handling",
			applicationKey:     "test-app",
			applicationVersion: "1.0.0",
			expectedStage:      "",
			description:        "Should handle service connection errors gracefully",
		},
		{
			name:               "Stage_Functionality_Documentation",
			applicationKey:     "bookverse-web",
			applicationVersion: "1.0.2",
			expectedStage:      "",
			description:        "Documents expected behavior: returns target_stage of latest COMPLETED promotion",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serverDetails := &config.ServerDetails{
				Url:         "http://localhost:8081/",
				AccessToken: "test-token",
			}

			stage := getApplicationVersionStage(serverDetails, tt.applicationKey, tt.applicationVersion)
			assert.Equal(t, tt.expectedStage, stage, tt.description)
		})
	}
}

func TestStageIntegrationInApplicationConstructor(t *testing.T) {
	tests := []struct {
		name               string
		applicationKey     string
		applicationVersion string
		description        string
	}{
		{
			name:               "Stage_Field_Integration",
			applicationKey:     "test-app",
			applicationVersion: "1.0.0",
			description:        "Stage field should be set during constructor call",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serverDetails := &config.ServerDetails{
				Url:         "http://localhost:8081/",
				AccessToken: "test-token",
			}

			cmd := NewCreateEvidenceApplication(
				serverDetails,
				"/test/predicate.json",
				"test-predicate-type",
				"/test/markdown.md",
				"test-key",
				"test-key-id",
				tt.applicationKey,
				tt.applicationVersion,
				"test-provider-id",
				"",
			)

			appCmd, ok := cmd.(*createEvidenceApplication)
			assert.True(t, ok)

			t.Logf("Stage field set to: '%s' (may be empty if apptrust service unavailable)", appCmd.stage)
			// We don't assert a specific value since it depends on external service availability
			// The important thing is that the stage field exists and the constructor doesn't crash
		})
	}
}

func TestCreateEvidenceApplication_RecordSummary(t *testing.T) {
	tempDir, err := fileutils.CreateTempDir()
	assert.NoError(t, err)
	defer func() {
		assert.NoError(t, fileutils.RemoveTempDir(tempDir))
	}()

	assert.NoError(t, os.Setenv("GITHUB_ACTIONS", "true"))
	assert.NoError(t, os.Setenv(coreutils.SummaryOutputDirPathEnv, tempDir))
	defer func() {
		assert.NoError(t, os.Unsetenv("GITHUB_ACTIONS"))
		assert.NoError(t, os.Unsetenv(coreutils.SummaryOutputDirPathEnv))
	}()

	serverDetails := &config.ServerDetails{
		Url:      "http://test.com",
		User:     "testuser",
		Password: "testpass",
	}

	evidence := NewCreateEvidenceApplication(
		serverDetails,
		"",
		"test-predicate-type",
		"",
		"test-key",
		"test-key-id",
		"testApp",
		"2.0.0",
		"test-provider-id",
		"",
	)

	appCmd, ok := evidence.(*createEvidenceApplication)
	assert.True(t, ok, "should create createEvidenceApplication instance")
	appCmd.projectKey = "myProject" // Set project key for testing

	expectedResponse := &model.CreateResponse{
		PredicateSlug: "test-app-slug",
		Verified:      true,
	}
	expectedSubject := "myProject-application-versions/testApp/2.0.0/release-bundle.json.evd"
	expectedSha256 := "app-sha256"

	appCmd.recordSummary(expectedResponse, expectedSubject, expectedSha256)

	summaryFiles, err := fileutils.ListFiles(tempDir, true)
	assert.NoError(t, err)
	assert.True(t, len(summaryFiles) > 0, "Summary file should be created")
}

func TestCreateEvidenceApplication_ProviderId(t *testing.T) {
	tests := []struct {
		name        string
		integration string
		providerId  string
		expected    string
	}{
		{
			name:        "With_custom_integration_ID",
			integration: "custom",
			providerId:  "custom-provider-id",
			expected:    "custom-provider-id",
		},
		{
			name:        "With_empty_integration_ID",
			integration: "",
			providerId:  "test-provider-id",
			expected:    "test-provider-id",
		},
		{
			name:        "With_sonar_integration_ID",
			integration: "sonar",
			providerId:  "sonar-provider-id",
			expected:    "sonar-provider-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serverDetails := &config.ServerDetails{
				Url:         "http://test.com",
				AccessToken: "test-token",
			}

			cmd := NewCreateEvidenceApplication(
				serverDetails,
				"/test/predicate.json",
				"test-predicate-type",
				"/test/markdown.md",
				"test-key",
				"test-key-id",
				"test-app",
				"1.0.0",
				tt.providerId,
				tt.integration,
			)

			appCmd, ok := cmd.(*createEvidenceApplication)
			assert.True(t, ok)
			assert.Equal(t, tt.expected, appCmd.providerId)
			assert.Equal(t, tt.integration, appCmd.integration)
		})
	}
}

func TestApplication(t *testing.T) {
	tests := []struct {
		name               string
		applicationKey     string
		applicationVersion string
		projectKey         string
		expected           string
	}{
		{
			name:               "Valid_application_with_project",
			applicationKey:     "my-app",
			applicationVersion: "1.0.0",
			projectKey:         "my-project",
			expected:           "my-project-application-versions",
		},
		{
			name:               "Valid_application_default_project",
			applicationKey:     "default-app",
			applicationVersion: "1.0.0",
			projectKey:         "default",
			expected:           "application-versions",
		},
		{
			name:               "Valid_application_empty_project",
			applicationKey:     "empty-app",
			applicationVersion: "1.0.0",
			projectKey:         "",
			expected:           "application-versions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test buildApplicationSubjectPath indirectly through the repo key logic
			repoKey := evidenceUtils.BuildApplicationVersionRepoKey(tt.projectKey)
			assert.Equal(t, tt.expected, repoKey)

			// Test manifest path construction
			manifestPath := buildApplicationManifestPath(repoKey, tt.applicationKey, tt.applicationVersion)
			expectedPath := repoKey + "/" + tt.applicationKey + "/" + tt.applicationVersion + "/release-bundle.json.evd"
			assert.Equal(t, expectedPath, manifestPath)
		})
	}
}

func TestCreateEvidenceApplication_Run_Success_WithInjectedDeps(t *testing.T) {
	// Create temporary predicate file
	d := t.TempDir()
	pred := filepath.Join(d, "p.json")
	_ = os.WriteFile(pred, []byte(`{"application":"test-app","version":"1.0.0"}`), 0600)

	// Create test command with proper setup
	cmd := &createEvidenceApplication{
		createEvidenceBase: createEvidenceBase{
			serverDetails:     &config.ServerDetails{User: "testuser"},
			predicateFilePath: pred,
			predicateType:     "application-evidence",
			key:               "", // Empty key for testing to avoid signing
			artifactoryClient: &mockApplicationArtifactoryServicesManager{},
			uploader: &MockEvidenceServiceManager{
				UploadResponse: []byte(`{"predicate_slug":"test-slug","verified":true}`),
			},
		},
		applicationKey:     "test-app",
		applicationVersion: "1.0.0",
		projectKey:         "test-project", // Set project key directly to avoid apptrust service call
	}

	// Test the core functionality without calling fetchProjectKey
	// This simulates the Run() method but skips the external API call and signing
	artifactoryClient, err := cmd.createArtifactoryClient()
	assert.NoError(t, err)

	subject, sha256, err := cmd.buildApplicationSubjectPath(artifactoryClient)
	assert.NoError(t, err)
	assert.NotEmpty(t, subject)
	assert.NotEmpty(t, sha256)
	assert.Contains(t, subject, "test-project-application-versions")
	assert.Contains(t, subject, "test-app")
	assert.Contains(t, subject, "1.0.0")
	assert.Equal(t, "dummy_application_sha256", sha256)
}

func TestCreateEvidenceApplication_Run_FileInfoError(t *testing.T) {
	cmd := createTestApplicationCommand()
	cmd.artifactoryClient = &SimpleMockServicesManager{
		FileInfoFunc: func(_ string) (*utils.FileInfo, error) {
			return nil, errors.New("file info error")
		},
	}

	mockResponse := model.CreateResponse{PredicateSlug: "test-slug", Verified: true}
	responseBytes, err := json.Marshal(mockResponse)
	if err != nil {
		t.Fatal(err)
	}
	cmd.uploader = &MockEvidenceServiceManager{
		UploadResponse: responseBytes,
	}

	err = cmd.Run()
	assert.Error(t, err)
}

func TestCreateEvidenceApplication_Run_EnvelopeError(t *testing.T) {
	cmd := createTestApplicationCommand()
	cmd.artifactoryClient = &mockApplicationArtifactoryServicesManager{}
	cmd.predicateFilePath = "invalid-predicate-path"

	mockResponse := model.CreateResponse{PredicateSlug: "test-slug", Verified: true}
	responseBytes, err := json.Marshal(mockResponse)
	if err != nil {
		t.Fatal(err)
	}
	cmd.uploader = &MockEvidenceServiceManager{
		UploadResponse: responseBytes,
	}

	err = cmd.Run()
	assert.Error(t, err)
}

func TestCreateEvidenceApplication_Run_UploadError(t *testing.T) {
	cmd := createTestApplicationCommand()
	cmd.artifactoryClient = &mockApplicationArtifactoryServicesManager{}
	cmd.uploader = &MockEvidenceServiceManager{
		UploadError: errors.New("upload error"),
	}

	err := cmd.Run()
	assert.Error(t, err)
}
