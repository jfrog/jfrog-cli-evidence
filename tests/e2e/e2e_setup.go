package e2e

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	rtUtils "github.com/jfrog/jfrog-cli-core/v2/artifactory/utils"
	"github.com/jfrog/jfrog-cli-core/v2/plugins"
	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	coreTests "github.com/jfrog/jfrog-cli-core/v2/utils/tests"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli"
	"github.com/jfrog/jfrog-client-go/artifactory"
	clientUtils "github.com/jfrog/jfrog-client-go/utils"
)

// Test configuration flags
var (
	JfrogUrl            *string
	JfrogUser           *string
	JfrogPassword       *string
	JfrogAccessToken    *string
	EvidenceAccessToken *string
)

const (
	Out = "out"
)

func init() {
	JfrogUrl = flag.String("jfrog.url", "http://localhost:8082", "JFrog platform url")
	JfrogUser = flag.String("jfrog.user", "admin", "JFrog platform username")
	JfrogPassword = flag.String("jfrog.password", "password", "JFrog platform password")
	JfrogAccessToken = flag.String("jfrog.adminToken", "", "JFrog platform admin token (for Artifactory)")
	EvidenceAccessToken = flag.String("jfrog.evidenceToken", "", "Evidence service access token")
}

var (
	artifactoryDetails *config.ServerDetails
	EvidenceCli        *coreTests.JfrogCli                    // Exported for use in tests subpackage
	ArtifactoryClient  artifactory.ArtifactoryServicesManager // Exported for use in tests subpackage
)

// SetupE2ETests initializes the E2E test environment
// Exported so subpackages can call it from their TestMain
func SetupE2ETests() {
	flag.Parse()
	initEvidenceCli()
	initArtifactoryClient()
}

// TearDownE2ETests cleans up the E2E test environment
// Exported so subpackages can call it from their TestMain
func TearDownE2ETests() {
	CleanFileSystem()
}

// CleanFileSystem removes temporary test files
func CleanFileSystem() {
	if _, err := os.Stat(Out); err == nil {
		err := os.RemoveAll(Out)
		if err != nil {
			return
		}
	}
}

func initEvidenceCli() {
	if EvidenceCli != nil {
		return
	}
	EvidenceCli = coreTests.NewJfrogCli(execEvidenceCliMain, "jfrog-evidence", authenticateEvidence())
}

func initArtifactoryClient() {
	if ArtifactoryClient != nil {
		return
	}

	updateArtifactoryDetails()

	var err error
	ArtifactoryClient, err = rtUtils.CreateServiceManager(artifactoryDetails, -1, 0, false)
	if err != nil {
		fmt.Printf("ERROR: Failed to create Artifactory services manager: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Artifactory services manager initialized")
}

// getTokenFilePath returns the absolute path to the .access_token file
// regardless of where the tests are run from
func getTokenFilePath() string {
	// Get the path to this source file (e2e_setup.go)
	_, filename, _, _ := runtime.Caller(0)
	// Get the e2e directory (where this file is located)
	e2eDir := filepath.Dir(filename)
	// Build the path to .access_token in the same directory
	return filepath.Join(e2eDir, ".access_token")
}

func authenticateEvidence() string {
	*JfrogUrl = clientUtils.AddTrailingSlashIfNeeded(*JfrogUrl)

	// Priority: Evidence token from flag > Token file (local Docker)
	var token string
	
	if *EvidenceAccessToken != "" {
		// Use dedicated Evidence token (SaaS environment)
		token = *EvidenceAccessToken
		fmt.Printf("✓ Using Evidence access token from flag\n")
	} else {
		// Try to read from file (local Docker environment)
		tokenFile := getTokenFilePath()
		if data, err := os.ReadFile(tokenFile); err == nil {
			token = strings.TrimSpace(string(data))
			if token != "" && token != "null" {
				fmt.Printf("✓ Using Evidence access token from %s\n", tokenFile)
			}
		}
	}

	if token == "" {
		fmt.Printf("ERROR: Evidence service requires an access token.\n")
		fmt.Printf("Options:\n")
		fmt.Printf("  1. For Docker: Run bootstrap: make start-e2e-env\n")
		fmt.Printf("  2. For SaaS: Provide flag: --jfrog.evidenceToken=YOUR_TOKEN\n")
		os.Exit(1)
	}

	fmt.Printf("✓ Evidence authentication configured for: %s\n", *JfrogUrl)

	evidenceDetails := &config.ServerDetails{
		Url:         *JfrogUrl,
		AccessToken: token,
	}
	return fmt.Sprintf("--url=%s --access-token=%s", evidenceDetails.Url, evidenceDetails.AccessToken)
}

func updateArtifactoryDetails() {
	*JfrogUrl = clientUtils.AddTrailingSlashIfNeeded(*JfrogUrl)
	artifactoryUrl := clientUtils.AddTrailingSlashIfNeeded(*JfrogUrl) + "artifactory/"
	
	artifactoryDetails = &config.ServerDetails{
		Url:            *JfrogUrl,
		ArtifactoryUrl: artifactoryUrl,
	}
	
	// Use access token if provided (SaaS), otherwise use username/password (localhost Docker)
	if *JfrogAccessToken != "" {
		artifactoryDetails.AccessToken = *JfrogAccessToken
		fmt.Printf("✓ Artifactory authentication configured with access token for: %s\n", artifactoryUrl)
	} else {
		artifactoryDetails.User = *JfrogUser
		artifactoryDetails.Password = *JfrogPassword
		fmt.Printf("✓ Artifactory authentication configured with username/password for: %s\n", artifactoryUrl)
	}
}

func execEvidenceCliMain() error {
	plugins.PluginMain(cli.GetStandaloneEvidenceApp())
	return nil
}
