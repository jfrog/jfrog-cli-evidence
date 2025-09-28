package generate

import (
	"os"
	"testing"

	"github.com/jfrog/jfrog-cli-evidence/evidence/cryptox"
	"github.com/stretchr/testify/assert"
)

func TestGenerateECDSAKeyPair(t *testing.T) {
	// Test key generation
	privateKeyPEM, publicKeyPEM, err := cryptox.GenerateECDSAKeyPair()
	assert.NoError(t, err)
	assert.NotEmpty(t, privateKeyPEM)
	assert.NotEmpty(t, publicKeyPEM)

	// Verify the public key can be loaded
	publicKey, err := cryptox.LoadKey([]byte(publicKeyPEM))
	assert.NoError(t, err)
	assert.Equal(t, cryptox.ECDSAKeyType, publicKey.KeyType)
	assert.Equal(t, cryptox.ECDSAKeyScheme, publicKey.Scheme)
	assert.Empty(t, publicKey.KeyVal.Private) // Should not contain private key
	assert.NotEmpty(t, publicKey.KeyVal.Public)

	// Verify the private key has the expected PEM structure (unencrypted)
	assert.Contains(t, privateKeyPEM, "-----BEGIN PRIVATE KEY-----")
	assert.Contains(t, privateKeyPEM, "-----END PRIVATE KEY-----")
	assert.NotContains(t, privateKeyPEM, "Proc-Type: 4,ENCRYPTED") // Should NOT be encrypted

	// Verify the private key can be loaded
	privateKey, err := cryptox.LoadKey([]byte(privateKeyPEM))
	assert.NoError(t, err)
	assert.Equal(t, cryptox.ECDSAKeyType, privateKey.KeyType)
	assert.Equal(t, cryptox.ECDSAKeyScheme, privateKey.Scheme)
	assert.NotEmpty(t, privateKey.KeyVal.Private)
	assert.NotEmpty(t, privateKey.KeyVal.Public)
}

func TestGenerateKeyPairCommand(t *testing.T) {
	// Clean up any existing files
	defer func() {
		os.Remove("evd.key")
		os.Remove("evd.pub")
	}()

	cmd := NewGenerateKeyPairCommand(nil, false, "test-alias", false, "", false)
	assert.NotNil(t, cmd)
	assert.Equal(t, "generate-key-pair", cmd.CommandName())

	// Test Run without upload
	err := cmd.Run()
	assert.NoError(t, err)

	// Verify files were created
	_, err = os.Stat("evd.key")
	assert.NoError(t, err)
	_, err = os.Stat("evd.pub")
	assert.NoError(t, err)

	// Verify file permissions
	info, _ := os.Stat("evd.key")
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
	
	info, _ = os.Stat("evd.pub")
	assert.Equal(t, os.FileMode(0644), info.Mode().Perm())

	// Load and verify the generated keys are ECDSA
	publicKeyData, err := os.ReadFile("evd.pub")
	assert.NoError(t, err)
	publicKey, err := cryptox.LoadKey(publicKeyData)
	assert.NoError(t, err)
	assert.Equal(t, cryptox.ECDSAKeyType, publicKey.KeyType)
	assert.Equal(t, cryptox.ECDSAKeyScheme, publicKey.Scheme)
}

func TestGenerateKeyPairCommandDuplicateValidation(t *testing.T) {
	// This test requires a live Artifactory instance
	t.Skip("Integration test - requires live Artifactory instance")
}

func TestGenerateECDSAKeyPairWithPassword(t *testing.T) {
	// Mock password function that returns a test password
	mockPasswordFunc := func(confirm bool) ([]byte, error) {
		return []byte("testpassword"), nil
	}

	// Test encrypted key generation
	privateKeyPEM, publicKeyPEM, err := cryptox.GenerateECDSAKeyPairWithPassword(mockPasswordFunc)
	assert.NoError(t, err)
	assert.NotEmpty(t, privateKeyPEM)
	assert.NotEmpty(t, publicKeyPEM)

	// Verify the private key has the expected PEM structure (encrypted)
	assert.Contains(t, privateKeyPEM, "-----BEGIN ENCRYPTED PRIVATE KEY-----")
	assert.Contains(t, privateKeyPEM, "-----END ENCRYPTED PRIVATE KEY-----")

	// Verify the public key can be loaded
	publicKey, err := cryptox.LoadKey([]byte(publicKeyPEM))
	assert.NoError(t, err)
	assert.Equal(t, cryptox.ECDSAKeyType, publicKey.KeyType)
	assert.Equal(t, cryptox.ECDSAKeyScheme, publicKey.Scheme)
	assert.Empty(t, publicKey.KeyVal.Private) // Should not contain private key
	assert.NotEmpty(t, publicKey.KeyVal.Public)
}