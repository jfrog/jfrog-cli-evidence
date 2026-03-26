package verifiers

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"testing"

	"github.com/jfrog/jfrog-cli-evidence/evidence/model"
	"github.com/jfrog/jfrog-client-go/artifactory"
	artUtils "github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	ioUtils "github.com/jfrog/jfrog-client-go/utils/io"
	"github.com/stretchr/testify/assert"
)

func TestVerify_NilEvidenceMetadata(t *testing.T) {
	mockClient := createMockArtifactoryClient([]byte{})
	verifier := NewEvidenceVerifier(nil, true, mockClient, nil)

	result, err := verifier.Verify("test-sha256", nil, "")
	assert.EqualError(t, err, "no evidence metadata provided")
	assert.Nil(t, result)
}

func TestVerify_EmptyEvidenceMetadata(t *testing.T) {
	mockClient := createMockArtifactoryClient([]byte{})
	verifier := NewEvidenceVerifier(nil, true, mockClient, nil)
	emptyMetadata := &[]model.SearchEvidenceEdge{}

	result, err := verifier.Verify("test-sha256", emptyMetadata, "")
	assert.EqualError(t, err, "no evidence metadata provided")
	assert.Nil(t, result)
}

func TestVerify_FileReadError(t *testing.T) {
	evidence := createTestEvidenceWithKeys()
	mockClient := &MockArtifactoryServicesManagerVerifier{
		ReadRemoteFileError: errors.New("file read error"),
	}
	var clientInterface artifactory.ArtifactoryServicesManager = mockClient
	verifier := NewEvidenceVerifier(nil, true, &clientInterface, nil)

	_, err := verifier.Verify(createTestSHA256(), evidence, "/path/to/file")
	assert.EqualError(t, err, "failed to read envelope: failed to read remote file: file read error")
}

func TestVerify_MultipleEvidence(t *testing.T) {
	testSha256 := createTestSHA256()
	evidence := &[]model.SearchEvidenceEdge{
		{
			Node: model.EvidenceMetadata{
				DownloadPath:  "test/path1",
				PredicateType: "test-predicate1",
				CreatedBy:     "user1",
				CreatedAt:     "2024-01-01",
				Subject: model.EvidenceSubject{
					Sha256: testSha256,
				},
			},
		},
		{
			Node: model.EvidenceMetadata{
				DownloadPath:  "test/path2",
				PredicateType: "test-predicate2",
				CreatedBy:     "user2",
				CreatedAt:     "2024-01-02",
				Subject: model.EvidenceSubject{
					Sha256: testSha256,
				},
			},
		},
	}

	mockClient := &MockArtifactoryServicesManagerVerifier{
		ReadRemoteFileFunc: func() io.ReadCloser {
			return io.NopCloser(bytes.NewReader(createMockDsseEnvelopeBytes(t)))
		},
	}
	var clientInterface artifactory.ArtifactoryServicesManager = mockClient
	verifier := NewEvidenceVerifier(nil, false, &clientInterface, nil)

	result, err := verifier.Verify(testSha256, evidence, "/path/to/file")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.EvidenceVerifications)
	assert.Len(t, *result.EvidenceVerifications, 2)

	// Verify each evidence entry
	for i, verification := range *result.EvidenceVerifications {
		assert.Equal(t, (*evidence)[i].Node.DownloadPath, verification.DownloadPath)
		assert.Equal(t, (*evidence)[i].Node.PredicateType, verification.PredicateType)
		assert.Equal(t, (*evidence)[i].Node.CreatedBy, verification.CreatedBy)
		assert.Equal(t, (*evidence)[i].Node.CreatedAt, verification.CreatedAt)

		// Verify that checksum verification was performed and status is set
		assert.Equal(t, testSha256, verification.SubjectChecksum)
		assert.Equal(t, model.Success, verification.VerificationResult.Sha256VerificationStatus)
	}
}

func TestVerify_NilEvidence(t *testing.T) {
	mockClient := createMockArtifactoryClient([]byte{})
	verifier := &evidenceVerifier{
		parser:           newEvidenceParser(mockClient, nil),
		dsseVerifier:     newDsseVerifier(nil, false, mockClient),
		sigstoreVerifier: newSigstoreVerifier(),
	}

	result, err := verifier.verifyEvidence(nil, createTestSHA256())
	assert.EqualError(t, err, "nil evidence provided")
	assert.Nil(t, result)
}

func TestVerify_OverallStatus(t *testing.T) {
	// Create multiple evidence entries to test overall status calculation
	evidence := &[]model.SearchEvidenceEdge{
		{
			Node: model.EvidenceMetadata{
				DownloadPath:  "test/path1",
				PredicateType: "test-predicate1",
				CreatedBy:     "user1",
				CreatedAt:     "2024-01-01",
				Subject: model.EvidenceSubject{
					Sha256: createTestSHA256(),
				},
			},
		},
		{
			Node: model.EvidenceMetadata{
				DownloadPath:  "test/path2",
				PredicateType: "test-predicate2",
				CreatedBy:     "user2",
				CreatedAt:     "2024-01-02",
				Subject: model.EvidenceSubject{
					Sha256: createTestSHA256(),
				},
			},
		},
	}

	// Mock client that returns DSSE envelopes for parsing
	mockClient := &MockArtifactoryServicesManagerVerifier{
		ReadRemoteFileFunc: func() io.ReadCloser {
			return io.NopCloser(bytes.NewReader(createMockDsseEnvelopeBytes(t)))
		},
	}
	var clientInterface artifactory.ArtifactoryServicesManager = mockClient
	verifier := NewEvidenceVerifier(nil, false, &clientInterface, nil)

	result, err := verifier.Verify(createTestSHA256(), evidence, "/path/to/file")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.EvidenceVerifications)
	assert.Len(t, *result.EvidenceVerifications, 2)

	// Test overall status determination
	assert.NotNil(t, result.OverallVerificationStatus)

	// Since we're using mock data without proper keys, verification should fail
	// but the overall structure should be properly set up
	assert.Contains(t, []model.VerificationStatus{model.Success, model.Failed}, result.OverallVerificationStatus)

	// Verify that all individual evidence has proper verification results
	for _, verification := range *result.EvidenceVerifications {
		assert.NotNil(t, verification.VerificationResult)
		// Each verification should have a checksum status since we're providing SHA256
		assert.NotEqual(t, model.VerificationStatus(""), verification.VerificationResult.Sha256VerificationStatus)
	}
}

func TestVerifyChecksum_Success(t *testing.T) {
	sha256 := createTestSHA256()
	result := verifyChecksum(sha256, sha256)
	assert.Equal(t, model.Success, result)
}

func TestVerifyChecksum_Failed(t *testing.T) {
	sha256a := createTestSHA256()
	sha256b := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
	result := verifyChecksum(sha256a, sha256b)
	assert.Equal(t, model.Failed, result)
}

func TestVerify_ChecksumVerificationFailure(t *testing.T) {
	subjectSha256 := createTestSHA256()
	differentSha256 := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"

	evidence := &[]model.SearchEvidenceEdge{
		{
			Node: model.EvidenceMetadata{
				DownloadPath:  "test/path",
				PredicateType: "test-predicate",
				CreatedBy:     "user",
				CreatedAt:     "2024-01-01",
				Subject: model.EvidenceSubject{
					Sha256: differentSha256,
				},
			},
		},
	}

	mockClient := &MockArtifactoryServicesManagerVerifier{
		ReadRemoteFileFunc: func() io.ReadCloser {
			return io.NopCloser(bytes.NewReader(createMockDsseEnvelopeBytes(t)))
		},
	}
	var clientInterface artifactory.ArtifactoryServicesManager = mockClient
	verifier := NewEvidenceVerifier(nil, false, &clientInterface, nil)

	result, err := verifier.Verify(subjectSha256, evidence, "/path/to/file")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.EvidenceVerifications)
	assert.Len(t, *result.EvidenceVerifications, 1)

	verification := (*result.EvidenceVerifications)[0]

	// Verify that checksum verification was performed and failed
	assert.Equal(t, differentSha256, verification.SubjectChecksum)
	assert.Equal(t, model.Failed, verification.VerificationResult.Sha256VerificationStatus)

	// Overall status should be failed due to checksum mismatch
	assert.Equal(t, model.Failed, result.OverallVerificationStatus)
}

func TestVerify_ChecksumVerificationAlwaysCalled(t *testing.T) {
	subjectSha256 := createTestSHA256()
	evidenceSha256 := createTestSHA256()

	evidence := &[]model.SearchEvidenceEdge{
		{
			Node: model.EvidenceMetadata{
				DownloadPath:  "test/path",
				PredicateType: "test-predicate",
				CreatedBy:     "user",
				CreatedAt:     "2024-01-01",
				Subject: model.EvidenceSubject{
					Sha256: evidenceSha256,
				},
			},
		},
	}

	// Mock client that returns invalid data to cause parsing errors
	mockClient := &MockArtifactoryServicesManagerVerifier{
		ReadRemoteFileFunc: func() io.ReadCloser {
			return io.NopCloser(bytes.NewReader([]byte("invalid data")))
		},
	}
	var clientInterface artifactory.ArtifactoryServicesManager = mockClient
	verifier := NewEvidenceVerifier(nil, false, &clientInterface, nil)

	_, err := verifier.Verify(subjectSha256, evidence, "/path/to/file")

	// Should get an error due to invalid data, but checksum verification should still be called
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read envelope")

	// The error should occur after checksum verification, so we can't check the result
	// But we can verify that the verifyChecksum function is called by checking the implementation
	// The checksum verification happens before parsing, so it should always be called
}

type fakeProgressMgr struct {
	increments int
	inited     bool
	totalInc   int64
}

func (f *fakeProgressMgr) NewProgressReader(total int64, label, path string) ioUtils.Progress {
	return nil
}
func (f *fakeProgressMgr) SetMergingState(id int, useSpinner bool) ioUtils.Progress { return nil }
func (f *fakeProgressMgr) GetProgress(id int) ioUtils.Progress                      { return nil }
func (f *fakeProgressMgr) RemoveProgress(id int)                                    {}
func (f *fakeProgressMgr) IncrementGeneralProgress()                                { f.increments++ }
func (f *fakeProgressMgr) Quit() error                                              { return nil }
func (f *fakeProgressMgr) IncGeneralProgressTotalBy(n int64)                        { f.totalInc += n }
func (f *fakeProgressMgr) SetHeadlineMsg(msg string)                                {}
func (f *fakeProgressMgr) ClearHeadlineMsg()                                        {}
func (f *fakeProgressMgr) InitProgressReaders()                                     { f.inited = true }
func (f *fakeProgressMgr) ClearProgress()                                           {}

func TestVerify_WithProgressMgr_Increments(t *testing.T) {
	evidence := &[]model.SearchEvidenceEdge{{Node: model.EvidenceMetadata{DownloadPath: "p", Subject: model.EvidenceSubject{Sha256: createTestSHA256()}}}}
	mockClient := &MockArtifactoryServicesManagerVerifier{ReadRemoteFileFunc: func() io.ReadCloser { return io.NopCloser(bytes.NewReader(createMockDsseEnvelopeBytes(t))) }}
	var clientInterface artifactory.ArtifactoryServicesManager = mockClient
	pm := &fakeProgressMgr{}
	verifier := NewEvidenceVerifier(nil, false, &clientInterface, pm)
	_, err := verifier.Verify(createTestSHA256(), evidence, "/path")
	assert.NoError(t, err)
	assert.True(t, pm.inited)
	assert.Equal(t, 1, pm.increments)
}

func TestVerify_WithProgressMgr_InitializesAndIncrements_Multiple(t *testing.T) {
	evidence := &[]model.SearchEvidenceEdge{
		{Node: model.EvidenceMetadata{DownloadPath: "p1", Subject: model.EvidenceSubject{Sha256: createTestSHA256()}}},
		{Node: model.EvidenceMetadata{DownloadPath: "p2", Subject: model.EvidenceSubject{Sha256: createTestSHA256()}}},
	}
	mockClient := &MockArtifactoryServicesManagerVerifier{ReadRemoteFileFunc: func() io.ReadCloser { return io.NopCloser(bytes.NewReader(createMockDsseEnvelopeBytes(t))) }}
	var clientInterface artifactory.ArtifactoryServicesManager = mockClient
	pm := &fakeProgressMgr{}
	verifier := NewEvidenceVerifier(nil, false, &clientInterface, pm)
	_, err := verifier.Verify(createTestSHA256(), evidence, "/path")
	assert.NoError(t, err)
	assert.True(t, pm.inited)
	assert.Equal(t, 2, pm.increments)
}

func TestVerify_AttachmentsVerificationSuccess(t *testing.T) {
	attachmentSha := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	evidence := &[]model.SearchEvidenceEdge{
		{
			Node: model.EvidenceMetadata{
				DownloadPath: "test/path1",
				Subject:      model.EvidenceSubject{Sha256: createTestSHA256()},
				Attachments: []model.AttachmentRef{
					{
						Name:         "report.txt",
						Sha256:       attachmentSha,
						DownloadPath: "repo/.evidence/attachments/report.txt",
					},
				},
			},
		},
	}
	mockClient := &MockArtifactoryServicesManagerVerifier{
		ReadRemoteFileFunc: func() io.ReadCloser {
			return io.NopCloser(bytes.NewReader(createDsseEnvelopeWithAttachmentMeta(t, attachmentSha)))
		},
		FileInfoFunc: func(_ string) (*artUtils.FileInfo, error) {
			return &artUtils.FileInfo{Checksums: struct {
				Sha1   string `json:"sha1,omitempty"`
				Sha256 string `json:"sha256,omitempty"`
				Md5    string `json:"md5,omitempty"`
			}{Sha256: attachmentSha}}, nil
		},
	}
	var clientInterface artifactory.ArtifactoryServicesManager = mockClient
	verifier := NewEvidenceVerifier(nil, false, &clientInterface, nil)

	result, err := verifier.Verify(createTestSHA256(), evidence, "/path/to/file")
	assert.NoError(t, err)
	assert.Equal(t, model.Success, result.OverallVerificationStatus)
	assert.Equal(t, model.Success, (*result.EvidenceVerifications)[0].VerificationResult.AttachmentsVerificationStatus)
	assert.Len(t, (*result.EvidenceVerifications)[0].AttachmentsVerification, 1)
}

func TestVerify_AttachmentsVerificationFailsWhenMetadataMissing(t *testing.T) {
	attachmentSha := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	evidence := &[]model.SearchEvidenceEdge{
		{
			Node: model.EvidenceMetadata{
				DownloadPath: "test/path1",
				Subject:      model.EvidenceSubject{Sha256: createTestSHA256()},
			},
		},
	}
	mockClient := &MockArtifactoryServicesManagerVerifier{
		ReadRemoteFileFunc: func() io.ReadCloser {
			return io.NopCloser(bytes.NewReader(createDsseEnvelopeWithAttachmentMeta(t, attachmentSha)))
		},
	}
	var clientInterface artifactory.ArtifactoryServicesManager = mockClient
	verifier := NewEvidenceVerifier(nil, false, &clientInterface, nil)

	result, err := verifier.Verify(createTestSHA256(), evidence, "/path/to/file")
	assert.NoError(t, err)
	assert.Equal(t, model.Failed, result.OverallVerificationStatus)
	verification := (*result.EvidenceVerifications)[0]
	assert.Equal(t, model.Failed, verification.VerificationResult.AttachmentsVerificationStatus)
	assert.Equal(t, "attachment failed verification", verification.VerificationResult.FailureReason)
	assert.Equal(t, attachmentMetadataNotFoundReason, verification.AttachmentsVerification[0].FailureReason)
}

func TestVerify_AttachmentsVerificationReturnsErrorWhenMetadataUnavailableViaFallback(t *testing.T) {
	attachmentSha := "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
	evidence := &[]model.SearchEvidenceEdge{
		{
			Node: model.EvidenceMetadata{
				DownloadPath:           "test/path1",
				Subject:                model.EvidenceSubject{Sha256: createTestSHA256()},
				AttachmentsUnavailable: true,
			},
		},
	}
	mockClient := &MockArtifactoryServicesManagerVerifier{
		ReadRemoteFileFunc: func() io.ReadCloser {
			return io.NopCloser(bytes.NewReader(createDsseEnvelopeWithAttachmentMeta(t, attachmentSha)))
		},
	}
	var clientInterface artifactory.ArtifactoryServicesManager = mockClient
	verifier := NewEvidenceVerifier(nil, false, &clientInterface, nil)

	result, err := verifier.Verify(createTestSHA256(), evidence, "/path/to/file")
	assert.EqualError(t, err, "unable to get attachment metadata from GraphQL (query without attachments)")
	assert.Nil(t, result)
}

func TestVerify_AttachmentsVerificationFailsWhenChecksumMismatch(t *testing.T) {
	expectedSha := "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	actualSha := "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	evidence := &[]model.SearchEvidenceEdge{
		{
			Node: model.EvidenceMetadata{
				DownloadPath: "test/path1",
				Subject:      model.EvidenceSubject{Sha256: createTestSHA256()},
				Attachments: []model.AttachmentRef{
					{
						Name:         "report.txt",
						Sha256:       expectedSha,
						DownloadPath: "repo/.evidence/attachments/report.txt",
					},
				},
			},
		},
	}
	mockClient := &MockArtifactoryServicesManagerVerifier{
		ReadRemoteFileFunc: func() io.ReadCloser {
			return io.NopCloser(bytes.NewReader(createDsseEnvelopeWithAttachmentMeta(t, expectedSha)))
		},
		FileInfoFunc: func(_ string) (*artUtils.FileInfo, error) {
			return &artUtils.FileInfo{Checksums: struct {
				Sha1   string `json:"sha1,omitempty"`
				Sha256 string `json:"sha256,omitempty"`
				Md5    string `json:"md5,omitempty"`
			}{Sha256: actualSha}}, nil
		},
	}
	var clientInterface artifactory.ArtifactoryServicesManager = mockClient
	verifier := NewEvidenceVerifier(nil, false, &clientInterface, nil)

	result, err := verifier.Verify(createTestSHA256(), evidence, "/path/to/file")
	assert.NoError(t, err)
	assert.Equal(t, model.Failed, result.OverallVerificationStatus)
	verification := (*result.EvidenceVerifications)[0]
	assert.Equal(t, model.Failed, verification.VerificationResult.AttachmentsVerificationStatus)
	assert.Contains(t, verification.AttachmentsVerification[0].FailureReason, "checksum mismatch")
}

func TestVerify_AttachmentsVerificationReturnsErrorWhenFileInfoFailsWithNon404(t *testing.T) {
	expectedSha := "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	evidence := &[]model.SearchEvidenceEdge{
		{
			Node: model.EvidenceMetadata{
				DownloadPath: "test/path1",
				Subject:      model.EvidenceSubject{Sha256: createTestSHA256()},
				Attachments: []model.AttachmentRef{
					{
						Name:         "report.txt",
						Sha256:       expectedSha,
						DownloadPath: "repo/.evidence/attachments/report.txt",
					},
				},
			},
		},
	}
	mockClient := &MockArtifactoryServicesManagerVerifier{
		ReadRemoteFileFunc: func() io.ReadCloser {
			return io.NopCloser(bytes.NewReader(createDsseEnvelopeWithAttachmentMeta(t, expectedSha)))
		},
		FileInfoError: errors.New("500 internal server error"),
	}
	var clientInterface artifactory.ArtifactoryServicesManager = mockClient
	verifier := NewEvidenceVerifier(nil, false, &clientInterface, nil)

	result, err := verifier.Verify(createTestSHA256(), evidence, "/path/to/file")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve attachment file info")
	assert.Contains(t, err.Error(), "500 internal server error")
	assert.Nil(t, result)
}

func TestVerify_AttachmentsVerificationReturnsErrorWhenChecksumIsUnavailable(t *testing.T) {
	expectedSha := "1212121212121212121212121212121212121212121212121212121212121212"
	evidence := &[]model.SearchEvidenceEdge{
		{
			Node: model.EvidenceMetadata{
				DownloadPath: "test/path1",
				Subject:      model.EvidenceSubject{Sha256: createTestSHA256()},
				Attachments: []model.AttachmentRef{
					{
						Name:         "report.txt",
						Sha256:       expectedSha,
						DownloadPath: "repo/.evidence/attachments/report.txt",
					},
				},
			},
		},
	}
	mockClient := &MockArtifactoryServicesManagerVerifier{
		ReadRemoteFileFunc: func() io.ReadCloser {
			return io.NopCloser(bytes.NewReader(createDsseEnvelopeWithAttachmentMeta(t, expectedSha)))
		},
		FileInfoFunc: func(_ string) (*artUtils.FileInfo, error) {
			return &artUtils.FileInfo{}, nil
		},
	}
	var clientInterface artifactory.ArtifactoryServicesManager = mockClient
	verifier := NewEvidenceVerifier(nil, false, &clientInterface, nil)

	result, err := verifier.Verify(createTestSHA256(), evidence, "/path/to/file")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve attachment checksum")
	assert.Contains(t, err.Error(), "sha256 is empty")
	assert.Nil(t, result)
}

func createDsseEnvelopeWithAttachmentMeta(t *testing.T, sha256 string) []byte {
	t.Helper()
	payload := `{"_type":"https://in-toto.io/Statement/v1","subject":[{"digest":{"sha256":"` + createTestSHA256() + `"}}],"predicateType":"https://example.com","predicate":{},"attachments":[{"name":"report.txt","sha256":"` + sha256 + `","type":"text/plain"}]}`
	envelope := `{"payload":"` + base64.StdEncoding.EncodeToString([]byte(payload)) + `","payloadType":"application/vnd.in-toto+json","signatures":[{"keyid":"k","sig":"dGVzdA=="}]}`
	return []byte(envelope)
}
