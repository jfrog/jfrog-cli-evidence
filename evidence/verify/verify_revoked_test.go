package verify

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli-evidence/evidence/model"
	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// UT-C1: verify never opts in to revoked evidence. The server hides revoked evidence by default,
// so a revoked evidence is "not found" for verify.

const (
	noEvidenceResponse     = `{"data":{"evidence":{"searchEvidence":{"edges":[]}}}}`
	activeEvidenceResponse = `{"data":{"evidence":{"searchEvidence":{"edges":[{"cursor":"1","node":{"downloadPath":"repo/.evidence/active.json","verified":true,"subject":{"sha256":"abc"}}}]}}}}`
)

func assertNoRevokedOptIn(t *testing.T, query string) {
	t.Helper()
	assert.NotContains(t, query, "includeRevoked")
	assert.NotContains(t, query, "revoked")
	assert.NotContains(t, query, "revocation")
}

func TestVerifyQueries_NeverContainIncludeRevoked(t *testing.T) {
	for _, useArtifactoryKeys := range []bool{false, true} {
		for _, includeAttachments := range []bool{false, true} {
			name := fmt.Sprintf("artifactoryKeys=%t/attachments=%t", useArtifactoryKeys, includeAttachments)
			t.Run(name, func(t *testing.T) {
				base := &verifyEvidenceBase{useArtifactoryKeys: useArtifactoryKeys}
				assertNoRevokedOptIn(t, base.buildSearchEvidenceQuery(includeAttachments))

				entity := &verifyEvidenceEntity{
					verifyEvidenceBase: verifyEvidenceBase{useArtifactoryKeys: useArtifactoryKeys},
					EntitySubject:      model.EntitySubject{EntityType: "gitCommit", EntityID: "abc123", ProjectKey: "proj"},
				}
				assertNoRevokedOptIn(t, entity.buildSearchEvidenceByEntityQuery(includeAttachments))
			})
		}
	}
}

func TestVerifyQueryEvidenceMetadata_SentQueriesNeverContainIncludeRevoked(t *testing.T) {
	mockManager := &MockOneModelManagerWithQueryCapture{
		GraphqlErrors:   []error{errors.New(`Cannot query field "attachments" on type "Evidence"`)},
		GraphqlResponse: []byte(activeEvidenceResponse),
	}
	v := &verifyEvidenceBase{oneModelClient: mockManager}

	_, err := v.queryEvidenceMetadata("repo", "dir", "app.tgz")
	require.NoError(t, err)

	require.Len(t, mockManager.CapturedQueries, 2, "initial query and attachments fallback")
	for _, query := range mockManager.CapturedQueries {
		assertNoRevokedOptIn(t, string(query))
	}
}

func TestVerifyQueryEvidenceMetadata_OnlyRevokedEvidence_NoEvidenceFound(t *testing.T) {
	mockManager := &MockOneModelManagerWithQueryCapture{GraphqlResponse: []byte(noEvidenceResponse)}
	v := &verifyEvidenceBase{oneModelClient: mockManager}

	_, err := v.queryEvidenceMetadata("repo", "dir", "app.tgz")

	require.Error(t, err)
	assert.Equal(t, "no evidence found for the given subject", err.Error())
	assertNoRevokedOptIn(t, string(mockManager.CapturedQuery))
}

func TestVerifyEvidenceEntity_OnlyRevokedEvidence_NoEvidenceFound(t *testing.T) {
	mockOneModel := &mockOneModelManagerEntity{GraphqlResponse: []byte(noEvidenceResponse)}
	mockVerifier := &mockVerifierEntity{}
	artClient := artifactory.ArtifactoryServicesManager(&artifactory.EmptyArtifactoryServicesManager{})
	cmd := &verifyEvidenceEntity{
		verifyEvidenceBase: verifyEvidenceBase{
			serverDetails:     &config.ServerDetails{Url: "https://example.jfrog.io/"},
			artifactoryClient: &artClient,
			oneModelClient:    mockOneModel,
			verifier:          mockVerifier,
			format:            "json",
		},
		EntitySubject: model.EntitySubject{EntityType: "gitCommit", EntityID: "abc123", ProjectKey: "proj"},
	}

	err := cmd.Run()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no evidence found for the given subject")
	assert.Empty(t, mockVerifier.lastSubjectPath, "nothing is verified when only revoked evidence exists")
	assertNoRevokedOptIn(t, string(mockOneModel.lastQuery))
}

func TestVerifyEvidence_MixedEvidence_VerifiesOnlyReturnedActiveEvidence(t *testing.T) {
	mockManager := &MockOneModelManagerWithQueryCapture{GraphqlResponse: []byte(activeEvidenceResponse)}
	verifier := &MockEvidenceVerifier{Result: &model.VerificationResponse{OverallVerificationStatus: model.Success}}
	v := &verifyEvidenceBase{oneModelClient: mockManager, verifier: verifier, format: "json"}

	edges, err := v.queryEvidenceMetadata("repo", "dir", "app.tgz")
	require.NoError(t, err)
	require.NoError(t, v.verifyEvidence(nil, edges, model.SubjectDigest{Type: "sha256", Value: "abc"}, "repo/dir/app.tgz"))

	require.NotNil(t, verifier.LastMeta)
	require.Len(t, *verifier.LastMeta, 1)
	assert.Equal(t, "repo/.evidence/active.json", (*verifier.LastMeta)[0].Node.DownloadPath)
	assertNoRevokedOptIn(t, string(mockManager.CapturedQuery))
}
