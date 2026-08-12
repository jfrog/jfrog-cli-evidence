package verifiers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/jfrog/jfrog-cli-evidence/evidence/dsse"
	"github.com/jfrog/jfrog-cli-evidence/evidence/model"
	"github.com/jfrog/jfrog-cli-evidence/evidence/sigstore"
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
//
// The signed statement is read from either a plain DSSE envelope or the DSSE envelope embedded
// in a Sigstore bundle.
func verifySignedSubjectDigest(expected model.SubjectDigest, result *model.EvidenceVerification) {
	statement, err := decodeSignedStatementFromVerification(result)
	if err != nil {
		recordSubjectDigestStatus(expected, result, model.Failed)
		setFailureReason(result, err.Error())
		return
	}

	if matched := findMatchingSubjectDigest(statement, expected); matched != nil {
		result.SignedSubjectDigest = matched
		recordSubjectDigestStatus(expected, result, model.Success)
		return
	}

	// Surface every digest from the statement so reports and failure reasons can show what was
	// actually signed when verification fails.
	found := allSubjectDigests(statement)
	result.SignedSubjectDigest = legacySubjectDigest(found)
	result.AvailableSubjectDigests = found
	recordSubjectDigestStatus(expected, result, model.Failed)
	setFailureReason(result, subjectDigestMismatchError(expected, found).Error())
}

// recordSubjectDigestStatus always records the signed-statement outcome on
// SubjectDigestVerificationStatus. For content (sha256) subjects it also mirrors the outcome onto
// Sha256VerificationStatus.
func recordSubjectDigestStatus(expected model.SubjectDigest, result *model.EvidenceVerification, status model.VerificationStatus) {
	result.VerificationResult.SubjectDigestVerificationStatus = status
	if expected.IsSha256() {
		result.VerificationResult.Sha256VerificationStatus = status
	}
}

// findMatchingSubjectDigest returns the subject digest map that contains the expected entry,
// or nil when no subject matches.
func findMatchingSubjectDigest(statement *signedStatement, expected model.SubjectDigest) map[string]string {
	for _, subject := range statement.Subject {
		for digestType, digestValue := range subject.Digest {
			if digestType == expected.Type && digestValue == expected.Value {
				return subject.Digest
			}
		}
	}
	return nil
}

// allSubjectDigests returns every non-empty subject digest map from the statement, in order.
func allSubjectDigests(statement *signedStatement) []map[string]string {
	digests := make([]map[string]string, 0, len(statement.Subject))
	for _, subject := range statement.Subject {
		if len(subject.Digest) > 0 {
			digests = append(digests, subject.Digest)
		}
	}
	return digests
}

// legacySubjectDigest preserves the collapsed signedSubjectDigest failure output: a single
// map with first-wins values per digest type. AvailableSubjectDigests is the authoritative,
// non-lossy representation when multiple subjects contain the same digest type.
func legacySubjectDigest(digests []map[string]string) map[string]string {
	if len(digests) == 0 {
		return nil
	}
	legacy := make(map[string]string)
	for _, digest := range digests {
		for digestType, digestValue := range digest {
			if _, exists := legacy[digestType]; !exists {
				legacy[digestType] = digestValue
			}
		}
	}
	return legacy
}

func decodeSignedStatementFromVerification(result *model.EvidenceVerification) (*signedStatement, error) {
	payload, err := signedStatementPayload(result)
	if err != nil {
		return nil, err
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

func signedStatementPayload(result *model.EvidenceVerification) ([]byte, error) {
	if result == nil {
		return nil, fmt.Errorf("the evidence does not contain a signed in-toto statement, so the subject digest could not be verified")
	}
	if result.DsseEnvelope != nil {
		return decodeDssePayload(result.DsseEnvelope)
	}
	if result.SigstoreBundle != nil {
		protoEnvelope, err := sigstore.GetDSSEEnvelope(result.SigstoreBundle)
		if err != nil {
			return nil, fmt.Errorf("failed to read the signed statement from the Sigstore bundle: %w", err)
		}
		payload := protoEnvelope.GetPayload()
		if len(payload) == 0 {
			return nil, fmt.Errorf("the Sigstore bundle DSSE envelope does not contain a payload")
		}
		return payload, nil
	}
	return nil, fmt.Errorf("the evidence does not contain a signed in-toto statement, so the subject digest could not be verified")
}

func decodeDssePayload(envelope *dsse.Envelope) ([]byte, error) {
	payload, err := base64.StdEncoding.DecodeString(envelope.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to decode the signed statement payload: %w", err)
	}
	return payload, nil
}

func subjectDigestMismatchError(expected model.SubjectDigest, found []map[string]string) error {
	if len(found) == 0 {
		return fmt.Errorf("the signed in-toto statement does not contain the expected subject digest %s %q", expected.Type, expected.Value)
	}
	foundJSON, err := json.Marshal(found)
	if err != nil {
		return fmt.Errorf("the signed in-toto statement does not contain the expected subject digest %s %q", expected.Type, expected.Value)
	}
	return fmt.Errorf("the signed in-toto statement does not contain the expected subject digest %s %q; found: %s",
		expected.Type, expected.Value, foundJSON)
}

func setFailureReason(result *model.EvidenceVerification, reason string) {
	if result.VerificationResult.FailureReason == "" {
		result.VerificationResult.FailureReason = reason
		return
	}
	result.VerificationResult.FailureReason += "; " + reason
}
