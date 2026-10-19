package verify

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	buildinfo "github.com/jfrog/build-info-go/entities"
	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli-core/v2/utils/coreutils"
	"github.com/jfrog/jfrog-cli-evidence/evidence/model"
	"github.com/jfrog/jfrog-cli-evidence/evidence/utils"
	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/artifactory/services"
	servicesUtils "github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockArtifactoryServicesManagerBuild embeds EmptyArtifactoryServicesManager and overrides methods for testing
type MockArtifactoryServicesManagerBuild struct {
	artifactory.EmptyArtifactoryServicesManager
	AqlResponse         string
	AqlError            error
	ReadRemoteFileData  []byte
	ReadRemoteFileError error
	// BuildRuns maps a build number to the start time, in Unix milliseconds, of its latest run.
	// GetBuildInfo reports build numbers that are not in the map as not found.
	BuildRuns         map[string]int64
	GetBuildInfoError error
	// BuildInfoFiles, when set, are the files in the build-info repository. Aql then returns the
	// files whose repo, path and name all equal the values in the query, instead of AqlResponse.
	BuildInfoFiles []*servicesUtils.ResultItem
	// Recorded calls
	AqlQueries        []string
	BuildInfoRequests []services.BuildInfoParams
}

func (m *MockArtifactoryServicesManagerBuild) Aql(query string) (io.ReadCloser, error) {
	m.AqlQueries = append(m.AqlQueries, query)
	if m.AqlError != nil {
		return nil, m.AqlError
	}
	if m.BuildInfoFiles == nil {
		return io.NopCloser(bytes.NewBufferString(m.AqlResponse)), nil
	}
	matches := []*servicesUtils.ResultItem{}
	for _, file := range m.BuildInfoFiles {
		matched := true
		for _, field := range [][2]string{{"repo", file.Repo}, {"path", file.Path}, {"name", file.Name}} {
			condition, err := utils.FormatAqlQuery(`"`+field[0]+`":%s`, field[1])
			if err != nil {
				return nil, err
			}
			matched = matched && strings.Contains(query, condition)
		}
		if matched {
			matches = append(matches, file)
		}
	}
	response, err := json.Marshal(utils.AqlResult{Results: matches})
	if err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(response)), nil
}

func (m *MockArtifactoryServicesManagerBuild) GetBuildInfo(params services.BuildInfoParams) (*buildinfo.PublishedBuildInfo, bool, error) {
	m.BuildInfoRequests = append(m.BuildInfoRequests, params)
	if m.GetBuildInfoError != nil {
		return nil, false, m.GetBuildInfoError
	}
	started, ok := m.BuildRuns[params.BuildNumber]
	if !ok {
		return nil, false, nil
	}
	return &buildinfo.PublishedBuildInfo{
		BuildInfo: buildinfo.BuildInfo{
			Name:    params.BuildName,
			Number:  params.BuildNumber,
			Started: time.UnixMilli(started).UTC().Format(utils.IsoDateTimeLayout),
		},
	}, true, nil
}

func (m *MockArtifactoryServicesManagerBuild) ReadRemoteFile(_ string) (io.ReadCloser, error) {
	if m.ReadRemoteFileError != nil {
		return nil, m.ReadRemoteFileError
	}
	if m.ReadRemoteFileData != nil {
		// Return a fresh reader each time to avoid EOF issues with multiple reads
		return io.NopCloser(bytes.NewReader(m.ReadRemoteFileData)), nil
	}
	// Return empty response to avoid method not implemented error
	return io.NopCloser(bytes.NewReader([]byte(`{"payload":"","payloadType":"","signatures":[]}`))), nil
}

// MockOneModelManagerBuild for build tests
type MockOneModelManagerBuild struct {
	GraphqlResponse []byte
	GraphqlError    error
}

func (m *MockOneModelManagerBuild) GraphqlQuery(_ []byte) ([]byte, error) {
	if m.GraphqlError != nil {
		return nil, m.GraphqlError
	}
	return m.GraphqlResponse, nil
}

// MockVerifyEvidenceBaseBuild for testing verifyEvidence method
type MockVerifyEvidenceBaseBuild struct {
	mock.Mock
	verifyEvidenceBase
}

func (m *MockVerifyEvidenceBaseBuild) verifyEvidence(client *artifactory.ArtifactoryServicesManager, metadata *[]model.SearchEvidenceEdge, sha256, subjectPath string) error {
	args := m.Called(client, metadata, sha256, subjectPath)
	return args.Error(0)
}

// MockVerifier for testing with testify - implements VerifierInterface
type MockVerifier struct {
	mock.Mock
}

func (m *MockVerifier) Verify(expectedSubject model.SubjectDigest, evidenceMetadata *[]model.SearchEvidenceEdge, subjectPath string) (*model.VerificationResponse, error) {
	args := m.Called(expectedSubject, evidenceMetadata, subjectPath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	result := args.Get(0)
	resp, ok := result.(*model.VerificationResponse)
	if !ok && result != nil {
		return nil, args.Error(1)
	}
	return resp, args.Error(1)
}

func TestNewVerifyEvidenceBuild(t *testing.T) {
	serverDetails := &config.ServerDetails{}
	project := "test-project"
	buildName := "test-build"
	buildNumber := "1"
	format := "json"
	keys := []string{"key1", "key2"}

	cmd := NewVerifyEvidenceBuild(serverDetails, project, buildName, buildNumber, format, keys, true)
	verifyCmd, ok := cmd.(*verifyEvidenceBuild)
	assert.True(t, ok)

	// Test verifyEvidenceBase fields
	assert.Equal(t, serverDetails, verifyCmd.serverDetails)
	assert.Equal(t, format, verifyCmd.format)
	assert.Equal(t, keys, verifyCmd.keys)
	assert.True(t, verifyCmd.useArtifactoryKeys)

	// Test verifyEvidenceBuild fields
	assert.Equal(t, project, verifyCmd.project)
	assert.Equal(t, buildName, verifyCmd.buildName)
	assert.Equal(t, buildNumber, verifyCmd.buildNumber)
}

func TestVerifyEvidenceBuild_CommandName(t *testing.T) {
	cmd := &verifyEvidenceBuild{}
	assert.Equal(t, "verify-evidence-build", cmd.CommandName())
}

func TestVerifyEvidenceBuild_ServerDetails(t *testing.T) {
	serverDetails := &config.ServerDetails{Url: "test.com"}
	cmd := &verifyEvidenceBuild{
		verifyEvidenceBase: verifyEvidenceBase{serverDetails: serverDetails},
	}

	result, err := cmd.ServerDetails()
	assert.NoError(t, err)
	assert.Equal(t, serverDetails, result)
}

func TestVerifyEvidenceBuild_Run_AqlError(t *testing.T) {
	// Mock Artifactory client with error
	mockClient := &MockArtifactoryServicesManagerBuild{
		AqlError:  errors.New("aql query failed"),
		BuildRuns: map[string]int64{"1": 1234567890},
	}

	// Create build verifier
	buildVerifier := &verifyEvidenceBuild{
		verifyEvidenceBase: verifyEvidenceBase{
			serverDetails: &config.ServerDetails{},
			artifactoryClient: func() *artifactory.ArtifactoryServicesManager {
				c := artifactory.ArtifactoryServicesManager(mockClient)
				return &c
			}(),
		},
		project:     "test-project",
		buildName:   "test-build",
		buildNumber: "1",
	}

	err := buildVerifier.Run()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to execute AQL query")
}

func TestVerifyEvidenceBuild_Run_NoBuildFound(t *testing.T) {
	// Mock Artifactory client whose build-info API knows no builds
	mockClient := &MockArtifactoryServicesManagerBuild{}

	// Create build verifier
	buildVerifier := &verifyEvidenceBuild{
		verifyEvidenceBase: verifyEvidenceBase{
			serverDetails: &config.ServerDetails{},
			artifactoryClient: func() *artifactory.ArtifactoryServicesManager {
				c := artifactory.ArtifactoryServicesManager(mockClient)
				return &c
			}(),
		},
		project:     "test-project",
		buildName:   "test-build",
		buildNumber: "1",
	}

	err := buildVerifier.Run()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no build found")
	assert.Empty(t, mockClient.AqlQueries, "no build-info file should be looked up for a build that does not exist")
}

func TestVerifyEvidenceBuild_Run_QueryEvidenceMetadataError(t *testing.T) {
	// Mock AQL response with build info file
	aqlResult := `{"results":[{"sha256":"test-sha256","name":"1-1234567890.json"}]}`

	// Mock Artifactory client
	mockClient := &MockArtifactoryServicesManagerBuild{
		AqlResponse: aqlResult,
		BuildRuns:   map[string]int64{"1": 1234567890},
	}

	// Mock OneModel client with error
	mockOneModel := &MockOneModelManagerBuild{
		GraphqlError: errors.New("graphql query failed"),
	}

	// Create build verifier
	buildVerifier := &verifyEvidenceBuild{
		verifyEvidenceBase: verifyEvidenceBase{
			serverDetails: &config.ServerDetails{},
			artifactoryClient: func() *artifactory.ArtifactoryServicesManager {
				c := artifactory.ArtifactoryServicesManager(mockClient)
				return &c
			}(),
			oneModelClient: mockOneModel,
		},
		project:     "test-project",
		buildName:   "test-build",
		buildNumber: "1",
	}

	err := buildVerifier.Run()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "graphql query failed")
}

func TestVerifyEvidenceBuild_Run_VerifyEvidenceError(t *testing.T) {
	// Mock AQL response with build info file
	aqlResult := `{"results":[{"sha256":"test-sha256","name":"1-1234567890.json"}]}`

	// Mock Artifactory client
	mockClient := &MockArtifactoryServicesManagerBuild{
		AqlResponse: aqlResult,
		BuildRuns:   map[string]int64{"1": 1234567890},
	}

	// Mock OneModel client for evidence metadata
	mockOneModel := &MockOneModelManagerBuild{
		GraphqlResponse: []byte(`{"data":{"evidence":{"searchEvidence":{"edges":[{"node":{"subject":{"sha256":"test-sha256"}}}]}}}}`),
	}

	// Mock the base verification with error
	mockBase := &MockVerifyEvidenceBaseBuild{}
	base := &verifyEvidenceBase{
		serverDetails: &config.ServerDetails{},
		artifactoryClient: func() *artifactory.ArtifactoryServicesManager {
			c := artifactory.ArtifactoryServicesManager(mockClient)
			return &c
		}(),
		oneModelClient: mockOneModel,
	}
	mockBase.verifyEvidenceBase = *base
	mockBase.On("verifyEvidence", mock.Anything, mock.Anything, "test-sha256", mock.Anything).Return(errors.New("verification failed"))

	// Test direct method call
	err := mockBase.verifyEvidence(nil, &[]model.SearchEvidenceEdge{{}}, "test-sha256", "/test/path")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "verification failed")
	mockBase.AssertExpectations(t)
}

func TestVerifyEvidenceBuild_Run_CreateArtifactoryClientError(t *testing.T) {
	// Create build verifier with invalid server configuration that would cause client creation to fail
	buildVerifier := &verifyEvidenceBuild{
		verifyEvidenceBase: verifyEvidenceBase{
			serverDetails: &config.ServerDetails{
				Url: "invalid-url", // Invalid URL that should cause client creation to fail
			},
		},
		project:     "test-project",
		buildName:   "test-build",
		buildNumber: "1",
	}

	err := buildVerifier.Run()
	assert.Error(t, err)
	// Just verify an error occurs - the specific error depends on when the invalid config is detected
}

func TestVerifyEvidenceBuild_ProjectBuildRepoKey(t *testing.T) {
	// Test different project scenarios for repo key building
	testCases := []struct {
		name            string
		project         string
		buildName       string
		buildNumber     string
		expectedRepoKey string
	}{
		{
			name:            "Empty project",
			project:         "",
			buildName:       "test-build",
			buildNumber:     "1",
			expectedRepoKey: "artifactory-build-info",
		},
		{
			name:            "Default project",
			project:         "default",
			buildName:       "test-build",
			buildNumber:     "1",
			expectedRepoKey: "artifactory-build-info",
		},
		{
			name:            "Custom project",
			project:         "my-project",
			buildName:       "test-build",
			buildNumber:     "1",
			expectedRepoKey: "my-project-build-info",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Mock AQL response with no results to test repo key logic
			aqlResult := `{"results":[]}`

			// Mock Artifactory client
			mockClient := &MockArtifactoryServicesManagerBuild{
				AqlResponse: aqlResult,
				BuildRuns:   map[string]int64{tc.buildNumber: 1234567890},
			}

			// Create build verifier
			buildVerifier := &verifyEvidenceBuild{
				verifyEvidenceBase: verifyEvidenceBase{
					serverDetails: &config.ServerDetails{},
					artifactoryClient: func() *artifactory.ArtifactoryServicesManager {
						c := artifactory.ArtifactoryServicesManager(mockClient)
						return &c
					}(),
				},
				project:     tc.project,
				buildName:   tc.buildName,
				buildNumber: tc.buildNumber,
			}

			// Run should fail with "no build found" but this validates the repo key logic
			err := buildVerifier.Run()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "no build found")
			require.Len(t, mockClient.BuildInfoRequests, 1)
			assert.Equal(t, tc.project, mockClient.BuildInfoRequests[0].ProjectKey)
			require.Len(t, mockClient.AqlQueries, 1)
			assert.Contains(t, mockClient.AqlQueries[0], `"repo":"`+tc.expectedRepoKey+`"`)
		})
	}
}

func TestVerifyEvidenceBuild_Run_Success(t *testing.T) {
	// Mock AQL response with build info file
	aqlResult := `{"results":[{"sha256":"test-sha256","name":"1-1234567890.json"}]}`

	// Mock Artifactory client
	mockClient := &MockArtifactoryServicesManagerBuild{
		AqlResponse: aqlResult,
		BuildRuns:   map[string]int64{"1": 1234567890},
	}

	// Mock OneModel client for evidence metadata
	mockOneModel := &MockOneModelManagerBuild{
		GraphqlResponse: []byte(`{"data":{"evidence":{"searchEvidence":{"edges":[{"node":{"subject":{"sha256":"test-sha256"},"downloadPath":"/evidence/path"}}]}}}}`),
	}

	// Create mock verifier using testify
	mockVerifier := new(MockVerifier)

	// Set up expectations for the mock verifier
	expectedResponse := &model.VerificationResponse{
		Subject: model.Subject{
			Path:   "/test/subject/path",
			Sha256: "test-sha256",
		},
		EvidenceVerifications:     &[]model.EvidenceVerification{},
		OverallVerificationStatus: model.Success,
	}
	mockVerifier.On("Verify", model.SubjectDigest{Type: model.Sha256DigestType, Value: "test-sha256"}, mock.AnythingOfType("*[]model.SearchEvidenceEdge"), mock.AnythingOfType("string")).Return(expectedResponse, nil)

	// Create build verifier with mock verifier
	buildVerifier := &verifyEvidenceBuild{
		verifyEvidenceBase: verifyEvidenceBase{
			serverDetails: &config.ServerDetails{},
			artifactoryClient: func() *artifactory.ArtifactoryServicesManager {
				c := artifactory.ArtifactoryServicesManager(mockClient)
				return &c
			}(),
			oneModelClient: mockOneModel,
			format:         "json",
			keys:           []string{},
			verifier:       mockVerifier, // Inject our mock verifier
		},
		project:     "test-project",
		buildName:   "test-build",
		buildNumber: "1",
	}

	// Call the actual Run method
	err := buildVerifier.Run()

	// Should succeed with our mock verifier returning VERIFIED status
	assert.NoError(t, err)

	// Verify that the mock verifier was called with expected parameters
	mockVerifier.AssertExpectations(t)
	mockVerifier.AssertCalled(t, "Verify", model.SubjectDigest{Type: model.Sha256DigestType, Value: "test-sha256"}, mock.AnythingOfType("*[]model.SearchEvidenceEdge"), mock.AnythingOfType("string"))
}

func TestVerifyEvidenceBuild_Run_VerificationFailed(t *testing.T) {
	// Mock AQL response with build info file
	aqlResult := `{"results":[{"sha256":"test-sha256","name":"1-1234567890.json"}]}`

	// Mock Artifactory client
	mockClient := &MockArtifactoryServicesManagerBuild{
		AqlResponse: aqlResult,
		BuildRuns:   map[string]int64{"1": 1234567890},
	}

	// Mock OneModel client for evidence metadata
	mockOneModel := &MockOneModelManagerBuild{
		GraphqlResponse: []byte(`{"data":{"evidence":{"searchEvidence":{"edges":[{"node":{"subject":{"sha256":"test-sha256"},"downloadPath":"/evidence/path"}}]}}}}`),
	}

	// Create mock verifier using testify
	mockVerifier := new(MockVerifier)

	// Set up expectations for the mock verifier to return FAILED status
	expectedResponse := &model.VerificationResponse{
		Subject: model.Subject{
			Path:   "/test/subject/path",
			Sha256: "test-sha256",
		},
		EvidenceVerifications:     &[]model.EvidenceVerification{},
		OverallVerificationStatus: model.Failed,
	}
	mockVerifier.On("Verify", model.SubjectDigest{Type: model.Sha256DigestType, Value: "test-sha256"}, mock.AnythingOfType("*[]model.SearchEvidenceEdge"), mock.AnythingOfType("string")).Return(expectedResponse, nil)

	// Create build verifier with mock verifier
	buildVerifier := &verifyEvidenceBuild{
		verifyEvidenceBase: verifyEvidenceBase{
			serverDetails: &config.ServerDetails{},
			artifactoryClient: func() *artifactory.ArtifactoryServicesManager {
				c := artifactory.ArtifactoryServicesManager(mockClient)
				return &c
			}(),
			oneModelClient: mockOneModel,
			format:         "json",
			keys:           []string{},
			verifier:       mockVerifier, // Inject our mock verifier
		},
		project:     "test-project",
		buildName:   "test-build",
		buildNumber: "1",
	}

	// Call the actual Run method
	err := buildVerifier.Run()

	// Should fail with ExitCodeFailNoOp due to FAILED verification status
	assert.Error(t, err)
	var cliErr coreutils.CliError
	assert.ErrorAs(t, err, &cliErr)
	assert.Equal(t, coreutils.ExitCodeError, cliErr.ExitCode)

	// Verify that the mock verifier was called with expected parameters
	mockVerifier.AssertExpectations(t)
	mockVerifier.AssertCalled(t, "Verify", model.SubjectDigest{Type: model.Sha256DigestType, Value: "test-sha256"}, mock.AnythingOfType("*[]model.SearchEvidenceEdge"), mock.AnythingOfType("string"))
}

func TestVerifyEvidenceBuild_Run_VerificationError(t *testing.T) {
	// Mock AQL response with build info file
	aqlResult := `{"results":[{"sha256":"test-sha256","name":"1-1234567890.json"}]}`

	// Mock Artifactory client
	mockClient := &MockArtifactoryServicesManagerBuild{
		AqlResponse: aqlResult,
		BuildRuns:   map[string]int64{"1": 1234567890},
	}

	// Mock OneModel client for evidence metadata
	mockOneModel := &MockOneModelManagerBuild{
		GraphqlResponse: []byte(`{"data":{"evidence":{"searchEvidence":{"edges":[{"node":{"subject":{"sha256":"test-sha256"},"downloadPath":"/evidence/path"}}]}}}}`),
	}

	// Create mock verifier using testify
	mockVerifier := new(MockVerifier)

	// Set up expectations for the mock verifier to return an error
	mockVerifier.On("Verify", model.SubjectDigest{Type: model.Sha256DigestType, Value: "test-sha256"}, mock.AnythingOfType("*[]model.SearchEvidenceEdge"), mock.AnythingOfType("string")).Return((*model.VerificationResponse)(nil), errors.New("verification error"))

	// Create build verifier with mock verifier
	buildVerifier := &verifyEvidenceBuild{
		verifyEvidenceBase: verifyEvidenceBase{
			serverDetails: &config.ServerDetails{},
			artifactoryClient: func() *artifactory.ArtifactoryServicesManager {
				c := artifactory.ArtifactoryServicesManager(mockClient)
				return &c
			}(),
			oneModelClient: mockOneModel,
			format:         "json",
			keys:           []string{},
			verifier:       mockVerifier, // Inject our mock verifier
		},
		project:     "test-project",
		buildName:   "test-build",
		buildNumber: "1",
	}

	// Call the actual Run method
	err := buildVerifier.Run()

	// Should return the verification error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "verification error")

	// Verify that the mock verifier was called with expected parameters
	mockVerifier.AssertExpectations(t)
	mockVerifier.AssertCalled(t, "Verify", model.SubjectDigest{Type: model.Sha256DigestType, Value: "test-sha256"}, mock.AnythingOfType("*[]model.SearchEvidenceEdge"), mock.AnythingOfType("string"))
}

type MockVerifierBuild struct{ mock.Mock }

func (m *MockVerifierBuild) Verify(expectedSubject model.SubjectDigest, evidenceMetadata *[]model.SearchEvidenceEdge, subjectPath string) (*model.VerificationResponse, error) {
	args := m.Called(expectedSubject, evidenceMetadata, subjectPath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	resp, _ := args.Get(0).(*model.VerificationResponse)
	return resp, args.Error(1)
}

func TestVerifyEvidenceBuild_Progress_Success(t *testing.T) {
	// Mock data
	aqlResult := `{"results":[{"sha256":"sha","name":"1-1234567890.json"}]}`
	mockClient := &MockArtifactoryServicesManagerBuild{AqlResponse: aqlResult, BuildRuns: map[string]int64{"1": 1234567890}}
	mockOneModel := &MockOneModelManagerBuild{GraphqlResponse: []byte(`{"data":{"evidence":{"searchEvidence":{"edges":[{"node":{"subject":{"sha256":"sha"},"downloadPath":"/evidence"}}]}}}}`)}
	mockVerifier := &MockVerifierBuild{}
	expected := &model.VerificationResponse{OverallVerificationStatus: model.Success, Subject: model.Subject{Path: "repo/build/file", Sha256: "sha"}}
	mockVerifier.On("Verify", model.SubjectDigest{Type: model.Sha256DigestType, Value: "sha"}, mock.AnythingOfType("*[]model.SearchEvidenceEdge"), mock.AnythingOfType("string")).Return(expected, nil)

	cmd := &verifyEvidenceBuild{
		verifyEvidenceBase: verifyEvidenceBase{
			serverDetails:  &config.ServerDetails{},
			format:         "json",
			verifier:       mockVerifier,
			oneModelClient: mockOneModel,
		},
		project:     "p",
		buildName:   "b",
		buildNumber: "1",
	}
	var clientInterface artifactory.ArtifactoryServicesManager = mockClient
	cmd.artifactoryClient = &clientInterface
	pm := &fakeProgress{}
	cmd.progressMgr = pm

	err := cmd.Run()
	assert.NoError(t, err)
	assert.True(t, pm.quitCalled)
	assert.True(t, len(pm.headlines) >= 1)
}

const (
	testBuildInfoRepo = "artifactory-build-info"
	testBuildName     = "test-build"
)

func buildInfoFile(name, sha256 string) *servicesUtils.ResultItem {
	return &servicesUtils.ResultItem{Repo: testBuildInfoRepo, Path: testBuildName, Name: name, Sha256: sha256}
}

// newBuildVerifierForTest returns a verifier for testBuildName in the default project, backed by the given mocks.
func newBuildVerifierForTest(client *MockArtifactoryServicesManagerBuild, verifier *MockVerifier, buildNumber string) *verifyEvidenceBuild {
	var clientInterface artifactory.ArtifactoryServicesManager = client
	return &verifyEvidenceBuild{
		verifyEvidenceBase: verifyEvidenceBase{
			serverDetails:     &config.ServerDetails{},
			artifactoryClient: &clientInterface,
			oneModelClient: &MockOneModelManagerBuild{
				GraphqlResponse: []byte(`{"data":{"evidence":{"searchEvidence":{"edges":[{"node":{"downloadPath":"/evidence/path"}}]}}}}`),
			},
			format:   "json",
			verifier: verifier,
		},
		buildName:   testBuildName,
		buildNumber: buildNumber,
	}
}

func TestVerifyEvidenceBuild_Run_SelectsExactBuildNumber(t *testing.T) {
	testCases := []struct {
		name               string
		buildRuns          map[string]int64
		buildInfoFiles     []*servicesUtils.ResultItem
		buildNumber        string
		expectedSubject    string
		expectedSha256     string
		expectedLookupName string
	}{
		{
			name:      "Build 4 is not confused with build 45",
			buildRuns: map[string]int64{"4": 1700000000000, "45": 1700000000045},
			buildInfoFiles: []*servicesUtils.ResultItem{
				buildInfoFile("4-1700000000000.json", "sha-4"),
				buildInfoFile("45-1700000000045.json", "sha-45"),
			},
			buildNumber:     "4",
			expectedSubject: "artifactory-build-info/test-build/4-1700000000000.json",
			expectedSha256:  "sha-4",
		},
		{
			name:      "Build 4 is not confused with build 4-rc",
			buildRuns: map[string]int64{"4": 1700000000000, "4-rc": 1700000000099},
			buildInfoFiles: []*servicesUtils.ResultItem{
				buildInfoFile("4-1700000000000.json", "sha-4"),
				buildInfoFile("4-rc-1700000000099.json", "sha-4-rc"),
			},
			buildNumber:     "4",
			expectedSubject: "artifactory-build-info/test-build/4-1700000000000.json",
			expectedSha256:  "sha-4",
		},
		{
			name:      "Build 4-rc is not confused with build 4",
			buildRuns: map[string]int64{"4": 1700000000000, "4-rc": 1700000000099},
			buildInfoFiles: []*servicesUtils.ResultItem{
				buildInfoFile("4-1700000000000.json", "sha-4"),
				buildInfoFile("4-rc-1700000000099.json", "sha-4-rc"),
			},
			buildNumber:     "4-rc",
			expectedSubject: "artifactory-build-info/test-build/4-rc-1700000000099.json",
			expectedSha256:  "sha-4-rc",
		},
		{
			name:      "Latest run of the build number is verified",
			buildRuns: map[string]int64{"4": 1800000000000},
			buildInfoFiles: []*servicesUtils.ResultItem{
				buildInfoFile("4-1700000000000.json", "sha-4-old"),
				buildInfoFile("4-1800000000000.json", "sha-4-latest"),
			},
			buildNumber:     "4",
			expectedSubject: "artifactory-build-info/test-build/4-1800000000000.json",
			expectedSha256:  "sha-4-latest",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &MockArtifactoryServicesManagerBuild{BuildRuns: tc.buildRuns, BuildInfoFiles: tc.buildInfoFiles}
			expectedDigest := model.SubjectDigest{Type: model.Sha256DigestType, Value: tc.expectedSha256}
			mockVerifier := new(MockVerifier)
			mockVerifier.On("Verify", expectedDigest, mock.AnythingOfType("*[]model.SearchEvidenceEdge"), tc.expectedSubject).
				Return(&model.VerificationResponse{OverallVerificationStatus: model.Success}, nil)

			err := newBuildVerifierForTest(mockClient, mockVerifier, tc.buildNumber).Run()

			assert.NoError(t, err)
			mockVerifier.AssertExpectations(t)
			require.Len(t, mockClient.BuildInfoRequests, 1)
			assert.Equal(t, tc.buildNumber, mockClient.BuildInfoRequests[0].BuildNumber)
			assert.Equal(t, testBuildName, mockClient.BuildInfoRequests[0].BuildName)
		})
	}
}

func TestVerifyEvidenceBuild_Run_OnlyLongerBuildNumberExists(t *testing.T) {
	// Only build 45 exists; verifying build 4 must not fall back to it.
	mockClient := &MockArtifactoryServicesManagerBuild{
		BuildRuns:      map[string]int64{"45": 1700000000045},
		BuildInfoFiles: []*servicesUtils.ResultItem{buildInfoFile("45-1700000000045.json", "sha-45")},
	}
	mockVerifier := new(MockVerifier)

	err := newBuildVerifierForTest(mockClient, mockVerifier, "4").Run()

	assert.EqualError(t, err, `no build found for build name "test-build" and build number "4"`)
	mockVerifier.AssertNotCalled(t, "Verify", mock.Anything, mock.Anything, mock.Anything)
}

func TestVerifyEvidenceBuild_Run_BuildInfoFileMissing(t *testing.T) {
	// The build-info API knows build 4, but its build-info file is not in the repository.
	// Only build 45's file is, and it must not be used instead.
	mockClient := &MockArtifactoryServicesManagerBuild{
		BuildRuns:      map[string]int64{"4": 1700000000000},
		BuildInfoFiles: []*servicesUtils.ResultItem{buildInfoFile("45-1700000000045.json", "sha-45")},
	}
	mockVerifier := new(MockVerifier)

	err := newBuildVerifierForTest(mockClient, mockVerifier, "4").Run()

	assert.EqualError(t, err, `no build found for build name "test-build" and build number "4": build-info file artifactory-build-info/test-build/4-1700000000000.json does not exist`)
	mockVerifier.AssertNotCalled(t, "Verify", mock.Anything, mock.Anything, mock.Anything)
}

func TestVerifyEvidenceBuild_Run_IgnoresAqlResultWithOtherName(t *testing.T) {
	// Even if Artifactory returned a file with another name, it must not be verified.
	mockClient := &MockArtifactoryServicesManagerBuild{
		BuildRuns:   map[string]int64{"4": 1700000000000},
		AqlResponse: `{"results":[{"repo":"artifactory-build-info","path":"test-build","name":"45-1700000000045.json","sha256":"sha-45"}]}`,
	}
	mockVerifier := new(MockVerifier)

	err := newBuildVerifierForTest(mockClient, mockVerifier, "4").Run()

	assert.ErrorContains(t, err, "no build found")
	mockVerifier.AssertNotCalled(t, "Verify", mock.Anything, mock.Anything, mock.Anything)
}

func TestVerifyEvidenceBuild_Run_GetBuildInfoError(t *testing.T) {
	mockClient := &MockArtifactoryServicesManagerBuild{GetBuildInfoError: errors.New("connection refused")}
	mockVerifier := new(MockVerifier)

	err := newBuildVerifierForTest(mockClient, mockVerifier, "4").Run()

	assert.ErrorContains(t, err, "connection refused")
	assert.NotContains(t, err.Error(), "no build found")
	assert.Empty(t, mockClient.AqlQueries)
}

func TestVerifyEvidenceBuild_Run_EscapesAqlValues(t *testing.T) {
	buildName := `build"},"name":{"$match":"*`
	buildNumber := `4"}).include("*`
	mockClient := &MockArtifactoryServicesManagerBuild{
		BuildRuns:   map[string]int64{buildNumber: 1700000000000},
		AqlResponse: `{"results":[]}`,
	}
	verifier := newBuildVerifierForTest(mockClient, new(MockVerifier), buildNumber)
	verifier.buildName = buildName

	err := verifier.Run()

	assert.ErrorContains(t, err, "no build found")
	require.Len(t, mockClient.AqlQueries, 1)
	expectedQuery := `items.find({"repo":"artifactory-build-info",` +
		`"path":"build\"},\"name\":{\"$match\":\"*",` +
		`"name":"4\"}).include(\"*-1700000000000.json"})` +
		`.include("repo","path","name","sha256")`
	assert.Equal(t, expectedQuery, mockClient.AqlQueries[0])
}
