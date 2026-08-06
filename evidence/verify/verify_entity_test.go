package verify

import (
	"testing"

	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli-evidence/evidence/model"
	evidenceutils "github.com/jfrog/jfrog-cli-evidence/evidence/utils"
	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockOneModelManagerEntity struct {
	GraphqlResponse []byte
	GraphqlError    error
	lastQuery       []byte
}

func (m *mockOneModelManagerEntity) GraphqlQuery(query []byte) ([]byte, error) {
	m.lastQuery = append([]byte{}, query...)
	if m.GraphqlError != nil {
		return nil, m.GraphqlError
	}
	return m.GraphqlResponse, nil
}

type mockVerifierEntity struct {
	lastSha256      string
	lastSubjectPath string
	response        *model.VerificationResponse
	err             error
}

func (m *mockVerifierEntity) Verify(subjectSha256 string, _ *[]model.SearchEvidenceEdge, subjectPath string) (*model.VerificationResponse, error) {
	m.lastSha256 = subjectSha256
	m.lastSubjectPath = subjectPath
	if m.err != nil {
		return nil, m.err
	}
	if m.response != nil {
		return m.response, nil
	}
	return &model.VerificationResponse{OverallVerificationStatus: model.Success}, nil
}

func TestVerifyEvidenceEntity_Run(t *testing.T) {
	mockOneModel := &mockOneModelManagerEntity{
		GraphqlResponse: []byte(`{"data":{"evidence":{"searchEvidence":{"edges":[{"node":{"subject":{"sha256":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},"downloadPath":"/evidence/path"}}]}}}}`),
	}
	mockVerifier := &mockVerifierEntity{}
	mockArtifactory := &artifactory.EmptyArtifactoryServicesManager{}
	artClient := artifactory.ArtifactoryServicesManager(mockArtifactory)
	cmd := &verifyEvidenceEntity{
		verifyEvidenceBase: verifyEvidenceBase{
			serverDetails:     &config.ServerDetails{Url: "https://example.jfrog.io/"},
			artifactoryClient: &artClient,
			oneModelClient:    mockOneModel,
			verifier:          mockVerifier,
			format:            "json",
		},
		entityType: "gitCommit",
		entityID:   "abc123",
		projectKey: "proj",
	}

	err := cmd.Run()
	require.NoError(t, err)
	assert.Contains(t, string(mockOneModel.lastQuery), `entityType: \"gitCommit\"`)
	assert.Contains(t, string(mockOneModel.lastQuery), `projectKey: \"proj\"`)
	assert.Equal(t, evidenceutils.EmptySubjectSha256, mockVerifier.lastSha256)
	assert.Equal(t, "gitCommit/abc123", mockVerifier.lastSubjectPath)
}

func TestVerifyEvidenceEntity_NoEvidence(t *testing.T) {
	mockArtifactory := &artifactory.EmptyArtifactoryServicesManager{}
	artClient := artifactory.ArtifactoryServicesManager(mockArtifactory)
	cmd := &verifyEvidenceEntity{
		verifyEvidenceBase: verifyEvidenceBase{
			serverDetails:     &config.ServerDetails{Url: "https://example.jfrog.io/"},
			artifactoryClient: &artClient,
			oneModelClient:    &mockOneModelManagerEntity{GraphqlResponse: []byte(`{"data":{"evidence":{"searchEvidence":{"edges":[]}}}}`)},
			format:            "json",
		},
		entityType: "gitCommit",
		entityID:   "abc123",
	}
	err := cmd.Run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no evidence found")
}

func TestNewVerifyEvidenceEntity(t *testing.T) {
	cmd := NewVerifyEvidenceEntity(&config.ServerDetails{}, "gitCommit", "abc", "repo", "", "", "json", nil, false)
	entityCmd, ok := cmd.(*verifyEvidenceEntity)
	require.True(t, ok)
	assert.Equal(t, "verify-evidence-entity", entityCmd.CommandName())
	assert.Equal(t, "repo", entityCmd.entityRepo)
}
