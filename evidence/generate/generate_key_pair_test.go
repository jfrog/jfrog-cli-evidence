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
		os.Remove("test-key.key")
		os.Remove("test-key.pub")
	}()

	cmd := NewGenerateKeyPairCommand(nil, false, "test-alias", true, "", "test-key") // uploadPublicKey=false, force=true, keyFileName="test-key"
	assert.NotNil(t, cmd)
	assert.Equal(t, "generate-key-pair", cmd.CommandName())

	// Test Run without upload
	err := cmd.Run()
	assert.NoError(t, err)

	// Verify files were created
	_, err = os.Stat("test-key.key")
	assert.NoError(t, err)
	_, err = os.Stat("test-key.pub")
	assert.NoError(t, err)

	// Verify file permissions
	info, _ := os.Stat("test-key.key")
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())

	info, _ = os.Stat("test-key.pub")
	assert.Equal(t, os.FileMode(0644), info.Mode().Perm())

	// Load and verify the generated keys are ECDSA
	publicKeyData, err := os.ReadFile("test-key.pub")
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

func TestEncryptedKeyRejection(t *testing.T) {
	// Test that encrypted keys are properly rejected
	encryptedKeyPEM := `-----BEGIN ENCRYPTED PRIVATE KEY-----
MIIFHDBOBgkqhkiG9w0BBQ0wQTApBgkqhkiG9w0BBQwwHAQIAgICAgICAgICAgICAgID
AgAMAwUGCCqGSM49BAMCAA==
-----END ENCRYPTED PRIVATE KEY-----`

	_, err := cryptox.LoadKey([]byte(encryptedKeyPEM))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "encrypted private keys are not supported")
}

func TestGenerateKeyPairCommandWithOutputDir(t *testing.T) {
	// Clean up any existing files
	defer func() {
		os.RemoveAll("test-output")
	}()

	cmd := NewGenerateKeyPairCommand(nil, false, "test-alias", true, "test-output", "custom-key") // uploadPublicKey=false, force=true, outputDir="test-output", keyFileName="custom-key"
	assert.NotNil(t, cmd)

	// Test Run without upload
	err := cmd.Run()
	assert.NoError(t, err)

	// Verify files were created in the output directory
	_, err = os.Stat("test-output/custom-key.key")
	assert.NoError(t, err)
	_, err = os.Stat("test-output/custom-key.pub")
	assert.NoError(t, err)

	// Verify file permissions
	info, _ := os.Stat("test-output/custom-key.key")
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())

	info, _ = os.Stat("test-output/custom-key.pub")
	assert.Equal(t, os.FileMode(0644), info.Mode().Perm())
}
