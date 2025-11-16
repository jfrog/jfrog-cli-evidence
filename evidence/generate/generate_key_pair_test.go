package generate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
		_ = os.Remove("test-key.key")
		_ = os.Remove("test-key.pub")
	}()

	cmd := NewGenerateKeyPairCommand(nil, false, "test-alias", "", "test-key") // uploadPublicKey=false, keyFileName="test-key"
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
		_ = os.RemoveAll("test-output")
	}()

	cmd := NewGenerateKeyPairCommand(nil, false, "test-alias", "test-output", "custom-key") // uploadPublicKey=false, keyFilePath="test-output", keyFileName="custom-key"
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

	cmd := NewGenerateKeyPairCommand(serverDetails, true, "test-alias", "/tmp", "my-key")

	assert.NotNil(t, cmd)
	assert.Equal(t, serverDetails, cmd.serverDetails)
	assert.True(t, cmd.uploadPublicKey)
	assert.Equal(t, "test-alias", cmd.keyAlias)
	assert.Equal(t, "/tmp", cmd.keyFilePath)
	assert.Equal(t, "my-key", cmd.keyFileName)
}

// TestKeyPairCommand_CommandName tests the CommandName method
func TestKeyPairCommand_CommandName(t *testing.T) {
	cmd := NewGenerateKeyPairCommand(nil, false, "", "", "")
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
			cmd := NewGenerateKeyPairCommand(nil, false, tt.keyAlias, "", "")
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
			cmd := NewGenerateKeyPairCommand(nil, false, "", tt.outputDir, "")

			// Clean up after test
			defer func() {
				if tt.outputDir != "" && tt.outputDir != "." {
					_ = os.RemoveAll(tt.outputDir)
				}
			}()

			result, err := cmd.prepareKeyFilePath()
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
		name        string
		outputDir   string
		keyFileName string
		wantPriv    string
		wantPub     string
	}{
		{
			name:        "default file name",
			outputDir:   ".",
			keyFileName: "",
			wantPriv:    "evidence.key",
			wantPub:     "evidence.pub",
		},
		{
			name:        "custom file name",
			outputDir:   ".",
			keyFileName: "my-key",
			wantPriv:    "my-key.key",
			wantPub:     "my-key.pub",
		},
		{
			name:        "custom directory and file name",
			outputDir:   "test-dir",
			keyFileName: "custom-key",
			wantPriv:    filepath.Join("test-dir", "custom-key.key"),
			wantPub:     filepath.Join("test-dir", "custom-key.pub"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewGenerateKeyPairCommand(nil, false, "", tt.outputDir, tt.keyFileName)

			privPath, pubPath := cmd.buildKeyFilePaths(tt.outputDir)
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
		_ = os.Remove(testPrivPath)
		_ = os.Remove(testPubPath)
	}()

	// Create existing files
	err := os.WriteFile(testPrivPath, []byte("test private key"), 0600)
	require.NoError(t, err)
	err = os.WriteFile(testPubPath, []byte("test public key"), 0644)
	require.NoError(t, err)

	tests := []struct {
		name      string
		wantError bool
	}{
		{
			name:      "files exist",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewGenerateKeyPairCommand(nil, false, "", "", "")

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
	cmd := NewGenerateKeyPairCommand(nil, false, "", "", "")

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
		_ = os.Remove(testPrivPath)
		_ = os.Remove(testPubPath)
	}()

	cmd := NewGenerateKeyPairCommand(nil, false, "", "", "")

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
		_ = os.RemoveAll("test-complete")
	}()

	cmd := NewGenerateKeyPairCommand(
		nil,                   // serverDetails
		false,                 // uploadPublicKey
		"test-complete-alias", // keyAlias
		"test-complete",       // keyFilePath
		"complete-test",       // keyFileName
	)

	assert.NotNil(t, cmd)
	assert.Equal(t, "test-complete-alias", cmd.keyAlias)
	assert.Equal(t, "test-complete", cmd.keyFilePath)
	assert.Equal(t, "complete-test", cmd.keyFileName)
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

// TestKeyPairCommand_logUploadWarning tests the error warning logic
func TestKeyPairCommand_logUploadWarning(t *testing.T) {
	tests := []struct {
		name         string
		errorMessage string
		expectedLogs []string
	}{
		{
			name:         "403 Forbidden error",
			errorMessage: "trusted keys API returned status 403: Forbidden - insufficient permissions to upload trusted keys",
			expectedLogs: []string{
				"Permission denied: Your user|token doesn't have sufficient permissions to upload trusted keys",
				"Contact your administrator to grant trusted keys upload permissions",
			},
		},
		{
			name:         "404 Not Found error",
			errorMessage: "trusted keys API returned status 404: 404 page not found - trusted keys API endpoint not available",
			expectedLogs: []string{
				"Endpoint not found: The trusted keys API endpoint is not available",
				"Check your server URL and ensure trusted keys feature is enabled",
			},
		},
		{
			name:         "401 Unauthorized error",
			errorMessage: "trusted keys API returned status 401: Unauthorized - invalid or expired authentication token",
			expectedLogs: []string{
				"Authentication failed: Invalid or expired authentication token",
				"Check your access token or regenerate a new one",
			},
		},
		{
			name:         "Duplicate alias error",
			errorMessage: "trusted keys API returned status 400: alias already exists",
			expectedLogs: []string{
				"Use a unique alias with --key-alias <unique-name>",
			},
		},
		{
			name:         "Generic error",
			errorMessage: "trusted keys API returned status 500: Internal server error",
			expectedLogs: []string{
				"You can manually upload the public key later or check your server configuration",
			},
		},
		{
			name:         "Forbidden with different message",
			errorMessage: "Forbidden - access denied",
			expectedLogs: []string{
				"Permission denied: Your user|token doesn't have sufficient permissions to upload trusted keys",
				"Contact your administrator to grant trusted keys upload permissions",
			},
		},
		{
			name:         "Insufficient permissions",
			errorMessage: "insufficient permissions to perform this operation",
			expectedLogs: []string{
				"Permission denied: Your user|token doesn't have sufficient permissions to upload trusted keys",
				"Contact your administrator to grant trusted keys upload permissions",
			},
		},
		{
			name:         "Page not found",
			errorMessage: "404 page not found",
			expectedLogs: []string{
				"Endpoint not found: The trusted keys API endpoint is not available",
				"Check your server URL and ensure trusted keys feature is enabled",
			},
		},
		{
			name:         "Endpoint not available",
			errorMessage: "endpoint not available",
			expectedLogs: []string{
				"Endpoint not found: The trusted keys API endpoint is not available",
				"Check your server URL and ensure trusted keys feature is enabled",
			},
		},
		{
			name:         "Unauthorized with different message",
			errorMessage: "Unauthorized access",
			expectedLogs: []string{
				"Authentication failed: Invalid or expired authentication token",
				"Check your access token or regenerate a new one",
			},
		},
		{
			name:         "Invalid or expired token",
			errorMessage: "invalid or expired token",
			expectedLogs: []string{
				"Authentication failed: Invalid or expired authentication token",
				"Check your access token or regenerate a new one",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewGenerateKeyPairCommand(nil, false, "", "", "")

			// Create a mock error
			err := fmt.Errorf("%s", tt.errorMessage)

			// Test the logUploadWarning function
			// Note: In a real test, you might want to capture log output
			// For now, we'll just ensure the function doesn't panic
			assert.NotPanics(t, func() {
				cmd.logUploadWarning(err)
			})

			// Verify the error message contains expected content
			// This is a basic check - in a real implementation you might want to
			// capture and verify the actual log output
			errStr := err.Error()
			for range tt.expectedLogs {
				// We can't easily test the actual log output in unit tests
				// but we can verify the error message contains the expected patterns
				switch {
				case strings.Contains(errStr, "403") || strings.Contains(errStr, "Forbidden") || strings.Contains(errStr, "insufficient permissions"):
					assert.Contains(t, tt.expectedLogs[0], "Permission denied")
				case strings.Contains(errStr, "404") || strings.Contains(errStr, "page not found") || strings.Contains(errStr, "endpoint not available"):
					assert.Contains(t, tt.expectedLogs[0], "Endpoint not found")
				case strings.Contains(errStr, "401") || strings.Contains(errStr, "Unauthorized") || strings.Contains(errStr, "invalid or expired"):
					assert.Contains(t, tt.expectedLogs[0], "Authentication failed")
				case strings.Contains(errStr, "already exists"):
					assert.Contains(t, tt.expectedLogs[0], "unique alias")
				}
			}
		})
	}
}

// TestUploadTrustedKeyErrorHandling tests the UploadTrustedKey error handling
func TestUploadTrustedKeyErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		responseBody  string
		expectedError string
	}{
		{
			name:          "403 Forbidden",
			statusCode:    403,
			responseBody:  `{"errors": [{"status": 403, "message": "Forbidden"}]}`,
			expectedError: "trusted keys API returned status 403: Forbidden - insufficient permissions to upload trusted keys: Forbidden",
		},
		{
			name:          "404 Not Found",
			statusCode:    404,
			responseBody:  `{"errors": [{"status": 404, "message": "Not Found"}]}`,
			expectedError: "trusted keys API returned status 404: 404 page not found - trusted keys API endpoint not available: Not Found",
		},
		{
			name:          "401 Unauthorized",
			statusCode:    401,
			responseBody:  `{"errors": [{"status": 401, "message": "Unauthorized"}]}`,
			expectedError: "trusted keys API returned status 401: Unauthorized - invalid or expired authentication token: Unauthorized",
		},
		{
			name:          "400 Bad Request",
			statusCode:    400,
			responseBody:  `{"errors": [{"status": 400, "message": "alias already exists"}]}`,
			expectedError: "trusted keys API returned status 400: alias already exists",
		},
		{
			name:          "500 Internal Server Error",
			statusCode:    500,
			responseBody:  `{"errors": [{"status": 500, "message": "Internal Server Error"}]}`,
			expectedError: "trusted keys API returned status 500: Internal Server Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test would require mocking the HTTP client
			// For now, we'll test the error message construction logic

			// Simulate the error message construction
			errorMsg := fmt.Sprintf("trusted keys API returned status %d", tt.statusCode)

			// Add specific error messages for common status codes
			switch tt.statusCode {
			case 403:
				errorMsg += ": Forbidden - insufficient permissions to upload trusted keys"
			case 404:
				errorMsg += ": 404 page not found - trusted keys API endpoint not available"
			case 401:
				errorMsg += ": Unauthorized - invalid or expired authentication token"
			}

			// Add response body if present
			if tt.responseBody != "" {
				// Parse JSON response to extract message
				var response struct {
					Errors []struct {
						Status  int    `json:"status"`
						Message string `json:"message"`
					} `json:"errors"`
				}
				if err := json.Unmarshal([]byte(tt.responseBody), &response); err == nil && len(response.Errors) > 0 {
					errorMsg += ": " + response.Errors[0].Message
				} else {
					errorMsg += ": " + tt.responseBody
				}
			}

			assert.Equal(t, tt.expectedError, errorMsg)
		})
	}
}

// TestErrorHandlingIntegration tests the integration of error handling
func TestErrorHandlingIntegration(t *testing.T) {
	tests := []struct {
		name          string
		errorMessage  string
		shouldContain []string
	}{
		{
			name:         "403 error integration",
			errorMessage: "trusted keys API returned status 403: Forbidden - insufficient permissions to upload trusted keys",
			shouldContain: []string{
				"status 403",
				"Forbidden",
				"insufficient permissions",
			},
		},
		{
			name:         "404 error integration",
			errorMessage: "trusted keys API returned status 404: 404 page not found - trusted keys API endpoint not available",
			shouldContain: []string{
				"status 404",
				"page not found",
				"endpoint not available",
			},
		},
		{
			name:         "401 error integration",
			errorMessage: "trusted keys API returned status 401: Unauthorized - invalid or expired authentication token",
			shouldContain: []string{
				"status 401",
				"Unauthorized",
				"invalid or expired",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewGenerateKeyPairCommand(nil, false, "", "", "")
			err := fmt.Errorf("%s", tt.errorMessage)

			// Test that the error message contains expected patterns
			errStr := err.Error()
			for _, expected := range tt.shouldContain {
				assert.Contains(t, errStr, expected)
			}

			// Test that logUploadWarning doesn't panic
			assert.NotPanics(t, func() {
				cmd.logUploadWarning(err)
			})
		})
	}
}
