package create

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli-evidence/evidence/client"
	"github.com/jfrog/jfrog-cli-evidence/evidence/dsse"
	"github.com/jfrog/jfrog-cli-evidence/evidence/model"
	"github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type captureEvidenceServiceClient struct {
	createReq  client.CreateEntityEvidenceRequest
	uploadBody []byte
	uploadResp []byte
	uploadErr  error
}

func (c *captureEvidenceServiceClient) PrepareEvidence(client.PrepareEvidenceRequest, bool) (*client.PrepareEvidenceResponse, error) {
	return nil, nil
}

func (c *captureEvidenceServiceClient) UploadPreparedSignedEvidence(string, []byte) ([]byte, error) {
	return nil, nil
}

func (c *captureEvidenceServiceClient) CreateEntityEvidence(request client.CreateEntityEvidenceRequest, payload []byte) ([]byte, error) {
	c.createReq = request
	c.uploadBody = payload
	if c.uploadErr != nil {
		return nil, c.uploadErr
	}
	return c.uploadResp, nil
}

func TestCreateEvidenceEntity_Run_SigstoreBundle(t *testing.T) {
	statement := map[string]any{
		"_type": "https://in-toto.io/Statement/v1",
		"subject": []any{
			map[string]any{
				"digest": map[string]any{"gitCommit": "abc123"},
				"name":   "abc123",
			},
		},
		"predicateType": "https://example.com/v1",
		"predicate":     map[string]any{},
	}
	statementBytes, err := json.Marshal(statement)
	require.NoError(t, err)
	payload := base64.StdEncoding.EncodeToString(statementBytes)
	bundleJSON := `{
        "mediaType": "application/vnd.dev.sigstore.bundle+json;version=0.2",
        "verificationMaterial": {"certificate": {"rawBytes": "dGVzdC1jZXJ0"}},
        "dsseEnvelope": {"payload": "` + payload + `", "payloadType": "application/vnd.in-toto+json", "signatures": [{"sig": "dGVzdC1zaWduYXR1cmU=", "keyid": "id"}]}
    }`
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "bundle.json")
	require.NoError(t, os.WriteFile(bundlePath, []byte(bundleJSON), 0o600))

	evidenceServiceClient := &captureEvidenceServiceClient{
		uploadResp: []byte(`{"verified":true,"predicate_slug":"example"}`),
	}
	cmd := &createEvidenceEntity{
		createEvidenceBase: createEvidenceBase{
			serverDetails:         &config.ServerDetails{Url: "https://example.jfrog.io/"},
			sigstoreBundlePath:    bundlePath,
			providerId:            "ci",
			evidenceServiceClient: evidenceServiceClient,
		},
		EntitySubject: model.EntitySubject{
			EntityType: "gitCommit",
			EntityID:   "abc123",
			ProjectKey: "proj",
		},
	}

	require.NoError(t, cmd.Run())
	assert.Equal(t, client.CreateEntityEvidenceRequest{
		EntityType: "gitCommit",
		EntityID:   "abc123",
		ProjectKey: "proj",
		ProviderID: "ci",
	}, evidenceServiceClient.createReq)
	assert.Equal(t, []byte(bundleJSON), evidenceServiceClient.uploadBody)
	require.Len(t, cmd.CollectedResponses(), 1)
	assert.True(t, cmd.CollectedResponses()[0].Verified)
}

func TestCreateEvidenceEntity_Run_SignsClientBuiltStatement(t *testing.T) {
	dir := t.TempDir()
	predicatePath := filepath.Join(dir, "predicate.json")
	require.NoError(t, os.WriteFile(predicatePath, []byte(`{"result":"ok"}`), 0o600))
	keyContent, err := os.ReadFile(filepath.Join("../..", "tests/testdata/ecdsa_key.pem"))
	require.NoError(t, err)

	evidenceServiceClient := &captureEvidenceServiceClient{
		uploadResp: []byte(`{"verified":true,"predicate_slug":"example"}`),
	}
	cmd := &createEvidenceEntity{
		createEvidenceBase: createEvidenceBase{
			serverDetails:         &config.ServerDetails{Url: "https://example.jfrog.io/", User: "alice"},
			predicateFilePath:     predicatePath,
			predicateType:         "https://example.com/v1",
			providerId:            "ci",
			key:                   string(keyContent),
			evidenceServiceClient: evidenceServiceClient,
		},
		EntitySubject: model.EntitySubject{
			EntityType: "gitCommit",
			EntityID:   "abc123",
			ProjectKey: "proj",
		},
	}

	require.NoError(t, cmd.Run())
	assert.Equal(t, client.CreateEntityEvidenceRequest{
		EntityType: "gitCommit",
		EntityID:   "abc123",
		ProjectKey: "proj",
		ProviderID: "ci",
	}, evidenceServiceClient.createReq)

	statement := signedStatementFromUpload(t, evidenceServiceClient.uploadBody)
	assert.Equal(t, "https://in-toto.io/Statement/v1", statement["_type"])
	assert.Equal(t, "https://example.com/v1", statement["predicateType"])
	assert.Equal(t, "alice", statement["createdBy"])
	assert.JSONEq(t, `{"result":"ok"}`, mustJSON(t, statement["predicate"]))
	assert.Equal(t, map[string]any{"gitCommit": "abc123"}, subjectDigest(t, statement))
	_, hasSha256 := subjectDigest(t, statement)["sha256"]
	assert.False(t, hasSha256)
	require.Len(t, cmd.CollectedResponses(), 1)
	assert.True(t, cmd.CollectedResponses()[0].Verified)
}

func TestCreateEvidenceEntity_Run_IncludesAttachmentInSignedStatement(t *testing.T) {
	dir := t.TempDir()
	predicatePath := filepath.Join(dir, "predicate.json")
	require.NoError(t, os.WriteFile(predicatePath, []byte(`{"result":"ok"}`), 0o600))
	keyContent, err := os.ReadFile(filepath.Join("../..", "tests/testdata/ecdsa_key.pem"))
	require.NoError(t, err)

	evidenceServiceClient := &captureEvidenceServiceClient{
		uploadResp: []byte(`{"verified":true,"predicate_slug":"example"}`),
	}
	cmd := &createEvidenceEntity{
		createEvidenceBase: createEvidenceBase{
			serverDetails:         &config.ServerDetails{Url: "https://example.jfrog.io/", User: "alice"},
			predicateFilePath:     predicatePath,
			predicateType:         "https://example.com/v1",
			key:                   string(keyContent),
			attachArtifactoryPath: "reports-local/reports/a.json",
			artifactoryClient: &SimpleMockServicesManager{
				FileInfoFunc: func(string) (*utils.FileInfo, error) {
					return NewFileInfoBuilder().WithSha256("deadbeef").Build(), nil
				},
			},
			evidenceServiceClient: evidenceServiceClient,
		},
		EntitySubject: model.EntitySubject{
			EntityType: "gitCommit",
			EntityID:   "abc123",
		},
	}

	require.NoError(t, cmd.Run())
	statement := signedStatementFromUpload(t, evidenceServiceClient.uploadBody)
	assert.Equal(t, map[string]any{"gitCommit": "abc123"}, subjectDigest(t, statement))
	attachments, ok := statement["attachments"].([]any)
	require.True(t, ok)
	require.Len(t, attachments, 1)
	assert.Equal(t, "a.json", attachments[0].(map[string]any)["name"])
	assert.Equal(t, "deadbeef", attachments[0].(map[string]any)["sha256"])

	var uploaded map[string]any
	require.NoError(t, json.Unmarshal(evidenceServiceClient.uploadBody, &uploaded))
	assert.JSONEq(t, `[{"repository":"reports-local","path":"reports/a.json","sha256":"deadbeef"}]`, mustJSON(t, uploaded["attachments"]))
}

func TestBuildEntityStatement_ReadsMarkdown(t *testing.T) {
	dir := t.TempDir()
	predicatePath := filepath.Join(dir, "predicate.json")
	markdownPath := filepath.Join(dir, "notes.md")
	require.NoError(t, os.WriteFile(predicatePath, []byte(`{}`), 0o600))
	require.NoError(t, os.WriteFile(markdownPath, []byte("# hi"), 0o600))

	cmd := &createEvidenceEntity{
		createEvidenceBase: createEvidenceBase{
			serverDetails:     &config.ServerDetails{User: "alice"},
			predicateFilePath: predicatePath,
			predicateType:     "https://example.com/v1",
			markdownFilePath:  markdownPath,
		},
		EntitySubject: model.EntitySubject{
			EntityType: "languageModel",
			EntityID:   "model-1",
			EntityRepo: "models-entity",
		},
	}
	statementBytes, err := cmd.buildEntityStatement(nil)
	require.NoError(t, err)

	var statement map[string]any
	require.NoError(t, json.Unmarshal(statementBytes, &statement))
	assert.Equal(t, "# hi", statement["markdown"])
	assert.Equal(t, map[string]any{"languageModel": "model-1"}, subjectDigest(t, statement))
	assert.Equal(t, "alice", statement["createdBy"])
}

func TestBuildEntityStatement_EmbedsMarkdownImages(t *testing.T) {
	dir := t.TempDir()
	predicatePath := filepath.Join(dir, "predicate.json")
	markdownPath := filepath.Join(dir, "notes.md")
	imagePath := filepath.Join(dir, "image.svg")
	require.NoError(t, os.WriteFile(predicatePath, []byte(`{}`), 0o600))
	require.NoError(t, os.WriteFile(markdownPath, []byte("![image](./image.svg)"), 0o600))
	require.NoError(t, os.WriteFile(imagePath, []byte(`<svg width="10" height="10" xmlns="http://www.w3.org/2000/svg"><rect width="10" height="10" fill="red"/></svg>`), 0o600))

	cmd := &createEvidenceEntity{
		createEvidenceBase: createEvidenceBase{
			serverDetails:     &config.ServerDetails{User: "alice"},
			predicateFilePath: predicatePath,
			predicateType:     "https://example.com/v1",
			markdownFilePath:  markdownPath,
		},
		EntitySubject: model.EntitySubject{
			EntityType: "languageModel",
			EntityID:   "model-1",
		},
	}
	statementBytes, err := cmd.buildEntityStatement(nil)
	require.NoError(t, err)

	var statement map[string]any
	require.NoError(t, json.Unmarshal(statementBytes, &statement))
	assert.Equal(t, "![image](data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMTAiIGhlaWdodD0iMTAiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyI+PHJlY3Qgd2lkdGg9IjEwIiBoZWlnaHQ9IjEwIiBmaWxsPSJyZWQiLz48L3N2Zz4=)", statement["markdown"])
}

func signedStatementFromUpload(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var envelope dsse.Envelope
	require.NoError(t, json.Unmarshal(body, &envelope))
	assert.Equal(t, "application/vnd.in-toto+json", envelope.PayloadType)
	require.NotEmpty(t, envelope.Signatures)
	payload, err := base64.StdEncoding.DecodeString(envelope.Payload)
	require.NoError(t, err)
	var statement map[string]any
	require.NoError(t, json.Unmarshal(payload, &statement))
	return statement
}

func subjectDigest(t *testing.T, statement map[string]any) map[string]any {
	t.Helper()
	subjects, ok := statement["subject"].([]any)
	require.True(t, ok)
	require.Len(t, subjects, 1)
	digest, ok := subjects[0].(map[string]any)["digest"].(map[string]any)
	require.True(t, ok)
	return digest
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	require.NoError(t, err)
	return string(raw)
}
