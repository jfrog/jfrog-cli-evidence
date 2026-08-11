package client

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	evidenceutils "github.com/jfrog/jfrog-cli-evidence/evidence/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetVersion(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/evidence/api/v1/system/version", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(" 7.1269.0 \n"))
	}))
	defer testServer.Close()

	client := newTestEvidenceClient(t, testServer.URL+"/evidence/")
	version, err := client.GetVersion()
	require.NoError(t, err)
	assert.Equal(t, "7.1269.0", version)
}

func TestEnsureFeatureSupported(t *testing.T) {
	tests := []struct {
		name          string
		version       string
		status        int
		errorContains string
	}{
		{name: "supported release", version: "7.1269.0", status: http.StatusOK},
		{name: "newer release", version: "7.1300.0", status: http.StatusOK},
		{name: "snapshot skipped", version: "7.x-SNAPSHOT-master-1", status: http.StatusOK},
		{name: "development skipped", version: "development", status: http.StatusOK},
		{name: "too old", version: "7.1268.0", status: http.StatusOK, errorContains: "7.1269.0"},
		{name: "version endpoint missing", status: http.StatusNotFound, errorContains: "failed to determine Evidence version"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/evidence/api/v1/system/version", r.URL.Path)
				w.WriteHeader(tt.status)
				if tt.status == http.StatusOK {
					_, _ = w.Write([]byte(tt.version))
				} else {
					_, _ = w.Write([]byte("not found"))
				}
			}))
			defer testServer.Close()

			serverDetails := &config.ServerDetails{
				Url:         testServer.URL + "/",
				EvidenceUrl: testServer.URL + "/evidence/",
				AccessToken: "token",
			}
			err := EnsureFeatureSupported(serverDetails, evidenceutils.FeatureEntityAPI)
			if tt.errorContains == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			}
		})
	}
}

func TestEnsureFeatureSupported_UnknownFeature(t *testing.T) {
	err := EnsureFeatureSupported(&config.ServerDetails{}, evidenceutils.EvidenceFeature("unknown"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown Evidence feature")
}
