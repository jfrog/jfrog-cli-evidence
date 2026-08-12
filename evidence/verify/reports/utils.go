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

// formatAvailableSubjectDigests renders every subject digest map from a failed verification.
func formatAvailableSubjectDigests(digests []map[string]string) []string {
	if len(digests) == 0 {
		return nil
	}
	formatted := make([]string, 0, len(digests))
	for _, digest := range digests {
		formatted = append(formatted, formatSignedSubjectDigests(digest)...)
	}
	return formatted
}

func verifyNotEmptyResponse(result *model.VerificationResponse) error {
	if result == nil {
		return fmt.Errorf("verification response is empty")
	}
	return nil
}

// subjectDigestsFromVerifications collects distinct subject digests reported by the given
// verifications. AvailableSubjectDigests takes precedence over the collapsed SignedSubjectDigest.
func subjectDigestsFromVerifications(verifications *[]model.EvidenceVerification) []string {
	if verifications == nil {
		return nil
	}
	seen := map[string]bool{}
	var digests []string
	for _, verification := range *verifications {
		entries := formatAvailableSubjectDigests(verification.AvailableSubjectDigests)
		if len(entries) == 0 {
			entries = formatSignedSubjectDigests(verification.SignedSubjectDigest)
		}
		for _, digest := range entries {
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
	// SubjectDigestVerificationStatus is always set from the signed statement. Sha256VerificationStatus
	// is additionally set for content subjects. Evidence whose subject was not verified is never
	// reported as verified.
	subjectStatusOk := (v.VerificationResult.Sha256VerificationStatus == model.Success ||
		v.VerificationResult.SubjectDigestVerificationStatus == model.Success) &&
		v.VerificationResult.Sha256VerificationStatus != model.Failed &&
		v.VerificationResult.SubjectDigestVerificationStatus != model.Failed
	return subjectStatusOk &&
		attachmentsStatusOk &&
		(v.VerificationResult.SignaturesVerificationStatus == model.Success ||
			v.VerificationResult.SigstoreBundleVerificationStatus == model.Success)
}
