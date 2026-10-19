package resolvers

import (
	"net/url"
	"testing"
	"time"

	"github.com/jfrog/jfrog-client-go/artifactory"
	artauth "github.com/jfrog/jfrog-client-go/artifactory/auth"
	"github.com/jfrog/jfrog-client-go/auth"
	"github.com/jfrog/jfrog-client-go/config"
	"github.com/jfrog/jfrog-client-go/http/jfroghttpclient"
	"github.com/jfrog/jfrog-client-go/utils/io/httputils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildUrl(t *testing.T) {
	platform := &mockServiceDetails{url: "https://test.rt.io/artifactory"}

	testCases := []struct {
		name     string
		details  auth.ServiceDetails
		subject  string
		checksum string
		expected string
	}{
		{
			name:     "Same host as the configured platform",
			details:  platform,
			subject:  "test.rt.io/local-test-repo/local-image-a:v2.0",
			checksum: "sha256:1234567890abcdef",
			expected: "https://test.rt.io/v2/local-test-repo/local-image-a/manifests/sha256:1234567890abcdef",
		},
		{
			name:     "Docker subdomain of the configured platform",
			details:  platform,
			subject:  "docker-local.test.rt.io/local-image-a:v2.0",
			checksum: "sha256:1234567890abcdef",
			expected: "https://docker-local.test.rt.io/v2/local-image-a/manifests/sha256:1234567890abcdef",
		},
		{
			name:     "Explicit port on the platform host is preserved",
			details:  platform,
			subject:  "test.rt.io:8443/repo/image:v1",
			checksum: "sha256:1234567890abcdef",
			expected: "https://test.rt.io:8443/v2/repo/image/manifests/sha256:1234567890abcdef",
		},
		{
			name:     "Empty domain uses the configured platform host",
			details:  platform,
			subject:  "my-image:latest",
			checksum: "sha256:1234567890abcdef",
			expected: "https://test.rt.io/v2/my-image/manifests/sha256:1234567890abcdef",
		},
		{
			name:     "Empty domain keeps the configured port",
			details:  &mockServiceDetails{url: "http://localhost:8082/artifactory"},
			subject:  "my-image:latest",
			checksum: "sha256:1234567890abcdef",
			expected: "http://localhost:8082/v2/my-image/manifests/sha256:1234567890abcdef",
		},
		{
			name:     "HTTP platform URL keeps its scheme",
			details:  &mockServiceDetails{url: "http://test.rt.io/artifactory"},
			subject:  "test.rt.io/repo/image:v1",
			checksum: "sha256:1234567890abcdef",
			expected: "http://test.rt.io/v2/repo/image/manifests/sha256:1234567890abcdef",
		},
		{
			name:     "Platform host comparison is case-insensitive",
			details:  &mockServiceDetails{url: "https://Test.RT.io/artifactory"},
			subject:  "test.rt.io/repo/image:v1",
			checksum: "sha256:1234567890abcdef",
			expected: "https://test.rt.io/v2/repo/image/manifests/sha256:1234567890abcdef",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			domain, path, err := parseOciSubject(tc.subject)
			require.NoError(t, err)
			result, err := buildContainerUrl(domain, path, tc.checksum, tc.details)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)

			parsed, err := url.Parse(result)
			require.NoError(t, err)
			platformURL, err := url.Parse(tc.details.GetUrl())
			require.NoError(t, err)
			assert.True(t, registryHostAllowed(canonicalHostname(parsed.Hostname()), canonicalHostname(platformURL.Hostname())))
		})
	}
}

func TestBuildUrlRejectsUntrustedRegistry(t *testing.T) {
	platform := &mockServiceDetails{url: "https://test.rt.io/artifactory"}

	testCases := []struct {
		name    string
		subject string
	}{
		{name: "Foreign registry", subject: "evil.example/anything:latest"},
		{name: "Docker Hub host", subject: "docker.io/library/nginx:latest"},
		{name: "Suffix of the platform host", subject: "test.rt.io.evil.example/x:v1"},
		{name: "Host that only shares a suffix label", subject: "nottest.rt.io/x:v1"},
		{name: "Different JFrog host", subject: "docker.jfrog.dev/sigs:07bfa335a98e6df33f4c5f933553cb4d2bc7c00211e6dc52341289645496dd9d"},
		{name: "Name parsed as a registry without a platform domain", subject: "local-test-registry/local-image-a:v2.0"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			domain, path, err := parseOciSubject(tc.subject)
			require.NoError(t, err)
			result, err := buildContainerUrl(domain, path, "sha256:1234567890abcdef", platform)
			require.Error(t, err)
			assert.Empty(t, result)
			assert.ErrorIs(t, err, errRegistryHostNotAllowed)
		})
	}
}

func TestBuildUrlRejectsMalformedRegistryHost(t *testing.T) {
	platform := &mockServiceDetails{url: "https://test.rt.io/artifactory"}
	hosts := []string{
		"user:pass@evil.example",
		"evil.example#@test.rt.io",
		"test.rt.io/evil.example",
		"evil.example?x=1",
	}
	for _, host := range hosts {
		t.Run(host, func(t *testing.T) {
			result, err := buildContainerUrl(host, "repo/image", "sha256:1234567890abcdef", platform)
			require.Error(t, err)
			assert.Empty(t, result)
		})
	}
}

func TestResolveDoesNotContactForeignRegistry(t *testing.T) {
	details := artauth.NewArtifactoryDetails()
	details.SetUrl("https://test.rt.io/artifactory")
	details.SetAccessToken("platform-token")
	cfg, err := config.NewConfigBuilder().SetServiceDetails(details).Build()
	require.NoError(t, err)

	resolver := NewOciSubjectResolver("evil.example/anything:latest", &hostGuardManager{cfg: cfg})
	_, err = resolver.Resolve("sha256:1234567890abcdef")
	require.Error(t, err)
	assert.ErrorIs(t, err, errRegistryHostNotAllowed)
	assert.Contains(t, err.Error(), "evil.example")
}

type hostGuardManager struct {
	artifactory.EmptyArtifactoryServicesManager
	cfg config.Config
}

func (m *hostGuardManager) GetConfig() config.Config { return m.cfg }

func (m *hostGuardManager) Client() *jfroghttpclient.JfrogHttpClient {
	panic("HTTP client must not be called for a rejected registry host")
}

type mockServiceDetails struct {
	url string
}

func (m *mockServiceDetails) GetUrl() string {
	return m.url
}

func (m *mockServiceDetails) GetUser() string                                              { return "" }
func (m *mockServiceDetails) GetPassword() string                                          { return "" }
func (m *mockServiceDetails) GetApiKey() string                                            { return "" }
func (m *mockServiceDetails) GetAccessToken() string                                       { return "" }
func (m *mockServiceDetails) GetPreRequestFunctions() []auth.ServiceDetailsPreRequestFunc  { return nil }
func (m *mockServiceDetails) GetClientCertPath() string                                    { return "" }
func (m *mockServiceDetails) GetClientCertKeyPath() string                                 { return "" }
func (m *mockServiceDetails) GetSshUrl() string                                            { return "" }
func (m *mockServiceDetails) GetSshKeyPath() string                                        { return "" }
func (m *mockServiceDetails) GetSshPassphrase() string                                     { return "" }
func (m *mockServiceDetails) GetSshAuthHeaders() map[string]string                         { return nil }
func (m *mockServiceDetails) GetClient() *jfroghttpclient.JfrogHttpClient                  { return nil }
func (m *mockServiceDetails) GetVersion() (string, error)                                  { return "", nil }
func (m *mockServiceDetails) SetUrl(url string)                                            { m.url = url }
func (m *mockServiceDetails) SetUser(user string)                                          {}
func (m *mockServiceDetails) SetPassword(password string)                                  {}
func (m *mockServiceDetails) SetApiKey(apiKey string)                                      {}
func (m *mockServiceDetails) SetAccessToken(accessToken string)                            {}
func (m *mockServiceDetails) AppendPreRequestFunction(auth.ServiceDetailsPreRequestFunc)   {}
func (m *mockServiceDetails) SetClientCertPath(certificatePath string)                     {}
func (m *mockServiceDetails) SetClientCertKeyPath(certificatePath string)                  {}
func (m *mockServiceDetails) SetSshUrl(url string)                                         {}
func (m *mockServiceDetails) SetSshKeyPath(sshKeyPath string)                              {}
func (m *mockServiceDetails) SetSshPassphrase(sshPassphrase string)                        {}
func (m *mockServiceDetails) SetSshAuthHeaders(sshAuthHeaders map[string]string)           {}
func (m *mockServiceDetails) SetClient(client *jfroghttpclient.JfrogHttpClient)            {}
func (m *mockServiceDetails) SetDialTimeout(dialTimeout time.Duration)                     {}
func (m *mockServiceDetails) SetOverallRequestTimeout(overallRequestTimeout time.Duration) {}
func (m *mockServiceDetails) IsSshAuthHeaderSet() bool                                     { return false }
func (m *mockServiceDetails) IsSshAuthentication() bool                                    { return false }
func (m *mockServiceDetails) AuthenticateSsh(sshKey, sshPassphrase string) error           { return nil }
func (m *mockServiceDetails) InitSsh() error                                               { return nil }
func (m *mockServiceDetails) RunPreRequestFunctions(httpClientDetails *httputils.HttpClientDetails) error {
	return nil
}
func (m *mockServiceDetails) CreateHttpClientDetails() httputils.HttpClientDetails {
	return httputils.HttpClientDetails{}
}
