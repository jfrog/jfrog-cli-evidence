package cryptox

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadKey(t *testing.T) {
	files, err := os.ReadDir("testdata")
	assert.NoError(t, err)
	assert.Equal(t, 20, len(files))
	var keyFiles []os.DirEntry
	keysToValidate := []string{"ecdsa-test-key-pem", "ed25519-test-key-pem", "rsa-test-key"}
	for _, file := range files {
		for _, key := range keysToValidate {
			if file.Name() == key {
				keyFiles = append(keyFiles, file)
			}
		}

	}
	assert.Equal(t, 3, len(keyFiles))

	for _, file := range keyFiles {
		keyFile, err := os.ReadFile(filepath.Join("testdata", file.Name()))
		assert.Nil(t, err)
		keys, err := ReadKey(keyFile)
		assert.Nil(t, err)
		assert.NotNil(t, keys)
	}
}

// TestReadKeyEncrypted tests reading encrypted private keys
func TestReadKeyEncrypted(t *testing.T) {
	testPassword := "testpassword123"

	// Generate encrypted key using the exported function
	mockPasswordFunc := func(confirm bool) ([]byte, error) {
		return []byte(testPassword), nil
	}

	// Generate encrypted key pair
	encryptedPrivateKeyPEM, _, err := GenerateECDSAKeyPairWithPassword(mockPasswordFunc)
	require.NoError(t, err)

	encryptedPEM := []byte(encryptedPrivateKeyPEM)

	// Test reading with environment variable password
	os.Setenv("JFROG_EVIDENCE_PASSWORD", testPassword)
	defer os.Unsetenv("JFROG_EVIDENCE_PASSWORD")

	key, err := ReadKey(encryptedPEM)
	assert.NoError(t, err)
	assert.NotNil(t, key)
	assert.Equal(t, ECDSAKeyType, key.KeyType)
	assert.Equal(t, ECDSAKeyScheme, key.Scheme)
	assert.NotEmpty(t, key.KeyVal.Private)
	assert.NotEmpty(t, key.KeyVal.Public)

	// Test reading with wrong password
	os.Setenv("JFROG_EVIDENCE_PASSWORD", "wrongpassword")
	_, err = ReadKey(encryptedPEM)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid password")
}

// TestReadKeyEncryptedCosignFormat tests reading Cosign-format encrypted keys
func TestReadKeyEncryptedCosignFormat(t *testing.T) {
	testPassword := "testpassword123"

	// Generate encrypted key using the exported function
	mockPasswordFunc := func(confirm bool) ([]byte, error) {
		return []byte(testPassword), nil
	}

	// Generate encrypted key pair
	encryptedPrivateKeyPEM, _, err := GenerateECDSAKeyPairWithPassword(mockPasswordFunc)
	require.NoError(t, err)

	// Convert to Cosign format by replacing the PEM type
	encryptedPEM := []byte(strings.Replace(encryptedPrivateKeyPEM, "ENCRYPTED PRIVATE KEY", "ENCRYPTED COSIGN PRIVATE KEY", -1))

	// Test reading with environment variable password
	os.Setenv("JFROG_EVIDENCE_PASSWORD", testPassword)
	defer os.Unsetenv("JFROG_EVIDENCE_PASSWORD")

	key, err := ReadKey(encryptedPEM)
	assert.NoError(t, err)
	assert.NotNil(t, key)
	assert.Equal(t, ECDSAKeyType, key.KeyType)
	assert.Equal(t, ECDSAKeyScheme, key.Scheme)
}

// TestReadKeyEncryptedSigstoreFormat tests reading Sigstore-format encrypted keys
func TestReadKeyEncryptedSigstoreFormat(t *testing.T) {
	testPassword := "testpassword123"

	// Generate encrypted key using the exported function
	mockPasswordFunc := func(confirm bool) ([]byte, error) {
		return []byte(testPassword), nil
	}

	// Generate encrypted key pair
	encryptedPrivateKeyPEM, _, err := GenerateECDSAKeyPairWithPassword(mockPasswordFunc)
	require.NoError(t, err)

	// Convert to Sigstore format by replacing the PEM type
	encryptedPEM := []byte(strings.Replace(encryptedPrivateKeyPEM, "ENCRYPTED PRIVATE KEY", "ENCRYPTED SIGSTORE PRIVATE KEY", -1))

	// Test reading with environment variable password
	os.Setenv("JFROG_EVIDENCE_PASSWORD", testPassword)
	defer os.Unsetenv("JFROG_EVIDENCE_PASSWORD")

	key, err := ReadKey(encryptedPEM)
	assert.NoError(t, err)
	assert.NotNil(t, key)
	assert.Equal(t, ECDSAKeyType, key.KeyType)
	assert.Equal(t, ECDSAKeyScheme, key.Scheme)
}

// TestReadKeyErrorHandling tests error handling scenarios
func TestReadKeyErrorHandling(t *testing.T) {
	// Test with invalid PEM data
	invalidPEM := []byte("invalid pem data")
	_, err := ReadKey(invalidPEM)
	assert.Error(t, err)

	// Test with empty data
	emptyData := []byte("")
	_, err = ReadKey(emptyData)
	assert.Error(t, err)

	// Test with nil data
	_, err = ReadKey(nil)
	assert.Error(t, err)
}

// TestSignerToSSLibKey tests the signerToSSLibKey conversion function
func TestSignerToSSLibKey(t *testing.T) {
	// Generate a test ECDSA key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	// Create original PEM for testing
	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	require.NoError(t, err)
	originalPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	// Convert signer to SSLibKey
	ssLibKey, err := signerToSSLibKey(privateKey, originalPEM)
	assert.NoError(t, err)
	assert.NotNil(t, ssLibKey)
	assert.Equal(t, ECDSAKeyType, ssLibKey.KeyType)
	assert.Equal(t, ECDSAKeyScheme, ssLibKey.Scheme)
	assert.NotEmpty(t, ssLibKey.KeyVal.Public)
	assert.NotEmpty(t, ssLibKey.KeyVal.Private)
	assert.Equal(t, strings.TrimSpace(string(originalPEM)), ssLibKey.KeyVal.Private)
}
