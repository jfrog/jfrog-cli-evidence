package verifiers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/jfrog/jfrog-cli-evidence/evidence/dsse"
	"github.com/jfrog/jfrog-cli-evidence/evidence/model"
)

// signedStatement is the subset of the in-toto statement required to inspect the subject
// digests that were actually signed. The digest is intentionally a generic map, since entity
// subjects are identified by their entity type rather than by a content checksum.
type signedStatement struct {
	Subject []struct {
		Digest map[string]string `json:"digest"`
	} `json:"subject"`
}

// verifySignedSubjectDigest checks that the signed statement carries the expected subject
// digest and records the outcome on the verification result. A subject digest that cannot be
// read, or that does not carry the expected entry, fails verification: reporting such evidence
// as verified would claim more than was actually checked.
func verifySignedSubjectDigest(expected model.SubjectDigest, result *model.EvidenceVerification) {
	digest, err := findSignedSubjectDigest(result.DsseEnvelope, expected)
	if err != nil {
		result.VerificationResult.SubjectDigestVerificationStatus = model.Failed
		setFailureReason(result, err.Error())
		return
	}
	result.SignedSubjectDigest = digest
	result.VerificationResult.SubjectDigestVerificationStatus = model.Success
}

func findSignedSubjectDigest(envelope *dsse.Envelope, expected model.SubjectDigest) (map[string]string, error) {
	if envelope == nil {
		return nil, fmt.Errorf("the evidence does not contain a signed in-toto statement, so the subject %s %q could not be verified", expected.Type, expected.Value)
	}
	statement, err := decodeSignedStatement(envelope)
	if err != nil {
		return nil, err
	}
	for _, subject := range statement.Subject {
		for digestType, digestValue := range subject.Digest {
			if digestType == expected.Type && digestValue == expected.Value {
				return subject.Digest, nil
			}
		}
	}
	return nil, fmt.Errorf("the signed in-toto statement does not contain the expected subject digest %s %q", expected.Type, expected.Value)
}

func decodeSignedStatement(envelope *dsse.Envelope) (*signedStatement, error) {
	payload, err := base64.StdEncoding.DecodeString(envelope.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to decode the signed statement payload: %w", err)
	}
	statement := &signedStatement{}
	if err := json.Unmarshal(payload, statement); err != nil {
		return nil, fmt.Errorf("failed to read the signed in-toto statement: %w", err)
	}
	if len(statement.Subject) == 0 {
		return nil, fmt.Errorf("the signed in-toto statement does not contain a subject")
	}
	return statement, nil
}

func setFailureReason(result *model.EvidenceVerification, reason string) {
	if result.VerificationResult.FailureReason == "" {
		result.VerificationResult.FailureReason = reason
		return
	}
	result.VerificationResult.FailureReason += "; " + reason
}
