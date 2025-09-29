package cryptox

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// Test data for encryption/decryption tests
func generateTestECDSAKey() ([]byte, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, err
	}

	return privateKeyBytes, nil
}

// TestEncryptDecryptPrivateKey tests the encryption and decryption of private keys
func TestEncryptDecryptPrivateKey(t *testing.T) {
	testPassword := "testpassword123"

	// Generate encrypted key using the exported function
	mockPasswordFunc := func(confirm bool) ([]byte, error) {
		return []byte(testPassword), nil
	}

	// Generate encrypted key pair
	encryptedPrivateKeyPEM, _, err := GenerateECDSAKeyPairWithPassword(mockPasswordFunc)
	assert.NoError(t, err)
	assert.NotEmpty(t, encryptedPrivateKeyPEM)
	assert.Contains(t, encryptedPrivateKeyPEM, "-----BEGIN ENCRYPTED PRIVATE KEY-----")

	// Test that the encrypted key can be loaded with correct password
	signer, err := LoadPrivateKey([]byte(encryptedPrivateKeyPEM), []byte(testPassword))
	assert.NoError(t, err)
	assert.NotNil(t, signer)

	// Test that loading with wrong password fails
	_, err = LoadPrivateKey([]byte(encryptedPrivateKeyPEM), []byte("wrongpassword"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid password")
}

// TestLoadPrivateKey tests loading encrypted private keys
func TestLoadPrivateKey(t *testing.T) {
	testPassword := "testpassword123"

	// Generate encrypted key using the exported function
	mockPasswordFunc := func(confirm bool) ([]byte, error) {
		return []byte(testPassword), nil
	}

	// Generate encrypted key pair
	encryptedPrivateKeyPEM, _, err := GenerateECDSAKeyPairWithPassword(mockPasswordFunc)
	require.NoError(t, err)

	encryptedPEM := []byte(encryptedPrivateKeyPEM)

	// Test loading with correct password
	signer, err := LoadPrivateKey(encryptedPEM, []byte(testPassword))
	assert.NoError(t, err)
	assert.NotNil(t, signer)

	// Verify it's an ECDSA key
	loadedKey, ok := signer.(*ecdsa.PrivateKey)
	assert.True(t, ok)
	assert.NotNil(t, loadedKey)

	// Test loading with wrong password
	wrongPassword := []byte("wrongpassword")
	_, err = LoadPrivateKey(encryptedPEM, wrongPassword)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid password")

	// Test loading with invalid PEM
	invalidPEM := []byte("invalid pem data")
	_, err = LoadPrivateKey(invalidPEM, []byte(testPassword))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid PEM block")

	// Test loading with unsupported PEM type
	unsupportedPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "UNSUPPORTED KEY TYPE",
		Bytes: []byte("dummy data"),
	})
	_, err = LoadPrivateKey(unsupportedPEM, []byte(testPassword))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported PEM type")
}

// TestGetPassword tests the GetPassword function with environment variable
func TestGetPassword(t *testing.T) {
	// Test with environment variable set
	testPassword := "test-env-password"
	os.Setenv("JFROG_EVIDENCE_PASSWORD", testPassword)
	defer os.Unsetenv("JFROG_EVIDENCE_PASSWORD")

	password, err := GetPassword(false)
	assert.NoError(t, err)
	assert.Equal(t, []byte(testPassword), password)

	// Test with environment variable unset (will fail in non-terminal environment)
	os.Unsetenv("JFROG_EVIDENCE_PASSWORD")
	_, err = GetPassword(false)
	assert.Error(t, err) // Should fail in test environment (not a terminal)
	assert.Contains(t, err.Error(), "password required but not in terminal")
}

// TestIsTerminal tests the IsTerminal function
func TestIsTerminal(t *testing.T) {
	// In test environment, this should return false
	isTerminal := IsTerminal()
	assert.False(t, isTerminal) // Test environment is not a terminal
}

// TestDecodeAndParsePEMEncrypted tests parsing encrypted PEM blocks
func TestDecodeAndParsePEMEncrypted(t *testing.T) {
	// Create an encrypted PEM block
	encryptedPEM := `-----BEGIN ENCRYPTED PRIVATE KEY-----
TWV0YWRhdGE=
-----END ENCRYPTED PRIVATE KEY-----`

	_, _, err := decodeAndParsePEM([]byte(encryptedPEM))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "encrypted private key requires password")

	// Test with Cosign encrypted key type
	cosignEncryptedPEM := `-----BEGIN ENCRYPTED COSIGN PRIVATE KEY-----
TWV0YWRhdGE=
-----END ENCRYPTED COSIGN PRIVATE KEY-----`

	_, _, err = decodeAndParsePEM([]byte(cosignEncryptedPEM))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "encrypted private key requires password")

	// Test with Sigstore encrypted key type
	sigstoreEncryptedPEM := `-----BEGIN ENCRYPTED SIGSTORE PRIVATE KEY-----
TWV0YWRhdGE=
-----END ENCRYPTED SIGSTORE PRIVATE KEY-----`

	_, _, err = decodeAndParsePEM([]byte(sigstoreEncryptedPEM))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "encrypted private key requires password")
}

// TestPasswordFuncType tests the PasswordFunc type
func TestPasswordFuncType(t *testing.T) {
	// Test successful password function
	successFunc := func(confirm bool) ([]byte, error) {
		return []byte("testpassword"), nil
	}

	password, err := successFunc(false)
	assert.NoError(t, err)
	assert.Equal(t, []byte("testpassword"), password)

	// Test error password function
	errorFunc := func(confirm bool) ([]byte, error) {
		return nil, errors.New("password error")
	}

	_, err = errorFunc(false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "password error")
}

// TestEncryptionConstants tests that our encryption constants are correct
func TestEncryptionConstants(t *testing.T) {
	assert.Equal(t, "ENCRYPTED PRIVATE KEY", EncryptedPrivateKeyPemType)
	assert.Equal(t, "ENCRYPTED COSIGN PRIVATE KEY", EncryptedCosignPrivateKeyPemType)
	assert.Equal(t, "ENCRYPTED SIGSTORE PRIVATE KEY", EncryptedSigstorePrivateKeyPemType)
}

// TestDecryptPrivateKeyErrorHandling tests various error scenarios
func TestDecryptPrivateKeyErrorHandling(t *testing.T) {
	// Test with invalid encrypted PEM data
	invalidPEM := pem.EncodeToMemory(&pem.Block{
		Type:  EncryptedPrivateKeyPemType,
		Bytes: []byte("invalid encrypted data"),
	})
	password := []byte("testpassword")

	_, err := LoadPrivateKey(invalidPEM, password)
	assert.Error(t, err)
	// The error should be user-friendly for decryption failures
	assert.True(t, strings.Contains(err.Error(), "invalid password") || strings.Contains(err.Error(), "failed to decrypt"))
}

// TestGeneratePEMBlockEdgeCases tests edge cases for PEM block generation
func TestGeneratePEMBlockEdgeCases(t *testing.T) {
	// Test with empty key bytes
	emptyPEM := generatePEMBlock([]byte{}, "TEST TYPE")
	assert.Contains(t, string(emptyPEM), "-----BEGIN TEST TYPE-----")
	assert.Contains(t, string(emptyPEM), "-----END TEST TYPE-----")

	// Test with nil key bytes
	nilPEM := generatePEMBlock(nil, "TEST TYPE")
	assert.Contains(t, string(nilPEM), "-----BEGIN TEST TYPE-----")
	assert.Contains(t, string(nilPEM), "-----END TEST TYPE-----")
}
