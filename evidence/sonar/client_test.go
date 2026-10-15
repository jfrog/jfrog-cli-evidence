package sonar

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetSonarIntotoStatement_LocalhostUsesServerFormat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/dop-translation/jfrog-evidence/task-123", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"_type":"https://in-toto.io/Statement/v1","predicateType":"https://sonar.com/evidence/sonarqube/v1"}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	assert.NoError(t, err)

	result, err := client.GetSonarIntotoStatement("task-123")
	assert.NoError(t, err)
	assert.Contains(t, string(result), "in-toto.io/Statement/v1")
}

func TestGetSonarIntotoStatement_SelfHostedUsesOnlyConfiguredHost(t *testing.T) {
	var requestedHosts []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedHosts = append(requestedHosts, r.Host)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"_type":"https://in-toto.io/Statement/v1"}`))
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	assert.NoError(t, err)
	configuredURL := "http://localhost:" + serverURL.Port()
	configuredParsedURL, err := url.Parse(configuredURL)
	assert.NoError(t, err)

	client, err := NewClient(configuredURL, "test-token")
	assert.NoError(t, err)

	_, err = client.GetSonarIntotoStatement("task-123")
	assert.NoError(t, err)
	assert.Equal(t, []string{configuredParsedURL.Host}, requestedHosts)
}

func TestPrepareCloudFormatURL_UsesOnlyKnownSonarCloudHosts(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		expected string
	}{
		{
			name:     "EU region",
			baseURL:  "https://sonarcloud.io",
			expected: "https://api.sonarcloud.io/dop-translation/jfrog-evidence/task-123",
		},
		{
			name:     "US region",
			baseURL:  "https://sonarqube.us",
			expected: "https://api.sonarqube.us/dop-translation/jfrog-evidence/task-123",
		},
		{
			name:     "self-hosted",
			baseURL:  "https://sonar.company.example",
			expected: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &httpClient{baseURL: test.baseURL}
			assert.Equal(t, test.expected, client.prepareCloudFormatUrl("task-123"))
		})
	}
}

func TestGetSonarIntotoStatement_ServerFormat404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/dop-translation/jfrog-evidence/task-123", r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	assert.NoError(t, err)

	_, err = client.GetSonarIntotoStatement("task-123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 404")
}

func TestGetSonarIntotoStatement_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"server error"}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	assert.NoError(t, err)

	_, err = client.GetSonarIntotoStatement("task-123")
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "status 500"))
}

func TestGetSonarIntotoStatement_EmptyTaskID(t *testing.T) {
	client, err := NewClient("https://test.example.com", "test-token")
	assert.NoError(t, err)

	result, err := client.GetSonarIntotoStatement("")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "missing ce task id")
}

func TestGetSonarIntotoStatement_AuthorizationHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		assert.Equal(t, "Bearer test-token", authHeader, "Should send Bearer token")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"_type":"https://in-toto.io/Statement/v1"}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	assert.NoError(t, err)

	result, err := client.GetSonarIntotoStatement("task-123")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTaskDetails_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/ce/task", r.URL.Path)
		assert.Equal(t, "task-123", r.URL.Query().Get("id"))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"task":{"id":"task-123","status":"SUCCESS"}}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	assert.NoError(t, err)

	details, err := client.GetTaskDetails("task-123")
	assert.NoError(t, err)
	assert.NotNil(t, details)
	assert.Equal(t, "task-123", details.Task.ID)
	assert.Equal(t, "SUCCESS", details.Task.Status)
}

func TestGetTaskDetails_EmptyTaskID(t *testing.T) {
	client, err := NewClient("https://test.example.com", "test-token")
	assert.NoError(t, err)

	details, err := client.GetTaskDetails("")
	assert.NoError(t, err)
	assert.Nil(t, details)
}

func TestGetTaskDetails_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"server error"}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	assert.NoError(t, err)

	details, err := client.GetTaskDetails("task-123")
	assert.Error(t, err)
	assert.Nil(t, details)
}

func TestGetSonarIntotoStatement_URLEncoding(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "task")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"_type":"https://in-toto.io/Statement/v1"}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	assert.NoError(t, err)

	result, err := client.GetSonarIntotoStatement("task/123+special")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, string(result), "in-toto.io/Statement/v1")
}
