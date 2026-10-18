package create

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/artifactory/services"
	"github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	"github.com/jfrog/jfrog-client-go/utils/io/content"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAttachmentArtifactoryTempPath(t *testing.T) {
	tests := []struct {
		name         string
		target       string
		localPath    string
		expectedPath string
		expectErr    bool
	}{
		{name: "repo only", target: "repo", localPath: "/tmp/file.txt", expectedPath: "repo/file.txt"},
		{name: "directory target", target: "repo/dir/", localPath: "/tmp/file.txt", expectedPath: "repo/dir/file.txt"},
		{name: "explicit filename", target: "repo/dir/custom.bin", localPath: "/tmp/file.txt", expectedPath: "repo/dir/custom.bin"},
		{name: "invalid leading slash", target: "/repo/dir/", localPath: "/tmp/file.txt", expectErr: true},
		{name: "wildcard star", target: "repo/dir/*", localPath: "/tmp/file.txt", expectErr: true},
		{name: "wildcard question", target: "repo/dir/file?.txt", localPath: "/tmp/file.txt", expectErr: true},
		{name: "parent segment", target: "repo/../other/file.txt", localPath: "/tmp/file.txt", expectErr: true},
		{name: "empty segment", target: "repo//file.txt", localPath: "/tmp/file.txt", expectErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parseAttachmentArtifactoryTempPath(tt.target, tt.localPath)
			if tt.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expectedPath, parsed.TargetPath)
		})
	}
}

func TestResolveAttachment_NoAttachments_SkipsVersionCheck(t *testing.T) {
	versionCalled := false
	mock := &SimpleMockServicesManager{
		GetVersionFunc: func() (string, error) {
			versionCalled = true
			return "7.999.0", nil
		},
	}
	c := &createEvidenceBase{}
	att, cleanup, err := c.resolveAttachment(mock)
	require.NoError(t, err)
	assert.Nil(t, att)
	assert.Nil(t, cleanup)
	assert.False(t, versionCalled, "GetVersion should not be called when no attachments are requested")
}

func TestResolveAttachment_WithAttachment_Success(t *testing.T) {
	mock := &SimpleMockServicesManager{
		FileInfoFunc: func(_ string) (*utils.FileInfo, error) {
			return NewFileInfoBuilder().WithSha256("abc123").Build(), nil
		},
	}
	c := &createEvidenceBase{attachArtifactoryPath: "repo/path/file.txt"}
	att, _, err := c.resolveAttachment(mock)
	require.NoError(t, err)
	require.NotNil(t, att)
	assert.Equal(t, "abc123", att.Sha256)
}

func TestWrapCreatePayloadWithAttachments(t *testing.T) {
	base := &createEvidenceBase{}
	envelope := []byte(`{"payload":"abc","payloadType":"application/vnd.in-toto+json","signatures":[]}`)
	att := &statementAttachment{Repository: "repo", Path: "a/b.txt", Sha256: "sha"}
	wrapped, err := base.wrapCreatePayloadWithAttachments(envelope, att)
	require.NoError(t, err)
	assert.Contains(t, string(wrapped), `"attachments"`)
	assert.Contains(t, string(wrapped), `"repository":"repo"`)
	assert.Contains(t, string(wrapped), `"path":"a/b.txt"`)
	assert.Contains(t, string(wrapped), `"sha256":"sha"`)
}

func TestUploadLocalAttachment_RefusesExistingArtifact(t *testing.T) {
	uploaded := false
	mock := &SimpleMockServicesManager{
		FileInfoFunc: func(string) (*utils.FileInfo, error) {
			return NewFileInfoBuilder().WithSha256("existing").Build(), nil
		},
		UploadFilesFunc: func(artifactory.UploadServiceOptions, ...services.UploadParams) (int, int, error) {
			uploaded = true
			return 1, 0, nil
		},
	}
	c := &createEvidenceBase{
		attachLocalPath:           writeTempAttachment(t),
		attachArtifactoryTempPath: "repo/prod/app.jar",
	}
	_, _, err := c.uploadLocalAttachment(mock)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "refusing to upload")
	assert.False(t, uploaded)
}

func TestUploadLocalAttachment_FailsClosedWhenLookupErrors(t *testing.T) {
	uploaded := false
	mock := &SimpleMockServicesManager{
		FileInfoFunc: func(string) (*utils.FileInfo, error) {
			return nil, errors.New("connection reset")
		},
		UploadFilesFunc: func(artifactory.UploadServiceOptions, ...services.UploadParams) (int, int, error) {
			uploaded = true
			return 1, 0, nil
		},
	}
	c := &createEvidenceBase{
		attachLocalPath:           writeTempAttachment(t),
		attachArtifactoryTempPath: "repo/prod/app.jar",
	}
	_, _, err := c.uploadLocalAttachment(mock)
	require.Error(t, err)
	assert.False(t, uploaded)
}

func TestUploadLocalAttachment_CleanupDeletesOnlyMatchingChecksum(t *testing.T) {
	const sha = "abc123"
	calls := 0
	var deleted services.DeleteParams
	deleteCalled := false
	mock := &SimpleMockServicesManager{
		FileInfoFunc: func(string) (*utils.FileInfo, error) {
			calls++
			if calls == 1 {
				return nil, errors.New("server response: 404 Not Found")
			}
			return NewFileInfoBuilder().WithSha256(sha).Build(), nil
		},
		UploadFilesFunc: func(_ artifactory.UploadServiceOptions, params ...services.UploadParams) (int, int, error) {
			require.Len(t, params, 1)
			assert.Equal(t, "repo/dir/note.txt", params[0].Target)
			return 1, 0, nil
		},
		GetPathsToDeleteFunc: func(params services.DeleteParams) (*content.ContentReader, error) {
			deleted = params
			return content.NewEmptyContentReader("results"), nil
		},
		DeleteFilesFunc: func(*content.ContentReader) (int, error) {
			deleteCalled = true
			return 1, nil
		},
	}
	c := &createEvidenceBase{
		attachLocalPath:           writeTempAttachment(t),
		attachArtifactoryTempPath: "repo/dir/note.txt",
	}
	att, cleanup, err := c.uploadLocalAttachment(mock)
	require.NoError(t, err)
	require.NotNil(t, cleanup)
	assert.Equal(t, sha, att.Sha256)
	cleanup()
	require.True(t, deleteCalled)
	assert.Empty(t, deleted.Pattern)
	assertAttachmentAQL(t, deleted.Aql.ItemsFind, map[string]string{
		"repo":   "repo",
		"path":   "dir",
		"name":   "note.txt",
		"sha256": sha,
		"type":   "file",
	})
}

func TestUploadLocalAttachment_CleanupSkipsChecksumMismatch(t *testing.T) {
	calls := 0
	deleteCalled := false
	mock := &SimpleMockServicesManager{
		FileInfoFunc: func(string) (*utils.FileInfo, error) {
			calls++
			switch calls {
			case 1:
				return nil, errors.New("server response: 404 Not Found")
			case 2:
				return NewFileInfoBuilder().WithSha256("uploaded").Build(), nil
			default:
				return NewFileInfoBuilder().WithSha256("someone-else").Build(), nil
			}
		},
		UploadFilesFunc: func(artifactory.UploadServiceOptions, ...services.UploadParams) (int, int, error) {
			return 1, 0, nil
		},
		GetPathsToDeleteFunc: func(services.DeleteParams) (*content.ContentReader, error) {
			deleteCalled = true
			return content.NewEmptyContentReader("results"), nil
		},
	}
	c := &createEvidenceBase{
		attachLocalPath:           writeTempAttachment(t),
		attachArtifactoryTempPath: "repo/dir/note.txt",
	}
	_, cleanup, err := c.uploadLocalAttachment(mock)
	require.NoError(t, err)
	cleanup()
	assert.False(t, deleteCalled)
}

func TestExactAttachmentCleanupAQL(t *testing.T) {
	raw, err := exactAttachmentCleanupAQL(`repo/dir/a"b.txt`, "deadbeef")
	require.NoError(t, err)
	assert.Contains(t, raw, `a\"b.txt`)
	assertAttachmentAQL(t, raw, map[string]string{
		"repo":   "repo",
		"path":   "dir",
		"name":   `a"b.txt`,
		"sha256": "deadbeef",
		"type":   "file",
	})

	root, err := exactAttachmentCleanupAQL("repo/app.jar", "sha")
	require.NoError(t, err)
	assertAttachmentAQL(t, root, map[string]string{
		"repo":   "repo",
		"path":   ".",
		"name":   "app.jar",
		"sha256": "sha",
		"type":   "file",
	})
}

func writeTempAttachment(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "note.txt")
	require.NoError(t, os.WriteFile(path, []byte("attachment"), 0600))
	return path
}

func assertAttachmentAQL(t *testing.T, raw string, expected map[string]string) {
	t.Helper()
	var find map[string]string
	require.NoError(t, json.Unmarshal([]byte(raw), &find))
	assert.Equal(t, expected, find)
}
