package cryptox

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculateKeyID(t *testing.T) {
	key := &SSLibKey{
		KeyIDHashAlgorithms: nil,
		KeyType:             "rsa",
		KeyVal:              KeyVal{},
		Scheme:              "",
		KeyID:               "",
	}
	keyID, err := calculateKeyID(key)
	assert.NoError(t, err)
	// Check if the returned key ID matches the expected one
	// #nosec G101 - False positive - Not a real password
	expectedKeyID := "f97abd1db1e58debee59bf72ce05a31c77f58df54e3ff47eb532270e37f2f12b" // replace with the expected key ID
	if keyID != expectedKeyID {
		t.Errorf("Expected '%s', got '%s'", expectedKeyID, keyID)
	}
}

func TestGeneratePEMBlock(t *testing.T) {
	pem := generatePEMBlock([]byte("key"), "pemType")
	assert.Equal(t, "-----BEGIN pemType-----\na2V5\n-----END pemType-----\n", string(pem))
}

func TestDecodeParsePEM(t *testing.T) {
	pem, _, err := decodeAndParsePEM(rsaPrivateKey)
	assert.NoError(t, err)
	assert.Equal(t, "RSA PRIVATE KEY", pem.Type)
}

func TestParsePEMKey(t *testing.T) {
	pem, _, err := decodeAndParsePEM(rsaPrivateKey)
	assert.NoError(t, err)
	key, err := parsePEMKey(pem.Bytes)
	assert.NoError(t, err)
	assert.NotNil(t, key)
}

func TestHashBeforeSigning(t *testing.T) {
	// Call hashBeforeSigning with a known payload and a SHA256 hash function
	payload := "test payload"
	hash := hashBeforeSigning([]byte(payload), sha256.New())

	// Check if the returned hash matches the expected one
	// #nosec G101 - False positive - Not a real password
	expectedHash := "813ca5285c28ccee5cab8b10ebda9c908fd6d78ed9dc94cc65ea6cb67a7f13ae" // SHA256 hash of "test payload"
	if hex.EncodeToString(hash) != expectedHash {
		t.Errorf("Expected '%s', got '%s'", expectedHash, hex.EncodeToString(hash))
	}
}

// TestIsEncryptedPEMType tests the encrypted PEM type detection
func TestIsEncryptedPEMType(t *testing.T) {
	tests := []struct {
		name     string
		pemType  string
		expected bool
	}{
		{
			name:     "encrypted private key",
			pemType:  "ENCRYPTED PRIVATE KEY",
			expected: true,
		},
		{
			name:     "encrypted cosign private key",
			pemType:  "ENCRYPTED COSIGN PRIVATE KEY",
			expected: true,
		},
		{
			name:     "encrypted sigstore private key",
			pemType:  "ENCRYPTED SIGSTORE PRIVATE KEY",
			expected: true,
		},
		{
			name:     "private key with encrypted marker",
			pemType:  "PRIVATE KEY",
			expected: false, // This should be false for unencrypted keys
		},
		{
			name:     "public key",
			pemType:  "PUBLIC KEY",
			expected: false,
		},
		{
			name:     "empty type",
			pemType:  "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isEncryptedPEMType(tt.pemType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGenerateFingerprint tests fingerprint generation
func TestGenerateFingerprint(t *testing.T) {
	// Generate a test ECDSA key pair
	_, publicKeyPEM, err := GenerateECDSAKeyPair()
	assert.NoError(t, err)

	// Load the public key
	publicKey, err := LoadKey([]byte(publicKeyPEM))
	assert.NoError(t, err)

	// Create a signer to get the public key
	signer, err := NewECDSASignerVerifierFromSSLibKey(publicKey)
	assert.NoError(t, err)

	// Generate fingerprint
	fingerprint, err := GenerateFingerprint(signer.Public())
	assert.NoError(t, err)
	assert.NotEmpty(t, fingerprint)
	assert.Len(t, fingerprint, 44) // Base64 encoded SHA256 (32 bytes = 44 chars)
}

// TestGenerateFingerprint_NilKey tests fingerprint generation with nil key
func TestGenerateFingerprint_NilKey(t *testing.T) {
	fingerprint, err := GenerateFingerprint(nil)
	assert.Error(t, err)
	assert.Empty(t, fingerprint)
	assert.Contains(t, err.Error(), "public key not available")
}

// TestCreateVerifier tests verifier creation for different key types
func TestCreateVerifier(t *testing.T) {
	tests := []struct {
		name     string
		keyType  string
		wantErr  bool
	}{
		{
			name:    "unsupported key type",
			keyType: "UNSUPPORTED",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock key
			key := &SSLibKey{
				KeyType: tt.keyType,
				Scheme:  "test-scheme",
				KeyVal: KeyVal{
					Public: "test-public-key",
				},
			}

			verifiers, err := CreateVerifier(key)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, verifiers)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, verifiers)
				assert.Len(t, verifiers, 1)
			}
		})
	}
}

// TestErrorConstants tests that error constants are properly defined
func TestErrorConstants(t *testing.T) {
	assert.NotNil(t, ErrNoPEMBlock)
	assert.NotNil(t, ErrFailedPEMParsing)
	assert.NotNil(t, ErrEncryptedKeyNotSupported)
	
	assert.Contains(t, ErrNoPEMBlock.Error(), "PEM block")
	assert.Contains(t, ErrFailedPEMParsing.Error(), "PEM")
	assert.Contains(t, ErrEncryptedKeyNotSupported.Error(), "encrypted private keys are not supported")
}

// TestKeyConstants tests key type constants
func TestKeyConstants(t *testing.T) {
	assert.Equal(t, "ecdsa", ECDSAKeyType)
	assert.Equal(t, "ecdsa-sha2-nistp256", ECDSAKeyScheme)
	assert.Equal(t, "rsa", RSAKeyType)
	assert.Equal(t, "rsassa-pss-sha256", RSAKeyScheme)
	assert.Equal(t, "ed25519", ED25519KeyType)
	assert.Equal(t, "ed25519", ED25519KeyScheme)
	assert.Equal(t, "PUBLIC KEY", PublicKeyPEM)
}
