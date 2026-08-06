package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli-core/v2/utils/coreutils"
	"github.com/jfrog/jfrog-client-go/auth"
	clientConfig "github.com/jfrog/jfrog-client-go/config"
	"github.com/jfrog/jfrog-client-go/http/jfroghttpclient"
	clientutils "github.com/jfrog/jfrog-client-go/utils"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"github.com/jfrog/jfrog-client-go/utils/log"
)

const prepareEvidenceAPI = "api/v1/evidence/prepare"

type SubjectType string

const (
	SubjectTypeArtifact           SubjectType = "artifact"
	SubjectTypeBuild              SubjectType = "build"
	SubjectTypePackage            SubjectType = "package"
	SubjectTypeReleaseBundle      SubjectType = "release_bundle"
	SubjectTypeApplicationVersion SubjectType = "application_version"
	SubjectTypeEntity             SubjectType = "entity"
)

// PrepareEvidenceRequest contains the data used by Evidence to generate an in-toto statement for signing.
// For entity subjects, at most one of EntityRepo, ProjectKey, and ApplicationKey may be set.
type PrepareEvidenceRequest struct {
	Predicate      json.RawMessage             `json:"predicate"`
	PredicateType  string                      `json:"predicate_type"`
	Markdown       string                      `json:"markdown,omitempty"`
	ProviderID     string                      `json:"provider_id,omitempty"`
	Subject        PrepareEvidenceSubject      `json:"subject"`
	EntityRepo     string                      `json:"entity_repo,omitempty"`
	ProjectKey     string                      `json:"project_key,omitempty"`
	ApplicationKey string                      `json:"application_key,omitempty"`
	Attachments    []PrepareEvidenceAttachment `json:"attachments,omitempty"`
}

// PrepareEvidenceSubject identifies the subject for which Evidence prepares the statement.
type PrepareEvidenceSubject struct {
	SubjectType   SubjectType `json:"subject_type"`
	SubjectSHA256 string      `json:"subject_sha256,omitempty"`

	RepoPath string `json:"repo_path,omitempty"`

	EntityType string `json:"entity_type,omitempty"`
	EntityID   string `json:"entity_id,omitempty"`

	BuildName      string `json:"build_name,omitempty"`
	BuildNumber    string `json:"build_number,omitempty"`
	BuildTimestamp string `json:"build_timestamp,omitempty"`

	ReleaseBundleName    string `json:"release_bundle_name,omitempty"`
	ReleaseBundleVersion string `json:"release_bundle_version,omitempty"`

	ApplicationKey     string `json:"application_key,omitempty"`
	ApplicationVersion string `json:"application_version,omitempty"`

	PackageRepo    string `json:"package_repo,omitempty"`
	PackageName    string `json:"package_name,omitempty"`
	PackageVersion string `json:"package_version,omitempty"`
}

// PrepareEvidenceAttachment identifies an attachment in Artifactory.
// SHA256 is optional in a prepare request and is resolved by Evidence.
type PrepareEvidenceAttachment struct {
	Repository string `json:"repository"`
	Path       string `json:"path"`
	SHA256     string `json:"sha256,omitempty"`
}

type PrepareEvidenceResponse struct {
	PostURL                   string                      `json:"post_url"`
	DSSEPayload               string                      `json:"dsse_payload"`
	DSSEPayloadType           string                      `json:"dsse_payload_type"`
	PreAuthenticationEncoding string                      `json:"pre_authentication_encoding,omitempty"`
	Attachments               []PrepareEvidenceAttachment `json:"attachments,omitempty"`
}

// EvidenceClient talks to Evidence REST APIs that are not yet available in jfrog-client-go.
type EvidenceClient struct {
	httpClient *jfroghttpclient.JfrogHttpClient
	details    auth.ServiceDetails
}

// NewEvidenceClient builds a client using the same Evidence URL/auth as CreateEvidenceServiceManager.
func NewEvidenceClient(serverDetails *config.ServerDetails) (*EvidenceClient, error) {
	certsPath, err := coreutils.GetJfrogCertsDir()
	if err != nil {
		return nil, err
	}
	evdAuth, err := serverDetails.CreateEvidenceAuthConfig()
	if err != nil {
		return nil, err
	}
	serviceConfig, err := clientConfig.NewConfigBuilder().
		SetServiceDetails(evdAuth).
		SetCertificatesPath(certsPath).
		SetInsecureTls(serverDetails.InsecureTls).
		Build()
	if err != nil {
		return nil, err
	}
	httpClient, err := jfroghttpclient.JfrogClientBuilder().
		SetCertificatesPath(serviceConfig.GetCertificatesPath()).
		SetInsecureTls(serviceConfig.IsInsecureTls()).
		SetClientCertPath(evdAuth.GetClientCertPath()).
		SetClientCertKeyPath(evdAuth.GetClientCertKeyPath()).
		AppendPreRequestInterceptor(evdAuth.RunPreRequestFunctions).
		SetContext(serviceConfig.GetContext()).
		SetDialTimeout(serviceConfig.GetDialTimeout()).
		SetOverallRequestTimeout(serviceConfig.GetOverallRequestTimeout()).
		SetRetries(serviceConfig.GetHttpRetries()).
		SetRetryWaitMilliSecs(serviceConfig.GetHttpRetryWaitMilliSecs()).
		Build()
	if err != nil {
		return nil, err
	}
	return &EvidenceClient{
		httpClient: httpClient,
		details:    evdAuth,
	}, nil
}

// PrepareEvidence asks Evidence to generate an in-toto statement for external signing.
func (c *EvidenceClient) PrepareEvidence(request PrepareEvidenceRequest, includePAE bool) (*PrepareEvidenceResponse, error) {
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, errorutils.CheckError(err)
	}

	queryParams := make(map[string]string)
	if includePAE {
		queryParams["include_pae"] = "true"
	}
	requestFullURL, err := clientutils.BuildUrl(c.details.GetUrl(), prepareEvidenceAPI, queryParams)
	if err != nil {
		return nil, errorutils.CheckError(err)
	}

	httpClientDetails := c.details.CreateHttpClientDetails()
	httpClientDetails.SetContentTypeApplicationJson()

	log.Debug("Preparing Evidence for signing")
	resp, body, err := c.httpClient.SendPost(requestFullURL, requestBody, &httpClientDetails)
	if err != nil {
		return nil, err
	}
	if err = errorutils.CheckResponseStatusWithBody(resp, body, http.StatusOK); err != nil {
		return nil, err
	}

	var response PrepareEvidenceResponse
	if err = json.Unmarshal(body, &response); err != nil {
		return nil, errorutils.CheckError(err)
	}
	return &response, nil
}

// UploadPreparedSignedEvidence uploads a signed DSSE envelope to the root-relative
// post URL returned by PrepareEvidence.
func (c *EvidenceClient) UploadPreparedSignedEvidence(postURL string, signedEnvelope []byte) ([]byte, error) {
	requestURL, err := c.resolvePreparedEvidencePostURL(postURL)
	if err != nil {
		return nil, err
	}

	httpClientDetails := c.details.CreateHttpClientDetails()
	httpClientDetails.SetContentTypeApplicationJson()

	log.Debug("Uploading prepared signed Evidence")
	resp, body, err := c.httpClient.SendPost(requestURL, signedEnvelope, &httpClientDetails)
	if err != nil {
		return nil, err
	}
	return body, errorutils.CheckResponseStatusWithBody(resp, body, http.StatusOK, http.StatusCreated)
}

func (c *EvidenceClient) resolvePreparedEvidencePostURL(postURL string) (string, error) {
	post, err := url.Parse(postURL)
	if err != nil {
		return "", errorutils.CheckError(err)
	}
	if postURL == "" || post.IsAbs() || post.Host != "" || post.Path == "" || post.Path[0] != '/' {
		return "", fmt.Errorf("prepared Evidence post URL must be a non-empty root-relative URL")
	}

	base, err := url.Parse(c.details.GetUrl())
	if err != nil {
		return "", errorutils.CheckError(err)
	}
	if base.Scheme == "" || base.Host == "" {
		return "", fmt.Errorf("evidence URL must include a scheme and host")
	}

	post.Scheme = base.Scheme
	post.Host = base.Host
	post.User = base.User
	return post.String(), nil
}
