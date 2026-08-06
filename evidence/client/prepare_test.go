package client

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	artifactoryAuth "github.com/jfrog/jfrog-client-go/artifactory/auth"
	"github.com/jfrog/jfrog-client-go/http/jfroghttpclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestEvidenceClient(t *testing.T, baseURL string) *EvidenceClient {
	t.Helper()
	details := artifactoryAuth.NewArtifactoryDetails()
	details.SetUrl(baseURL)
	client, err := jfroghttpclient.JfrogClientBuilder().Build()
	require.NoError(t, err)
	return &EvidenceClient{httpClient: client, details: details}
}

func TestPrepareEvidence(t *testing.T) {
	tests := []struct {
		name        string
		includePAE  bool
		expectedPAE string
	}{
		{name: "without PAE"},
		{name: "with PAE", includePAE: true, expectedPAE: "DSSEv1 28 application/vnd.in-toto+json 7 payload"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/evidence/api/v1/evidence/prepare", r.URL.Path)
				assert.Equal(t, test.includePAE, r.URL.Query().Get("include_pae") == "true")

				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				assert.JSONEq(t, `{
					"predicate": {"result": "passed"},
					"predicate_type": "https://example.com/predicate/v1",
					"provider_id": "ci",
					"subject": {
						"subject_type": "entity",
						"entity_type": "gitCommit",
						"entity_id": "abc123"
					},
					"project_key": "proj",
					"attachments": [{
						"repository": "reports-local",
						"path": "reports/result.json"
					}]
				}`, string(body))

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				response := map[string]any{
					"post_url":          "/evidence/api/v1/entity/gitCommit/abc123?project=proj",
					"dsse_payload":      "cGF5bG9hZA==",
					"dsse_payload_type": "application/vnd.in-toto+json",
					"attachments": []map[string]string{{
						"repository": "reports-local",
						"path":       "reports/result.json",
						"sha256":     "abc",
					}},
				}
				if test.expectedPAE != "" {
					response["pre_authentication_encoding"] = test.expectedPAE
				}
				require.NoError(t, json.NewEncoder(w).Encode(response))
			}))
			defer testServer.Close()

			client := newTestEvidenceClient(t, testServer.URL+"/evidence/")
			response, err := client.PrepareEvidence(PrepareEvidenceRequest{
				Predicate:     json.RawMessage(`{"result":"passed"}`),
				PredicateType: "https://example.com/predicate/v1",
				ProviderID:    "ci",
				Subject: PrepareEvidenceSubject{
					SubjectType: SubjectTypeEntity,
					EntityType:  "gitCommit",
					EntityID:    "abc123",
				},
				ProjectKey: "proj",
				Attachments: []PrepareEvidenceAttachment{{
					Repository: "reports-local",
					Path:       "reports/result.json",
				}},
			}, test.includePAE)
			require.NoError(t, err)
			assert.Equal(t, "/evidence/api/v1/entity/gitCommit/abc123?project=proj", response.PostURL)
			assert.Equal(t, "cGF5bG9hZA==", response.DSSEPayload)
			assert.Equal(t, "application/vnd.in-toto+json", response.DSSEPayloadType)
			assert.Equal(t, test.expectedPAE, response.PreAuthenticationEncoding)
			require.Len(t, response.Attachments, 1)
			assert.Equal(t, "abc", response.Attachments[0].SHA256)
		})
	}
}

func TestUploadPreparedSignedEvidence(t *testing.T) {
	signedEnvelope := []byte(`{"payload":"cGF5bG9hZA==","payloadType":"application/vnd.in-toto+json","signatures":[{"keyid":"key","sig":"signature"}]}`)

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/evidence/api/v1/entity/gitCommit/abc123", r.URL.Path)
		assert.Equal(t, "proj", r.URL.Query().Get("project"))
		assert.Equal(t, "ci provider", r.URL.Query().Get("providerId"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, signedEnvelope, body)

		w.WriteHeader(http.StatusCreated)
		_, err = w.Write([]byte(`{"verified":true}`))
		require.NoError(t, err)
	}))
	defer testServer.Close()

	client := newTestEvidenceClient(t, testServer.URL+"/evidence/")
	body, err := client.UploadPreparedSignedEvidence(
		"/evidence/api/v1/entity/gitCommit/abc123?project=proj&providerId=ci+provider",
		signedEnvelope,
	)
	require.NoError(t, err)
	assert.JSONEq(t, `{"verified":true}`, string(body))
}

func TestUploadPreparedSignedEvidence_RejectsNonRootRelativeURL(t *testing.T) {
	client := newTestEvidenceClient(t, "https://example.jfrog.io/evidence/")
	for _, postURL := range []string{
		"",
		"api/v1/entity/gitCommit/abc123",
		"https://other.example/evidence/api/v1/entity/gitCommit/abc123",
	} {
		t.Run(postURL, func(t *testing.T) {
			_, err := client.UploadPreparedSignedEvidence(postURL, []byte(`{}`))
			require.Error(t, err)
			assert.Contains(t, err.Error(), "root-relative")
		})
	}
}
