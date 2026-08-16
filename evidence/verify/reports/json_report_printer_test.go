package reports

import (
	"testing"

	"github.com/jfrog/jfrog-cli-evidence/evidence/model"
	"github.com/stretchr/testify/assert"
)

func TestJsonPrinter_Success(t *testing.T) {
	resp := &model.VerificationResponse{
		Subject: model.Subject{
			Sha256: "test-checksum",
		},
		OverallVerificationStatus: model.Success,
		EvidenceVerifications: &[]model.EvidenceVerification{{
			VerificationResult: model.EvidenceVerificationResult{
				Sha256VerificationStatus:     model.Success,
				SignaturesVerificationStatus: model.Success,
				SignaturesVerificationNote:   "recorded verification note",
			},
		}},
	}

	out := captureOutput(func() {
		err := JsonReportPrinter.Print(resp)
		assert.NoError(t, err)
	})

	assert.Contains(t, out, `"signaturesVerificationNote": "recorded verification note"`)
	// The status enum stays machine-readable; the inline caveat is plaintext-only.
	assert.Contains(t, out, `"signaturesVerificationStatus": "success"`)
	assert.NotContains(t, out, "recorded result, not verified in this run")
	assert.NotContains(t, out, `"source"`)
}

func TestJsonPrinter_NilResponse(t *testing.T) {
	err := JsonReportPrinter.Print(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "verification response is empty")
}
