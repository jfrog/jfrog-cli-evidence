package reports

import (
	"testing"

	"github.com/jfrog/jfrog-cli-evidence/evidence/model"
	"github.com/stretchr/testify/assert"
)

func TestPlaintext_Print_Success(t *testing.T) {
	resp := &model.VerificationResponse{
		OverallVerificationStatus: model.Success,
		EvidenceVerifications: &[]model.EvidenceVerification{{
			PredicateType: "test-type",
			CreatedBy:     "test-user",
			CreatedAt:     "2024-01-01T00:00:00Z",
			VerificationResult: model.EvidenceVerificationResult{
				SignaturesVerificationStatus: model.Success,
				Sha256VerificationStatus:     model.Success,
			},
		}},
	}

	out := captureOutput(func() {
		err := PlaintextReportPrinter.Print(resp)
		assert.NoError(t, err)
	})
	assert.Contains(t, out, "Verification passed for 1 out of 1 evidence")
}

func TestPlaintext_Print_SeveralEvidence_Success(t *testing.T) {
	resp := &model.VerificationResponse{
		Subject: model.Subject{
			Sha256: "test-checksum",
			Path:   "test-file.txt",
		},
		OverallVerificationStatus: model.Success,
		EvidenceVerifications: &[]model.EvidenceVerification{{
			PredicateType: "test-type",
			CreatedBy:     "test-user",
			CreatedAt:     "2024-01-01T00:00:00Z",
			VerificationResult: model.EvidenceVerificationResult{
				SignaturesVerificationStatus: model.Success,
				Sha256VerificationStatus:     model.Success,
			},
		}, {
			PredicateType: "test-type-2",
			CreatedBy:     "test-user-2",
			CreatedAt:     "2024-01-02T00:00:00Z",
			VerificationResult: model.EvidenceVerificationResult{
				SignaturesVerificationStatus: model.Success,
				Sha256VerificationStatus:     model.Success,
			},
		}},
	}

	out := captureOutput(func() {
		err := PlaintextReportPrinter.Print(resp)
		assert.NoError(t, err)
	})
	assert.Contains(t, out, "Subject sha256:        test-checksum")
	assert.Contains(t, out, "Subject:               test-file.txt")
	assert.Contains(t, out, "Loaded 2 evidence")
	assert.Contains(t, out, "Verification passed for 2 out of 2 evidence")
}

func TestPlaintext_Print_SigstoreBundleSuccess(t *testing.T) {
	resp := &model.VerificationResponse{
		Subject: model.Subject{
			Sha256: "test-checksum",
			Path:   "test-file.txt",
		},
		OverallVerificationStatus: model.Success,
		EvidenceVerifications: &[]model.EvidenceVerification{{
			MediaType:       model.SigstoreBundle,
			PredicateType:   "test-predicate",
			CreatedBy:       "test-user",
			CreatedAt:       "2024-01-01T00:00:00Z",
			SubjectChecksum: "test-checksum",
			VerificationResult: model.EvidenceVerificationResult{
				SigstoreBundleVerificationStatus: model.Success,
				Sha256VerificationStatus:         model.Success,
			},
		}},
	}

	out := captureOutput(func() {
		err := PlaintextReportPrinter.Print(resp)
		assert.NoError(t, err)
	})

	assert.Contains(t, out, "Subject sha256:        test-checksum")
	assert.Contains(t, out, "Subject:               test-file.txt")
	assert.Contains(t, out, "Loaded 1 evidence")
	assert.Contains(t, out, "Verification passed for 1 out of 1 evidence")
	assert.Contains(t, out, "- Evidence 1:")
	assert.Contains(t, out, "- Media type:                      sigstore.bundle")
	assert.Contains(t, out, "- Predicate type:                  test-predicate")
	assert.Contains(t, out, "- Evidence subject sha256:         test-checksum")
	assert.Contains(t, out, "- Sigstore verification status:")
	expected := PlaintextReportPrinter.getColoredStatus(model.Success)
	assert.Contains(t, out, expected)
}

func TestPlaintext_Print_NilResponse(t *testing.T) {
	err := PlaintextReportPrinter.Print(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "verification response is empty")
}

func TestPlaintext_Print_WithFullDetails(t *testing.T) {
	resp := &model.VerificationResponse{
		Subject: model.Subject{
			Sha256: "test-checksum",
			Path:   "test/path",
		},
		OverallVerificationStatus: model.Success,
		EvidenceVerifications: &[]model.EvidenceVerification{{
			SubjectChecksum: "test-checksum",
			PredicateType:   "test-type",
			CreatedBy:       "test-user",
			CreatedAt:       "2024-01-01T00:00:00Z",
			MediaType:       model.SimpleDSSE,
			VerificationResult: model.EvidenceVerificationResult{
				Sha256VerificationStatus:     model.Success,
				SignaturesVerificationStatus: model.Success,
				KeySource:                    "test-key-source",
				KeyFingerprint:               "test-fingerprint",
			},
		}},
	}

	out := captureOutput(func() {
		err := PlaintextReportPrinter.Print(resp)
		assert.NoError(t, err)
	})
	assert.Contains(t, out, "Subject sha256:        test-checksum")
	assert.Contains(t, out, "Subject:               test/path")
	assert.Contains(t, out, "Media type:                      evidence.dsse")
	assert.Contains(t, out, "Key source:                      test-key-source")
	assert.Contains(t, out, "Key fingerprint:                 test-fingerprint")
}

func TestGetColoredStatus_AllStatuses(t *testing.T) {
	assert.Equal(t, PlaintextReportPrinter.success, PlaintextReportPrinter.getColoredStatus(model.Success))
	assert.Equal(t, PlaintextReportPrinter.failed, PlaintextReportPrinter.getColoredStatus(model.Failed))
}

func TestPlaintext_Print_AttachmentStatusSuccessWithoutDetails(t *testing.T) {
	resp := &model.VerificationResponse{
		OverallVerificationStatus: model.Success,
		EvidenceVerifications: &[]model.EvidenceVerification{{
			PredicateType: "test-type",
			VerificationResult: model.EvidenceVerificationResult{
				SignaturesVerificationStatus:  model.Success,
				Sha256VerificationStatus:      model.Success,
				AttachmentsVerificationStatus: model.Success,
			},
			AttachmentsVerification: []model.AttachmentVerification{{
				Name:               "report.txt",
				VerificationStatus: model.Success,
			}},
		}},
	}

	out := captureOutput(func() {
		err := PlaintextReportPrinter.Print(resp)
		assert.NoError(t, err)
	})
	assert.Contains(t, out, "Attachments verification status:")
	assert.NotContains(t, out, "Attachments:                    ")
}

func TestPlaintext_Print_AttachmentStatusFailureWithDetails(t *testing.T) {
	resp := &model.VerificationResponse{
		OverallVerificationStatus: model.Failed,
		EvidenceVerifications: &[]model.EvidenceVerification{{
			PredicateType: "test-type",
			VerificationResult: model.EvidenceVerificationResult{
				SignaturesVerificationStatus:  model.Success,
				Sha256VerificationStatus:      model.Success,
				AttachmentsVerificationStatus: model.Failed,
			},
			AttachmentsVerification: []model.AttachmentVerification{
				{Name: "report.pdf", VerificationStatus: model.Success},
				{Name: "contract.pdf", VerificationStatus: model.Failed, FailureReason: "checksum mismatch"},
			},
		}},
	}

	out := captureOutput(func() {
		err := PlaintextReportPrinter.Print(resp)
		assert.NoError(t, err)
	})
	assert.Contains(t, out, "Attachments verification status:")
	assert.Contains(t, out, "Attachments summary:             1/2 verified")
	assert.Contains(t, out, "• contract.pdf "+PlaintextReportPrinter.failed)
	assert.Contains(t, out, "checksum mismatch")
}
