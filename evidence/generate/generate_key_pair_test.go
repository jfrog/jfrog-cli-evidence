package generate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cryptox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// TestNewGenerateKeyPairCommand tests the constructor function
func TestNewGenerateKeyPairCommand(t *testing.T) {
	serverDetails := &config.ServerDetails{
		Url: "https://test.jfrog.io",
	}

	cmd := NewGenerateKeyPairCommand(serverDetails, true, "test-alias", false, "/tmp", "my-key")
	
	assert.NotNil(t, cmd)
	assert.Equal(t, serverDetails, cmd.serverDetails)
	assert.True(t, cmd.uploadPublicKey)
	assert.Equal(t, "test-alias", cmd.keyAlias)
	assert.False(t, cmd.forceOverwrite)
	assert.Equal(t, "/tmp", cmd.outputDir)
	assert.Equal(t, "my-key", cmd.keyFileName)
}

// TestKeyPairCommand_CommandName tests the CommandName method
func TestKeyPairCommand_CommandName(t *testing.T) {
	cmd := NewGenerateKeyPairCommand(nil, false, "", false, "", "")
	assert.Equal(t, "generate-key-pair", cmd.CommandName())
}

// TestKeyPairCommand_generateOrGetAlias tests alias generation logic
func TestKeyPairCommand_generateOrGetAlias(t *testing.T) {
	tests := []struct {
		name     string
		keyAlias string
		want     string
	}{
		{
			name:     "custom alias provided",
			keyAlias: "my-custom-alias",
			want:     "my-custom-alias",
		},
		{
			name:     "empty alias generates timestamp-based",
			keyAlias: "",
			want:     "", // Will be generated dynamically
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewGenerateKeyPairCommand(nil, false, tt.keyAlias, false, "", "")
			result := cmd.generateOrGetAlias()
			
			if tt.keyAlias != "" {
				assert.Equal(t, tt.want, result)
			} else {
				// Should generate timestamp-based alias
				assert.Contains(t, result, "evd-key-")
				assert.Len(t, result, 23) // "evd-key-" + "YYYYMMDD-HHMMSS" = 8 + 15 = 23
			}
		})
	}
}

// TestKeyPairCommand_prepareOutputDirectory tests directory preparation
func TestKeyPairCommand_prepareOutputDirectory(t *testing.T) {
	tests := []struct {
		name      string
		outputDir string
		want      string
	}{
		{
			name:      "empty output dir defaults to current",
			outputDir: "",
			want:      ".",
		},
		{
			name:      "current directory",
			outputDir: ".",
			want:      ".",
		},
		{
			name:      "custom directory",
			outputDir: "test-dir",
			want:      "test-dir",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewGenerateKeyPairCommand(nil, false, "", false, tt.outputDir, "")
			
			// Clean up after test
			defer func() {
				if tt.outputDir != "" && tt.outputDir != "." {
					os.RemoveAll(tt.outputDir)
				}
			}()

			result, err := cmd.prepareOutputDirectory()
			assert.NoError(t, err)
			assert.Equal(t, tt.want, result)

			// Verify directory was created if needed
			if tt.outputDir != "" && tt.outputDir != "." {
				_, err := os.Stat(tt.outputDir)
				assert.NoError(t, err)
			}
		})
	}
}

// TestKeyPairCommand_buildKeyFilePaths tests file path construction
func TestKeyPairCommand_buildKeyFilePaths(t *testing.T) {
	tests := []struct {
		name       string
		outputDir  string
		keyFileName string
		wantPriv   string
		wantPub    string
	}{
		{
			name:       "default file name",
			outputDir:  ".",
			keyFileName: "",
			wantPriv:   "evidence.key",
			wantPub:    "evidence.pub",
		},
		{
			name:       "custom file name",
			outputDir:  ".",
			keyFileName: "my-key",
			wantPriv:   "my-key.key",
			wantPub:    "my-key.pub",
		},
		{
			name:       "custom directory and file name",
			outputDir:  "test-dir",
			keyFileName: "custom-key",
			wantPriv:   filepath.Join("test-dir", "custom-key.key"),
			wantPub:    filepath.Join("test-dir", "custom-key.pub"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewGenerateKeyPairCommand(nil, false, "", false, tt.outputDir, tt.keyFileName)
			
			privPath, pubPath, err := cmd.buildKeyFilePaths(tt.outputDir)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantPriv, privPath)
			assert.Equal(t, tt.wantPub, pubPath)
		})
	}
}

// TestKeyPairCommand_validateExistingFiles tests file validation logic
func TestKeyPairCommand_validateExistingFiles(t *testing.T) {
	// Create test files
	testPrivPath := "test-validate.key"
	testPubPath := "test-validate.pub"
	
	// Clean up after test
	defer func() {
		os.Remove(testPrivPath)
		os.Remove(testPubPath)
	}()

	// Create existing files
	err := os.WriteFile(testPrivPath, []byte("test private key"), 0600)
	require.NoError(t, err)
	err = os.WriteFile(testPubPath, []byte("test public key"), 0644)
	require.NoError(t, err)

	tests := []struct {
		name          string
		forceOverwrite bool
		wantError     bool
	}{
		{
			name:          "files exist without force",
			forceOverwrite: false,
			wantError:     true,
		},
		{
			name:          "files exist with force",
			forceOverwrite: true,
			wantError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewGenerateKeyPairCommand(nil, false, "", tt.forceOverwrite, "", "")
			
			err := cmd.validateExistingFiles(testPrivPath, testPubPath)
			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "already exists")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestKeyPairCommand_generateKeyPair tests key generation
func TestKeyPairCommand_generateKeyPair(t *testing.T) {
	cmd := NewGenerateKeyPairCommand(nil, false, "", false, "", "")
	
	privPEM, pubPEM, err := cmd.generateKeyPair()
	assert.NoError(t, err)
	assert.NotEmpty(t, privPEM)
	assert.NotEmpty(t, pubPEM)

	// Verify PEM format
	assert.Contains(t, privPEM, "-----BEGIN PRIVATE KEY-----")
	assert.Contains(t, privPEM, "-----END PRIVATE KEY-----")
	assert.Contains(t, pubPEM, "-----BEGIN PUBLIC KEY-----")
	assert.Contains(t, pubPEM, "-----END PUBLIC KEY-----")

	// Verify keys can be loaded
	privKey, err := cryptox.LoadKey([]byte(privPEM))
	assert.NoError(t, err)
	assert.Equal(t, cryptox.ECDSAKeyType, privKey.KeyType)

	pubKey, err := cryptox.LoadKey([]byte(pubPEM))
	assert.NoError(t, err)
	assert.Equal(t, cryptox.ECDSAKeyType, pubKey.KeyType)
}

// TestKeyPairCommand_writeKeyFiles tests file writing with permissions
func TestKeyPairCommand_writeKeyFiles(t *testing.T) {
	testPrivPath := "test-write.key"
	testPubPath := "test-write.pub"
	
	// Clean up after test
	defer func() {
		os.Remove(testPrivPath)
		os.Remove(testPubPath)
	}()

	cmd := NewGenerateKeyPairCommand(nil, false, "", false, "", "")
	
	// Generate test keys
	privPEM, pubPEM, err := cmd.generateKeyPair()
	require.NoError(t, err)

	// Write files
	err = cmd.writeKeyFiles(privPEM, pubPEM, testPrivPath, testPubPath)
	assert.NoError(t, err)

	// Verify files exist
	_, err = os.Stat(testPrivPath)
	assert.NoError(t, err)
	_, err = os.Stat(testPubPath)
	assert.NoError(t, err)

	// Verify permissions
	privInfo, err := os.Stat(testPrivPath)
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(PrivateKeyPermissions), privInfo.Mode().Perm())

	pubInfo, err := os.Stat(testPubPath)
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(PublicKeyPermissions), pubInfo.Mode().Perm())

	// Verify content
	privContent, err := os.ReadFile(testPrivPath)
	assert.NoError(t, err)
	assert.Equal(t, privPEM, string(privContent))

	pubContent, err := os.ReadFile(testPubPath)
	assert.NoError(t, err)
	assert.Equal(t, pubPEM, string(pubContent))
}

// TestConstants tests that constants are properly defined
func TestConstants(t *testing.T) {
	assert.Equal(t, 1, DefaultRetries)
	assert.Equal(t, 0, DefaultTimeout)
	assert.Equal(t, 0, DefaultThreads)
	assert.False(t, DefaultDryRun)
	assert.Equal(t, os.FileMode(0600), os.FileMode(PrivateKeyPermissions))
	assert.Equal(t, os.FileMode(0644), os.FileMode(PublicKeyPermissions))
	assert.Equal(t, os.FileMode(0755), os.FileMode(DirectoryPermissions))
}

// TestGenerateKeyPairCommandWithAllFlags tests the complete workflow with all flags
func TestGenerateKeyPairCommandWithAllFlags(t *testing.T) {
	// Clean up after test
	defer func() {
		os.RemoveAll("test-complete")
	}()

	cmd := NewGenerateKeyPairCommand(
		nil,                    // serverDetails
		false,                  // uploadPublicKey
		"test-complete-alias",  // keyAlias
		true,                   // forceOverwrite
		"test-complete",        // outputDir
		"complete-test",        // keyFileName
	)

	assert.NotNil(t, cmd)
	assert.Equal(t, "test-complete-alias", cmd.keyAlias)
	assert.Equal(t, "test-complete", cmd.outputDir)
	assert.Equal(t, "complete-test", cmd.keyFileName)
	assert.True(t, cmd.forceOverwrite)
	assert.False(t, cmd.uploadPublicKey)

	// Test Run
	err := cmd.Run()
	assert.NoError(t, err)

	// Verify files were created
	_, err = os.Stat("test-complete/complete-test.key")
	assert.NoError(t, err)
	_, err = os.Stat("test-complete/complete-test.pub")
	assert.NoError(t, err)

	// Verify file permissions
	privInfo, err := os.Stat("test-complete/complete-test.key")
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(PrivateKeyPermissions), privInfo.Mode().Perm())

	pubInfo, err := os.Stat("test-complete/complete-test.pub")
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(PublicKeyPermissions), pubInfo.Mode().Perm())
}

// TestGenerateKeyPairCommandFileOverwrite tests file overwrite scenarios
func TestGenerateKeyPairCommandFileOverwrite(t *testing.T) {
	testPrivPath := "test-overwrite.key"
	testPubPath := "test-overwrite.pub"
	
	// Clean up after test
	defer func() {
		os.Remove(testPrivPath)
		os.Remove(testPubPath)
	}()

	// Create existing files
	err := os.WriteFile(testPrivPath, []byte("old private key"), 0600)
	require.NoError(t, err)
	err = os.WriteFile(testPubPath, []byte("old public key"), 0644)
	require.NoError(t, err)

	// Test without force (should fail)
	cmd1 := NewGenerateKeyPairCommand(nil, false, "", false, "", "test-overwrite")
	err = cmd1.Run()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	// Test with force (should succeed)
	cmd2 := NewGenerateKeyPairCommand(nil, false, "", true, "", "test-overwrite")
	err = cmd2.Run()
	assert.NoError(t, err)

	// Verify files were overwritten
	privContent, err := os.ReadFile(testPrivPath)
	assert.NoError(t, err)
	assert.NotEqual(t, "old private key", string(privContent))
	assert.Contains(t, string(privContent), "-----BEGIN PRIVATE KEY-----")

	pubContent, err := os.ReadFile(testPubPath)
	assert.NoError(t, err)
	assert.NotEqual(t, "old public key", string(pubContent))
	assert.Contains(t, string(pubContent), "-----BEGIN PUBLIC KEY-----")
}
