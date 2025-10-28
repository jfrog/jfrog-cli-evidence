package verifiers

import (
	"crypto/x509"
	"fmt"

	"github.com/jfrog/jfrog-cli-evidence/evidence/verify/verifiers/ca"
	v1 "github.com/sigstore/protobuf-specs/gen/pb-go/bundle/v1"

	"github.com/jfrog/jfrog-cli-evidence/evidence/model"
	"github.com/pkg/errors"
	"github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/verify"
)

const (
	sigstoreKeySource = "Sigstore Bundle Key"
	gitHubIssuerOrg   = "GitHub, Inc."
	publicGood        = "public-good"
)

type sigstoreVerifierInterface interface {
	verify(result *model.EvidenceVerification) error
}

type sigstoreVerifier struct {
	rootCertificateProvider ca.TUFRootCertificateProvider
}

func newSigstoreVerifier() sigstoreVerifierInterface {
	return &sigstoreVerifier{}
}

func (v *sigstoreVerifier) verify(result *model.EvidenceVerification) error {
	if result == nil || result.SigstoreBundle == nil {
		return fmt.Errorf("empty evidence verification or Sigstore bundle provided for verification")
	}

	if v.rootCertificateProvider == nil {
		v.rootCertificateProvider = ca.NewTUFRootCertificateProvider()
	}

	protoBundle := result.SigstoreBundle.Bundle
	if protoBundle == nil {
		return errors.New("invalid bundle: missing protobuf bundle")
	}

	issuer, err := extractIssuerFromBundle(protoBundle)
	if err != nil {
		return fmt.Errorf("failed to extract issuer from bundle: %v", err)
	}

	// Load the appropriate trustedMaterial based on the issuer
	trustedMaterial, verifierConfig, err := v.prepareVerificationData(issuer)
	if err != nil {
		return err
	}

	verifier, err := verify.NewVerifier(trustedMaterial, verifierConfig...)
	if err != nil {
		return fmt.Errorf("failed to create signature verifier: %v", err)
	}

	bundleToVerify, err := bundle.NewBundle(protoBundle)
	if err != nil {
		return errors.Wrap(err, "failed to create bundle for verification")
	}

	policy := verify.NewPolicy(
		verify.WithoutArtifactUnsafe(),   // Skip artifact verification due to separate digest verification
		verify.WithoutIdentitiesUnsafe(), // Skip identity verification for now
	)

	verificationResult, err := verifier.Verify(bundleToVerify, policy)
	if err != nil {
		result.VerificationResult.SigstoreBundleVerificationStatus = model.Failed
		result.VerificationResult.FailureReason = err.Error()
		return nil //nolint:nilerr
	}
	result.VerificationResult.KeySource = sigstoreKeySource
	result.VerificationResult.SigstoreBundleVerificationStatus = model.Success
	result.VerificationResult.SigstoreBundleVerificationResult = verificationResult
	return nil
}

func (v *sigstoreVerifier) prepareVerificationData(issuer string) (root.TrustedMaterial, []verify.VerifierOption, error) {
	switch issuer {
	case gitHubIssuerOrg:
		trustedMaterial, err := v.rootCertificateProvider.LoadTUFRootGithubCertificate()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load GitHub TUF root trustedMaterial: %v", err)
		}
		verifierConfig := []verify.VerifierOption{
			verify.WithSignedTimestamps(1),
		}
		return trustedMaterial, verifierConfig, nil
	case publicGood:
		trustedMaterial, err := v.rootCertificateProvider.LoadTUFRootCertificate()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load TUF root trustedMaterial: %v", err)
		}
		verifierConfig := []verify.VerifierOption{
			verify.WithSignedCertificateTimestamps(1),
			verify.WithObserverTimestamps(1),
			verify.WithTransparencyLog(1),
		}

		return trustedMaterial, verifierConfig, nil
	default:
		return nil, nil, fmt.Errorf("unsupported issuer: %s", issuer)
	}

}

func extractIssuerFromBundle(bundle *v1.Bundle) (string, error) {
	if bundle == nil || bundle.VerificationMaterial == nil {
		return "", fmt.Errorf("bundle has no verification material")
	}

	vm := bundle.VerificationMaterial
	var certDER []byte

	switch content := vm.Content.(type) {
	case *v1.VerificationMaterial_X509CertificateChain:
		if content.X509CertificateChain != nil &&
			len(content.X509CertificateChain.Certificates) > 0 {
			certDER = content.X509CertificateChain.Certificates[0].RawBytes
		}
	case *v1.VerificationMaterial_Certificate:
		if content.Certificate != nil {
			certDER = content.Certificate.RawBytes
		}
	default:
		return "", fmt.Errorf("unsupported verification material type: %T", content)
	}

	if len(certDER) == 0 {
		return "", fmt.Errorf("no certificate found in bundle")
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return "", err
	}

	if len(cert.Issuer.Organization) > 0 {
		return cert.Issuer.Organization[0], nil
	}

	return "", fmt.Errorf("no organization found in bundle")
}
