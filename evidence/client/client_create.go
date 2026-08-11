package client

import (
	"fmt"
	"net/http"
	"net/url"

	clientutils "github.com/jfrog/jfrog-client-go/utils"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"github.com/jfrog/jfrog-client-go/utils/log"
)

// CreateEntityEvidenceRequest identifies the entity subject and optional Artifactory scope /
// provider for a create call. At most one of EntityRepo, ProjectKey, and ApplicationKey may be set.
type CreateEntityEvidenceRequest struct {
	EntityType     string
	EntityID       string
	EntityRepo     string
	ProjectKey     string
	ApplicationKey string
	ProviderID     string
}

// UploadPreparedSignedEvidence uploads a signed DSSE envelope to the root-relative
// post URL returned by PrepareEvidence.
func (c *EvidenceClient) UploadPreparedSignedEvidence(postURL string, signedEnvelope []byte) ([]byte, error) {
	requestURL, err := c.resolveEvidencePostURL(postURL)
	if err != nil {
		return nil, err
	}
	log.Debug("Uploading prepared signed Evidence")
	return c.createEvidence(requestURL, signedEnvelope)
}

// CreateEntityEvidence POSTs a signed DSSE envelope or Sigstore bundle to
// /evidence/api/v1/entity/{type}/{id}.
func (c *EvidenceClient) CreateEntityEvidence(request CreateEntityEvidenceRequest, payload []byte) ([]byte, error) {
	requestURL, err := c.buildEntityEvidenceURL(request)
	if err != nil {
		return nil, err
	}
	log.Debug("Creating entity Evidence at:", requestURL)
	return c.createEvidence(requestURL, payload)
}

func (c *EvidenceClient) buildEntityEvidenceURL(request CreateEntityEvidenceRequest) (string, error) {
	queryParams := make(map[string]string)
	switch {
	case request.EntityRepo != "":
		queryParams["repo"] = request.EntityRepo
	case request.ProjectKey != "":
		queryParams["project"] = request.ProjectKey
	case request.ApplicationKey != "":
		queryParams["application"] = request.ApplicationKey
	}
	if request.ProviderID != "" {
		queryParams["providerId"] = request.ProviderID
	}
	apiPath := fmt.Sprintf("api/v1/entity/%s/%s", request.EntityType, request.EntityID)
	requestURL, err := clientutils.BuildUrl(c.details.GetUrl(), apiPath, queryParams)
	if err != nil {
		return "", errorutils.CheckError(err)
	}
	return requestURL, nil
}

func (c *EvidenceClient) createEvidence(requestURL string, payload []byte) ([]byte, error) {
	httpClientDetails := c.details.CreateHttpClientDetails()
	httpClientDetails.SetContentTypeApplicationJson()

	resp, body, err := c.httpClient.SendPost(requestURL, payload, &httpClientDetails)
	if err != nil {
		return nil, err
	}
	return body, errorutils.CheckResponseStatusWithBody(resp, body, http.StatusOK, http.StatusCreated)
}

func (c *EvidenceClient) resolveEvidencePostURL(postURL string) (string, error) {
	post, err := url.Parse(postURL)
	if err != nil {
		return "", errorutils.CheckError(err)
	}
	if postURL == "" || post.IsAbs() || post.Host != "" || post.Path == "" || post.Path[0] != '/' {
		return "", fmt.Errorf("Evidence post URL must be a non-empty root-relative URL")
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
