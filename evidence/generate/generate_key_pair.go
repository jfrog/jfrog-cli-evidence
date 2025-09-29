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
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"github.com/jfrog/jfrog-client-go/utils/log"
)

type KeyPairCommand struct {
	serverDetails   *config.ServerDetails
	uploadPublicKey bool
	keyAlias        string
	forceOverwrite  bool
	outputDir       string
	keyFileName     string
}

func NewGenerateKeyPairCommand(serverDetails *config.ServerDetails, uploadPublicKey bool, keyAlias string, forceOverwrite bool, outputDir string, keyFileName string) *KeyPairCommand {
	return &KeyPairCommand{
		serverDetails:   serverDetails,
		uploadPublicKey: uploadPublicKey,
		keyAlias:        keyAlias,
		forceOverwrite:  forceOverwrite,
		outputDir:       outputDir,
		keyFileName:     keyFileName,
	}
}

// Run executes the key pair generation
func (cmd *KeyPairCommand) Run() error {
	log.Info("🔑 JFrog Evidence Key Pair Generation")
	log.Info("Generating ECDSA P-256 key pair for evidence signing...")

	alias := cmd.generateOrGetAlias()

	outputDir, err := cmd.prepareOutputDirectory()
	if err != nil {
		return err
	}

	privateKeyPath, publicKeyPath, err := cmd.buildKeyFilePaths(outputDir)
	if err != nil {
		return err
	}

	if err := cmd.validateExistingFiles(privateKeyPath, publicKeyPath); err != nil {
		return err
	}

	if err := cmd.preValidateAliasIfNeeded(alias); err != nil {
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

// generateOrGetAlias generates a timestamp-based alias if none provided
func (cmd *KeyPairCommand) generateOrGetAlias() string {
	if cmd.keyAlias != "" {
		return cmd.keyAlias
	}
	timestamp := time.Now().Format("20060102-150405")
	return fmt.Sprintf("evd-key-%s", timestamp)
}

// prepareOutputDirectory creates the output directory if needed
func (cmd *KeyPairCommand) prepareOutputDirectory() (string, error) {
	outputDir := cmd.outputDir
	if outputDir == "" {
		outputDir = "." // Current directory
	}

	if outputDir != "." {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return "", errorutils.CheckError(fmt.Errorf("failed to create output directory %s: %w", outputDir, err))
		}
		log.Info(fmt.Sprintf("📁 Output directory: %s", outputDir))
	}

	return outputDir, nil
}

// buildKeyFilePaths constructs the full paths for private and public key files
func (cmd *KeyPairCommand) buildKeyFilePaths(outputDir string) (string, string, error) {
	keyFileName := cmd.keyFileName
	if keyFileName == "" {
		keyFileName = "evidence" // Default file name
	}

	privateKeyPath := filepath.Join(outputDir, keyFileName+".key")
	publicKeyPath := filepath.Join(outputDir, keyFileName+".pub")

	return privateKeyPath, publicKeyPath, nil
}

// validateExistingFiles checks if key files already exist and handles overwrite logic
func (cmd *KeyPairCommand) validateExistingFiles(privateKeyPath, publicKeyPath string) error {
	log.Debug("Checking for existing key files...")

	if _, err := os.Stat(privateKeyPath); err == nil {
		if cmd.forceOverwrite {
			log.Info("🔄 Overwriting existing private key file (--force enabled)")
		} else {
			return errorutils.CheckError(fmt.Errorf("private key file %s already exists - please remove it first, use --force to overwrite, or use a different location", privateKeyPath))
		}
	}

	if _, err := os.Stat(publicKeyPath); err == nil {
		if cmd.forceOverwrite {
			log.Info("🔄 Overwriting existing public key file (--force enabled)")
		} else {
			return errorutils.CheckError(fmt.Errorf("public key file %s already exists - please remove it first, use --force to overwrite, or use a different location", publicKeyPath))
		}
	}

	return nil
}

// preValidateAliasIfNeeded validates alias availability if upload is requested
func (cmd *KeyPairCommand) preValidateAliasIfNeeded(alias string) error {
	if cmd.uploadPublicKey {
		return cmd.preValidateAlias(alias)
	}
	return nil
}

// generateKeyPair creates the ECDSA key pair
func (cmd *KeyPairCommand) generateKeyPair() (string, string, error) {
	privateKeyPEM, publicKeyPEM, err := cryptox.GenerateECDSAKeyPair()
	if err != nil {
		return "", "", errorutils.CheckError(fmt.Errorf("key generation failed: %w", err))
	}
	return privateKeyPEM, publicKeyPEM, nil
}

// writeKeyFiles writes the private and public keys to their respective files
func (cmd *KeyPairCommand) writeKeyFiles(privateKeyPEM, publicKeyPEM, privateKeyPath, publicKeyPath string) error {
	log.Info("Writing key files...")

	// Write private key to file with restricted permissions
	if err := os.WriteFile(privateKeyPath, []byte(privateKeyPEM), 0600); err != nil {
		return errorutils.CheckError(fmt.Errorf("failed to write private key to %s: %w", privateKeyPath, err))
	}
	log.Debug(fmt.Sprintf("Private key written to %s with permissions 600", privateKeyPath))

	// Write public key to file
	if err := os.WriteFile(publicKeyPath, []byte(publicKeyPEM), 0644); err != nil {
		return errorutils.CheckError(fmt.Errorf("failed to write public key to %s: %w", publicKeyPath, err))
	}
	log.Debug(fmt.Sprintf("Public key written to %s with permissions 644", publicKeyPath))

	return nil
}

// logSuccess logs the successful creation of key files and alias
func (cmd *KeyPairCommand) logSuccess(alias, privateKeyPath, publicKeyPath string) {
	log.Info(fmt.Sprintf("✅ Private key saved: %s", privateKeyPath))
	log.Info(fmt.Sprintf("✅ Public key saved: %s", publicKeyPath))
	log.Info(fmt.Sprintf("✅ Key alias: %s", alias))
}

// uploadPublicKeyIfNeeded uploads the public key to JFrog platform if requested
func (cmd *KeyPairCommand) uploadPublicKeyIfNeeded(publicKeyPEM, alias string) error {
	if !cmd.uploadPublicKey {
		return nil
	}

	log.Info("Uploading public key to JFrog platform trusted keys...")
	return cmd.uploadToTrustedKeys(publicKeyPEM, alias)
}

// logUploadWarning logs appropriate warning messages for upload failures
func (cmd *KeyPairCommand) logUploadWarning(err error) {
	log.Warn("❌ Failed to upload public key to JFrog platform:", err.Error())
	log.Warn("⚠️ Key pair was generated successfully, but trusted keys upload failed")

	if strings.Contains(err.Error(), "already exists") {
		log.Warn("💡 To resolve: Use a unique alias with --key-alias <unique-name>")
	} else {
		log.Warn("💡 You can manually upload the public key later or check your server configuration")
	}
}

// logCompletion logs the final success message
func (cmd *KeyPairCommand) logCompletion() {
	log.Info("🎉 Key pair generation completed successfully!")
	log.Info("Now you can use the private key for signing evidence with JFrog CLI")
}

// uploadToTrustedKeys uploads the public key to JFrog trusted keys API
func (cmd *KeyPairCommand) uploadToTrustedKeys(publicKeyPEM string, alias string) error {
	if cmd.serverDetails == nil {
		return errorutils.CheckError(fmt.Errorf("server details required for uploading to trusted keys"))
	}

	// Use the provided alias
	log.Debug(fmt.Sprintf("Using alias for upload: %s", alias))

	log.Debug("Creating Artifactory service manager for trusted keys upload...")
	// Create Artifactory service manager
	serviceManager, err := utils.CreateUploadServiceManager(cmd.serverDetails, 1, 0, 0, false, nil)
	if err != nil {
		return errorutils.CheckError(fmt.Errorf("failed to create service manager: %w", err))
	}

	log.Debug(fmt.Sprintf("Uploading public key with alias '%s' to trusted keys API...", alias))
	// Upload the key using the utility function
	response, err := cmd.UploadTrustedKey(&serviceManager, alias, publicKeyPEM)
	if err != nil {
		return errorutils.CheckError(fmt.Errorf("trusted keys API upload failed: %w", err))
	}

	log.Debug(fmt.Sprintf("Trusted keys upload response: %+v", response))
	return nil
}

// preValidateAlias validates the alias before key generation to fail fast
func (cmd *KeyPairCommand) preValidateAlias(alias string) error {
	if cmd.serverDetails == nil {
		return errorutils.CheckError(fmt.Errorf("server details required for uploading to trusted keys"))
	}

	// Use the pre-generated alias for validation
	log.Info(fmt.Sprintf("🔍 Validating alias '%s' availability...", alias))

	// Create Artifactory service manager
	serviceManager, err := utils.CreateUploadServiceManager(cmd.serverDetails, 1, 0, 0, false, nil)
	if err != nil {
		return errorutils.CheckError(fmt.Errorf("failed to create service manager for alias validation: %w", err))
	}

	// Check if alias exists
	exists, err := serviceManager.CheckAliasExists(alias)
	if err != nil {
		log.Warn("⚠️ Could not validate alias availability - proceeding with generation")
		log.Debug(fmt.Sprintf("Alias validation error: %v", err))
		return nil // Don't fail on validation errors, just warn
	}

	if exists {
		if cmd.keyAlias == "" {
			// This shouldn't happen with timestamp-based aliases, but handle gracefully
			return errorutils.CheckError(fmt.Errorf("default alias '%s' already exists - please specify a unique alias with --key-alias", alias))
		} else {
			return errorutils.CheckError(fmt.Errorf("alias '%s' already exists - please choose a different alias", alias))
		}
	}

	log.Info("✅ Alias is available")
	return nil
}

// CommandName returns the command name for error handling
func (cmd *KeyPairCommand) CommandName() string {
	return "generate-key-pair"
}

// UploadTrustedKey uploads a public key to the JFrog trusted keys API using the ArtifactoryServicesManager
func (cmd *KeyPairCommand) UploadTrustedKey(serviceManager *artifactory.ArtifactoryServicesManager, alias, publicKey string) (*services.TrustedKeyResponse, error) {
	if serviceManager == nil {
		return nil, errorutils.CheckErrorf("artifactory services manager cannot be nil")
	}
	if alias == "" {
		return nil, errorutils.CheckErrorf("key alias cannot be empty")
	}
	if publicKey == "" {
		return nil, errorutils.CheckErrorf("public key cannot be empty")
	}

	// Prepare the parameters
	params := services.TrustedKeyParams{
		Alias:     alias,
		PublicKey: publicKey,
	}

	// Use the service manager to upload the trusted key
	return (*serviceManager).UploadTrustedKey(params)
}
