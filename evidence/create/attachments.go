package create

import (
	"encoding/json"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/jfrog/jfrog-cli-evidence/evidence/intoto"
	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/artifactory/services"
	artUtils "github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"github.com/jfrog/jfrog-client-go/utils/log"
)

type statementAttachment struct {
	Repository string
	Path       string
	Sha256     string
	Name       string
	Type       string
}

type parsedTarget struct {
	Repository string
	TargetPath string
}

func (c *createEvidenceBase) resolveAttachment(client artifactory.ArtifactoryServicesManager) (*statementAttachment, func(), error) {
	if c.attachLocalPath == "" && c.attachArtifactoryPath == "" {
		return nil, nil, nil
	}
	if c.attachArtifactoryPath != "" {
		att, err := c.resolveExistingArtifactoryAttachment(client, c.attachArtifactoryPath)
		return att, nil, err
	}
	return c.uploadLocalAttachment(client)
}

func (c *createEvidenceBase) resolveExistingArtifactoryAttachment(client artifactory.ArtifactoryServicesManager, repoPath string) (*statementAttachment, error) {
	repository, path, err := splitRepoPath(repoPath)
	if err != nil {
		return nil, err
	}
	if strings.HasSuffix(path, "/") {
		return nil, errorutils.CheckErrorf("invalid --attach-artifactory value '%s': expected a file path, got a directory", repoPath)
	}
	fileInfo, err := client.FileInfo(repoPath)
	if err != nil {
		return nil, errorutils.CheckErrorf("failed to resolve --attach-artifactory '%s': %v", repoPath, err)
	}
	if fileInfo == nil || fileInfo.Checksums.Sha256 == "" {
		return nil, errorutils.CheckErrorf("invalid --attach-artifactory value '%s': path must point to a file with sha256", repoPath)
	}
	return &statementAttachment{
		Repository: repository,
		Path:       path,
		Sha256:     fileInfo.Checksums.Sha256,
		Name:       filepath.Base(path),
		Type:       detectMimeType(path),
	}, nil
}

func (c *createEvidenceBase) uploadLocalAttachment(client artifactory.ArtifactoryServicesManager) (*statementAttachment, func(), error) {
	if _, err := os.Stat(c.attachLocalPath); err != nil {
		return nil, nil, errorutils.CheckErrorf("failed to read --attach-local file '%s': %v", c.attachLocalPath, err)
	}
	target, err := parseAttachmentTempTarget(c.attachTempTarget, c.attachLocalPath)
	if err != nil {
		return nil, nil, err
	}

	uploadParams := services.UploadParams{
		CommonParams: &artUtils.CommonParams{
			Pattern: c.attachLocalPath,
			Target:  target.TargetPath,
		},
	}
	uploaded, failed, err := client.UploadFiles(artifactory.UploadServiceOptions{}, uploadParams)
	if err != nil {
		return nil, nil, errorutils.CheckErrorf("failed to upload --attach-local file to '%s': %v", target.TargetPath, err)
	}
	if failed > 0 || uploaded == 0 {
		return nil, nil, errorutils.CheckErrorf("failed to upload --attach-local file to '%s'", target.TargetPath)
	}

	repository, path, err := splitRepoPath(target.TargetPath)
	if err != nil {
		return nil, nil, err
	}
	fileInfo, err := client.FileInfo(target.TargetPath)
	if err != nil {
		return nil, nil, errorutils.CheckErrorf("failed to resolve uploaded attachment '%s': %v", target.TargetPath, err)
	}
	if fileInfo == nil || fileInfo.Checksums.Sha256 == "" {
		return nil, nil, errorutils.CheckErrorf("uploaded attachment '%s' is invalid: sha256 checksum is missing", target.TargetPath)
	}

	cleanup := func() {
		deleteParams := services.NewDeleteParams()
		deleteParams.Pattern = target.TargetPath
		reader, err := client.GetPathsToDelete(deleteParams)
		if err != nil {
			log.Warn("Failed to create cleanup plan for temporary attachment:", target.TargetPath, "error:", err)
			return
		}
		if reader == nil {
			return
		}
		defer func() {
			_ = reader.Close()
		}()
		if _, err = client.DeleteFiles(reader); err != nil {
			log.Warn("Failed to cleanup temporary attachment:", target.TargetPath, "error:", err)
		}
	}

	return &statementAttachment{
		Repository: repository,
		Path:       path,
		Sha256:     fileInfo.Checksums.Sha256,
		Name:       filepath.Base(path),
		Type:       detectMimeType(c.attachLocalPath),
	}, cleanup, nil
}

func toStatementAttachmentMeta(att *statementAttachment) []intoto.Attachment {
	if att == nil {
		return nil
	}
	return []intoto.Attachment{{
		Name:   att.Name,
		Sha256: att.Sha256,
		Type:   att.Type,
	}}
}

func (c *createEvidenceBase) wrapCreatePayloadWithAttachments(envelopeBytes []byte, att *statementAttachment) ([]byte, error) {
	if att == nil {
		return envelopeBytes, nil
	}
	var payload map[string]any
	if err := jsonUnmarshal(envelopeBytes, &payload); err != nil {
		return nil, err
	}
	payload["attachments"] = []map[string]string{{
		"repository": att.Repository,
		"path":       att.Path,
		"sha256":     att.Sha256,
	}}
	return jsonMarshal(payload)
}

func parseAttachmentTempTarget(target, localFilePath string) (*parsedTarget, error) {
	if target == "" {
		return nil, errorutils.CheckErrorf("--attach-temp-target cannot be empty")
	}
	if strings.HasPrefix(target, "/") {
		return nil, errorutils.CheckErrorf("invalid --attach-temp-target '%s': leading '/' is not allowed", target)
	}
	segments := strings.Split(target, "/")
	if len(segments) == 0 || segments[0] == "" {
		return nil, errorutils.CheckErrorf("invalid --attach-temp-target '%s': repository segment is required", target)
	}
	repo := segments[0]

	isDirectoryInput := len(segments) == 1 || strings.HasSuffix(target, "/")
	localName := filepath.Base(localFilePath)
	var finalPath string
	if isDirectoryInput {
		if len(segments) == 1 {
			finalPath = fmt.Sprintf("%s/%s", repo, localName)
		} else {
			finalPath = fmt.Sprintf("%s%s", target, localName)
		}
	} else {
		finalPath = target
	}

	return &parsedTarget{
		Repository: repo,
		TargetPath: finalPath,
	}, nil
}

func splitRepoPath(repoPath string) (string, string, error) {
	parts := strings.SplitN(repoPath, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", errorutils.CheckErrorf("invalid repository path '%s': expected <repo>/<path>", repoPath)
	}
	return parts[0], parts[1], nil
}

func detectMimeType(path string) string {
	ext := filepath.Ext(path)
	if ext == "" {
		return ""
	}
	return mime.TypeByExtension(ext)
}

// indirection for tests
var (
	jsonUnmarshal = json.Unmarshal
	jsonMarshal   = json.Marshal
)
