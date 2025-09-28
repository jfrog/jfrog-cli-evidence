package generate

import (
	"fmt"
	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/artifactory/services"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jfrog/jfrog-cli-core/v2/artifactory/utils"
	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cryptox"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"github.com/jfrog/jfrog-client-go/utils/log"
)

// KeyPairCommand generates an ECDSA P-256 key pair
type KeyPairCommand struct {
	serverDetails     *config.ServerDetails
	uploadPublicKey   bool
	keyAlias          string
	forceOverwrite    bool
	outputDir         string
	encryptPrivateKey bool
}

// NewGenerateKeyPairCommand creates a new instance of GenerateKeyPairCommand
func NewGenerateKeyPairCommand(serverDetails *config.ServerDetails, uploadPublicKey bool, keyAlias string, forceOverwrite bool, outputDir string, encryptPrivateKey bool) *KeyPairCommand {
	return &KeyPairCommand{
		serverDetails:     serverDetails,
		uploadPublicKey:   uploadPublicKey,
		keyAlias:          keyAlias,
		forceOverwrite:    forceOverwrite,
		outputDir:         outputDir,
		encryptPrivateKey: encryptPrivateKey,
	}
}

// Run executes the key pair generation
func (cmd *KeyPairCommand) Run() error {
	log.Info("🔑 JFrog Evidence Key Pair Generation")
	log.Info("Generating ECDSA P-256 key pair for evidence signing...")

	// Determine output directory and key file paths
	outputDir := cmd.outputDir
	if outputDir == "" {
		outputDir = "." // Current directory
	}

	// Create output directory if it doesn't exist
	if outputDir != "." {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return errorutils.CheckError(fmt.Errorf("failed to create output directory %s: %w", outputDir, err))
		}
		log.Info(fmt.Sprintf("📁 Output directory: %s", outputDir))
	}

	// Build key file paths
	privateKeyPath := filepath.Join(outputDir, "evidence.key")
	publicKeyPath := filepath.Join(outputDir, "evidence.pub")

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

	// Pre-validate alias if upload is requested (fail fast)
	if cmd.uploadPublicKey {
		if err := cmd.preValidateAlias(); err != nil {
			return err
		}
	}

	// Generate the key pair
	var privateKeyPEM, publicKeyPEM string
	var err error

	if cmd.encryptPrivateKey {
		log.Info("🔐 Private key will be encrypted with password")
		privateKeyPEM, publicKeyPEM, err = cryptox.GenerateECDSAKeyPairWithPassword(cryptox.GetPassword)
	} else {
		log.Info("🔓 Private key will be stored unencrypted (protected by file permissions)")
		privateKeyPEM, publicKeyPEM, err = cryptox.GenerateECDSAKeyPair()
	}

	if err != nil {
		return errorutils.CheckError(fmt.Errorf("key generation failed: %w", err))
	}

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

	log.Info(fmt.Sprintf("✅  Private key saved: %s", privateKeyPath))
	log.Info(fmt.Sprintf("✅  Public key saved: %s", publicKeyPath))

	// Display the alias that would be used for upload
	alias := cmd.keyAlias
	if alias == "" {
		// Generate the same default alias that would be used for upload
		timestamp := time.Now().Format("20060102-150405")
		alias = fmt.Sprintf("evd-key-%s", timestamp)
		log.Info(fmt.Sprintf("✅  Key alias: %s", alias))
	} else {
		log.Info(fmt.Sprintf("✅  Key alias: %s", alias))
	}

	// Upload to trusted keys API if requested
	if cmd.uploadPublicKey {
		log.Info("Uploading public key to JFrog platform trusted keys...")
		if err := cmd.uploadToTrustedKeys(publicKeyPEM); err != nil {
			// Don't fail the whole command if upload fails, just warn
			log.Warn("❌ Failed to upload public key to JFrog platform:", err.Error())
			log.Warn("⚠️  Key pair was generated successfully, but trusted keys upload failed")
			if strings.Contains(err.Error(), "already exists") {
				log.Warn("💡 To resolve: Use a unique alias with --key-alias <unique-name>")
			} else {
				log.Warn("💡 You can manually upload the public key later or check your server configuration")
			}
		} else {
			log.Info("✅ Public key successfully uploaded to JFrog platform trusted keys")
		}
	}

	log.Info("🎉 Key pair generation completed successfully!")
	log.Info("Now you can use the private key for signing evidence with JFrog CLI")

	return nil
}

// uploadToTrustedKeys uploads the public key to JFrog trusted keys API
func (cmd *KeyPairCommand) uploadToTrustedKeys(publicKeyPEM string) error {
	if cmd.serverDetails == nil {
		return errorutils.CheckError(fmt.Errorf("server details required for uploading to trusted keys"))
	}

	// Use a default alias if none provided
	alias := cmd.keyAlias
	if alias == "" {
		// Generate a unique default alias with timestamp
		timestamp := time.Now().Format("20060102-150405")
		alias = fmt.Sprintf("evd-key-%s", timestamp)
		log.Info(fmt.Sprintf("💡 No alias provided, using generated alias: %s", alias))
	} else {
		log.Debug(fmt.Sprintf("Using provided key alias: %s", alias))
	}

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
func (cmd *KeyPairCommand) preValidateAlias() error {
	if cmd.serverDetails == nil {
		return errorutils.CheckError(fmt.Errorf("server details required for uploading to trusted keys"))
	}

	// Determine the alias that will be used
	alias := cmd.keyAlias
	if alias == "" {
		// Generate the same default alias that would be used later
		timestamp := time.Now().Format("20060102-150405")
		alias = fmt.Sprintf("evd-key-%s", timestamp)
	}

	log.Info(fmt.Sprintf("🔍 Validating alias '%s' availability...", alias))

	// Create Artifactory service manager
	serviceManager, err := utils.CreateUploadServiceManager(cmd.serverDetails, 1, 0, 0, false, nil)
	if err != nil {
		return errorutils.CheckError(fmt.Errorf("failed to create service manager for alias validation: %w", err))
	}

	// Check if alias exists
	exists, err := serviceManager.CheckAliasExists(alias)
	if err != nil {
		log.Warn("⚠️  Could not validate alias availability - proceeding with generation")
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
