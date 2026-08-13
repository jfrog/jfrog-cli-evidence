package verifiers

import (
	"fmt"
	"os"

	"github.com/jfrog/jfrog-cli-evidence/evidence/cryptox"
	"github.com/jfrog/jfrog-cli-evidence/evidence/dsse"
	"github.com/jfrog/jfrog-cli-evidence/evidence/model"
	"github.com/jfrog/jfrog-client-go/artifactory"
	clientLog "github.com/jfrog/jfrog-client-go/utils/log"
)

const localKeySource = "User Provided Key"
const artifactoryKeySource = "Artifactory Key"

const (
	federatedVerificationNote  = "signature was cryptographically verified on the federated JPD where the evidence was created, and the verified result was replicated with the evidence to this JPD; to verify locally, provide a public key via --public-keys or upload it to Artifactory and use --use-artifactory-keys"
	noPublicKeyAvailableReason = "no public key available"
)

type dsseVerifierInterface interface {
	verify(evidence *model.SearchEvidenceEdge, result *model.EvidenceVerification) error
}

type dsseVerifier struct {
	keys               []string
	useArtifactoryKeys bool
	localKeys          []dsse.Verifier
	artifactoryClient  *artifactory.ArtifactoryServicesManager
	attachmentVerifier attachmentVerifierInterface
}

func newDsseVerifier(keys []string, useArtifactoryKeys bool, client *artifactory.ArtifactoryServicesManager) dsseVerifierInterface {
	return &dsseVerifier{
		keys:               keys,
		useArtifactoryKeys: useArtifactoryKeys,
		artifactoryClient:  client,
		attachmentVerifier: newAttachmentVerifier(*client),
	}
}

func (v *dsseVerifier) verify(evidence *model.SearchEvidenceEdge, result *model.EvidenceVerification) error {
	if evidence == nil || result == nil {
		return fmt.Errorf("empty evidence or result provided for DSSE verification")
	}
	localVerifiers, err := v.getLocalVerifiers()
	if err != nil && v.keys != nil && len(v.keys) > 0 {
		return err
	}

	var artifactoryVerifiers []dsse.Verifier
	if v.useArtifactoryKeys {
		artifactoryVerifiers, err = getArtifactoryVerifiers(evidence)
		if err != nil {
			return err
		}
	}

	var failureReason string
	if len(localVerifiers) > 0 || len(artifactoryVerifiers) > 0 {
		v.verifyWithKeys(localVerifiers, artifactoryVerifiers, result)
	} else {
		failureReason = verifyViaEvidenceVerified(evidence, result)
	}

	if err := v.attachmentVerifier.verify(evidence, result); err != nil {
		return err
	}
	// An attachment failure carries a more specific reason, so it keeps the field.
	if failureReason != "" && result.VerificationResult.FailureReason == "" {
		result.VerificationResult.FailureReason = failureReason
	}
	return nil
}

func (v *dsseVerifier) verifyWithKeys(localVerifiers, artifactoryVerifiers []dsse.Verifier, result *model.EvidenceVerification) {
	if len(localVerifiers) > 0 && verifyEnvelope(localVerifiers, result.DsseEnvelope, result) {
		result.VerificationResult.KeySource = localKeySource
		return
	}
	if len(artifactoryVerifiers) > 0 && verifyEnvelope(artifactoryVerifiers, result.DsseEnvelope, result) {
		result.VerificationResult.KeySource = artifactoryKeySource
		return
	}
	// Keys were present but signature invalid — do not fall back to the federated verified flag.
	if result.VerificationResult.SignaturesVerificationStatus == "" {
		result.VerificationResult.SignaturesVerificationStatus = model.Failed
	}
}

// verifyViaEvidenceVerified trusts Evidence GraphQL verified (DB) when no public key is available.
// That flag is what federated ingest stored from evidence.verified; on an edge after Distribution
// override it is false, so verify stays not verified without a Trusted Key.
func verifyViaEvidenceVerified(evidence *model.SearchEvidenceEdge, result *model.EvidenceVerification) string {
	result.VerificationResult.SignaturesVerificationStatus = model.Failed
	if evidence != nil && evidence.Node.Verified {
		result.VerificationResult.SignaturesVerificationStatus = model.Success
		result.VerificationResult.SignaturesVerificationNote = federatedVerificationNote
		return ""
	}
	return noPublicKeyAvailableReason
}

func (v *dsseVerifier) getLocalVerifiers() ([]dsse.Verifier, error) {
	if v.localKeys != nil {
		return v.localKeys, nil
	}
	var keys []dsse.Verifier
	for _, keyPath := range v.keys {
		if keyPath == "" {
			continue
		}
		keyFile, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read key %s: %w", keyPath, err)
		}
		loadedKey, err := cryptox.ReadPublicKey(keyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load key %s: %w", keyPath, err)
		}
		if loadedKey == nil {
			return nil, fmt.Errorf("key is null or empty %s", keyPath)
		}
		verifier, err := cryptox.CreateVerifier(loadedKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create verifier for key %s: %w", keyPath, err)
		}
		keys = append(keys, verifier...)
	}
	v.localKeys = keys
	return keys, nil
}

// verifyEnvelope returns true if verification succeeded, false otherwise. Uses pointer for result.
func verifyEnvelope(verifiers []dsse.Verifier, envelope *dsse.Envelope, result *model.EvidenceVerification) bool {
	// formal check for empty result
	if result == nil {
		return false
	}
	if verifiers == nil || envelope == nil {
		result.VerificationResult.SignaturesVerificationStatus = model.Failed
		return false
	}
	for _, verifier := range verifiers {
		if err := envelope.Verify(verifier); err == nil {
			result.VerificationResult.SignaturesVerificationStatus = model.Success
			fingerprint, err := cryptox.GenerateFingerprint(verifier.Public())
			if err != nil {
				clientLog.Warn("Failed to generate fingerprint for the key: %s", verifier.Public())
			} else {
				result.VerificationResult.KeyFingerprint = fingerprint
			}
			return true
		}
	}
	result.VerificationResult.SignaturesVerificationStatus = model.Failed
	return false
}

func getArtifactoryVerifiers(evidence *model.SearchEvidenceEdge) ([]dsse.Verifier, error) {
	if evidence == nil {
		return nil, fmt.Errorf("empty evidence provided for artifactory verifier retrieval")
	}
	evidenceSigningKey := evidence.Node.SigningKey
	if evidenceSigningKey.PublicKey == "" {
		return []dsse.Verifier{}, nil
	}
	key, err := cryptox.LoadKey([]byte(evidenceSigningKey.PublicKey))
	if err != nil {
		return nil, fmt.Errorf("failed to load artifactory key: %w", err)
	}
	verifier, err := cryptox.CreateVerifier(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create verifier for evidence predicate type: %s", evidence.Node.PredicateType)
	}
	return verifier, nil
}
