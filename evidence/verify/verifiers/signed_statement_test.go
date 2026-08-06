package verifiers

import (
	"encoding/base64"
	"testing"

	"github.com/jfrog/jfrog-cli-evidence/evidence/dsse"
	"github.com/jfrog/jfrog-cli-evidence/evidence/model"
	"github.com/stretchr/testify/assert"
)

const testGitCommitID = "c1712d8f3dddb5c3bd6eb8edf85fa9279cdc14b7"

func entityEnvelope(t *testing.T, digest string) *dsse.Envelope {
	t.Helper()
	payload := `{"_type":"https://in-toto.io/Statement/v1","subject":[{"digest":` + digest + `}],"predicateType":"https://jfrog.com/evidence/commit-approval/v1","predicate":{}}`
	return &dsse.Envelope{
		Payload:     base64.StdEncoding.EncodeToString([]byte(payload)),
		PayloadType: "application/vnd.in-toto+json",
	}
}

func TestVerifySignedSubjectDigest_EntityDigestPresent(t *testing.T) {
	result := &model.EvidenceVerification{
		DsseEnvelope: entityEnvelope(t, `{"gitCommit":"`+testGitCommitID+`"}`),
	}

	verifySignedSubjectDigest(model.SubjectDigest{Type: "gitCommit", Value: testGitCommitID}, result)

	assert.Equal(t, model.Success, result.VerificationResult.SubjectDigestVerificationStatus)
	assert.Equal(t, map[string]string{"gitCommit": testGitCommitID}, result.SignedSubjectDigest)
	assert.Empty(t, result.VerificationResult.FailureReason)
}

func TestVerifySignedSubjectDigest_EntityTypeCaseMismatchFails(t *testing.T) {
	result := &model.EvidenceVerification{
		DsseEnvelope: entityEnvelope(t, `{"gitcommit":"`+testGitCommitID+`"}`),
	}

	verifySignedSubjectDigest(model.SubjectDigest{Type: "gitCommit", Value: testGitCommitID}, result)

	assert.Equal(t, model.Failed, result.VerificationResult.SubjectDigestVerificationStatus)
}

func TestVerifySignedSubjectDigest_DigestFoundAmongMultipleSubjects(t *testing.T) {
	payload := `{"_type":"https://in-toto.io/Statement/v1","subject":[{"digest":{"gitCommit":"0000000000000000000000000000000000000000"}},{"digest":{"gitCommit":"` + testGitCommitID + `"}}]}`
	result := &model.EvidenceVerification{
		DsseEnvelope: &dsse.Envelope{Payload: base64.StdEncoding.EncodeToString([]byte(payload))},
	}

	verifySignedSubjectDigest(model.SubjectDigest{Type: "gitCommit", Value: testGitCommitID}, result)

	assert.Equal(t, model.Success, result.VerificationResult.SubjectDigestVerificationStatus)
	assert.Equal(t, testGitCommitID, result.SignedSubjectDigest["gitCommit"])
}

func TestVerifySignedSubjectDigest_EntityIDMismatchFails(t *testing.T) {
	result := &model.EvidenceVerification{
		DsseEnvelope: entityEnvelope(t, `{"gitCommit":"0000000000000000000000000000000000000000"}`),
	}

	verifySignedSubjectDigest(model.SubjectDigest{Type: "gitCommit", Value: testGitCommitID}, result)

	assert.Equal(t, model.Failed, result.VerificationResult.SubjectDigestVerificationStatus)
	assert.Empty(t, result.SignedSubjectDigest)
	assert.Contains(t, result.VerificationResult.FailureReason, "does not contain the expected subject digest")
	assert.Contains(t, result.VerificationResult.FailureReason, testGitCommitID)
}

func TestVerifySignedSubjectDigest_EntityTypeMismatchFails(t *testing.T) {
	result := &model.EvidenceVerification{
		DsseEnvelope: entityEnvelope(t, `{"gitTag":"`+testGitCommitID+`"}`),
	}

	verifySignedSubjectDigest(model.SubjectDigest{Type: "gitCommit", Value: testGitCommitID}, result)

	assert.Equal(t, model.Failed, result.VerificationResult.SubjectDigestVerificationStatus)
}

func TestVerifySignedSubjectDigest_Sha256OnlyDigestFails(t *testing.T) {
	// Entity evidence whose signed statement carries only a content checksum proves nothing
	// about the entity, so it must not be reported as verified.
	result := &model.EvidenceVerification{
		DsseEnvelope: entityEnvelope(t, `{"sha256":"`+createTestSHA256()+`"}`),
	}

	verifySignedSubjectDigest(model.SubjectDigest{Type: "gitCommit", Value: testGitCommitID}, result)

	assert.Equal(t, model.Failed, result.VerificationResult.SubjectDigestVerificationStatus)
}

func TestVerifySignedSubjectDigest_NoSignedStatementFails(t *testing.T) {
	result := &model.EvidenceVerification{}

	verifySignedSubjectDigest(model.SubjectDigest{Type: "gitCommit", Value: testGitCommitID}, result)

	assert.Equal(t, model.Failed, result.VerificationResult.SubjectDigestVerificationStatus)
	assert.Contains(t, result.VerificationResult.FailureReason, "does not contain a signed in-toto statement")
}

func TestVerifySignedSubjectDigest_UndecodablePayloadFails(t *testing.T) {
	result := &model.EvidenceVerification{
		DsseEnvelope: &dsse.Envelope{Payload: "not-base64!"},
	}

	verifySignedSubjectDigest(model.SubjectDigest{Type: "gitCommit", Value: testGitCommitID}, result)

	assert.Equal(t, model.Failed, result.VerificationResult.SubjectDigestVerificationStatus)
	assert.Contains(t, result.VerificationResult.FailureReason, "failed to decode the signed statement payload")
}

func TestVerifySignedSubjectDigest_StatementWithoutSubjectFails(t *testing.T) {
	payload := `{"_type":"https://in-toto.io/Statement/v1","predicateType":"https://example.com","predicate":{}}`
	result := &model.EvidenceVerification{
		DsseEnvelope: &dsse.Envelope{Payload: base64.StdEncoding.EncodeToString([]byte(payload))},
	}

	verifySignedSubjectDigest(model.SubjectDigest{Type: "gitCommit", Value: testGitCommitID}, result)

	assert.Equal(t, model.Failed, result.VerificationResult.SubjectDigestVerificationStatus)
	assert.Contains(t, result.VerificationResult.FailureReason, "does not contain a subject")
}

func TestSetFailureReason_AppendsToExistingReason(t *testing.T) {
	result := &model.EvidenceVerification{}
	result.VerificationResult.FailureReason = "first reason"

	setFailureReason(result, "second reason")

	assert.Equal(t, "first reason; second reason", result.VerificationResult.FailureReason)
}
