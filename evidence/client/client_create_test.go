package client

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestBuildEntityEvidenceURL(t *testing.T) {
	client := newTestEvidenceClient(t, "https://example.jfrog.io/evidence/")
	tests := []struct {
		name    string
		request CreateEntityEvidenceRequest
		want    string
	}{
		{
			name: "project scope",
			request: CreateEntityEvidenceRequest{
				EntityType: "gitCommit",
				EntityID:   "abc123",
				ProjectKey: "proj",
				ProviderID: "ci",
			},
			want: "https://example.jfrog.io/evidence/api/v1/entity/gitCommit/abc123?project=proj&providerId=ci",
		},
		{
			name: "entity repo scope",
			request: CreateEntityEvidenceRequest{
				EntityType: "languageModel",
				EntityID:   "model-1",
				EntityRepo: "models-entity",
			},
			want: "https://example.jfrog.io/evidence/api/v1/entity/languageModel/model-1?repo=models-entity",
		},
		{
			name: "application scope",
			request: CreateEntityEvidenceRequest{
				EntityType:     "gitCommit",
				EntityID:       "abc123",
				ApplicationKey: "my-app",
			},
			want: "https://example.jfrog.io/evidence/api/v1/entity/gitCommit/abc123?application=my-app",
		},
		{
			name: "no scope",
			request: CreateEntityEvidenceRequest{
				EntityType: "gitCommit",
				EntityID:   "abc123",
			},
			want: "https://example.jfrog.io/evidence/api/v1/entity/gitCommit/abc123",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := client.buildEntityEvidenceURL(tt.request)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCreateEntityEvidence(t *testing.T) {
	payload := []byte(`{"mediaType":"application/vnd.dev.sigstore.bundle+json;version=0.2"}`)
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/evidence/api/v1/entity/gitCommit/abc123", r.URL.Path)
		assert.Equal(t, "proj", r.URL.Query().Get("project"))
		assert.Equal(t, "ci", r.URL.Query().Get("providerId"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, payload, body)

		w.WriteHeader(http.StatusCreated)
		_, err = w.Write([]byte(`{"verified":true}`))
		require.NoError(t, err)
	}))
	defer testServer.Close()

	client := newTestEvidenceClient(t, testServer.URL+"/evidence/")
	body, err := client.CreateEntityEvidence(CreateEntityEvidenceRequest{
		EntityType: "gitCommit",
		EntityID:   "abc123",
		ProjectKey: "proj",
		ProviderID: "ci",
	}, payload)
	require.NoError(t, err)
	assert.JSONEq(t, `{"verified":true}`, string(body))
}
