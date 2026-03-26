package reports

import (
	"fmt"

	"github.com/jfrog/jfrog-cli-evidence/evidence/model"
)

func verifyNotEmptyResponse(result *model.VerificationResponse) error {
	if result == nil {
		return fmt.Errorf("verification response is empty")
	}
	return nil
}

func IsVerificationSucceed(v model.EvidenceVerification) bool {
	attachmentsStatusOk := v.VerificationResult.AttachmentsVerificationStatus == "" || v.VerificationResult.AttachmentsVerificationStatus == model.Success
	return v.VerificationResult.Sha256VerificationStatus == model.Success &&
		attachmentsStatusOk &&
		(v.VerificationResult.SignaturesVerificationStatus == model.Success ||
			v.VerificationResult.SigstoreBundleVerificationStatus == model.Success)
}
