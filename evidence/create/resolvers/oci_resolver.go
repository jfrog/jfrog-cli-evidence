package resolvers

import (
	"errors"
	"fmt"
	"html"
	"net/url"
	"strings"

	"github.com/distribution/reference"
	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/auth"
)

// OciSubjectResolver handles resolution of Docker and OCI container image subjects
// Both Docker and OCI use the same container image format and registry protocols
type OciSubjectResolver struct {
	aqlResolver AqlResolver
	subject     string
	client      artifactory.ArtifactoryServicesManager
}

const registryHeader = "X-Artifactory-Docker-Registry"
const shaPrefix = "sha256:"

// errRegistryHostNotAllowed is returned when a container subject names a registry
// that is not the configured platform host. The manifest request carries platform
// credentials, and the subject can come from an untrusted sigstore bundle.
var errRegistryHostNotAllowed = errors.New("registry host is not the configured Artifactory host or a subdomain of it")

// NewOciSubjectResolver creates a new OciSubjectResolver instance
func NewOciSubjectResolver(subject string, client artifactory.ArtifactoryServicesManager) *OciSubjectResolver {
	return &OciSubjectResolver{
		subject:     subject,
		aqlResolver: NewAqlSubjectResolver(client),
		client:      client,
	}
}

// Resolve resolves a container image subject to repository paths using the checksum
func (d *OciSubjectResolver) Resolve(checksum string) ([]string, error) {
	if d.subject == "" {
		return nil, fmt.Errorf("subject cannot be empty")
	}
	if checksum == "" {
		return nil, fmt.Errorf("checksum cannot be empty")
	}
	if d.aqlResolver == nil || d.client == nil {
		return nil, fmt.Errorf("artifactory client is not properly initialized")
	}

	domain, path, err := parseOciSubject(d.subject)
	if err != nil {
		return nil, fmt.Errorf("failed to parse container subject: %w", err)
	}

	serviceDetails := d.client.GetConfig().GetServiceDetails()

	containerRegistryUrl, err := buildContainerUrl(domain, path, checksum, serviceDetails)
	if err != nil {
		return nil, fmt.Errorf("failed to build container registry URL from subject: %s, %w", d.subject, err)
	}

	details := serviceDetails.CreateHttpClientDetails()
	resp, _, err := d.client.Client().SendHead(containerRegistryUrl, &details)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve container repository using URL: %s, %w", containerRegistryUrl, err)
	}

	repo := ""
	if values, ok := resp.Header[registryHeader]; ok && len(values) > 0 {
		repo = values[0]
	}

	if repo == "" {
		return nil, fmt.Errorf("no repository found in response headers for subject: %s", d.subject)
	}

	return d.aqlResolver.Resolve(repo, path, checksum)
}

// buildContainerUrl constructs an OCI registry v2 manifest URL for the subject.
// The registry host is taken from the subject only when it is the configured
// platform host or a Docker subdomain of it. Otherwise the function fails
// before any request is made, so platform credentials are not sent elsewhere.
func buildContainerUrl(domain, path, checksum string, serviceDetails auth.ServiceDetails) (string, error) {
	repositoryPath, err := validateRepositoryPath(path)
	if err != nil {
		return "", err
	}
	normalized, err := normalizeChecksum(checksum)
	if err != nil {
		return "", err
	}

	artifactoryURL := serviceDetails.GetUrl()
	if artifactoryURL == "" {
		return "", fmt.Errorf("artifactory URL is not configured")
	}

	parsedURL, err := url.Parse(artifactoryURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse artifactory URL '%s': %w", artifactoryURL, err)
	}

	protocol := parsedURL.Scheme
	if protocol == "" {
		protocol = "https"
	}
	if protocol != "https" && protocol != "http" {
		return "", fmt.Errorf("artifactory URL scheme %q is not supported", protocol)
	}

	platformHost := canonicalHostname(parsedURL.Hostname())
	if platformHost == "" {
		return "", fmt.Errorf("cannot determine registry hostname from artifactory URL")
	}

	registry := domain
	if registry == "" {
		// Keep the configured port. Hostname() alone would drop it and could
		// send credentials to a different local listener.
		registry = parsedURL.Host
	}

	registryHost, err := parseRegistryHost(registry)
	if err != nil {
		return "", err
	}
	if !registryHostAllowed(registryHost.hostname, platformHost) {
		return "", fmt.Errorf("%w: registry %q, platform %q", errRegistryHostNotAllowed, registryHost.hostname, platformHost)
	}

	manifestURL := &url.URL{
		Scheme: protocol,
		Host:   registryHost.host,
		// EscapeString is a no-op for the allowlisted characters, and it keeps the
		// subject path and digest from being treated as raw markup in the URL.
		Path: html.EscapeString("/v2/" + repositoryPath + "/manifests/" + shaPrefix + normalized),
	}
	return manifestURL.String(), nil
}

type parsedRegistryHost struct {
	host     string
	hostname string
}

func parseRegistryHost(registry string) (parsedRegistryHost, error) {
	if registry == "" || strings.Contains(registry, "@") || strings.Contains(registry, "\\") || strings.Contains(registry, "/") {
		return parsedRegistryHost{}, fmt.Errorf("invalid registry host %q", registry)
	}

	parsed, err := url.Parse("https://" + registry)
	if err != nil {
		return parsedRegistryHost{}, fmt.Errorf("invalid registry host %q: %w", registry, err)
	}
	if parsed.User != nil || parsed.Hostname() == "" || (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return parsedRegistryHost{}, fmt.Errorf("invalid registry host %q", registry)
	}

	hostname := canonicalHostname(parsed.Hostname())
	if hostname == "" || strings.Contains(hostname, "..") {
		return parsedRegistryHost{}, fmt.Errorf("invalid registry host %q", registry)
	}
	return parsedRegistryHost{host: parsed.Host, hostname: hostname}, nil
}

func registryHostAllowed(registryHost, platformHost string) bool {
	if registryHost == "" || platformHost == "" {
		return false
	}
	if registryHost == platformHost {
		return true
	}
	return strings.HasSuffix(registryHost, "."+platformHost)
}

func canonicalHostname(host string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
}

func validateRepositoryPath(path string) (string, error) {
	if path == "" || strings.HasPrefix(path, "/") || strings.HasSuffix(path, "/") ||
		strings.Contains(path, "..") || strings.Contains(path, "//") {
		return "", fmt.Errorf("invalid repository path")
	}
	var sanitized strings.Builder
	sanitized.Grow(len(path))
	for _, r := range path {
		if !allowedRepositoryPathRune(r) {
			return "", fmt.Errorf("invalid repository path")
		}
		sanitized.WriteRune(r)
	}
	return sanitized.String(), nil
}

func allowedRepositoryPathRune(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' || r == '/'
}

func normalizeChecksum(checksum string) (string, error) {
	raw := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(checksum)), shaPrefix)
	if raw == "" {
		return "", fmt.Errorf("checksum cannot be empty")
	}
	var sanitized strings.Builder
	sanitized.Grow(len(raw))
	for _, r := range raw {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return "", fmt.Errorf("checksum must be a sha256 hex digest")
		}
		sanitized.WriteRune(r)
	}
	return sanitized.String(), nil
}

// parseOciSubject extracts registry domain and repository path from a OCI subject
func parseOciSubject(subject string) (string, string, error) {
	if subject == "" {
		return "", "", fmt.Errorf("subject cannot be empty")
	}

	ref, err := reference.Parse(subject)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse container reference '%s': %w", subject, err)
	}

	named, ok := ref.(reference.Named)
	if !ok {
		return "", "", fmt.Errorf("failed to parse container reference '%s': not a named reference", subject)
	}

	domain := reference.Domain(named)
	path := reference.Path(named)
	return domain, path, nil
}
