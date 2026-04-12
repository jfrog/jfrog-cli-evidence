package tests

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jfrog/jfrog-cli-evidence/tests/e2e/utils"
	clientutils "github.com/jfrog/jfrog-client-go/utils"
	"github.com/stretchr/testify/require"
)

type attachmentEvidenceFixture struct {
	SubjectRepoPath    string
	AttachmentRepoPath string
	PredicatePath      string
}

func (r *EvidenceE2ETestsRunner) RunCreateEvidenceWithAttachment(t *testing.T) {
	t.Log("=== Create Evidence - With Attachment Test ===")
	require.NotEmpty(t, SharedPrivateKeyPath, "shared key pair not initialized")

	ensureAttachmentSupportedArtifactoryVersion(t, r)
	ensureAttachmentSupportedEvidenceVersion(t, r)

	fixture := prepareAttachmentEvidenceFixture(t, r, "create-attachments")

	createOutput := r.EvidenceUserCLI.RunCliCmdWithOutput(t,
		"create",
		"--predicate", fixture.PredicatePath,
		"--predicate-type", "https://slsa.dev/provenance/v1",
		"--subject-repo-path", fixture.SubjectRepoPath,
		"--attach-artifactory-path", fixture.AttachmentRepoPath,
		"--key", SharedPrivateKeyPath,
		"--key-alias", SharedKeyAlias,
	)
	t.Logf("Evidence creation output: %s", createOutput)
	require.NotContains(t, createOutput, "Error", "evidence create with attachment should succeed")
	require.NotContains(t, createOutput, "Failed", "evidence create with attachment should succeed")

	getOutput := r.EvidenceAdminCLI.RunCliCmdWithOutput(t,
		"get",
		"--subject-repo-path", fixture.SubjectRepoPath,
		"--format", "json",
	)
	assertGetOutputContainsAttachment(t, getOutput)
	t.Log("=== ✅ Create Evidence with Attachment completed ===")
}

func prepareAttachmentEvidenceFixture(t *testing.T, r *EvidenceE2ETestsRunner, predicateKind string) attachmentEvidenceFixture {
	t.Helper()

	subjectRepo := utils.CreateTestRepositoryWithName(t, r.ServicesManager, "generic")
	subjectArtifactPath := utils.CreateTestArtifact(t, fmt.Sprintf("subject artifact for %s", predicateKind))
	subjectName := filepath.Base(subjectArtifactPath)
	subjectRepoPath := fmt.Sprintf("%s/%s", subjectRepo, subjectName)
	require.NoError(t, utils.UploadArtifact(r.ServicesManager, subjectArtifactPath, subjectRepoPath))

	attachmentRepo := utils.CreateTestRepositoryWithName(t, r.ServicesManager, "generic")
	attachmentPath := utils.CreateTestArtifact(t, fmt.Sprintf("attachment for %s", predicateKind))
	attachmentRepoPath := fmt.Sprintf("%s/attachments/report.txt", attachmentRepo)
	require.NoError(t, utils.UploadArtifact(r.ServicesManager, attachmentPath, attachmentRepoPath))

	tempDir := t.TempDir()
	predicatePath := filepath.Join(tempDir, "predicate.json")
	require.NoError(t, os.WriteFile(predicatePath, []byte(fmt.Sprintf(`{"type":"%s"}`, predicateKind)), 0644))

	return attachmentEvidenceFixture{
		SubjectRepoPath:    subjectRepoPath,
		AttachmentRepoPath: attachmentRepoPath,
		PredicatePath:      predicatePath,
	}
}

func assertGetOutputContainsAttachment(t *testing.T, getOutput string) {
	t.Helper()

	var jsonData map[string]any
	require.NoError(t, json.Unmarshal([]byte(getOutput), &jsonData), "get output should be valid JSON")

	resultMap, ok := jsonData["result"].(map[string]any)
	require.True(t, ok, "get output must include result")

	evidences, ok := resultMap["evidence"].([]any)
	require.True(t, ok, "get output must include evidence list")
	require.NotEmpty(t, evidences, "get output must include at least one evidence")

	firstEvidence, ok := evidences[0].(map[string]any)
	require.True(t, ok, "first evidence must be an object")

	attachments, exists := firstEvidence["attachments"]
	require.True(t, exists, "attachments should be present for evidence with attachment")
	require.NotEmpty(t, attachments, "attachments should not be empty")
}

func ensureAttachmentSupportedEvidenceVersion(t *testing.T, r *EvidenceE2ETestsRunner) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, getJfrogBaseURL(t, r)+"/evidence/api/v1/system/version", nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Skipf("skipping attachment e2e: failed to query evidence version: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	if resp.StatusCode != http.StatusOK {
		t.Skipf("skipping attachment e2e: evidence version endpoint returned %s", resp.Status)
	}

	version := strings.TrimSpace(string(body))
	if err = clientutils.ValidateMinimumVersion("JFrog Evidence", version, "7.646.1"); err != nil {
		t.Skipf("skipping attachment e2e: evidence version %q does not support attachments", version)
	}
}

func ensureAttachmentSupportedArtifactoryVersion(t *testing.T, r *EvidenceE2ETestsRunner) {
	t.Helper()
	require.NotNil(t, r)
	require.NotNil(t, r.ServicesManager, "services manager not initialized")

	version, err := r.ServicesManager.GetVersion()
	require.NoError(t, err, "failed to query Artifactory version")
	version = strings.TrimSpace(version)
	require.NotEmpty(t, version, "empty Artifactory version in response")

	if err = clientutils.ValidateMinimumVersion("JFrog Artifactory", version, "7.143.0"); err != nil {
		t.Skipf("skipping attachment e2e: Artifactory version %q is below required 7.143.0", version)
	}
}

func getJfrogBaseURL(t *testing.T, r *EvidenceE2ETestsRunner) string {
	t.Helper()
	require.NotNil(t, r)
	require.NotNil(t, r.ServicesManager, "services manager not initialized")

	config := r.ServicesManager.GetConfig()
	require.NotNil(t, config, "services manager config not initialized")

	serviceDetails := config.GetServiceDetails()
	require.NotNil(t, serviceDetails, "service details not initialized")

	baseURL := normalizePlatformURL(serviceDetails.GetUrl())
	require.NotEmpty(t, baseURL, "jfrog base url not configured")
	return baseURL
}

func normalizePlatformURL(rawURL string) string {
	baseURL := strings.TrimRight(rawURL, "/")
	return strings.TrimSuffix(baseURL, "/artifactory")
}
