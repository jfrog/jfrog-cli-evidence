package create

import (
	"testing"

	"github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAttachmentTempTarget(t *testing.T) {
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parseAttachmentTempTarget(tt.target, tt.localPath)
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
