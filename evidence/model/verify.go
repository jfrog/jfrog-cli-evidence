package model

import (
	"strings"

	"github.com/jfrog/jfrog-cli-evidence/evidence/dsse"
	"github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/sigstore/sigstore-go/pkg/verify"
)

const SchemaVersion = "1.3"

// Sha256DigestType is the digest type used for subjects identified by the sha256 of their content.
const Sha256DigestType = "sha256"

type VerificationResponse struct {
	// Update the schemaVersion value when this structure is updated.
	SchemaVersion             string                  `json:"schemaVersion"`
	Subject                   Subject                 `json:"subject"`
	EvidenceVerifications     *[]EvidenceVerification `json:"evidenceVerifications"`
	OverallVerificationStatus VerificationStatus      `json:"overallVerificationStatus"`
}

type Subject struct {
	Path   string `json:"path"`
	Sha256 string `json:"sha256,omitempty"`
}

// SubjectDigest is the digest entry a signed in-toto statement is expected to carry for
// the verified subject. Content based subjects use Sha256DigestType, while entity subjects
// use the entity type as the digest type and the entity id as the value.
type SubjectDigest struct {
	Type  string
	Value string
}

// IsSha256 reports whether the subject is identified by the sha256 of its content.
func (s SubjectDigest) IsSha256() bool {
	return s.Type == "" || strings.EqualFold(s.Type, Sha256DigestType)
}

type EvidenceVerification struct {
	MediaType       MediaType `json:"mediaType"`
	DownloadPath    string    `json:"downloadPath"`
	SubjectChecksum string    `json:"evidenceSubjectSha256,omitempty"`
	// SignedSubjectDigest is the subject digest found in the signed statement. It is reported
	// for subjects that are not identified by a content checksum, such as entity subjects.
	SignedSubjectDigest     map[string]string          `json:"signedSubjectDigest,omitempty"`
	PredicateType           string                     `json:"predicateType"`
	CreatedBy               string                     `json:"createdBy"`
	CreatedAt               string                     `json:"createdAt"`
	VerificationResult      EvidenceVerificationResult `json:"verificationResult"`
	AttachmentsVerification []AttachmentVerification   `json:"attachmentsVerification,omitempty"`
	DsseEnvelope            *dsse.Envelope             `json:"dsseEnvelope,omitempty"`
	SigstoreBundle          *bundle.Bundle             `json:"sigstoreBundle,omitempty"`
}

type EvidenceVerificationResult struct {
	Sha256VerificationStatus VerificationStatus `json:"sha256VerificationStatus,omitempty"`
	// SubjectDigestVerificationStatus reports whether the signed statement carries the expected
	// subject digest. It is set instead of Sha256VerificationStatus for entity subjects.
	SubjectDigestVerificationStatus  VerificationStatus         `json:"subjectDigestVerificationStatus,omitempty"`
	SignaturesVerificationStatus     VerificationStatus         `json:"signaturesVerificationStatus,omitempty"`
	SigstoreBundleVerificationStatus VerificationStatus         `json:"sigstoreBundleVerificationStatus,omitempty"`
	KeySource                        string                     `json:"keySource,omitempty"`
	KeyFingerprint                   string                     `json:"keyFingerprint,omitempty"`
	SigstoreBundleVerificationResult *verify.VerificationResult `json:"sigstoreBundleVerificationResult,omitempty"`
	AttachmentsVerificationStatus    VerificationStatus         `json:"attachmentsVerificationStatus,omitempty"`
	FailureReason                    string                     `json:"failureReason,omitempty"`
}

type AttachmentVerification struct {
	Name               string             `json:"name"`
	ExpectedSha256     string             `json:"expectedSha256,omitempty"`
	ActualSha256       string             `json:"actualSha256,omitempty"`
	DownloadPath       string             `json:"downloadPath,omitempty"`
	VerificationStatus VerificationStatus `json:"verificationStatus"`
	FailureReason      string             `json:"failureReason,omitempty"`
}

type VerificationStatus string

const (
	Success VerificationStatus = "success"
	Failed  VerificationStatus = "failed"
)

type MediaType string

const (
	SigstoreBundle MediaType = "sigstore.bundle"
	SimpleDSSE     MediaType = "evidence.dsse"
)
