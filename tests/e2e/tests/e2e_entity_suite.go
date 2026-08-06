package tests

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jfrog/jfrog-cli-evidence/tests/e2e"
	"github.com/jfrog/jfrog-cli-evidence/tests/e2e/utils"
	"github.com/stretchr/testify/require"
)

const (
	entityTypeGitCommit    = "gitCommit"
	entityTypeApplication  = "application"
	entityPredicateTypeURL = "https://jfrog.com/evidence/commit-approval/v1"
	// emptyPayloadSha256 is the sha256 of an empty payload, reported by the service as the
	// subject checksum of entity evidence. It must never surface as a verified checksum.
	emptyPayloadSha256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
)

// newGitCommitEntityID returns a 40-char hex SHA-1 suitable as a gitCommit entity id.
// Evidence validates gitCommit digests as 20-byte hex strings.
func newGitCommitEntityID(seed string) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", seed, time.Now().UnixNano())))
	return hex.EncodeToString(sum[:20])
}

func writeEntityPredicate(t *testing.T, tempDir string, values map[string]interface{}) string {
	t.Helper()
	predicateBytes, err := json.MarshalIndent(values, "", "  ")
	require.NoError(t, err)
	predicatePath := filepath.Join(tempDir, "predicate.json")
	require.NoError(t, os.WriteFile(predicatePath, predicateBytes, 0644))
	return predicatePath
}

// requireEntityDigestReported asserts that the verification report shows the entity digest read
// from the signed in-toto statement, and that it does not claim a sha256 was verified. Entity
// evidence carries no content checksum, so a sha256 status would be a false claim.
func requireEntityDigestReported(t *testing.T, verifyOutput, entityType, entityID string) {
	t.Helper()
	require.Contains(t, verifyOutput, fmt.Sprintf("%s: %s", entityType, entityID),
		"Verification should report the entity digest found in the signed statement")
	require.NotContains(t, verifyOutput, "Sha256 verification status",
		"Verification of an entity subject must not report a sha256 status")
	require.NotContains(t, verifyOutput, emptyPayloadSha256,
		"Verification must not report the sha256 of an empty payload as the subject checksum")
}

func requireSharedKeys(t *testing.T) {
	t.Helper()
	if SharedPrivateKeyPath == "" || SharedPublicKeyPath == "" {
		t.Fatalf("Shared key pair not initialized. Ensure PrepareTestsData() was called.")
	}
	t.Logf("Using shared key pair: %s (alias: %s)", SharedPrivateKeyPath, SharedKeyAlias)
}

func ensureDefaultGitCommitEntityRepo(t *testing.T, r *EvidenceE2ETestsRunner) string {
	t.Helper()
	repoKey := utils.DefaultEntityRepoKey(entityTypeGitCommit)
	utils.EnsureEntityRepository(t, r.ServicesManager, repoKey, "")
	return repoKey
}

func ensureProjectGitCommitEntityRepo(t *testing.T, r *EvidenceE2ETestsRunner) string {
	t.Helper()
	require.NotEmpty(t, e2e.ProjectKey, "Project key must be available from bootstrap")
	repoKey := utils.ProjectEntityRepoKey(e2e.ProjectKey, entityTypeGitCommit)
	utils.EnsureEntityRepository(t, r.ServicesManager, repoKey, e2e.ProjectKey)
	return repoKey
}

func ensureProjectApplicationEntityRepo(t *testing.T, r *EvidenceE2ETestsRunner) string {
	t.Helper()
	require.NotEmpty(t, e2e.ProjectKey, "Project key must be available from bootstrap")
	repoKey := utils.ProjectEntityRepoKey(e2e.ProjectKey, entityTypeApplication)
	utils.EnsureEntityRepository(t, r.ServicesManager, repoKey, e2e.ProjectKey)
	return repoKey
}

func registerEntityEvidenceCleanup(t *testing.T, r *EvidenceE2ETestsRunner, repoKey, entityType, entityID string) {
	t.Helper()
	t.Cleanup(func() {
		utils.CleanupEntityEvidence(t, r.ServicesManager, repoKey, entityType, entityID)
	})
}

// RunCreateEvidenceForEntity creates evidence for a gitCommit entity in the default
// gitCommit-entity repository (created if missing, left in place).
func (r *EvidenceE2ETestsRunner) RunCreateEvidenceForEntity(t *testing.T) {
	t.Log("=== Create Evidence - Entity (default scope) ===")
	requireSharedKeys(t)

	repoKey := ensureDefaultGitCommitEntityRepo(t, r)
	tempDir := t.TempDir()
	entityID := newGitCommitEntityID("e2e-git-commit")
	registerEntityEvidenceCleanup(t, r, repoKey, entityTypeGitCommit, entityID)

	t.Log("Step 1: Creating predicate...")
	predicatePath := writeEntityPredicate(t, tempDir, map[string]interface{}{
		"buildType":   "entity-test",
		"timestamp":   time.Now().Unix(),
		"environment": "e2e-test",
		"entityType":  entityTypeGitCommit,
		"entityId":    entityID,
	})
	t.Log("✓ Predicate created")

	t.Log("Step 2: Creating evidence for entity...")
	createOutput := RunCliCmdWithStdOutputAndErrOutput(t, r.EvidenceUserCLI,
		"create",
		"--predicate", predicatePath,
		"--predicate-type", entityPredicateTypeURL,
		"--entity-type", entityTypeGitCommit,
		"--entity-id", entityID,
		"--key", SharedPrivateKeyPath,
		"--key-alias", SharedKeyAlias,
	)
	t.Logf("Evidence creation output: %s", createOutput)
	require.NotContains(t, createOutput, "Error", "Evidence creation should not error")
	require.NotContains(t, createOutput, "Failed", "Evidence creation should not fail")
	t.Log("✓ Evidence created successfully")

	t.Log("Step 3: Getting evidence to validate creation...")
	getOutput := RunCliCmdWithStdOutputAndErrOutput(t, r.EvidenceAdminCLI,
		"get",
		"--entity-type", entityTypeGitCommit,
		"--entity-id", entityID,
	)
	t.Logf("Evidence get output: %s", getOutput)
	require.Contains(t, getOutput, entityTypeGitCommit, "Output should include entity type")
	require.Contains(t, getOutput, entityID, "Output should include entity id")
	t.Log("✓ Evidence retrieved successfully")

	t.Log("=== ✅ Create Evidence for Entity Test Completed Successfully! ===")
}

// RunCreateEvidenceForEntityWithProject creates evidence for a gitCommit entity
// scoped to the shared e2e project ({project}-gitCommit-entity).
func (r *EvidenceE2ETestsRunner) RunCreateEvidenceForEntityWithProject(t *testing.T) {
	t.Log("=== Create Evidence - Entity with Project ===")
	requireSharedKeys(t)

	repoKey := ensureProjectGitCommitEntityRepo(t, r)
	tempDir := t.TempDir()
	entityID := newGitCommitEntityID("e2e-project-git-commit")
	registerEntityEvidenceCleanup(t, r, repoKey, entityTypeGitCommit, entityID)

	t.Logf("Using project: %s", e2e.ProjectKey)
	t.Log("Step 1: Creating predicate...")
	predicatePath := writeEntityPredicate(t, tempDir, map[string]interface{}{
		"buildType":   "entity-project-test",
		"timestamp":   time.Now().Unix(),
		"environment": "e2e-test",
		"entityType":  entityTypeGitCommit,
		"entityId":    entityID,
		"project":     e2e.ProjectKey,
	})
	t.Log("✓ Predicate created")

	t.Log("Step 2: Creating evidence for project-scoped entity...")
	createOutput := RunCliCmdWithStdOutputAndErrOutput(t, r.EvidenceUserCLI,
		"create",
		"--predicate", predicatePath,
		"--predicate-type", entityPredicateTypeURL,
		"--entity-type", entityTypeGitCommit,
		"--entity-id", entityID,
		"--project", e2e.ProjectKey,
		"--key", SharedPrivateKeyPath,
		"--key-alias", SharedKeyAlias,
	)
	t.Logf("Evidence creation output: %s", createOutput)
	require.NotContains(t, createOutput, "Error", "Evidence creation should not error")
	require.NotContains(t, createOutput, "Failed", "Evidence creation should not fail")
	t.Log("✓ Evidence created successfully")

	t.Log("Step 3: Getting evidence to validate creation...")
	getOutput := RunCliCmdWithStdOutputAndErrOutput(t, r.EvidenceAdminCLI,
		"get",
		"--entity-type", entityTypeGitCommit,
		"--entity-id", entityID,
		"--project", e2e.ProjectKey,
	)
	t.Logf("Evidence get output: %s", getOutput)
	require.Contains(t, getOutput, entityTypeGitCommit, "Output should include entity type")
	require.Contains(t, getOutput, entityID, "Output should include entity id")
	t.Log("✓ Evidence retrieved successfully")

	t.Log("=== ✅ Create Evidence for Entity with Project Test Completed Successfully! ===")
}

// RunCreateEvidenceForApplicationEntity exercises the bare --application-key shorthand
// (entity-type=application, AppTrust-resolved project scope) for create, get, and verify.
func (r *EvidenceE2ETestsRunner) RunCreateEvidenceForApplicationEntity(t *testing.T) {
	t.Log("=== Create Evidence - Application Entity (--application-key shorthand) ===")
	requireSharedKeys(t)

	repoKey := ensureProjectApplicationEntityRepo(t, r)
	tempDir := t.TempDir()
	var applicationKey string
	t.Cleanup(func() {
		if applicationKey != "" {
			utils.CleanupEntityEvidence(t, r.ServicesManager, repoKey, entityTypeApplication, applicationKey)
			utils.CleanupTestApplication(t, r.ServicesManager, applicationKey, e2e.ProjectKey)
		}
	})

	t.Log("Step 1: Creating AppTrust application...")
	var applicationName string
	applicationKey, applicationName = utils.CreateTestApplication(t, r.ServicesManager, e2e.ProjectKey)
	t.Logf("✓ Application created: %s (%s)", applicationKey, applicationName)

	t.Log("Step 2: Creating predicate...")
	predicatePath := writeEntityPredicate(t, tempDir, map[string]interface{}{
		"buildType":      "application-entity-test",
		"timestamp":      time.Now().Unix(),
		"environment":    "e2e-test",
		"applicationKey": applicationKey,
		"project":        e2e.ProjectKey,
	})
	t.Log("✓ Predicate created")

	t.Log("Step 3: Creating evidence with bare --application-key...")
	createOutput := RunCliCmdWithStdOutputAndErrOutput(t, r.EvidenceUserCLI,
		"create",
		"--predicate", predicatePath,
		"--predicate-type", entityPredicateTypeURL,
		"--application-key", applicationKey,
		"--key", SharedPrivateKeyPath,
		"--key-alias", SharedKeyAlias,
	)
	t.Logf("Evidence creation output: %s", createOutput)
	require.NotContains(t, createOutput, "Error", "Evidence creation should not error")
	require.NotContains(t, createOutput, "Failed", "Evidence creation should not fail")
	t.Log("✓ Evidence created successfully")

	t.Log("Step 4: Getting evidence with bare --application-key...")
	getOutput := RunCliCmdWithStdOutputAndErrOutput(t, r.EvidenceUserCLI,
		"get",
		"--application-key", applicationKey,
	)
	t.Logf("Get evidence output: %s", getOutput)
	require.Contains(t, getOutput, entityTypeApplication, "Output should include application entity type")
	require.Contains(t, getOutput, applicationKey, "Output should include application key as entity id")
	t.Log("✅ Evidence retrieved with application-key shorthand")

	t.Log("Step 5: Verifying evidence with bare --application-key...")
	verifyOutput := RunCliCmdWithStdOutputAndErrOutput(t, r.EvidenceUserCLI,
		"verify",
		"--application-key", applicationKey,
		"--public-keys", SharedPublicKeyPath,
	)
	t.Logf("Verification output: %s", verifyOutput)
	require.Contains(t, verifyOutput, applicationKey, "Verification should reference the application entity")
	require.Contains(t, verifyOutput, "Verification passed", "Verification should pass")
	requireEntityDigestReported(t, verifyOutput, entityTypeApplication, applicationKey)
	t.Log("✅ Evidence verified with application-key shorthand")

	t.Log("=== ✅ Create Evidence for Application Entity Test Completed Successfully! ===")
}

// RunGetEvidenceForEntity creates entity evidence then retrieves it via get.
func (r *EvidenceE2ETestsRunner) RunGetEvidenceForEntity(t *testing.T) {
	t.Log("=== Get Evidence - Entity ===")
	requireSharedKeys(t)

	repoKey := ensureDefaultGitCommitEntityRepo(t, r)
	tempDir := t.TempDir()
	entityID := newGitCommitEntityID("e2e-get-git-commit")
	registerEntityEvidenceCleanup(t, r, repoKey, entityTypeGitCommit, entityID)

	t.Log("Step 1: Creating predicate...")
	predicatePath := writeEntityPredicate(t, tempDir, map[string]interface{}{
		"buildType":   "get-entity-test",
		"timestamp":   time.Now().Unix(),
		"environment": "e2e-get-test",
		"entityType":  entityTypeGitCommit,
		"entityId":    entityID,
	})
	t.Log("✓ Predicate created")

	t.Log("Step 2: Creating evidence using Admin CLI...")
	createOutput := RunCliCmdWithStdOutputAndErrOutput(t, r.EvidenceAdminCLI,
		"create",
		"--predicate", predicatePath,
		"--predicate-type", entityPredicateTypeURL,
		"--entity-type", entityTypeGitCommit,
		"--entity-id", entityID,
		"--key", SharedPrivateKeyPath,
		"--key-alias", SharedKeyAlias,
	)
	t.Logf("Evidence creation output: %s", createOutput)
	require.NotContains(t, createOutput, "Error", "Evidence creation should not error")
	require.NotContains(t, createOutput, "Failed", "Evidence creation should not fail")
	t.Log("✓ Evidence created successfully")

	t.Log("Step 3: Getting evidence using User CLI...")
	getOutput := RunCliCmdWithStdOutputAndErrOutput(t, r.EvidenceUserCLI,
		"get",
		"--entity-type", entityTypeGitCommit,
		"--entity-id", entityID,
	)
	t.Logf("Get evidence output: %s", getOutput)
	require.Contains(t, getOutput, `"type": "entity"`, "Output should be entity evidence")
	require.Contains(t, getOutput, entityTypeGitCommit, "Output should include entity type")
	require.Contains(t, getOutput, entityID, "Output should include entity id")
	t.Log("✅ User successfully retrieved entity evidence!")

	t.Log("=== ✅ Get Evidence for Entity Test Completed Successfully! ===")
}

// RunVerifyEvidenceForEntity creates entity evidence then verifies it.
func (r *EvidenceE2ETestsRunner) RunVerifyEvidenceForEntity(t *testing.T) {
	t.Log("=== Verify Evidence - Entity ===")
	requireSharedKeys(t)

	repoKey := ensureDefaultGitCommitEntityRepo(t, r)
	tempDir := t.TempDir()
	entityID := newGitCommitEntityID("e2e-verify-git-commit")
	subjectPath := fmt.Sprintf("%s/%s", entityTypeGitCommit, entityID)
	registerEntityEvidenceCleanup(t, r, repoKey, entityTypeGitCommit, entityID)

	t.Log("Step 1: Creating predicate...")
	predicatePath := writeEntityPredicate(t, tempDir, map[string]interface{}{
		"buildType":   "verify-entity-test",
		"timestamp":   time.Now().Unix(),
		"environment": "e2e-verify-test",
		"entityType":  entityTypeGitCommit,
		"entityId":    entityID,
	})
	t.Log("✓ Predicate created")

	t.Log("Step 2: Creating evidence using Admin CLI...")
	createOutput := RunCliCmdWithStdOutputAndErrOutput(t, r.EvidenceAdminCLI,
		"create",
		"--predicate", predicatePath,
		"--predicate-type", entityPredicateTypeURL,
		"--entity-type", entityTypeGitCommit,
		"--entity-id", entityID,
		"--key", SharedPrivateKeyPath,
		"--key-alias", SharedKeyAlias,
	)
	t.Logf("Evidence creation output: %s", createOutput)
	require.NotContains(t, createOutput, "Error", "Evidence creation should not error")
	require.NotContains(t, createOutput, "Failed", "Evidence creation should not fail")
	t.Log("✓ Evidence created successfully")

	t.Log("Step 3: Verifying evidence using User CLI...")
	verifyOutput := RunCliCmdWithStdOutputAndErrOutput(t, r.EvidenceUserCLI,
		"verify",
		"--entity-type", entityTypeGitCommit,
		"--entity-id", entityID,
		"--public-keys", SharedPublicKeyPath,
	)
	t.Logf("Verification output: %s", verifyOutput)
	require.Contains(t, verifyOutput, subjectPath, "Verification should reference the entity subject path")
	require.Contains(t, verifyOutput, "Verification passed", "Verification should pass")
	requireEntityDigestReported(t, verifyOutput, entityTypeGitCommit, entityID)
	t.Log("✅ User successfully verified entity evidence!")

	t.Log("=== ✅ Verify Evidence for Entity Test Completed Successfully! ===")
}
