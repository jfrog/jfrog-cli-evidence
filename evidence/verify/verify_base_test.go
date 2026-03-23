package verify

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli-core/v2/utils/coreutils"
	"github.com/jfrog/jfrog-cli-evidence/evidence/model"
	"github.com/jfrog/jfrog-cli-evidence/evidence/verify/reports"
	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/onemodel"
	ioUtils "github.com/jfrog/jfrog-client-go/utils/io"
	"github.com/stretchr/testify/assert"
)

// MockOneModelManagerBase for base tests
type MockOneModelManagerBase struct {
	GraphqlResponse []byte
	GraphqlError    error
}

func (m *MockOneModelManagerBase) GraphqlQuery(_ []byte) ([]byte, error) {
	if m.GraphqlError != nil {
		return nil, m.GraphqlError
	}
	return m.GraphqlResponse, nil
}

// MockOneModelManagerWithQueryCapture captures the GraphQL query for testing
type MockOneModelManagerWithQueryCapture struct {
	GraphqlResponse []byte
	GraphqlError    error
	GraphqlErrors   []error
	CapturedQuery   []byte
	CapturedQueries [][]byte
	callIndex       int
}

func (m *MockOneModelManagerWithQueryCapture) GraphqlQuery(query []byte) ([]byte, error) {
	m.CapturedQuery = query
	m.CapturedQueries = append(m.CapturedQueries, query)
	if m.callIndex < len(m.GraphqlErrors) && m.GraphqlErrors[m.callIndex] != nil {
		err := m.GraphqlErrors[m.callIndex]
		m.callIndex++
		return nil, err
	}
	m.callIndex++
	if m.GraphqlError != nil {
		return nil, m.GraphqlError
	}
	return m.GraphqlResponse, nil
}

// Satisfy interface for onemodel.Manager
var _ onemodel.Manager = (*MockOneModelManagerBase)(nil)

// MockEvidenceVerifier implements the EvidenceVerifierInterface for unit testing verifyEvidence
type MockEvidenceVerifier struct {
	Result   *model.VerificationResponse
	Err      error
	LastSha  string
	LastMeta *[]model.SearchEvidenceEdge
	LastPath string
}

func (m *MockEvidenceVerifier) Verify(subjectSha256 string, evidenceMetadata *[]model.SearchEvidenceEdge, subjectPath string) (*model.VerificationResponse, error) {
	m.LastSha = subjectSha256
	m.LastMeta = evidenceMetadata
	m.LastPath = subjectPath
	return m.Result, m.Err
}

func TestVerifyEvidenceBase_PrintVerifyResult_JSON(t *testing.T) {
	v := &verifyEvidenceBase{format: "json"}
	resp := &model.VerificationResponse{
		Subject: model.Subject{
			Sha256: "test-checksum",
		},
		OverallVerificationStatus: model.Success,
	}

	// For JSON output, just test that it doesn't return an error
	// since fmt.Println writes to stdout which we can't easily capture in tests
	err := v.printVerifyResult(resp)
	assert.NoError(t, err)
}

func TestVerifyEvidenceBase_PrintVerifyResult_Text_Success(t *testing.T) {
	v := &verifyEvidenceBase{format: "text"}
	resp := &model.VerificationResponse{
		OverallVerificationStatus: model.Success,
		EvidenceVerifications: &[]model.EvidenceVerification{{
			PredicateType: "test-type",
			CreatedBy:     "test-user",
			CreatedAt:     "2024-01-01T00:00:00Z",
			VerificationResult: model.EvidenceVerificationResult{
				Sha256VerificationStatus:     model.Success,
				SignaturesVerificationStatus: model.Success,
			},
		}},
	}

	// Test that the print function executes without error for successful verification
	err := v.printVerifyResult(resp)
	assert.NoError(t, err)
}

func TestVerifyEvidenceBase_UnknownFormat_DefaultsToText(t *testing.T) {
	v := &verifyEvidenceBase{format: "unknown"}
	resp := &model.VerificationResponse{
		OverallVerificationStatus: model.Success,
		EvidenceVerifications: &[]model.EvidenceVerification{{
			PredicateType: "test-type",
			CreatedBy:     "test-user",
			CreatedAt:     "2024-01-01T00:00:00Z",
			VerificationResult: model.EvidenceVerificationResult{
				Sha256VerificationStatus:     model.Success,
				SignaturesVerificationStatus: model.Success,
			},
		}},
	}

	// Test that unknown format defaults to text and executes without error
	err := v.printVerifyResult(resp)
	assert.NoError(t, err)
}

func TestVerifyEvidenceBase_CreateArtifactoryClient_Success(t *testing.T) {
	serverDetails := &config.ServerDetails{Url: "test.com"}
	v := &verifyEvidenceBase{serverDetails: serverDetails}

	// First call should create client
	client1, err := v.createArtifactoryClient()
	assert.NoError(t, err)
	assert.NotNil(t, client1)

	// Second call should return cached client
	client2, err := v.createArtifactoryClient()
	assert.NoError(t, err)
	assert.Equal(t, client1, client2)
}

func TestVerifyEvidenceBase_CreateArtifactoryClient_Error(t *testing.T) {
	// Test with invalid server configuration
	v := &verifyEvidenceBase{
		serverDetails: &config.ServerDetails{
			Url: "invalid-url", // Invalid URL that should cause client creation to fail
		},
	}

	// Client creation might succeed but subsequent operations would fail
	// Let's test that it doesn't panic and that we can call it
	client, err := v.createArtifactoryClient()
	// The behavior may vary - either it fails immediately or succeeds but fails later
	if err != nil {
		assert.Contains(t, err.Error(), "failed to create Artifactory client")
	} else {
		// If it succeeds, just verify we got a client
		assert.NotNil(t, client)
	}
}
func TestVerifyEvidenceBase_QueryEvidenceMetadata_SuccessWithPublicKey(t *testing.T) {
	mockManager := &MockOneModelManagerWithQueryCapture{
		GraphqlResponse: []byte(`{"data":{"evidence":{"searchEvidence":{"edges":[{"cursor":"c","node":{"downloadPath":"p","predicateType":"t","predicateCategory":"cat","createdAt":"now","createdBy":"me","subject":{"sha256":"abc"},"signingKey":{"alias":"a"}}}]}}}}`),
	}

	v := &verifyEvidenceBase{
		oneModelClient:     mockManager,
		useArtifactoryKeys: true,
	}
	edges, err := v.queryEvidenceMetadata("test-repo", "test/path", "test-file.txt")
	assert.NoError(t, err)

	if assert.NotNil(t, edges) && assert.Len(t, *edges, 1) {
		node := (*edges)[0].Node
		assert.Equal(t, "p", node.DownloadPath)
		assert.Equal(t, "t", node.PredicateType)
		assert.Equal(t, "cat", node.PredicateCategory)
		assert.Equal(t, "now", node.CreatedAt)
		assert.Equal(t, "me", node.CreatedBy)
		assert.Equal(t, "abc", node.Subject.Sha256)
		assert.Equal(t, "a", node.SigningKey.Alias)
	}
}

func TestVerifyEvidenceBase_QueryEvidenceMetadata_SuccessWithoutPublicKey(t *testing.T) {
	mockManager := &MockOneModelManagerWithQueryCapture{
		GraphqlResponse: []byte(`{"data":{"evidence":{"searchEvidence":{"edges":[{"cursor":"c","node":{"downloadPath":"p","predicateType":"t","predicateCategory":"cat","createdAt":"now","createdBy":"me","subject":{"sha256":"abc"},"signingKey":{"alias":"a"}}}]}}}}`),
	}

	v := &verifyEvidenceBase{
		oneModelClient:     mockManager,
		useArtifactoryKeys: false,
	}
	edges, err := v.queryEvidenceMetadata("test-repo", "test/path", "test-file.txt")
	assert.NoError(t, err)

	if assert.NotNil(t, edges) && assert.Len(t, *edges, 1) {
		edge := (*edges)[0]
		assert.Equal(t, "p", edge.Node.DownloadPath)
		assert.Equal(t, "t", edge.Node.PredicateType)
		assert.Equal(t, "abc", edge.Node.Subject.Sha256)
	}
}

func TestVerifyEvidenceBase_QueryEvidenceMetadata_GraphqlError(t *testing.T) {
	mockManager := &MockOneModelManagerWithQueryCapture{
		GraphqlError: errors.New("graphql query failed"),
	}

	v := &verifyEvidenceBase{oneModelClient: mockManager}
	_, err := v.queryEvidenceMetadata("test-repo", "test/path", "test-file.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error querying evidence from One-Model service")
}

func TestVerifyEvidenceBase_QueryEvidenceMetadata_UnmarshalError(t *testing.T) {
	mockManager := &MockOneModelManagerWithQueryCapture{
		GraphqlResponse: []byte("invalid json"),
	}

	v := &verifyEvidenceBase{oneModelClient: mockManager}
	_, err := v.queryEvidenceMetadata("test-repo", "test/path", "test-file.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal evidence metadata")
}

func TestVerifyEvidenceBase_QueryEvidenceMetadata_NoEdges(t *testing.T) {
	mockManager := &MockOneModelManagerWithQueryCapture{
		GraphqlResponse: []byte(`{"data":{"evidence":{"searchEvidence":{"edges":[]}}}}`),
	}

	v := &verifyEvidenceBase{oneModelClient: mockManager}
	_, err := v.queryEvidenceMetadata("test-repo", "test/path", "test-file.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no evidence found for the given subject")
}

func TestVerifyEvidenceBase_QueryEvidenceMetadata_CreateOneModelClient(t *testing.T) {
	// Test case where oneModelClient is nil and needs to be created
	v := &verifyEvidenceBase{
		serverDetails:  &config.ServerDetails{Url: "test.com"},
		oneModelClient: nil,
	}

	// This should fail when trying to query GraphQL with basic server config
	_, err := v.queryEvidenceMetadata("test-repo", "test/path", "test-file.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error querying evidence from One-Model service")
}

func TestVerifyEvidenceBase_SearchEvidenceQueryExactMatch(t *testing.T) {
	v := &verifyEvidenceBase{useArtifactoryKeys: true}
	builtQuery := v.buildSearchEvidenceQuery(true)
	expectedQuery := `{"query":"{ evidence { searchEvidence( where: { hasSubjectWith: { repositoryKey: \"%s\", path: \"%s\", name: \"%s\" }} ) { edges { cursor node { downloadPath predicateType createdAt createdBy subject { sha256 } attachments { name sha256 type downloadPath } signingKey {alias, publicKey} } } } } }"}`

	assert.Equal(t, expectedQuery, builtQuery,
		"Built query with publicKey+attachments has been modified. "+
			"If this change is intentional, please update this test.")

	formattedQuery := fmt.Sprintf(builtQuery, "test-repo", "test/path", "test-file.txt")
	assert.Contains(t, formattedQuery, "test-repo")
	assert.Contains(t, formattedQuery, "test/path")
	assert.Contains(t, formattedQuery, "test-file.txt")

	var jsonCheck any
	err := json.Unmarshal([]byte(formattedQuery), &jsonCheck)
	assert.NoError(t, err, "Formatted query should be valid JSON")
}

func TestVerifyEvidenceBase_SearchEvidenceQueryWithoutAttachments(t *testing.T) {
	v := &verifyEvidenceBase{useArtifactoryKeys: true}
	builtQuery := v.buildSearchEvidenceQuery(false)

	assert.NotContains(t, builtQuery, "attachments")
	assert.Contains(t, builtQuery, "signingKey {alias, publicKey}")

	formattedQuery := fmt.Sprintf(builtQuery, "test-repo", "test/path", "test-file.txt")
	var jsonCheck any
	err := json.Unmarshal([]byte(formattedQuery), &jsonCheck)
	assert.NoError(t, err, "Formatted query should be valid JSON")
}

func TestVerifyEvidenceBase_SearchEvidenceQueryWithoutPublicKey(t *testing.T) {
	v := &verifyEvidenceBase{useArtifactoryKeys: false}
	builtQuery := v.buildSearchEvidenceQuery(true)

	assert.Contains(t, builtQuery, "attachments")
	assert.NotContains(t, builtQuery, "publicKey")
	assert.NotContains(t, builtQuery, "signingKey")

	formattedQuery := fmt.Sprintf(builtQuery, "test-repo", "test/path", "test-file.txt")
	var jsonCheck any
	err := json.Unmarshal([]byte(formattedQuery), &jsonCheck)
	assert.NoError(t, err, "Formatted query should be valid JSON")
}

func TestVerifyEvidenceBase_Integration(t *testing.T) {
	// Test the integration of verifyEvidenceBase components
	v := &verifyEvidenceBase{
		serverDetails: &config.ServerDetails{Url: "test.com"},
		format:        "json",
		keys:          []string{"key1"},
	}

	// Verify the structure is correct
	assert.Equal(t, "test.com", v.serverDetails.Url)
	assert.Equal(t, "json", v.format)
	assert.Equal(t, []string{"key1"}, v.keys)
	assert.Nil(t, v.artifactoryClient)
	assert.Nil(t, v.oneModelClient)
}

func TestVerifyEvidenceBase_MultipleFormats(t *testing.T) {
	// Test different format scenarios
	testCases := []struct {
		name   string
		format string
	}{
		{
			name:   "JSON format",
			format: "json",
		},
		{
			name:   "Default format",
			format: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			v := &verifyEvidenceBase{format: tc.format}
			resp := &model.VerificationResponse{
				OverallVerificationStatus: model.Success,
				EvidenceVerifications: &[]model.EvidenceVerification{{
					PredicateType: "test-type",
					CreatedBy:     "test-user",
					CreatedAt:     "2024-01-01T00:00:00Z",
					VerificationResult: model.EvidenceVerificationResult{
						SignaturesVerificationStatus: model.Success,
					},
				}},
			}

			err := v.printVerifyResult(resp)
			assert.NoError(t, err)
		})
	}
}

func TestVerifyEvidenceBase_QueryEvidenceMetadata_QueryContainsPublicKey_WhenUseArtifactoryKeysTrue(t *testing.T) {
	mockManager := &MockOneModelManagerWithQueryCapture{
		GraphqlResponse: []byte(`{"data":{"evidence":{"searchEvidence":{"edges":[{"cursor":"c","node":{"downloadPath":"p","predicateType":"t","createdAt":"now","createdBy":"me","subject":{"sha256":"abc"},"signingKey":{"alias":"a","publicKey":"test-key"}}}]}}}}`),
	}

	v := &verifyEvidenceBase{
		oneModelClient:     mockManager,
		useArtifactoryKeys: true,
	}
	_, err := v.queryEvidenceMetadata("test-repo", "test/path", "test-file.txt")
	assert.NoError(t, err)

	// Verify that the captured query contains publicKey
	capturedQuery := string(mockManager.CapturedQuery)
	assert.Contains(t, capturedQuery, "publicKey", "Query should contain publicKey when useArtifactoryKeys is true")
	assert.Contains(t, capturedQuery, "signingKey", "Query should contain signingKey when useArtifactoryKeys is true")
}

func TestVerifyEvidenceBase_QueryEvidenceMetadata_QueryContainsPublicKey_WhenUseArtifactoryKeysFalse(t *testing.T) {
	mockManager := &MockOneModelManagerWithQueryCapture{
		GraphqlResponse: []byte(`{"data":{"evidence":{"searchEvidence":{"edges":[{"cursor":"c","node":{"downloadPath":"p","predicateType":"t","createdAt":"now","createdBy":"me","subject":{"sha256":"abc"}}}]}}}}`),
	}

	v := &verifyEvidenceBase{
		oneModelClient:     mockManager,
		useArtifactoryKeys: false,
	}
	_, err := v.queryEvidenceMetadata("test-repo", "test/path", "test-file.txt")
	assert.NoError(t, err)

	// Verify that the captured query does NOT contain publicKey or signingKey
	capturedQuery := string(mockManager.CapturedQuery)
	assert.NotContains(t, capturedQuery, "publicKey", "Query should NOT contain publicKey when useArtifactoryKeys is false")
	assert.NotContains(t, capturedQuery, "signingKey", "Query should NOT contain signingKey when useArtifactoryKeys is false")
}

func TestVerifyEvidenceBase_QueryEvidenceMetadata_QueryStructure_WithPublicKey(t *testing.T) {
	mockManager := &MockOneModelManagerWithQueryCapture{
		GraphqlResponse: []byte(`{"data":{"evidence":{"searchEvidence":{"edges":[{"cursor":"c","node":{"downloadPath":"p","predicateType":"t","createdAt":"now","createdBy":"me","subject":{"sha256":"abc"},"signingKey":{"alias":"a","publicKey":"test-key"}}}]}}}}`),
	}

	v := &verifyEvidenceBase{
		oneModelClient:     mockManager,
		useArtifactoryKeys: true,
	}
	_, err := v.queryEvidenceMetadata("test-repo", "test/path", "test-file.txt")
	assert.NoError(t, err)

	// Verify the query structure and parameters
	capturedQuery := string(mockManager.CapturedQuery)
	assert.Contains(t, capturedQuery, "test-repo", "Query should contain the repository parameter")
	assert.Contains(t, capturedQuery, "test/path", "Query should contain the path parameter")
	assert.Contains(t, capturedQuery, "test-file.txt", "Query should contain the name parameter")

	// Verify the GraphQL structure includes signingKey with publicKey
	assert.Contains(t, capturedQuery, "signingKey {alias, publicKey}", "Query should request signingKey with alias and publicKey")
}

func TestVerifyEvidenceBase_QueryEvidenceMetadata_QueryStructure_WithoutPublicKey(t *testing.T) {
	mockManager := &MockOneModelManagerWithQueryCapture{
		GraphqlResponse: []byte(`{"data":{"evidence":{"searchEvidence":{"edges":[{"cursor":"c","node":{"downloadPath":"p","predicateType":"t","createdAt":"now","createdBy":"me","subject":{"sha256":"abc"}}}]}}}}`),
	}

	v := &verifyEvidenceBase{
		oneModelClient:     mockManager,
		useArtifactoryKeys: false,
	}
	_, err := v.queryEvidenceMetadata("test-repo", "test/path", "test-file.txt")
	assert.NoError(t, err)

	// Verify the query structure and parameters
	capturedQuery := string(mockManager.CapturedQuery)
	assert.Contains(t, capturedQuery, "test-repo", "Query should contain the repository parameter")
	assert.Contains(t, capturedQuery, "test/path", "Query should contain the path parameter")
	assert.Contains(t, capturedQuery, "test-file.txt", "Query should contain the name parameter")

	// Verify the GraphQL structure does NOT include signingKey with publicKey
	assert.NotContains(t, capturedQuery, "signingKey {alias, publicKey}", "Query should NOT request signingKey with alias and publicKey when useArtifactoryKeys is false")
}

func TestVerifyEvidenceBase_QueryEvidenceMetadata_GraphqlValidationError_PublicKey(t *testing.T) {
	// Mock the GraphQL validation error for publicKey field
	graphqlError := fmt.Errorf(`{"errors":[{"message":"Cannot query field \"publicKey\" on type \"EvidenceSigningKey\"."}]}`)

	mockManager := &MockOneModelManagerWithQueryCapture{
		GraphqlError: graphqlError,
	}

	v := &verifyEvidenceBase{
		oneModelClient:     mockManager,
		useArtifactoryKeys: true,
	}
	_, err := v.queryEvidenceMetadata("test-repo", "test/path", "test-file.txt")
	assert.Error(t, err)

	// Check if the error contains the expected version requirement message
	assert.Contains(t, err.Error(), "the evidence service version should be at least 7.125.0")
	assert.Contains(t, err.Error(), "the onemodel version should be at least 1.55.0")
}

func TestVerifyEvidenceBase_QueryEvidenceMetadata_FallbackToQueryWithoutAttachmentsWhenUnsupported(t *testing.T) {
	mockManager := &MockOneModelManagerWithQueryCapture{
		GraphqlErrors: []error{
			fmt.Errorf(`{"errors":[{"message":"Cannot query field \"attachments\" on type \"EvidenceMetadata\"."}]}`),
			nil,
		},
		GraphqlResponse: []byte(`{"data":{"evidence":{"searchEvidence":{"edges":[{"cursor":"c","node":{"downloadPath":"p","predicateType":"t","createdAt":"now","createdBy":"me","subject":{"sha256":"abc"},"signingKey":{"alias":"a","publicKey":"pk"}}}]}}}}`),
	}
	v := &verifyEvidenceBase{
		oneModelClient:     mockManager,
		useArtifactoryKeys: true,
	}

	edges, err := v.queryEvidenceMetadata("test-repo", "test/path", "test-file.txt")
	assert.NoError(t, err)

	if assert.NotNil(t, edges) && assert.Len(t, *edges, 1) {
		edge := (*edges)[0]
		assert.Equal(t, "p", edge.Node.DownloadPath)
		assert.Equal(t, "t", edge.Node.PredicateType)
		assert.Equal(t, "me", edge.Node.CreatedBy)
		assert.Equal(t, "abc", edge.Node.Subject.Sha256)
		assert.Equal(t, "a", edge.Node.SigningKey.Alias)
	}

	assert.Len(t, mockManager.CapturedQueries, 2)
	assert.Contains(t, string(mockManager.CapturedQueries[0]), "attachments")
	assert.NotContains(t, string(mockManager.CapturedQueries[1]), "attachments")
}

func TestVerifyEvidenceBase_QueryEvidenceMetadata_FallbackWithoutAttachments_PublicKeyError(t *testing.T) {
	mockManager := &MockOneModelManagerWithQueryCapture{
		GraphqlErrors: []error{
			fmt.Errorf(`{"errors":[{"message":"Cannot query field \"attachments\" on type \"EvidenceMetadata\"."}]}`),
			fmt.Errorf(`{"errors":[{"message":"Cannot query field \"publicKey\" on type \"EvidenceSigningKey\"."}]}`),
		},
	}
	v := &verifyEvidenceBase{
		oneModelClient:     mockManager,
		useArtifactoryKeys: true,
	}

	_, err := v.queryEvidenceMetadata("test-repo", "test/path", "test-file.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "the evidence service version should be at least 7.125.0")
	assert.Contains(t, err.Error(), "the onemodel version should be at least 1.55.0")
	assert.Len(t, mockManager.CapturedQueries, 2)
	assert.Contains(t, string(mockManager.CapturedQueries[0]), "attachments")
	assert.NotContains(t, string(mockManager.CapturedQueries[1]), "attachments")
}

func TestIsVerificationSucceed(t *testing.T) {
	tests := []struct {
		name           string
		verification   model.EvidenceVerification
		expectedResult bool
		description    string
	}{
		{
			name: "DSSE_BothSuccess",
			verification: model.EvidenceVerification{
				MediaType: model.SimpleDSSE,
				VerificationResult: model.EvidenceVerificationResult{
					Sha256VerificationStatus:     model.Success,
					SignaturesVerificationStatus: model.Success,
				},
			},
			expectedResult: true,
			description:    "DSSE verification should succeed when both Sha256 and Signatures are success",
		},
		{
			name: "DSSE_Sha256Failed",
			verification: model.EvidenceVerification{
				MediaType: model.SimpleDSSE,
				VerificationResult: model.EvidenceVerificationResult{
					Sha256VerificationStatus:     model.Failed,
					SignaturesVerificationStatus: model.Success,
				},
			},
			expectedResult: false,
			description:    "DSSE verification should fail when Sha256 verification fails",
		},
		{
			name: "DSSE_SignaturesFailed",
			verification: model.EvidenceVerification{
				MediaType: model.SimpleDSSE,
				VerificationResult: model.EvidenceVerificationResult{
					Sha256VerificationStatus:     model.Success,
					SignaturesVerificationStatus: model.Failed,
				},
			},
			expectedResult: false,
			description:    "DSSE verification should fail when Signatures verification fails",
		},
		{
			name: "DSSE_BothFailed",
			verification: model.EvidenceVerification{
				MediaType: model.SimpleDSSE,
				VerificationResult: model.EvidenceVerificationResult{
					Sha256VerificationStatus:     model.Failed,
					SignaturesVerificationStatus: model.Failed,
				},
			},
			expectedResult: false,
			description:    "DSSE verification should fail when both Sha256 and Signatures verification fail",
		},
		{
			name: "SigstoreBundle_Success",
			verification: model.EvidenceVerification{
				MediaType: model.SigstoreBundle,
				VerificationResult: model.EvidenceVerificationResult{
					SigstoreBundleVerificationStatus: model.Success,
					Sha256VerificationStatus:         model.Success,
				},
			},
			expectedResult: true,
			description:    "Sigstore bundle verification should succeed when SigstoreBundleVerificationStatus is success",
		},
		{
			name: "SigstoreBundle_Failed",
			verification: model.EvidenceVerification{
				MediaType: model.SigstoreBundle,
				VerificationResult: model.EvidenceVerificationResult{
					SigstoreBundleVerificationStatus: model.Failed,
				},
			},
			expectedResult: false,
			description:    "Sigstore bundle verification should fail when SigstoreBundleVerificationStatus is failed",
		},
		{
			name: "SigstoreBundle_SuccessWithSignaturesFailed",
			verification: model.EvidenceVerification{
				MediaType: model.SigstoreBundle,
				VerificationResult: model.EvidenceVerificationResult{
					Sha256VerificationStatus:         model.Success,
					SignaturesVerificationStatus:     model.Failed,
					SigstoreBundleVerificationStatus: model.Success,
				},
			},
			expectedResult: true,
			description:    "Verification should succeed with Sigstore bundle success even if Signatures verification failed",
		},
		{
			name: "DSSE_SuccessWithSigstoreFailed",
			verification: model.EvidenceVerification{
				MediaType: model.SimpleDSSE,
				VerificationResult: model.EvidenceVerificationResult{
					Sha256VerificationStatus:         model.Success,
					SignaturesVerificationStatus:     model.Success,
					SigstoreBundleVerificationStatus: model.Failed,
				},
			},
			expectedResult: true,
			description:    "Verification should succeed with DSSE success even if Sigstore bundle field is failed",
		},
		{
			name: "DSSE_AttachmentsFailed",
			verification: model.EvidenceVerification{
				MediaType: model.SimpleDSSE,
				VerificationResult: model.EvidenceVerificationResult{
					Sha256VerificationStatus:      model.Success,
					SignaturesVerificationStatus:  model.Success,
					AttachmentsVerificationStatus: model.Failed,
				},
			},
			expectedResult: false,
			description:    "Verification should fail when attachments verification fails",
		},
		{
			name: "AllFieldsEmpty",
			verification: model.EvidenceVerification{
				VerificationResult: model.EvidenceVerificationResult{},
			},
			expectedResult: false,
			description:    "Verification should fail when all verification status fields are empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := reports.IsVerificationSucceed(tt.verification)
			assert.Equal(t, tt.expectedResult, result, tt.description)
		})
	}
}

func TestVerifyEvidence_UsesProvidedVerifier_Success(t *testing.T) {
	resp := &model.VerificationResponse{
		OverallVerificationStatus: model.Success,
		EvidenceVerifications:     &[]model.EvidenceVerification{{}},
	}
	mockVerifier := &MockEvidenceVerifier{Result: resp}

	v := &verifyEvidenceBase{
		verifier: mockVerifier,
		format:   "json",
	}

	metadata := []model.SearchEvidenceEdge{{}}
	err := v.verifyEvidence(nil, &metadata, "sha-123", "some/path/file")
	assert.NoError(t, err)
	assert.Equal(t, "sha-123", mockVerifier.LastSha)
	assert.Equal(t, &metadata, mockVerifier.LastMeta)
	assert.Equal(t, "some/path/file", mockVerifier.LastPath)
}

func TestVerifyEvidence_ReturnsVerifierError(t *testing.T) {
	mockVerifier := &MockEvidenceVerifier{Err: errors.New("verify failed")}
	v := &verifyEvidenceBase{verifier: mockVerifier}

	metadata := []model.SearchEvidenceEdge{{}}
	err := v.verifyEvidence(nil, &metadata, "sha-xyz", "subject/path")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "verify failed")
}

func TestVerifyEvidence_ReturnsCliErrorOnFailedStatus(t *testing.T) {
	resp := &model.VerificationResponse{
		OverallVerificationStatus: model.Failed,
		EvidenceVerifications:     &[]model.EvidenceVerification{{}},
	}
	mockVerifier := &MockEvidenceVerifier{Result: resp}
	v := &verifyEvidenceBase{verifier: mockVerifier, format: "text"}

	metadata := []model.SearchEvidenceEdge{{}}
	err := v.verifyEvidence(nil, &metadata, "sha-000", "subject")
	if assert.Error(t, err) {
		var cliError coreutils.CliError
		ok := errors.As(err, &cliError)
		assert.True(t, ok, "error should be of type CliError when overall status is failed")
	}
}

func TestVerifyEvidence_InitializesVerifierWhenNil_EmptyMetadataError(t *testing.T) {
	var mgr artifactory.ArtifactoryServicesManager
	client := &mgr

	v := &verifyEvidenceBase{verifier: nil, format: "json"}
	var emptyMetadata []model.SearchEvidenceEdge

	err := v.verifyEvidence(client, &emptyMetadata, "sha-empty", "subject")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no evidence metadata provided")
}

// fakeProgress implements ioUtils.ProgressMgr for testing headlines/totals/quit without side effects
type fakeProgress struct {
	headlines       []string
	totalIncrements int64
	quitCalled      bool
	initialized     bool
}

type fakeBar struct{ id int }

func (f *fakeProgress) NewProgressReader(_ int64, _, _ string) ioUtils.Progress {
	return &fakeBar{id: 1}
}
func (f *fakeProgress) SetMergingState(id int, _ bool) ioUtils.Progress {
	return &fakeBar{id: id}
}
func (f *fakeProgress) GetProgress(id int) ioUtils.Progress { return &fakeBar{id: id} }
func (f *fakeProgress) RemoveProgress(_ int)                {}
func (f *fakeProgress) IncrementGeneralProgress()           {}
func (f *fakeProgress) Quit() error                         { f.quitCalled = true; return nil }
func (f *fakeProgress) IncGeneralProgressTotalBy(n int64)   { f.totalIncrements += n }
func (f *fakeProgress) SetHeadlineMsg(msg string)           { f.headlines = append(f.headlines, msg) }
func (f *fakeProgress) ClearHeadlineMsg()                   {}
func (f *fakeProgress) InitProgressReaders()                { f.initialized = true }
func (f *fakeProgress) ClearProgress()                      {}

func (b *fakeBar) ActionWithProgress(reader io.Reader) io.Reader { return reader }
func (b *fakeBar) SetProgress(_ int64)                           {}
func (b *fakeBar) Abort()                                        {}
func (b *fakeBar) GetId() int                                    { return b.id }

func TestVerifyEvidenceBase_Progress_QueryEvidenceMetadata(t *testing.T) {
	pm := &fakeProgress{}
	mockManager := &MockOneModelManagerWithQueryCapture{GraphqlResponse: []byte(`{"data":{"evidence":{"searchEvidence":{"edges":[{"cursor":"c","node":{"downloadPath":"p","predicateType":"t","createdAt":"now","createdBy":"me","subject":{"sha256":"abc"}}}]}}}}`)}
	v := &verifyEvidenceBase{oneModelClient: mockManager, progressMgr: pm}
	_, err := v.queryEvidenceMetadata("repo", "p", "n")
	assert.NoError(t, err)
	assert.True(t, len(pm.headlines) >= 1)
}
