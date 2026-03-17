package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jfrog/jfrog-cli-evidence/tests/e2e"
	"github.com/jfrog/jfrog-cli-evidence/tests/e2e/utils"
	clientutils "github.com/jfrog/jfrog-client-go/utils"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"github.com/stretchr/testify/require"
)

func (r *EvidenceE2ETestsRunner) RunCreateEvidenceWithAttachmentPermissions(t *testing.T) {
	t.Log("=== Create Evidence - Attachment Permissions Test ===")
	require.NotEmpty(t, SharedPrivateKeyPath, "shared key pair not initialized")
	ensureAttachmentSupportedEvidenceVersion(t)

	t.Run("LocalAttachment_UserHasSubjectAndTempWritePermissions", func(t *testing.T) {
		r.runAttachmentCase(t, attachmentCase{
			name:               "local-attachment-write-allowed",
			useLocalAttachment: true,
			grantTempWrite:     true,
			expectError:        false,
		})
	})

	t.Run("LocalAttachment_UserHasSubjectButNoTempWritePermissions", func(t *testing.T) {
		r.runAttachmentCase(t, attachmentCase{
			name:               "local-attachment-write-denied",
			useLocalAttachment: true,
			grantTempWrite:     false,
			expectError:        true,
			errorContains: []string{
				"failed to upload --attach-local file",
				"403 Forbidden",
			},
		})
	})

	t.Run("ArtifactoryAttachment_UserHasSubjectAndAttachmentReadPermissions", func(t *testing.T) {
		r.runAttachmentCase(t, attachmentCase{
			name:                           "artifactory-attachment-read-allowed",
			useArtifactoryAttachment:       true,
			grantArtifactoryAttachmentRead: true,
			expectError:                    false,
		})
	})

	t.Run("ArtifactoryAttachment_UserHasSubjectButNoAttachmentReadPermissions", func(t *testing.T) {
		r.runAttachmentCase(t, attachmentCase{
			name:                           "artifactory-attachment-read-denied",
			useArtifactoryAttachment:       true,
			grantArtifactoryAttachmentRead: false,
			expectError:                    true,
			errorContains: []string{
				"failed to resolve --attach-artifactory",
				"403 Forbidden",
			},
		})
	})
}

type attachmentCase struct {
	name                           string
	useLocalAttachment             bool
	useArtifactoryAttachment       bool
	grantTempWrite                 bool
	grantArtifactoryAttachmentRead bool
	expectError                    bool
	errorContains                  []string
}

func (r *EvidenceE2ETestsRunner) runAttachmentCase(t *testing.T, tc attachmentCase) {
	t.Helper()

	subjectRepo := utils.CreateTestRepositoryWithName(t, r.ServicesManager, "generic")
	subjectArtifactPath := utils.CreateTestArtifact(t, "subject artifact for attachment permissions case")
	subjectArtifactName := filepath.Base(subjectArtifactPath)
	subjectRepoPath := fmt.Sprintf("%s/%s", subjectRepo, subjectArtifactName)
	require.NoError(t, utils.UploadArtifact(r.ServicesManager, subjectArtifactPath, subjectRepoPath))

	tmpRepo := utils.CreateTestRepositoryWithName(t, r.ServicesManager, "generic")
	tmpTarget := fmt.Sprintf("%s/tmp/", tmpRepo)
	localAttachmentPath := utils.CreateTestArtifact(t, "local attachment content")

	attachmentRepo := utils.CreateTestRepositoryWithName(t, r.ServicesManager, "generic")
	attachmentRepoPath := fmt.Sprintf("%s/attachments/preuploaded.txt", attachmentRepo)
	require.NoError(t, utils.UploadArtifact(r.ServicesManager, localAttachmentPath, attachmentRepoPath))

	tempDir := t.TempDir()
	predicatePath := filepath.Join(tempDir, "predicate.json")
	require.NoError(t, os.WriteFile(predicatePath, []byte(`{"type":"attachment-permissions-e2e"}`), 0644))

	username := fmt.Sprintf("att-e2e-%d", time.Now().UnixNano())
	password := "EvidenceTest123!"
	email := fmt.Sprintf("%s@jfrog.local", username)
	adminToken := mustGetAdminToken(t)
	baseURL := getJfrogBaseURL()

	require.NoError(t, createAccessUser(baseURL, adminToken, username, password, email))
	t.Cleanup(func() {
		_ = deleteAccessUser(baseURL, adminToken, username)
	})

	groupName := fmt.Sprintf("att-e2e-group-%d", time.Now().UnixNano())
	require.NoError(t, createAccessGroup(baseURL, adminToken, groupName, username))
	t.Cleanup(func() {
		_ = deleteAccessGroup(baseURL, adminToken, groupName)
	})

	subjectPermissionName := fmt.Sprintf("perm-subject-%d", time.Now().UnixNano())
	require.NoError(t, createAccessPermissionForGroup(
		baseURL,
		adminToken,
		subjectPermissionName,
		groupName,
		[]string{"READ", "ANNOTATE"},
		[]string{subjectRepo},
	))
	t.Cleanup(func() {
		_ = deleteAccessPermission(baseURL, adminToken, subjectPermissionName)
	})

	if tc.grantTempWrite {
		tmpPermissionName := fmt.Sprintf("perm-tmp-%d", time.Now().UnixNano())
		require.NoError(t, createAccessPermissionForGroup(
			baseURL,
			adminToken,
			tmpPermissionName,
			groupName,
			[]string{"READ", "WRITE", "DELETE"},
			[]string{tmpRepo},
		))
		t.Cleanup(func() {
			_ = deleteAccessPermission(baseURL, adminToken, tmpPermissionName)
		})
	}

	if tc.grantArtifactoryAttachmentRead {
		attachmentPermissionName := fmt.Sprintf("perm-att-read-%d", time.Now().UnixNano())
		require.NoError(t, createAccessPermissionForGroup(
			baseURL,
			adminToken,
			attachmentPermissionName,
			groupName,
			[]string{"READ"},
			[]string{attachmentRepo},
		))
		t.Cleanup(func() {
			_ = deleteAccessPermission(baseURL, adminToken, attachmentPermissionName)
		})
	}

	userToken, err := createGroupScopedUserToken(baseURL, adminToken, username, groupName)
	require.NoError(t, err)
	require.NotEmpty(t, userToken)

	createArgs := []string{
		"create",
		"--predicate", predicatePath,
		"--predicate-type", "https://slsa.dev/provenance/v1",
		"--subject-repo-path", subjectRepoPath,
		"--key", SharedPrivateKeyPath,
		"--key-alias", SharedKeyAlias,
	}
	if tc.useLocalAttachment {
		createArgs = append(createArgs, "--attach-local", localAttachmentPath, "--attach-temp-target", tmpTarget)
	}
	if tc.useArtifactoryAttachment {
		createArgs = append(createArgs, "--attach-artifactory", attachmentRepoPath)
	}

	output, runErr := runEvidenceCommandWithToken(userToken, createArgs...)
	combined := output
	if runErr != nil {
		combined = combined + "\n" + runErr.Error()
	}

	if tc.expectError {
		require.Error(t, runErr, "attachment case should fail")
		for _, msg := range tc.errorContains {
			require.Contains(t, combined, msg)
		}
		return
	}

	require.NoError(t, runErr, "attachment case should succeed")
	require.NotContains(t, combined, "Error")
	require.NotContains(t, combined, "Failed")
}

func runEvidenceCommandWithToken(token string, args ...string) (string, error) {
	_, currentFile, _, _ := runtime.Caller(0)
	binaryPath := filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "build", "jfrog-evidence")
	cmdArgs := append([]string{}, args...)
	cmdArgs = append(cmdArgs, fmt.Sprintf("--url=%s", *e2e.JfrogUrl), fmt.Sprintf("--access-token=%s", token))
	cmd := exec.Command(binaryPath, cmdArgs...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func getJfrogBaseURL() string {
	return strings.TrimRight(*e2e.JfrogUrl, "/")
}

func ensureAttachmentSupportedEvidenceVersion(t *testing.T) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, getJfrogBaseURL()+"/evidence/api/v1/system/version", nil)
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

func mustGetAdminToken(t *testing.T) string {
	t.Helper()
	if e2e.ArtifactoryAdminToken != nil && strings.TrimSpace(*e2e.ArtifactoryAdminToken) != "" {
		return strings.TrimSpace(*e2e.ArtifactoryAdminToken)
	}
	_, currentFile, _, _ := runtime.Caller(0)
	tokenPath := filepath.Join(filepath.Dir(currentFile), "..", "local", ".admin_token")
	data, err := os.ReadFile(tokenPath)
	require.NoError(t, err)
	token := strings.TrimSpace(string(data))
	require.NotEmpty(t, token)
	return token
}

func createAccessUser(baseURL, adminToken, username, password, email string) error {
	payload := map[string]any{
		"username":          username,
		"password":          password,
		"email":             email,
		"groups":            []string{},
		"profile_updatable": true,
		"admin":             false,
	}
	content, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, baseURL+"/access/api/v2/users", bytes.NewReader(content))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusConflict {
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	return errorutils.CheckErrorf("failed to create user '%s': %s (%s)", username, resp.Status, string(body))
}

func createGroupScopedUserToken(baseURL, adminToken, username, groupName string) (string, error) {
	form := fmt.Sprintf("username=%s&scope=applied-permissions/groups:%s&expires_in=0", username, groupName)
	req, err := http.NewRequest(http.MethodPost, baseURL+"/access/api/v1/tokens", strings.NewReader(form))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", errorutils.CheckErrorf("failed to create group-scoped token for user '%s': %s (%s)", username, resp.Status, string(body))
	}
	var parsed struct {
		AccessToken string `json:"access_token"`
	}
	if err = json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if strings.TrimSpace(parsed.AccessToken) == "" {
		return "", errorutils.CheckErrorf("empty access_token in token response for user '%s' and group '%s'", username, groupName)
	}
	return parsed.AccessToken, nil
}

func deleteAccessUser(baseURL, adminToken, username string) error {
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/access/api/v2/users/%s", baseURL, username), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+adminToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound {
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	return errorutils.CheckErrorf("failed to delete user '%s': %s (%s)", username, resp.Status, string(body))
}

func deleteAccessPermission(baseURL, adminToken, permissionName string) error {
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/access/api/v2/permissions/%s", baseURL, permissionName), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+adminToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound {
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	return errorutils.CheckErrorf("failed to delete permission '%s': %s (%s)", permissionName, resp.Status, string(body))
}

func createAccessGroup(baseURL, adminToken, groupName, username string) error {
	payload := map[string]any{
		"name":             groupName,
		"description":      "attachment e2e isolated group",
		"admin_privileges": false,
		"auto_join":        false,
		"members":          []string{username},
	}
	content, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, baseURL+"/access/api/v2/groups", bytes.NewReader(content))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusConflict {
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	return errorutils.CheckErrorf("failed to create group '%s': %s (%s)", groupName, resp.Status, string(body))
}

func deleteAccessGroup(baseURL, adminToken, groupName string) error {
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/access/api/v2/groups/%s", baseURL, groupName), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+adminToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound {
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	return errorutils.CheckErrorf("failed to delete group '%s': %s (%s)", groupName, resp.Status, string(body))
}

func createAccessPermissionForGroup(baseURL, adminToken, permissionName, groupName string, actions []string, repos []string) error {
	targets := map[string]map[string][]string{}
	for _, repo := range repos {
		targets[repo] = map[string][]string{
			"include_patterns": {"**"},
			"exclude_patterns": {},
		}
	}
	payload := map[string]any{
		"name": permissionName,
		"resources": map[string]any{
			"artifact": map[string]any{
				"actions": map[string]any{
					"users":  map[string][]string{},
					"groups": map[string][]string{groupName: actions},
				},
				"targets": targets,
			},
			"repository": map[string]any{
				"actions": map[string]any{
					"users":  map[string][]string{},
					"groups": map[string][]string{groupName: actions},
				},
				"targets": targets,
			},
		},
	}
	content, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	err = postAccessPermission(baseURL, adminToken, content)
	if err == nil {
		return nil
	}
	deleteErr := deleteAccessPermission(baseURL, adminToken, permissionName)
	if deleteErr != nil {
		return err
	}
	return postAccessPermission(baseURL, adminToken, content)
}

func postAccessPermission(baseURL, adminToken string, content []byte) error {
	req, err := http.NewRequest(http.MethodPost, baseURL+"/access/api/v2/permissions", bytes.NewReader(content))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK {
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	return errorutils.CheckErrorf("failed to create permission target: %s (%s)", resp.Status, string(body))
}
