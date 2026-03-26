package verifiers

import (
	"encoding/base64"
	"errors"
	"testing"

	"github.com/jfrog/jfrog-cli-evidence/evidence/dsse"
	"github.com/jfrog/jfrog-cli-evidence/evidence/model"
	"github.com/jfrog/jfrog-client-go/artifactory"
	artUtils "github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	"github.com/stretchr/testify/assert"
)

func TestAttachmentVerifier_Verify_Success(t *testing.T) {
	attachmentSha := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	evidence := &model.SearchEvidenceEdge{
		Node: model.EvidenceMetadata{
			Attachments: []model.AttachmentRef{
				{
					Name:         "report.txt",
					Sha256:       attachmentSha,
					DownloadPath: "repo/.evidence/attachments/report.txt",
				},
			},
		},
	}
	result := &model.EvidenceVerification{
		DsseEnvelope: dsseEnvelopeWithAttachment(t, attachmentSha),
	}

	mockClient := &MockArtifactoryServicesManagerVerifier{
		FileInfoFunc: func(_ string) (*artUtils.FileInfo, error) {
			return &artUtils.FileInfo{Checksums: struct {
				Sha1   string `json:"sha1,omitempty"`
				Sha256 string `json:"sha256,omitempty"`
				Md5    string `json:"md5,omitempty"`
			}{Sha256: attachmentSha}}, nil
		},
	}
	var clientInterface artifactory.ArtifactoryServicesManager = mockClient
	verifier := newAttachmentVerifier(clientInterface)

	err := verifier.verify(evidence, result)
	assert.NoError(t, err)
	assert.Equal(t, model.Success, result.VerificationResult.AttachmentsVerificationStatus)
	assert.Len(t, result.AttachmentsVerification, 1)
	assert.Equal(t, model.Success, result.AttachmentsVerification[0].VerificationStatus)
}

func TestAttachmentVerifier_Verify_ReturnsErrorOnNilInputs(t *testing.T) {
	mockClient := &MockArtifactoryServicesManagerVerifier{}
	var clientInterface artifactory.ArtifactoryServicesManager = mockClient
	verifier := newAttachmentVerifier(clientInterface)

	err := verifier.verify(nil, nil)
	assert.EqualError(t, err, "empty evidence or DSSE envelope provided for attachment verification")
}

func TestAttachmentVerifier_Verify_MetadataMissing(t *testing.T) {
	attachmentSha := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	evidence := &model.SearchEvidenceEdge{
		Node: model.EvidenceMetadata{},
	}
	result := &model.EvidenceVerification{
		DsseEnvelope: dsseEnvelopeWithAttachment(t, attachmentSha),
	}

	mockClient := &MockArtifactoryServicesManagerVerifier{}
	var clientInterface artifactory.ArtifactoryServicesManager = mockClient
	verifier := newAttachmentVerifier(clientInterface)

	err := verifier.verify(evidence, result)
	assert.NoError(t, err)
	assert.Equal(t, model.Failed, result.VerificationResult.AttachmentsVerificationStatus)
	assert.Equal(t, attachmentVerificationFailedReason, result.VerificationResult.FailureReason)
	assert.Len(t, result.AttachmentsVerification, 1)
	assert.Equal(t, attachmentMetadataNotFoundReason, result.AttachmentsVerification[0].FailureReason)
}

func TestAttachmentVerifier_Verify_ReturnsErrorWhenDssePayloadAttachmentsCannotBeParsed(t *testing.T) {
	evidence := &model.SearchEvidenceEdge{
		Node: model.EvidenceMetadata{},
	}
	result := &model.EvidenceVerification{
		DsseEnvelope: &dsse.Envelope{
			Payload: "not-base64",
		},
	}

	mockClient := &MockArtifactoryServicesManagerVerifier{}
	var clientInterface artifactory.ArtifactoryServicesManager = mockClient
	verifier := newAttachmentVerifier(clientInterface)

	err := verifier.verify(evidence, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse DSSE payload attachments")
}

func TestAttachmentVerifier_Verify_FileInfoNon404Error(t *testing.T) {
	attachmentSha := "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	evidence := &model.SearchEvidenceEdge{
		Node: model.EvidenceMetadata{
			Attachments: []model.AttachmentRef{
				{
					Name:         "report.txt",
					Sha256:       attachmentSha,
					DownloadPath: "repo/.evidence/attachments/report.txt",
				},
			},
		},
	}
	result := &model.EvidenceVerification{
		DsseEnvelope: dsseEnvelopeWithAttachment(t, attachmentSha),
	}

	mockClient := &MockArtifactoryServicesManagerVerifier{
		FileInfoError: errors.New("500 internal server error"),
	}
	var clientInterface artifactory.ArtifactoryServicesManager = mockClient
	verifier := newAttachmentVerifier(clientInterface)

	err := verifier.verify(evidence, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve attachment file info")
}

func TestAttachmentVerifier_Verify_EmptyChecksumReturnsError(t *testing.T) {
	attachmentSha := "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	evidence := &model.SearchEvidenceEdge{
		Node: model.EvidenceMetadata{
			Attachments: []model.AttachmentRef{
				{
					Name:         "report.txt",
					Sha256:       attachmentSha,
					DownloadPath: "repo/.evidence/attachments/report.txt",
				},
			},
		},
	}
	result := &model.EvidenceVerification{
		DsseEnvelope: dsseEnvelopeWithAttachment(t, attachmentSha),
	}

	mockClient := &MockArtifactoryServicesManagerVerifier{
		FileInfoFunc: func(_ string) (*artUtils.FileInfo, error) {
			return &artUtils.FileInfo{}, nil
		},
	}
	var clientInterface artifactory.ArtifactoryServicesManager = mockClient
	verifier := newAttachmentVerifier(clientInterface)

	err := verifier.verify(evidence, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve attachment checksum")
}

func TestIsAttachmentNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "standard Artifactory 404 response",
			err:      errors.New("server response: 404 Not Found"),
			expected: true,
		},
		{
			name:     "404 Not Found with body",
			err:      errors.New("server response: 404 Not Found\n{\"errors\":[{\"status\":404,\"message\":\"File not found.\"}]}"),
			expected: true,
		},
		{
			name:     "500 internal server error",
			err:      errors.New("server response: 500 Internal Server Error"),
			expected: false,
		},
		{
			name:     "error containing bare 404 without Not Found",
			err:      errors.New("returned 4040 bytes"),
			expected: false,
		},
		{
			name:     "error containing Not Found without 404",
			err:      errors.New("resource Not Found in cache"),
			expected: false,
		},
		{
			name:     "unrelated error",
			err:      errors.New("connection refused"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isAttachmentNotFoundError(tt.err))
		})
	}
}

func TestAttachmentVerifier_Verify_FileNotFound404(t *testing.T) {
	attachmentSha := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	evidence := &model.SearchEvidenceEdge{
		Node: model.EvidenceMetadata{
			Attachments: []model.AttachmentRef{
				{
					Name:         "report.txt",
					Sha256:       attachmentSha,
					DownloadPath: "repo/.evidence/attachments/report.txt",
				},
			},
		},
	}
	result := &model.EvidenceVerification{
		DsseEnvelope: dsseEnvelopeWithAttachment(t, attachmentSha),
	}

	mockClient := &MockArtifactoryServicesManagerVerifier{
		FileInfoError: errors.New("server response: 404 Not Found"),
	}
	var clientInterface artifactory.ArtifactoryServicesManager = mockClient
	verifier := newAttachmentVerifier(clientInterface)

	err := verifier.verify(evidence, result)
	assert.NoError(t, err)
	assert.Equal(t, model.Failed, result.VerificationResult.AttachmentsVerificationStatus)
	if assert.Len(t, result.AttachmentsVerification, 1) {
		assert.Equal(t, attachmentArtifactNotFoundReason, result.AttachmentsVerification[0].FailureReason)
		assert.Equal(t, model.Failed, result.AttachmentsVerification[0].VerificationStatus)
	}
}

func dsseEnvelopeWithAttachment(t *testing.T, sha256 string) *dsse.Envelope {
	t.Helper()
	payload := `{"_type":"https://in-toto.io/Statement/v1","subject":[{"digest":{"sha256":"` + createTestSHA256() + `"}}],"predicateType":"https://example.com","predicate":{},"attachments":[{"name":"report.txt","sha256":"` + sha256 + `","type":"text/plain"}]}`
	return &dsse.Envelope{
		Payload:     base64.StdEncoding.EncodeToString([]byte(payload)),
		PayloadType: "application/vnd.in-toto+json",
		Signatures: []dsse.Signature{
			{
				KeyId: "k",
				Sig:   base64.StdEncoding.EncodeToString([]byte("test")),
			},
		},
	}
}

func TestAttachmentVerifier_Verify_MetadataUnavailable_SignatureVerified_ReturnsError(t *testing.T) {
	attachmentSha := "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
	evidence := &model.SearchEvidenceEdge{
		Node: model.EvidenceMetadata{
			AttachmentsUnavailable: true,
		},
	}
	result := &model.EvidenceVerification{
		DsseEnvelope:       dsseEnvelopeWithAttachment(t, attachmentSha),
		VerificationResult: model.EvidenceVerificationResult{SignaturesVerificationStatus: model.Success},
	}

	mockClient := &MockArtifactoryServicesManagerVerifier{}
	var clientInterface artifactory.ArtifactoryServicesManager = mockClient
	verifier := newAttachmentVerifier(clientInterface)

	err := verifier.verify(evidence, result)
	assert.EqualError(t, err, attachmentMetadataUnavailableReason)
}

func TestAttachmentVerifier_Verify_MetadataUnavailable_SignatureFailed_FailsGracefully(t *testing.T) {
	attachmentSha := "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
	evidence := &model.SearchEvidenceEdge{
		Node: model.EvidenceMetadata{
			AttachmentsUnavailable: true,
		},
	}
	result := &model.EvidenceVerification{
		DsseEnvelope:       dsseEnvelopeWithAttachment(t, attachmentSha),
		VerificationResult: model.EvidenceVerificationResult{SignaturesVerificationStatus: model.Failed},
	}

	mockClient := &MockArtifactoryServicesManagerVerifier{}
	var clientInterface artifactory.ArtifactoryServicesManager = mockClient
	verifier := newAttachmentVerifier(clientInterface)

	err := verifier.verify(evidence, result)
	assert.NoError(t, err)
	assert.Equal(t, model.Failed, result.VerificationResult.AttachmentsVerificationStatus)
	assert.Equal(t, attachmentMetadataUnavailableReason, result.VerificationResult.FailureReason)
}
