package verifiers

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jfrog/jfrog-cli-evidence/evidence/intoto"
	"github.com/jfrog/jfrog-cli-evidence/evidence/model"
	"github.com/jfrog/jfrog-client-go/artifactory"
	artUtils "github.com/jfrog/jfrog-client-go/artifactory/services/utils"
)

const (
	attachmentMetadataNotFoundReason    = "attachment not found in Evidence"
	attachmentArtifactNotFoundReason    = "attachment not found in Artifactory"
	checksumMismatchReason              = "checksum mismatch"
	attachmentMetadataUnavailableReason = "unable to get attachment metadata from GraphQL (query without attachments)"
	attachmentVerificationFailedReason  = "attachment failed verification"
)

type attachmentVerifierInterface interface {
	verify(evidence *model.SearchEvidenceEdge, result *model.EvidenceVerification) error
}

type attachmentVerifier struct {
	artifactoryClient artifactory.ArtifactoryServicesManager
}

func newAttachmentVerifier(client artifactory.ArtifactoryServicesManager) attachmentVerifierInterface {
	return &attachmentVerifier{artifactoryClient: client}
}

func (v *attachmentVerifier) verify(evidence *model.SearchEvidenceEdge, result *model.EvidenceVerification) error {
	if evidence == nil || result == nil || result.DsseEnvelope == nil {
		return fmt.Errorf("empty evidence or DSSE envelope provided for attachment verification")
	}

	expectedAttachments, err := extractStatementAttachments(result.DsseEnvelope.Payload)
	if err != nil {
		return fmt.Errorf("failed to parse DSSE payload attachments: %w", err)
	}
	if len(expectedAttachments) == 0 {
		return nil
	}
	if evidence.Node.AttachmentsUnavailable {
		if result.VerificationResult.SignaturesVerificationStatus == model.Failed {
			result.VerificationResult.AttachmentsVerificationStatus = model.Failed
			result.VerificationResult.FailureReason = attachmentMetadataUnavailableReason
			return nil
		}
		return errors.New(attachmentMetadataUnavailableReason)
	}

	actualBySha := buildAttachmentMapBySha(evidence.Node.Attachments)
	verifications := make([]model.AttachmentVerification, 0, len(expectedAttachments))
	hasFailures := false

	for _, expected := range expectedAttachments {
		verification := model.AttachmentVerification{
			Name:               expected.Name,
			ExpectedSha256:     expected.Sha256,
			VerificationStatus: model.Success,
		}

		actualAttachment, ok := actualBySha[expected.Sha256]
		if !ok {
			verification.VerificationStatus = model.Failed
			verification.FailureReason = attachmentMetadataNotFoundReason
			verifications = append(verifications, verification)
			hasFailures = true
			continue
		}

		verification.DownloadPath = actualAttachment.DownloadPath
		fileInfo, fileInfoErr := v.artifactoryClient.FileInfo(actualAttachment.DownloadPath)
		if fileInfoErr != nil && isAttachmentNotFoundError(fileInfoErr) {
			verification.VerificationStatus = model.Failed
			verification.FailureReason = attachmentArtifactNotFoundReason
			verifications = append(verifications, verification)
			hasFailures = true
			continue
		}

		if err = handleAttachmentFileInfoErrors(actualAttachment.DownloadPath, fileInfo, fileInfoErr); err != nil {
			return err
		}

		verification.ActualSha256 = fileInfo.Checksums.Sha256
		if verification.ActualSha256 != expected.Sha256 {
			verification.VerificationStatus = model.Failed
			verification.FailureReason = checksumMismatchReason
			hasFailures = true
		}
		verifications = append(verifications, verification)
	}

	result.AttachmentsVerification = verifications
	if hasFailures {
		result.VerificationResult.AttachmentsVerificationStatus = model.Failed
		if result.VerificationResult.FailureReason == "" {
			result.VerificationResult.FailureReason = attachmentVerificationFailedReason
		}
		return nil
	}
	result.VerificationResult.AttachmentsVerificationStatus = model.Success
	return nil
}

func extractStatementAttachments(encodedPayload string) ([]intoto.Attachment, error) {
	payloadBytes, err := base64.StdEncoding.DecodeString(encodedPayload)
	if err != nil {
		return nil, err
	}
	statement := &intoto.Statement{}
	if err = json.Unmarshal(payloadBytes, statement); err != nil {
		return nil, err
	}
	return statement.Attachments, nil
}

func buildAttachmentMapBySha(attachments []model.AttachmentRef) map[string]model.AttachmentRef {
	if len(attachments) == 0 {
		return map[string]model.AttachmentRef{}
	}
	result := make(map[string]model.AttachmentRef, len(attachments))
	for _, attachment := range attachments {
		result[attachment.Sha256] = attachment
	}
	return result
}

func isAttachmentNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errString := err.Error()
	return strings.Contains(errString, "404 Not Found")
}

func handleAttachmentFileInfoErrors(downloadPath string, fileInfo *artUtils.FileInfo, fileInfoErr error) error {
	if fileInfoErr != nil {
		return fmt.Errorf("failed to resolve attachment file info for %s: %w", downloadPath, fileInfoErr)
	}
	if fileInfo == nil {
		return fmt.Errorf("failed to resolve attachment file info for %s: empty file info response", downloadPath)
	}
	if fileInfo.Checksums.Sha256 == "" {
		return fmt.Errorf("failed to resolve attachment checksum for %s: sha256 is empty", downloadPath)
	}
	return nil
}
