package create

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli-evidence/evidence/client"
	"github.com/jfrog/jfrog-cli-evidence/evidence/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type captureEvidenceServiceClient struct {
	prepareReq  client.PrepareEvidenceRequest
	prepareResp *client.PrepareEvidenceResponse
	prepareErr  error
	createReq   client.CreateEntityEvidenceRequest
	uploadURL   string
	uploadBody  []byte
	uploadResp  []byte
	uploadErr   error
}

func (c *captureEvidenceServiceClient) PrepareEvidence(request client.PrepareEvidenceRequest, _ bool) (*client.PrepareEvidenceResponse, error) {
	c.prepareReq = request
	if c.prepareErr != nil {
		return nil, c.prepareErr
	}
	return c.prepareResp, nil
}

func (c *captureEvidenceServiceClient) UploadPreparedSignedEvidence(postURL string, signedEnvelope []byte) ([]byte, error) {
	c.uploadURL = postURL
	c.uploadBody = signedEnvelope
	if c.uploadErr != nil {
		return nil, c.uploadErr
	}
	return c.uploadResp, nil
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

func TestCreateEvidenceEntity_Run_PrepareSignUpload(t *testing.T) {
	dir := t.TempDir()
	predicatePath := filepath.Join(dir, "predicate.json")
	require.NoError(t, os.WriteFile(predicatePath, []byte(`{"result":"ok"}`), 0o600))

	evidenceServiceClient := &captureEvidenceServiceClient{
		prepareResp: &client.PrepareEvidenceResponse{
			PostURL:         "/evidence/api/v1/entity/gitCommit/abc123?project=proj",
			DSSEPayload:     base64.StdEncoding.EncodeToString([]byte(`{"_type":"https://in-toto.io/Statement/v1","predicateType":"https://example.com/v1","predicate":{"result":"ok"},"subject":[{"digest":{"gitCommit":"abc123"}}]}`)),
			DSSEPayloadType: "application/vnd.in-toto+json",
			Attachments: []client.PrepareEvidenceAttachment{{
				Repository: "reports-local",
				Path:       "reports/a.txt",
				SHA256:     "deadbeef",
			}},
		},
		uploadResp: []byte(`{"verified":true,"predicate_slug":"example"}`),
	}

	cmd := &createEvidenceEntity{
		createEvidenceBase: createEvidenceBase{
			serverDetails:         &config.ServerDetails{Url: "https://example.jfrog.io/"},
			predicateFilePath:     predicatePath,
			predicateType:         "https://example.com/v1",
			providerId:            "ci",
			key:                   "not-a-real-key",
			evidenceServiceClient: evidenceServiceClient,
		},
		EntitySubject: model.EntitySubject{
			EntityType: "gitCommit",
			EntityID:   "abc123",
			ProjectKey: "proj",
		},
	}

	err := cmd.Run()
	require.Error(t, err) // signing key invalid
	assert.Equal(t, "gitCommit", evidenceServiceClient.prepareReq.Subject.EntityType)
	assert.Equal(t, "abc123", evidenceServiceClient.prepareReq.Subject.EntityID)
	assert.Equal(t, client.SubjectTypeEntity, evidenceServiceClient.prepareReq.Subject.SubjectType)
	assert.Equal(t, "proj", evidenceServiceClient.prepareReq.ProjectKey)
	assert.Equal(t, "ci", evidenceServiceClient.prepareReq.ProviderID)
	assert.JSONEq(t, `{"result":"ok"}`, string(evidenceServiceClient.prepareReq.Predicate))
}

func TestWrapEnvelopeWithAttachmentRefs(t *testing.T) {
	envelope := []byte(`{"payload":"abc","payloadType":"application/vnd.in-toto+json","signatures":[]}`)
	wrapped, err := wrapEnvelopeWithAttachmentRefs(envelope, toAttachmentRefs([]client.PrepareEvidenceAttachment{{
		Repository: "repo",
		Path:       "path/file.txt",
		SHA256:     "abc123",
	}}))
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"payload":"abc",
		"payloadType":"application/vnd.in-toto+json",
		"signatures":[],
		"attachments":[{"repository":"repo","path":"path/file.txt","sha256":"abc123"}]
	}`, string(wrapped))
}

func TestUploadPreparedEvidence_CollectsResponse(t *testing.T) {
	evidenceServiceClient := &captureEvidenceServiceClient{
		uploadResp: []byte(`{"verified":false,"predicate_slug":"slug"}`),
	}
	cmd := &createEvidenceEntity{
		createEvidenceBase: createEvidenceBase{evidenceServiceClient: evidenceServiceClient},
	}
	resp, err := cmd.uploadPreparedEvidence("/evidence/api/v1/entity/gitCommit/abc", []byte(`{"payload":"x"}`))
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.False(t, resp.Verified)
	assert.Equal(t, "slug", resp.PredicateSlug)
	require.Len(t, cmd.CollectedResponses(), 1)
	assert.Equal(t, &model.CreateResponse{Verified: false, PredicateSlug: "slug"}, cmd.CollectedResponses()[0])
	assert.Equal(t, "/evidence/api/v1/entity/gitCommit/abc", evidenceServiceClient.uploadURL)
}

func TestPrepareSignedStatement_ReadsMarkdown(t *testing.T) {
	dir := t.TempDir()
	predicatePath := filepath.Join(dir, "predicate.json")
	markdownPath := filepath.Join(dir, "notes.md")
	require.NoError(t, os.WriteFile(predicatePath, []byte(`{}`), 0o600))
	require.NoError(t, os.WriteFile(markdownPath, []byte("# hi"), 0o600))

	evidenceServiceClient := &captureEvidenceServiceClient{
		prepareResp: &client.PrepareEvidenceResponse{DSSEPayload: base64.StdEncoding.EncodeToString([]byte(`{}`))},
	}
	cmd := &createEvidenceEntity{
		createEvidenceBase: createEvidenceBase{
			predicateFilePath:     predicatePath,
			predicateType:         "https://example.com/v1",
			markdownFilePath:      markdownPath,
			evidenceServiceClient: evidenceServiceClient,
		},
		EntitySubject: model.EntitySubject{
			EntityType: "languageModel",
			EntityID:   "model-1",
			EntityRepo: "models-entity",
		},
	}
	_, err := cmd.buildEvidenceStatement(nil)
	require.NoError(t, err)
	assert.Equal(t, "# hi", evidenceServiceClient.prepareReq.Markdown)
	assert.Equal(t, "models-entity", evidenceServiceClient.prepareReq.EntityRepo)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(evidenceServiceClient.prepareReq.Predicate, &decoded))
}
