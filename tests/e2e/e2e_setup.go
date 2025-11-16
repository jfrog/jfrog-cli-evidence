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
	"github.com/jfrog/jfrog-client-go/access"
	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/lifecycle"
	clientUtils "github.com/jfrog/jfrog-client-go/utils"
)

var (
	JfrogUrl              *string
	JfrogUser             *string
	JfrogPassword         *string
	ArtifactoryAdminToken *string
	EvidenceAccessToken   *string
	ProjectToken          *string
	ProjectKeyFlag        *string
)

func init() {
	// Don't define flags in init() to allow parent packages to provide their own flags
	// Flags will be initialized in SetupE2ETests() if not already set
}

var (
	artifactoryDetails *config.ServerDetails
	EvidenceUserCli    *coreTests.JfrogCli
	EvidenceAdminCli   *coreTests.JfrogCli
	EvidenceProjectCli *coreTests.JfrogCli
	ProjectKey         string
	ArtifactoryClient  artifactory.ArtifactoryServicesManager
	AccessClient       *access.AccessServicesManager
	LifecycleClient    *lifecycle.LifecycleServicesManager
)

// SetupE2ETests initializes the E2E test environment for jfrog-cli-evidence local and jfrog-cli-evidence tests.
//
// This function is called from TestMain in the tests package and sets up all required
// CLI instances, service managers, and authentication for running E2E tests.
//
// # How it works:
//
// 1. Flag Initialization (initializeFlags):
//   - Conditionally defines command-line flags if not already set by a parent package
//   - This allows jfrog-cli to provide its own flags while jfrog-cli-evidence can run standalone
//   - Flags include: --jfrog.url, --jfrog.adminToken, --jfrog.evidenceToken, etc.
//
// 2. Flag Parsing (parseFlags):
//   - Parses os.Args to populate flag values from command-line arguments
//   - Essential for CI/SaaS environments where values are passed as test flags
//   - Safe to call multiple times (subsequent calls are no-ops)
//
// 3. CLI Initialization:
//   - EvidenceUserCli: Uses evidence user token for create/verify/get operations
//   - EvidenceAdminCli: Uses admin token for admin operations (key upload, etc.)
//   - EvidenceProjectCli: Uses project-scoped token for project-based tests (optional)
//
// 4. Service Manager Initialization:
//   - ArtifactoryClient: For direct Artifactory API calls (repos, artifacts, builds)
//   - AccessClient: For access/permissions management
//   - LifecycleClient: For release bundle operations
//
// Local Docker environment:
//
//	make start-e2e-env  # Starts containers and runs bootstrap
//	go test ./tests/e2e/...  # Uses tokens from local/.access_token files
func SetupE2ETests() {
	initializeFlags()
	parseFlags()
	initEvidenceUserCli()
	initEvidenceAdminCli()
	initEvidenceProjectCli()
	initArtifactoryClient()
	initAccessClient()
	initLifecycleClient()
}

// ValidateSetup checks if all required components are properly initialized
// Returns an error if any critical component is missing
// Note: EvidenceProjectCli is optional and not validated (used only for project-based tests)
func ValidateSetup() error {
	if EvidenceUserCli == nil {
		return fmt.Errorf("evidence User CLI not initialized")
	}
	if EvidenceAdminCli == nil {
		return fmt.Errorf("evidence Admin CLI not initialized")
	}
	if ArtifactoryClient == nil {
		return fmt.Errorf("artifactory services manager not initialized")
	}
	if AccessClient == nil {
		return fmt.Errorf("access services manager not initialized")
	}
	if LifecycleClient == nil {
		return fmt.Errorf("lifecycle services manager not initialized")
	}
	return nil
}

// initializeFlags initializes command-line flags if not already set by parent package
// This allows jfrog-cli to provide its own flag definitions and avoid conflicts
func initializeFlags() {
	if JfrogUrl == nil {
		JfrogUrl = flag.String("jfrog.url", "http://localhost:8082", "JFrog platform url")
	}
	if JfrogUser == nil {
		JfrogUser = flag.String("jfrog.user", "admin", "JFrog platform username")
	}
	if JfrogPassword == nil {
		JfrogPassword = flag.String("jfrog.password", "password", "JFrog platform password")
	}
	if ArtifactoryAdminToken == nil {
		ArtifactoryAdminToken = flag.String("jfrog.adminToken", "", "JFrog platform admin token (for Artifactory)")
	}
	if EvidenceAccessToken == nil {
		EvidenceAccessToken = flag.String("jfrog.evidenceToken", "", "Evidence service access token")
	}
	if ProjectToken == nil {
		ProjectToken = flag.String("jfrog.projectToken", "", "Project-scoped access token")
	}
	if ProjectKeyFlag == nil {
		ProjectKeyFlag = flag.String("jfrog.projectKey", "", "Project key for project-based tests")
	}
}

// parseFlags parses command-line flags to populate flag values
// This is essential for tests to read command-line arguments like:
//
//	go test ./tests/e2e/... --jfrog.url=https://myserver.com --jfrog.adminToken=abc123
//
// Without this, all flags would use their default values and command-line args would be ignored.
// Note: Calling flag.Parse() multiple times is safe (subsequent calls are no-ops)
func parseFlags() {
	flag.Parse()
}

// initEvidenceUserCli initializes the Evidence CLI with user/evidence token
// Used for regular evidence operations (create, verify, get)
func initEvidenceUserCli() {
	if EvidenceUserCli != nil {
		return
	}
	EvidenceUserCli = coreTests.NewJfrogCli(execEvidenceCliMain, "jfrog-evidence", authenticateEvidenceUser())
	fmt.Println("✓ Evidence User CLI initialized")
}

// initEvidenceAdminCli initializes the Evidence CLI with admin token
// Used for admin operations (key upload, etc.)
func initEvidenceAdminCli() {
	if EvidenceAdminCli != nil {
		return
	}
	EvidenceAdminCli = coreTests.NewJfrogCli(execEvidenceCliMain, "jfrog-evidence", authenticateEvidenceAdmin())
	fmt.Println("✓ Evidence Admin CLI initialized")
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

func initAccessClient() {
	if AccessClient != nil {
		return
	}

	updateArtifactoryDetails()

	var err error
	AccessClient, err = rtUtils.CreateAccessServiceManager(artifactoryDetails, false)
	if err != nil {
		fmt.Printf("ERROR: Failed to create Access services manager: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Access services manager initialized")
}

func initEvidenceProjectCli() {
	if EvidenceProjectCli != nil {
		return
	}

	*JfrogUrl = clientUtils.AddTrailingSlashIfNeeded(*JfrogUrl)

	var projectToken string
	var projectKey string

	// Priority: Flags (SaaS/CI) > Files (local Docker)
	if *ProjectToken != "" && *ProjectKeyFlag != "" {
		// Use project token and key from flags (SaaS environment)
		projectToken = *ProjectToken
		projectKey = *ProjectKeyFlag
		fmt.Printf("✓ Using project token from flag\n")
		fmt.Printf("✓ Using project key from flag: %s\n", projectKey)
	} else {
		// Try to read from files (local Docker environment)
		projectTokenFile := getProjectTokenFilePath()
		projectKeyFile := getProjectKeyFilePath()

		if data, err := os.ReadFile(projectTokenFile); err == nil {
			projectToken = strings.TrimSpace(string(data))
			if projectToken != "" {
				fmt.Printf("✓ Using project token from %s\n", projectTokenFile)
			}
		}

		if data, err := os.ReadFile(projectKeyFile); err == nil {
			projectKey = strings.TrimSpace(string(data))
			if projectKey != "" {
				fmt.Printf("✓ Using project key from %s: %s\n", projectKeyFile, projectKey)
			}
		}
	}

	// Set global project key
	ProjectKey = projectKey

	// Authenticate with project-scoped token (returns auth string like User CLI)
	authString := authenticateEvidenceProject(projectToken)

	// Initialize Evidence CLI with project-scoped authentication
	EvidenceProjectCli = coreTests.NewJfrogCli(execEvidenceCliMain, "jfrog-evidence", authString)
	fmt.Printf("✓ Evidence Project CLI initialized for project: %s\n", ProjectKey)
}

func initLifecycleClient() {
	if LifecycleClient != nil {
		return
	}

	updateArtifactoryDetails()

	lifecycleDetails := &config.ServerDetails{
		Url:         artifactoryDetails.Url,
		AccessToken: artifactoryDetails.AccessToken,
		User:        artifactoryDetails.User,
		Password:    artifactoryDetails.Password,
	}

	if lifecycleDetails.Url != "" {
		baseUrl := clientUtils.AddTrailingSlashIfNeeded(lifecycleDetails.Url)
		lifecycleDetails.ArtifactoryUrl = baseUrl + "artifactory/"
		lifecycleDetails.LifecycleUrl = baseUrl + "artifactory/"
		lifecycleDetails.Url = baseUrl

		fmt.Printf("Lifecycle URL configured: %s (embedded in Artifactory)\n", lifecycleDetails.LifecycleUrl)
	}

	var err error
	LifecycleClient, err = rtUtils.CreateLifecycleServiceManager(lifecycleDetails, false)
	if err != nil {
		fmt.Printf("ERROR: Failed to create Lifecycle services manager: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Lifecycle services manager initialized")
}

// getProjectTokenFilePath returns the absolute path to the .project_token file
func getProjectTokenFilePath() string {
	_, filename, _, _ := runtime.Caller(0)
	e2eDir := filepath.Dir(filename)
	return filepath.Join(e2eDir, "local", ".project_token")
}

// getProjectKeyFilePath returns the absolute path to the .project_key file
func getProjectKeyFilePath() string {
	_, filename, _, _ := runtime.Caller(0)
	e2eDir := filepath.Dir(filename)
	return filepath.Join(e2eDir, "local", ".project_key")
}

// authenticateEvidenceProject configures authentication for Evidence Project CLI
// Uses project-scoped token for role-based operations
func authenticateEvidenceProject(token string) string {
	*JfrogUrl = clientUtils.AddTrailingSlashIfNeeded(*JfrogUrl)

	if token == "" {
		fmt.Printf("ERROR: Project token is required for Evidence Project CLI.\n")
		os.Exit(1)
	}

	fmt.Printf("✓ Evidence project authentication configured for: %s\n", *JfrogUrl)

	// Return auth string in the same format as User/Admin CLI
	return fmt.Sprintf("--url=%s --access-token=%s", *JfrogUrl, token)
}

// getTokenFilePath returns the absolute path to the .access_token file
// regardless of where the tests are run from
func getTokenFilePath() string {
	// Get the path to this source file (e2e_setup.go)
	_, filename, _, _ := runtime.Caller(0)
	// Get the e2e directory (where this file is located)
	e2eDir := filepath.Dir(filename)
	// Build the path to .access_token in the local/ subdirectory
	return filepath.Join(e2eDir, "local", ".access_token")
}

// getAdminTokenFilePath returns the absolute path to the .admin_token file
// regardless of where the tests are run from
func getAdminTokenFilePath() string {
	// Get the path to this source file (e2e_setup.go)
	_, filename, _, _ := runtime.Caller(0)
	// Get the e2e directory (where this file is located)
	e2eDir := filepath.Dir(filename)
	// Build the path to .admin_token in the local/ subdirectory
	return filepath.Join(e2eDir, "local", ".admin_token")
}

// authenticateEvidenceUser configures authentication for Evidence User CLI
// Uses evidence/user token for regular evidence operations
func authenticateEvidenceUser() string {
	*JfrogUrl = clientUtils.AddTrailingSlashIfNeeded(*JfrogUrl)

	// Priority: Evidence token from flag > Token file (local Docker)
	var token string

	if *EvidenceAccessToken != "" {
		// Use dedicated Evidence token (SaaS environment)
		token = *EvidenceAccessToken
		fmt.Printf("✓ Using Evidence user token from flag\n")
	} else {
		// Try to read from file (local Docker environment)
		tokenFile := getTokenFilePath()
		if data, err := os.ReadFile(tokenFile); err == nil {
			token = strings.TrimSpace(string(data))
			if token != "" && token != "null" {
				fmt.Printf("✓ Using Evidence user token from %s\n", tokenFile)
			}
		}
	}

	if token == "" {
		fmt.Printf("ERROR: Evidence service requires an access token.\n")
	}

	fmt.Printf("✓ Evidence user authentication configured for: %s\n", *JfrogUrl)

	evidenceDetails := &config.ServerDetails{
		Url:         *JfrogUrl,
		AccessToken: token,
	}
	return fmt.Sprintf("--url=%s --access-token=%s", evidenceDetails.Url, evidenceDetails.AccessToken)
}

// authenticateEvidenceAdmin configures authentication for Evidence Admin CLI
// Uses admin token for privileged operations (key upload, etc.)
func authenticateEvidenceAdmin() string {
	*JfrogUrl = clientUtils.AddTrailingSlashIfNeeded(*JfrogUrl)

	var token string

	// Priority: Admin token flag > Admin token file (local Docker)
	if *ArtifactoryAdminToken != "" {
		// Use admin token (SaaS environment or explicit admin)
		token = *ArtifactoryAdminToken
		fmt.Printf("✓ Using admin token from flag for Evidence Admin CLI\n")
	} else {
		// Try to read from .admin_token file (local Docker environment)
		adminTokenFile := getAdminTokenFilePath()
		if data, err := os.ReadFile(adminTokenFile); err == nil {
			token = strings.TrimSpace(string(data))
			if token != "" && token != "null" {
				fmt.Printf("✓ Using admin token from %s (local Docker bootstrap)\n", adminTokenFile)
			}
		}
	}

	if token == "" {
		fmt.Printf("ERROR: Admin token required for Evidence Admin CLI.\n")
	}

	fmt.Printf("✓ Evidence admin authentication configured for: %s\n", *JfrogUrl)

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

	// Priority: Admin token flag > Admin token file > Username/password
	if *ArtifactoryAdminToken != "" {
		artifactoryDetails.AccessToken = *ArtifactoryAdminToken
		fmt.Printf("✓ Artifactory authentication configured with admin token from flag for: %s\n", artifactoryUrl)
	} else {
		// Try to read admin token from .admin_token file (local Docker environment)
		adminTokenFile := getAdminTokenFilePath()
		if data, err := os.ReadFile(adminTokenFile); err == nil {
			token := strings.TrimSpace(string(data))
			if token != "" && token != "null" {
				artifactoryDetails.AccessToken = token
				fmt.Printf("✓ Artifactory authentication configured with admin token from %s for: %s\n", adminTokenFile, artifactoryUrl)
				return
			}
		}

		// Fallback to username/password
		artifactoryDetails.User = *JfrogUser
		artifactoryDetails.Password = *JfrogPassword
		fmt.Printf("✓ Artifactory authentication configured with username/password for: %s\n", artifactoryUrl)
	}
}

func execEvidenceCliMain() error {
	plugins.PluginMain(cli.GetStandaloneEvidenceApp())
	return nil
}
