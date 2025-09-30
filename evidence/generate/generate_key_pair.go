package generate

import (
	"fmt"
	"github.com/jfrog/jfrog-cli-core/v2/artifactory/utils"
	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/artifactory/services"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cryptox"
	"github.com/jfrog/jfrog-client-go/utils/log"
)

// Service manager configuration constants
const (
	DefaultRetries = 1
	DefaultTimeout = 0
	DefaultThreads = 0
	DefaultDryRun  = false
)

// File permission constants
const (
	PrivateKeyPermissions = 0600
	PublicKeyPermissions  = 0644
	DirectoryPermissions  = 0755
)

// KeyPairCommand represents a command for generating ECDSA P-256 key pairs for evidence signing.
// It handles key generation, file management, and optional upload to JFrog platform trusted keys.
type KeyPairCommand struct {
	serverDetails   *config.ServerDetails
	uploadPublicKey bool
	keyAlias        string
	keyFilePath     string
	keyFileName     string
}

// NewGenerateKeyPairCommand creates a new KeyPairCommand instance with the provided configuration.
// Parameters:
//   - serverDetails: JFrog platform connection details (required for trusted keys upload)
//   - uploadPublicKey: whether to upload the generated public key to trusted keys
//   - keyAlias: custom alias for the key (empty for auto-generated timestamp-based alias)
//   - keyFilePath: directory to save key files (empty for current directory)
//   - keyFileName: base name for key files (empty for default "evidence")
func NewGenerateKeyPairCommand(serverDetails *config.ServerDetails, uploadPublicKey bool, keyAlias string, keyFilePath string, keyFileName string) *KeyPairCommand {
	return &KeyPairCommand{
		serverDetails:   serverDetails,
		uploadPublicKey: uploadPublicKey,
		keyAlias:        keyAlias,
		keyFilePath:     keyFilePath,
		keyFileName:     keyFileName,
	}
}

// Run executes the complete key pair generation workflow.
// It generates ECDSA P-256 keys, saves them to files with proper permissions,
// and optionally uploads the public key to JFrog platform trusted keys.
// Returns an error if any step fails.
func (cmd *KeyPairCommand) Run() error {
	log.Info("🔑 JFrog Evidence Key Pair Generation")
	log.Info("Generating ECDSA P-256 key pair for evidence signing...")

	alias := cmd.generateOrGetAlias()

	keyFilePath, err := cmd.prepareKeyFilePath()
	if err != nil {
		return err
	}

	privateKeyPath, publicKeyPath, err := cmd.buildKeyFilePaths(keyFilePath)
	if err != nil {
		return err
	}

	if err := cmd.validateExistingFiles(privateKeyPath, publicKeyPath); err != nil {
		return err
	}

	privateKeyPEM, publicKeyPEM, err := cmd.generateKeyPair()
	if err != nil {
		return err
	}

	if err := cmd.writeKeyFiles(privateKeyPEM, publicKeyPEM, privateKeyPath, publicKeyPath); err != nil {
		return err
	}

	cmd.logSuccess(alias, privateKeyPath, publicKeyPath)

	if err := cmd.uploadPublicKeyIfNeeded(publicKeyPEM, alias); err != nil {
		cmd.logUploadWarning(err)
	}

	cmd.logCompletion()
	return nil
}

// generateOrGetAlias generates a timestamp-based alias if none provided.
// If keyAlias is empty, it creates a unique alias using the format
// "evd-key-YYYYMMDD-HHMMSS" based on the current timestamp.
func (cmd *KeyPairCommand) generateOrGetAlias() string {
	if cmd.keyAlias != "" {
		return cmd.keyAlias
	}
	timestamp := time.Now().Format("20060102-150405")
	return fmt.Sprintf("evd-key-%s", timestamp)
}

// prepareKeyFilePath creates the key file path directory if needed.
// If keyFilePath is empty, it defaults to the current directory.
// Creates the directory with proper permissions if it doesn't exist.
func (cmd *KeyPairCommand) prepareKeyFilePath() (string, error) {
	keyFilePath := cmd.keyFilePath
	if keyFilePath == "" {
		keyFilePath = "." // Current directory
	}

	if keyFilePath != "." {
		if err := os.MkdirAll(keyFilePath, DirectoryPermissions); err != nil {
			return "", fmt.Errorf("failed to create key file path directory %s: %w", keyFilePath, err)
		}
		log.Info(fmt.Sprintf("📁 Key file path: %s", keyFilePath))
	}

	return keyFilePath, nil
}

// buildKeyFilePaths constructs the full paths for private and public key files.
// Uses the configured keyFileName or defaults to "evidence".
// Returns the private key path (.key) and public key path (.pub).
func (cmd *KeyPairCommand) buildKeyFilePaths(keyFilePath string) (string, string, error) {
	keyFileName := cmd.keyFileName
	if keyFileName == "" {
		keyFileName = "evidence" // Default file name
	}

	privateKeyPath := filepath.Join(keyFilePath, keyFileName+".key")
	publicKeyPath := filepath.Join(keyFilePath, keyFileName+".pub")

	return privateKeyPath, publicKeyPath, nil
}

// validateExistingFiles checks if key files already exist and returns an error with helpful message.
func (cmd *KeyPairCommand) validateExistingFiles(privateKeyPath, publicKeyPath string) error {
	log.Debug("Checking for existing key files...")

	if _, err := os.Stat(privateKeyPath); err == nil {
		return fmt.Errorf("private key file %s already exists - please remove it first or use a different location", privateKeyPath)
	}

	if _, err := os.Stat(publicKeyPath); err == nil {
		return fmt.Errorf("public key file %s already exists - please remove it first or use a different location", publicKeyPath)
	}

	return nil
}


// generateKeyPair creates a new ECDSA P-256 key pair using cryptox.GenerateECDSAKeyPair.
// Returns PEM-encoded private and public keys as strings.
func (cmd *KeyPairCommand) generateKeyPair() (string, string, error) {
	privateKeyPEM, publicKeyPEM, err := cryptox.GenerateECDSAKeyPair()
	if err != nil {
		return "", "", fmt.Errorf("key generation failed: %w", err)
	}
	return privateKeyPEM, publicKeyPEM, nil
}

// writeKeyFiles writes the private and public keys to their respective files.
// Sets proper file permissions: 0600 for private key (owner read/write only),
// 0644 for public key (owner read/write, group/other read).
func (cmd *KeyPairCommand) writeKeyFiles(privateKeyPEM, publicKeyPEM, privateKeyPath, publicKeyPath string) error {
	log.Info("Writing key files...")

	// Write private key to file with restricted permissions
	if err := os.WriteFile(privateKeyPath, []byte(privateKeyPEM), PrivateKeyPermissions); err != nil {
		return fmt.Errorf("failed to write private key to %s: %w", privateKeyPath, err)
	}
	log.Debug(fmt.Sprintf("Private key written to %s with permissions 600", privateKeyPath))

	// Write public key to file
	if err := os.WriteFile(publicKeyPath, []byte(publicKeyPEM), PublicKeyPermissions); err != nil {
		return fmt.Errorf("failed to write public key to %s: %w", publicKeyPath, err)
	}
	log.Debug(fmt.Sprintf("Public key written to %s with permissions 644", publicKeyPath))

	return nil
}

// logSuccess logs the successful creation of key files and alias information.
func (cmd *KeyPairCommand) logSuccess(alias, privateKeyPath, publicKeyPath string) {
	log.Info(fmt.Sprintf("✅ Private key saved: %s", privateKeyPath))
	log.Info(fmt.Sprintf("✅ Public key saved: %s", publicKeyPath))
	log.Info(fmt.Sprintf("✅ Key alias: %s", alias))
}

// uploadPublicKeyIfNeeded uploads the public key to JFrog platform if requested.
// Only performs upload when uploadPublicKey is true.
func (cmd *KeyPairCommand) uploadPublicKeyIfNeeded(publicKeyPEM, alias string) error {
	if !cmd.uploadPublicKey {
		return nil
	}

	log.Info("Uploading public key to JFrog platform trusted keys...")
	return cmd.uploadToTrustedKeys(publicKeyPEM, alias)
}

// logUploadWarning logs appropriate warning messages for upload failures.
// Provides specific guidance based on the error type (duplicate alias vs other issues).
func (cmd *KeyPairCommand) logUploadWarning(err error) {
	log.Warn("❌ Failed to upload public key to JFrog platform:", err.Error())
	log.Warn("⚠️ Key pair was generated successfully, but trusted keys upload failed")

	if strings.Contains(err.Error(), "already exists") {
		log.Warn("💡 To resolve: Use a unique alias with --key-alias <unique-name>")
	} else {
		log.Warn("💡 You can manually upload the public key later or check your server configuration")
	}
}

// logCompletion logs the final success message with usage instructions.
func (cmd *KeyPairCommand) logCompletion() {
	log.Info("🎉 Key pair generation completed successfully!")
	log.Info("Now you can use the private key for signing evidence with JFrog CLI")
}

// uploadToTrustedKeys uploads the public key to JFrog trusted keys API.
// Creates an Artifactory service manager and uses it to upload the key with the specified alias.
func (cmd *KeyPairCommand) uploadToTrustedKeys(publicKeyPEM string, alias string) error {
	if cmd.serverDetails == nil {
		return fmt.Errorf("server details required for uploading to trusted keys")
	}

	// Use the provided alias
	log.Debug(fmt.Sprintf("Using alias for upload: %s", alias))

	log.Debug("Creating Artifactory service manager for trusted keys upload...")
	// Create Artifactory service manager
	serviceManager, err := utils.CreateUploadServiceManager(cmd.serverDetails, DefaultRetries, DefaultTimeout, DefaultThreads, DefaultDryRun, nil)
	if err != nil {
		return fmt.Errorf("failed to create service manager: %w", err)
	}

	log.Debug(fmt.Sprintf("Uploading public key with alias '%s' to trusted keys API...", alias))
	// Upload the key using the utility function
	response, err := cmd.UploadTrustedKey(&serviceManager, alias, publicKeyPEM)
	if err != nil {
		return fmt.Errorf("trusted keys API upload failed: %w", err)
	}

	log.Debug(fmt.Sprintf("Trusted keys upload response: %+v", response))
	return nil
}


// CommandName returns the command name for error handling and logging purposes.
func (cmd *KeyPairCommand) CommandName() string {
	return "generate-key-pair"
}

// UploadTrustedKey uploads a public key to the JFrog trusted keys API using the ArtifactoryServicesManager.
// Validates input parameters and creates TrustedKeyParams for the upload operation.
func (cmd *KeyPairCommand) UploadTrustedKey(serviceManager *artifactory.ArtifactoryServicesManager, alias, publicKey string) (*services.TrustedKeyResponse, error) {
	if serviceManager == nil {
		return nil, fmt.Errorf("artifactory services manager cannot be nil")
	}
	if alias == "" {
		return nil, fmt.Errorf("key alias cannot be empty")
	}
	if publicKey == "" {
		return nil, fmt.Errorf("public key cannot be empty")
	}

	// Prepare the parameters
	params := services.TrustedKeyParams{
		Alias:     alias,
		PublicKey: publicKey,
	}

	// Use the service manager to upload the trusted key
	return (*serviceManager).UploadTrustedKey(params)
}
