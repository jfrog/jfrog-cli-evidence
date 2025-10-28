package verifiers

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/jfrog/jfrog-cli-evidence/evidence/model"
	v1 "github.com/sigstore/protobuf-specs/gen/pb-go/bundle/v1"
	common "github.com/sigstore/protobuf-specs/gen/pb-go/common/v1"
	dssepb "github.com/sigstore/protobuf-specs/gen/pb-go/dsse"
	"github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/stretchr/testify/assert"
)

func TestSigstoreVerifier_VerifyNilResult(t *testing.T) {
	verifier := &sigstoreVerifier{}

	err := verifier.verify(nil)
	assert.Error(t, err)
	assert.Equal(t, "empty evidence verification or Sigstore bundle provided for verification", err.Error())
}

func TestSigstoreVerifier_VerifyResultWithNilSigstoreBundle(t *testing.T) {
	verifier := &sigstoreVerifier{}

	result := &model.EvidenceVerification{
		SigstoreBundle:     nil,
		VerificationResult: model.EvidenceVerificationResult{},
	}

	err := verifier.verify(result)
	assert.Error(t, err)
	assert.Equal(t, "empty evidence verification or Sigstore bundle provided for verification", err.Error())
}

func TestSigstoreVerifier_VerifyNilProtobufBundle(t *testing.T) {
	mockProvider := &MockTUFRootCertificateProvider{}
	// No mock expectations needed - the bundle check happens first

	verifier := &sigstoreVerifier{
		rootCertificateProvider: mockProvider,
	}

	result := &model.EvidenceVerification{
		SigstoreBundle: &bundle.Bundle{
			Bundle: nil,
		},
		VerificationResult: model.EvidenceVerificationResult{},
	}

	err := verifier.verify(result)
	assert.Error(t, err)
	assert.Equal(t, "invalid bundle: missing protobuf bundle", err.Error())

	// Certificate provider should not be called when bundle is nil
}

func TestSigstoreVerifier_VerifyNilBundle(t *testing.T) {
	mockProvider := &MockTUFRootCertificateProvider{}
	// No mock expectations needed - the bundle is checked before certificate loading

	verifier := &sigstoreVerifier{
		rootCertificateProvider: mockProvider,
	}

	result := &model.EvidenceVerification{
		SigstoreBundle: &bundle.Bundle{
			Bundle: nil, // nil protobuf bundle
		},
		VerificationResult: model.EvidenceVerificationResult{},
	}

	err := verifier.verify(result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid bundle: missing protobuf bundle")

	// Certificate provider should not be called when bundle is nil
}

func TestSigstoreVerifier_VerifyTUFProviderError(t *testing.T) {
	mockProvider := &MockTUFRootCertificateProvider{}
	// Empty bundle will fail to extract issuer

	verifier := &sigstoreVerifier{
		rootCertificateProvider: mockProvider,
	}

	result := &model.EvidenceVerification{
		SigstoreBundle: &bundle.Bundle{
			Bundle: &v1.Bundle{}, // Empty but not nil - will fail issuer extraction
		},
		VerificationResult: model.EvidenceVerificationResult{},
	}

	err := verifier.verify(result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to extract issuer from bundle")

	// Verify that the mock provider was not called (failed earlier)
	mockProvider.AssertExpectations(t)
}

func TestSigstoreVerifier_Creation(t *testing.T) {
	verifier := newSigstoreVerifier()
	assert.NotNil(t, verifier)
}

func TestSigstoreVerifier_VerifyNilBundleAfterTUFSuccess(t *testing.T) {
	mockProvider := &MockTUFRootCertificateProvider{}
	// No mock expectations needed - bundle validation happens before TUF loading

	verifier := &sigstoreVerifier{
		rootCertificateProvider: mockProvider,
	}

	result := &model.EvidenceVerification{
		SigstoreBundle: &bundle.Bundle{
			Bundle: nil, // nil protobuf bundle
		},
		VerificationResult: model.EvidenceVerificationResult{},
	}

	err := verifier.verify(result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid bundle: missing protobuf bundle")

	// Certificate provider should not be called when bundle is nil
}

func TestExtractIssuerFromBundle(t *testing.T) {
	tests := []struct {
		name           string
		issuer         string
		expectedIssuer string
		expectError    bool
		errorContains  string
	}{
		{
			name:           "GitHub issuer",
			issuer:         "GitHub, Inc.",
			expectedIssuer: "GitHub, Inc.",
			expectError:    false,
		},
		{
			name:           "public-good issuer",
			issuer:         "public-good",
			expectedIssuer: "public-good",
			expectError:    false,
		},
		{
			name:           "Other issuer",
			issuer:         "Other Org",
			expectedIssuer: "Other Org",
			expectError:    false,
		},
		{
			name:          "Empty issuer returns error",
			issuer:        "",
			expectError:   true,
			errorContains: "no organization found in bundle",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bundleWithIssuer := createBundleWithIssuer(t, tt.issuer)
			issuer, err := extractIssuerFromBundle(bundleWithIssuer)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedIssuer, issuer)
			}
		})
	}
}

func TestExtractIssuerFromBundle_ErrorCases(t *testing.T) {
	tests := []struct {
		name          string
		bundle        *v1.Bundle
		errorContains string
	}{
		{
			name:          "Nil bundle",
			bundle:        nil,
			errorContains: "bundle has no verification material",
		},
		{
			name: "No verification material",
			bundle: &v1.Bundle{
				VerificationMaterial: nil,
			},
			errorContains: "bundle has no verification material",
		},
		{
			name: "Empty verification material",
			bundle: &v1.Bundle{
				VerificationMaterial: &v1.VerificationMaterial{},
			},
			errorContains: "unsupported verification material type",
		},
		{
			name: "Certificate with empty RawBytes",
			bundle: &v1.Bundle{
				VerificationMaterial: &v1.VerificationMaterial{
					Content: &v1.VerificationMaterial_Certificate{
						Certificate: &common.X509Certificate{
							RawBytes: []byte{},
						},
					},
				},
			},
			errorContains: "no certificate found in bundle",
		},
		{
			name: "Invalid certificate bytes",
			bundle: &v1.Bundle{
				VerificationMaterial: &v1.VerificationMaterial{
					Content: &v1.VerificationMaterial_Certificate{
						Certificate: &common.X509Certificate{
							RawBytes: []byte("invalid cert data"),
						},
					},
				},
			},
			errorContains: "", // Will get x509 parse error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issuer, err := extractIssuerFromBundle(tt.bundle)
			assert.Error(t, err)
			assert.Empty(t, issuer)
			if tt.errorContains != "" {
				assert.Contains(t, err.Error(), tt.errorContains)
			}
		})
	}
}

func TestSigstoreVerifier_InvalidBundleCreation(t *testing.T) {
	mockProvider := &MockTUFRootCertificateProvider{}
	// No mock needed - will fail at issuer extraction stage

	verifier := &sigstoreVerifier{
		rootCertificateProvider: mockProvider,
	}

	// Create a minimal invalid protobuf bundle that will fail at issuer extraction
	result := &model.EvidenceVerification{
		SigstoreBundle: &bundle.Bundle{
			Bundle: &v1.Bundle{
				MediaType: "invalid-media-type",
				// No VerificationMaterial - will fail at issuer extraction
			},
		},
		VerificationResult: model.EvidenceVerificationResult{},
	}

	err := verifier.verify(result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to extract issuer from bundle")

	mockProvider.AssertExpectations(t)
}

func TestSigstoreVerifier_PrepareVerificationData(t *testing.T) {
	tests := []struct {
		name                    string
		issuer                  string
		setupMock               func(*MockTUFRootCertificateProvider)
		expectError             bool
		errorMsg                string
		expectedVerifierOptions int
	}{
		{
			name:   "GitHub issuer loads GitHub certificate with signed timestamps",
			issuer: "GitHub, Inc.",
			setupMock: func(mockProvider *MockTUFRootCertificateProvider) {
				mockProvider.On("LoadTUFRootGithubCertificate").Return(nil, nil)
			},
			expectError:             false,
			expectedVerifierOptions: 1, // WithSignedTimestamps only
		},
		{
			name:   "public-good issuer loads public Sigstore with full options",
			issuer: "public-good",
			setupMock: func(mockProvider *MockTUFRootCertificateProvider) {
				mockProvider.On("LoadTUFRootCertificate").Return(nil, nil)
			},
			expectError:             false,
			expectedVerifierOptions: 3, // WithSignedCertificateTimestamps, WithObserverTimestamps, WithTransparencyLog
		},
		{
			name:        "Unsupported issuer returns error",
			issuer:      "Other Org",
			setupMock:   func(mockProvider *MockTUFRootCertificateProvider) {},
			expectError: true,
			errorMsg:    "unsupported issuer: Other Org",
		},
		{
			name:        "Empty issuer returns error",
			issuer:      "",
			setupMock:   func(mockProvider *MockTUFRootCertificateProvider) {},
			expectError: true,
			errorMsg:    "unsupported issuer:",
		},
		{
			name:   "GitHub certificate loading failure",
			issuer: "GitHub, Inc.",
			setupMock: func(mockProvider *MockTUFRootCertificateProvider) {
				mockProvider.On("LoadTUFRootGithubCertificate").Return(nil, errors.New("TUF error"))
			},
			expectError: true,
			errorMsg:    "failed to load GitHub TUF root trustedMaterial",
		},
		{
			name:   "Public Sigstore certificate loading failure",
			issuer: "public-good",
			setupMock: func(mockProvider *MockTUFRootCertificateProvider) {
				mockProvider.On("LoadTUFRootCertificate").Return(nil, errors.New("TUF error"))
			},
			expectError: true,
			errorMsg:    "failed to load TUF root trustedMaterial",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProvider := &MockTUFRootCertificateProvider{}
			tt.setupMock(mockProvider)

			verifier := &sigstoreVerifier{
				rootCertificateProvider: mockProvider,
			}

			trustedMaterial, verifierOptions, err := verifier.prepareVerificationData(tt.issuer)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, trustedMaterial)
				assert.Nil(t, verifierOptions)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedVerifierOptions, len(verifierOptions))
			}

			mockProvider.AssertExpectations(t)
		})
	}
}

func TestSigstoreVerifier_GitHubCertificateLoading(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*MockTUFRootCertificateProvider)
		issuer      string
		expectError bool
		errorMsg    string
	}{
		{
			name: "GitHub issuer triggers GitHub certificate loading - success",
			setupMock: func(mockProvider *MockTUFRootCertificateProvider) {
				// Should call LoadTUFRootGithubCertificate for GitHub issuer
				mockProvider.On("LoadTUFRootGithubCertificate").Return(nil, nil)
			},
			issuer:      "GitHub, Inc.",
			expectError: false,
		},
		{
			name: "GitHub issuer triggers GitHub certificate loading - failure",
			setupMock: func(mockProvider *MockTUFRootCertificateProvider) {
				// Should call LoadTUFRootGithubCertificate and return error
				mockProvider.On("LoadTUFRootGithubCertificate").Return(nil, errors.New("failed to load GitHub TUF"))
			},
			issuer:      "GitHub, Inc.",
			expectError: true,
			errorMsg:    "failed to load GitHub TUF root trustedMaterial",
		},
		{
			name: "public-good issuer uses public Sigstore certificate - success",
			setupMock: func(mockProvider *MockTUFRootCertificateProvider) {
				// Should call LoadTUFRootCertificate for public-good issuer
				mockProvider.On("LoadTUFRootCertificate").Return(nil, nil)
			},
			issuer:      "public-good",
			expectError: false,
		},
		{
			name: "public-good issuer uses public Sigstore certificate - failure",
			setupMock: func(mockProvider *MockTUFRootCertificateProvider) {
				// Should call LoadTUFRootCertificate and return error
				mockProvider.On("LoadTUFRootCertificate").Return(nil, errors.New("failed to load public TUF"))
			},
			issuer:      "public-good",
			expectError: true,
			errorMsg:    "failed to load TUF root trustedMaterial",
		},
		{
			name: "Unsupported issuer returns error",
			setupMock: func(mockProvider *MockTUFRootCertificateProvider) {
				// No certificate loading should be attempted for unsupported issuer
			},
			issuer:      "Other Org",
			expectError: true,
			errorMsg:    "unsupported issuer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProvider := &MockTUFRootCertificateProvider{}
			tt.setupMock(mockProvider)

			verifier := &sigstoreVerifier{
				rootCertificateProvider: mockProvider,
			}

			// Create a bundle with the specified issuer in the certificate
			protoBundle := createBundleWithIssuer(t, tt.issuer)

			result := &model.EvidenceVerification{
				SigstoreBundle: &bundle.Bundle{
					Bundle: protoBundle,
				},
				VerificationResult: model.EvidenceVerificationResult{},
			}

			err := verifier.verify(result)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			}

			// Verify that the correct certificate loading method was called
			mockProvider.AssertExpectations(t)
		})
	}
}

// createBundleWithIssuer creates a test bundle with a certificate containing the specified issuer organization
func createBundleWithIssuer(t *testing.T, issuer string) *v1.Bundle {
	// Generate CA key pair
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate CA RSA key: %v", err)
	}

	// Create CA certificate
	caIssuerName := pkix.Name{
		CommonName: "test-ca",
	}
	if issuer != "" {
		caIssuerName.Organization = []string{issuer}
	}

	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               caIssuerName,
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		IsCA:                  true,
		BasicConstraintsValid: true,
		MaxPathLen:            1,
	}

	// Self-sign CA certificate
	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("failed to create CA certificate: %v", err)
	}

	caCert, err := x509.ParseCertificate(caCertDER)
	if err != nil {
		t.Fatalf("failed to parse CA certificate: %v", err)
	}

	// Generate end-entity key pair
	leafKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate leaf RSA key: %v", err)
	}

	// Create end-entity certificate signed by CA
	leafTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			CommonName: "test-subject",
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
		BasicConstraintsValid: true,
	}

	// Sign the leaf certificate with the CA
	certDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, caCert, &leafKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("failed to create leaf certificate: %v", err)
	}

	// Create a protobuf bundle with the certificate
	return &v1.Bundle{
		MediaType: "application/vnd.dev.sigstore.bundle+json;version=0.2",
		VerificationMaterial: &v1.VerificationMaterial{
			Content: &v1.VerificationMaterial_Certificate{
				Certificate: &common.X509Certificate{
					RawBytes: certDER,
				},
			},
		},
		// Add minimal content to make it a valid bundle structure
		Content: &v1.Bundle_DsseEnvelope{
			DsseEnvelope: &dssepb.Envelope{
				Payload:     []byte("test-payload"),
				PayloadType: "application/vnd.in-toto+json",
				Signatures: []*dssepb.Signature{
					{
						Sig: []byte("test-signature"),
					},
				},
			},
		},
	}
}
