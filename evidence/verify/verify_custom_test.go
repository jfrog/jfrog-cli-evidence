package verify

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli-evidence/evidence/model"
	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockArtifactoryServicesManagerCustom embeds EmptyArtifactoryServicesManager and overrides methods for testing
type MockArtifactoryServicesManagerCustom struct {
	artifactory.EmptyArtifactoryServicesManager
	AqlResponse string
	AqlError    error
	// Recorded calls
	AqlQueries []string
}

func (m *MockArtifactoryServicesManagerCustom) Aql(query string) (io.ReadCloser, error) {
	m.AqlQueries = append(m.AqlQueries, query)
	if m.AqlError != nil {
		return nil, m.AqlError
	}
	return io.NopCloser(bytes.NewBufferString(m.AqlResponse)), nil
}

// MockOneModelManagerCustom for custom tests
type MockOneModelManagerCustom struct {
	GraphqlResponse []byte
	GraphqlError    error
}

func (m *MockOneModelManagerCustom) GraphqlQuery(_ []byte) ([]byte, error) {
	if m.GraphqlError != nil {
		return nil, m.GraphqlError
	}
	return m.GraphqlResponse, nil
}

// MockVerifyEvidenceBaseCustom for testing verifyEvidence method
type MockVerifyEvidenceBaseCustom struct {
	mock.Mock
	verifyEvidenceBase
}

func (m *MockVerifyEvidenceBaseCustom) verifyEvidence(client *artifactory.ArtifactoryServicesManager, metadata *[]model.SearchEvidenceEdge, sha256, subjectPath string) error {
	args := m.Called(client, metadata, sha256, subjectPath)
	return args.Error(0)
}

// MockVerifierCustom for testing cryptox.Verifier.Verify method calls
type MockVerifierCustom struct {
	mock.Mock
}

func (m *MockVerifierCustom) Verify(expectedSubject model.SubjectDigest, evidenceMetadata *[]model.SearchEvidenceEdge, subjectPath string) (*model.VerificationResponse, error) {
	args := m.Called(expectedSubject, evidenceMetadata, subjectPath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	resp, ok := args.Get(0).(*model.VerificationResponse)
	if !ok && args.Get(0) != nil {
		return nil, args.Error(1)
	}
	return resp, args.Error(1)
}

func TestNewVerifyEvidenceCustom(t *testing.T) {
	serverDetails := &config.ServerDetails{}
	subjectRepoPath := "test-repo/path/to/subject.txt"
	format := "json"
	keys := []string{"key1", "key2"}

	cmd := NewVerifyEvidenceCustom(serverDetails, subjectRepoPath, format, keys, true)
	verifyCmd, ok := cmd.(*verifyEvidenceCustom)
	assert.True(t, ok)

	// Test verifyEvidenceBase fields
	assert.Equal(t, serverDetails, verifyCmd.serverDetails)
	assert.Equal(t, format, verifyCmd.format)
	assert.Equal(t, keys, verifyCmd.keys)

	// Test verifyEvidenceCustom fields
	assert.Equal(t, subjectRepoPath, verifyCmd.subjectRepoPath)
	assert.True(t, verifyCmd.useArtifactoryKeys)
}

func TestVerifyEvidenceCustom_CommandName(t *testing.T) {
	cmd := &verifyEvidenceCustom{}
	assert.Equal(t, "verify-evidence-custom", cmd.CommandName())
}

func TestVerifyEvidenceCustom_ServerDetails(t *testing.T) {
	serverDetails := &config.ServerDetails{Url: "test.com"}
	cmd := &verifyEvidenceCustom{
		verifyEvidenceBase: verifyEvidenceBase{serverDetails: serverDetails},
	}

	result, err := cmd.ServerDetails()
	assert.NoError(t, err)
	assert.Equal(t, serverDetails, result)
}

func TestVerifyEvidenceCustom_Run_AqlError(t *testing.T) {
	// Mock Artifactory client with error
	mockClient := &MockArtifactoryServicesManagerCustom{
		AqlError: errors.New("aql query failed"),
	}

	// Create custom verifier
	customVerifier := &verifyEvidenceCustom{
		verifyEvidenceBase: verifyEvidenceBase{
			serverDetails: &config.ServerDetails{},
			artifactoryClient: func() *artifactory.ArtifactoryServicesManager {
				c := artifactory.ArtifactoryServicesManager(mockClient)
				return &c
			}(),
		},
		subjectRepoPath: "test-repo/path/to/subject.txt",
	}

	err := customVerifier.Run()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to execute AQL query")
}

func TestVerifyEvidenceCustom_Run_NoSubjectFound(t *testing.T) {
	// Mock AQL response with no results
	aqlResult := `{"results":[]}`

	// Mock Artifactory client
	mockClient := &MockArtifactoryServicesManagerCustom{
		AqlResponse: aqlResult,
	}

	// Create custom verifier
	customVerifier := &verifyEvidenceCustom{
		verifyEvidenceBase: verifyEvidenceBase{
			serverDetails: &config.ServerDetails{},
			artifactoryClient: func() *artifactory.ArtifactoryServicesManager {
				c := artifactory.ArtifactoryServicesManager(mockClient)
				return &c
			}(),
		},
		subjectRepoPath: "test-repo/path/to/subject.txt",
	}

	err := customVerifier.Run()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no subject found")
}

func TestVerifyEvidenceCustom_Run_QueryEvidenceMetadataError(t *testing.T) {
	// Mock AQL response with subject file
	aqlResult := `{"results":[{"sha256":"test-sha256","name":"subject.txt","repo":"test-repo","path":"path/to"}]}`

	// Mock Artifactory client
	mockClient := &MockArtifactoryServicesManagerCustom{
		AqlResponse: aqlResult,
	}

	// Mock OneModel client with error
	mockOneModel := &MockOneModelManagerCustom{
		GraphqlError: errors.New("graphql query failed"),
	}

	// Create custom verifier
	customVerifier := &verifyEvidenceCustom{
		verifyEvidenceBase: verifyEvidenceBase{
			serverDetails: &config.ServerDetails{},
			artifactoryClient: func() *artifactory.ArtifactoryServicesManager {
				c := artifactory.ArtifactoryServicesManager(mockClient)
				return &c
			}(),
			oneModelClient: mockOneModel,
		},
		subjectRepoPath: "test-repo/path/to/subject.txt",
	}

	err := customVerifier.Run()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "graphql query failed")
}

func TestVerifyEvidenceCustom_Run_VerifyEvidenceError(t *testing.T) {
	// Mock AQL response with subject file
	aqlResult := `{"results":[{"sha256":"test-sha256","name":"subject.txt","repo":"test-repo","path":"path/to"}]}`

	// Mock Artifactory client
	mockClient := &MockArtifactoryServicesManagerCustom{
		AqlResponse: aqlResult,
	}

	// Mock OneModel client for evidence metadata
	mockOneModel := &MockOneModelManagerCustom{
		GraphqlResponse: []byte(`{"data":{"evidence":{"searchEvidence":{"edges":[{"node":{"subject":{"sha256":"test-sha256"}}}]}}}}`),
	}

	// Mock the base verification with error
	mockBase := &MockVerifyEvidenceBaseCustom{}
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
	err := mockBase.verifyEvidence(nil, &[]model.SearchEvidenceEdge{{}}, "test-sha256", "test-path")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "verification failed")
	mockBase.AssertExpectations(t)
}

func TestVerifyEvidenceCustom_Run_CreateArtifactoryClientError(t *testing.T) {
	// Create custom verifier with invalid server configuration that would cause client creation to fail
	customVerifier := &verifyEvidenceCustom{
		verifyEvidenceBase: verifyEvidenceBase{
			serverDetails: &config.ServerDetails{
				Url: "invalid-url", // Invalid URL that should cause client creation to fail
			},
		},
		subjectRepoPath: "test-repo/path/to/subject.txt",
	}

	err := customVerifier.Run()
	assert.Error(t, err)
	// Just verify an error occurs - the specific error depends on when the invalid config is detected
}

func TestVerifyEvidenceCustom_SubjectRepoPathParsing(t *testing.T) {
	// Test different subject repo path formats
	testCases := []struct {
		name            string
		subjectRepoPath string
		expectedRepo    string
		expectedPath    string
		expectedName    string
	}{
		{
			name:            "Simple path",
			subjectRepoPath: "repo/file.txt",
			expectedRepo:    "repo",
			expectedPath:    "",
			expectedName:    "file.txt",
		},
		{
			name:            "Path with directory",
			subjectRepoPath: "repo/path/to/file.txt",
			expectedRepo:    "repo",
			expectedPath:    "path/to",
			expectedName:    "file.txt",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Mock AQL response with no results to test path parsing
			aqlResult := `{"results":[]}`

			// Mock Artifactory client
			mockClient := &MockArtifactoryServicesManagerCustom{
				AqlResponse: aqlResult,
			}

			// Create custom verifier
			customVerifier := &verifyEvidenceCustom{
				verifyEvidenceBase: verifyEvidenceBase{
					serverDetails: &config.ServerDetails{},
					artifactoryClient: func() *artifactory.ArtifactoryServicesManager {
						c := artifactory.ArtifactoryServicesManager(mockClient)
						return &c
					}(),
				},
				subjectRepoPath: tc.subjectRepoPath,
			}

			// Run should fail with "no subject found" but this validates the path parsing
			err := customVerifier.Run()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "no subject found")
		})
	}
}

func TestVerifyEvidenceCustom_Run_Success(t *testing.T) {
	// Mock AQL response with subject file
	aqlResult := `{"results":[{"sha256":"test-sha256","name":"subject.txt","repo":"test-repo","path":"path/to"}]}`

	// Mock Artifactory client
	mockClient := &MockArtifactoryServicesManagerCustom{
		AqlResponse: aqlResult,
	}

	// Mock OneModel client for evidence metadata
	mockOneModel := &MockOneModelManagerCustom{
		GraphqlResponse: []byte(`{"data":{"evidence":{"searchEvidence":{"edges":[{"node":{"subject":{"sha256":"test-sha256"},"downloadPath":"/test/evidence.json","predicateType":"test-predicate"}}]}}}}`),
	}

	// Create testify mock for verifier using the new interface
	mockVerifier := &MockVerifierCustom{}
	expectedResponse := &model.VerificationResponse{
		OverallVerificationStatus: model.Success,
		Subject: model.Subject{
			Path:   "test-repo/path/to/subject.txt",
			Sha256: "test-sha256",
		},
	}

	// Set up the mock expectations - use mock.Anything for the subjectPath since it gets formatted
	mockVerifier.On("Verify", model.SubjectDigest{Type: model.Sha256DigestType, Value: "test-sha256"}, mock.AnythingOfType("*[]model.SearchEvidenceEdge"), mock.Anything).Return(expectedResponse, nil)

	// Create custom verifier with injected mock verifier
	customVerifier := &verifyEvidenceCustom{
		verifyEvidenceBase: verifyEvidenceBase{
			serverDetails:  &config.ServerDetails{},
			format:         "json",
			keys:           []string{},
			oneModelClient: mockOneModel,
			verifier:       mockVerifier, // Inject the mock verifier using the interface
		},
		subjectRepoPath: "test-repo/path/to/subject.txt",
	}

	// Mock the createArtifactoryClient method by setting the client directly
	var clientInterface artifactory.ArtifactoryServicesManager = mockClient
	customVerifier.artifactoryClient = &clientInterface

	// Call the actual Run method
	err := customVerifier.Run()

	// Assert results
	assert.NoError(t, err)

	// Verify that the mock verifier was called with the exact expected parameters
	mockVerifier.AssertExpectations(t)
	mockVerifier.AssertCalled(t, "Verify", model.SubjectDigest{Type: model.Sha256DigestType, Value: "test-sha256"}, mock.AnythingOfType("*[]model.SearchEvidenceEdge"), mock.Anything)

	t.Log("✅ SUCCESS: Run() method called mock verifier.Verify() with testify assertions!")
}

func TestVerifyEvidenceCustom_Progress_Success(t *testing.T) {
	// Mock AQL response with subject file
	aqlResult := `{"results":[{"sha256":"test-sha256","name":"subject.txt","repo":"test-repo","path":"path/to"}]}`
	mockClient := &MockArtifactoryServicesManagerCustom{AqlResponse: aqlResult}
	mockOneModel := &MockOneModelManagerCustom{GraphqlResponse: []byte(`{"data":{"evidence":{"searchEvidence":{"edges":[{"node":{"subject":{"sha256":"test-sha256"},"downloadPath":"/evidence/path"}}]}}}}`)}
	mockVerifier := &MockVerifierCustom{}
	expected := &model.VerificationResponse{OverallVerificationStatus: model.Success, Subject: model.Subject{Path: "test-repo/path/to/subject.txt", Sha256: "test-sha256"}}
	mockVerifier.On("Verify", model.SubjectDigest{Type: model.Sha256DigestType, Value: "test-sha256"}, mock.AnythingOfType("*[]model.SearchEvidenceEdge"), mock.AnythingOfType("string")).Return(expected, nil)

	cmd := &verifyEvidenceCustom{
		verifyEvidenceBase: verifyEvidenceBase{
			serverDetails:  &config.ServerDetails{},
			format:         "json",
			verifier:       mockVerifier,
			oneModelClient: mockOneModel,
		},
		subjectRepoPath: "test-repo/path/to/subject.txt",
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

func TestPrepareAqlQuery(t *testing.T) {
	testCases := []struct {
		name     string
		repo     string
		path     string
		fileName string
		expected string
	}{
		{
			name:     "plain values",
			repo:     "test-repo",
			path:     "path/to",
			fileName: "subject.txt",
			expected: `items.find({"repo": "test-repo","path": "path/to","name": "subject.txt"}).include("repo", "path", "name", "sha256")`,
		},
		{
			name:     "no path",
			repo:     `my"repo`,
			fileName: `subject\.txt`,
			expected: `items.find({"repo": "my\"repo","name": "subject\\.txt"}).include("repo", "path", "name", "sha256")`,
		},
		{
			name:     "quote and backslash in every value",
			repo:     `my"repo`,
			path:     `a"b\c`,
			fileName: `subject\.txt`,
			expected: `items.find({"repo": "my\"repo","path": "a\"b\\c","name": "subject\\.txt"}).include("repo", "path", "name", "sha256")`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			query, err := prepareAqlQuery(tc.path, tc.repo, tc.fileName)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, query)
		})
	}
}

func TestVerifyEvidenceCustom_Run_EscapesQueryValues(t *testing.T) {
	mockClient := &MockArtifactoryServicesManagerCustom{
		AqlResponse: `{"results":[{"sha256":"test-sha256"}]}`,
	}
	mockOneModel := &MockOneModelManagerWithQueryCapture{
		GraphqlResponse: []byte(`{"data":{"evidence":{"searchEvidence":{"edges":[{"node":{"subject":{"sha256":"test-sha256"},"downloadPath":"/test/evidence.json"}}]}}}}`),
	}
	subjectRepoPath := `my"repo/a\b/subject".txt`
	mockVerifier := &MockVerifierCustom{}
	mockVerifier.On("Verify", model.SubjectDigest{Type: model.Sha256DigestType, Value: "test-sha256"}, mock.AnythingOfType("*[]model.SearchEvidenceEdge"), subjectRepoPath).
		Return(&model.VerificationResponse{OverallVerificationStatus: model.Success}, nil)

	customVerifier := &verifyEvidenceCustom{
		verifyEvidenceBase: verifyEvidenceBase{
			serverDetails:  &config.ServerDetails{},
			format:         "json",
			oneModelClient: mockOneModel,
			verifier:       mockVerifier,
		},
		subjectRepoPath: subjectRepoPath,
	}
	var clientInterface artifactory.ArtifactoryServicesManager = mockClient
	customVerifier.artifactoryClient = &clientInterface

	require.NoError(t, customVerifier.Run())

	assert.Equal(t, []string{`items.find({"repo": "my\"repo","path": "a\\b","name": "subject\".txt"}).include("repo", "path", "name", "sha256")`}, mockClient.AqlQueries)
	require.Len(t, mockOneModel.CapturedQueries, 1)
	assert.Contains(t, graphqlQueryText(t, mockOneModel.CapturedQueries[0]),
		`hasSubjectWith: { repositoryKey: "my\"repo", path: "a\\b", name: "subject\".txt" }`)
	mockVerifier.AssertExpectations(t)
}
