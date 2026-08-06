package reports

import (
	"fmt"
	"sort"

	"github.com/jfrog/jfrog-cli-evidence/evidence/model"
)

// formatSignedSubjectDigests renders the subject digests read from the signed statement as
// "type: value" entries, sorted so the output is stable.
func formatSignedSubjectDigests(digest map[string]string) []string {
	if len(digest) == 0 {
		return nil
	}
	types := make([]string, 0, len(digest))
	for digestType := range digest {
		types = append(types, digestType)
	}
	sort.Strings(types)
	formatted := make([]string, 0, len(types))
	for _, digestType := range types {
		formatted = append(formatted, fmt.Sprintf("%s: %s", digestType, digest[digestType]))
	}
	return formatted
}

func verifyNotEmptyResponse(result *model.VerificationResponse) error {
	if result == nil {
		return fmt.Errorf("verification response is empty")
	}
	return nil
}

// signedSubjectDigestsFromVerifications collects the distinct signed subject digests reported
// by the given verifications.
func signedSubjectDigestsFromVerifications(verifications *[]model.EvidenceVerification) []string {
	if verifications == nil {
		return nil
	}
	seen := map[string]bool{}
	var digests []string
	for _, verification := range *verifications {
		for _, digest := range formatSignedSubjectDigests(verification.SignedSubjectDigest) {
			if !seen[digest] {
				seen[digest] = true
				digests = append(digests, digest)
			}
		}
	}
	return digests
}

func IsVerificationSucceed(v model.EvidenceVerification) bool {
	attachmentsStatusOk := v.VerificationResult.AttachmentsVerificationStatus == "" || v.VerificationResult.AttachmentsVerificationStatus == model.Success
	// The subject must be verified either by its content checksum or by the digest carried in the
	// signed statement. Evidence whose subject was not verified is never reported as verified.
	subjectStatusOk := v.VerificationResult.Sha256VerificationStatus == model.Success ||
		v.VerificationResult.SubjectDigestVerificationStatus == model.Success
	return subjectStatusOk &&
		attachmentsStatusOk &&
		(v.VerificationResult.SignaturesVerificationStatus == model.Success ||
			v.VerificationResult.SigstoreBundleVerificationStatus == model.Success)
}
