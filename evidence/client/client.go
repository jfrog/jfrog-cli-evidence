package client

import (
	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli-core/v2/utils/coreutils"
	"github.com/jfrog/jfrog-client-go/auth"
	clientConfig "github.com/jfrog/jfrog-client-go/config"
	"github.com/jfrog/jfrog-client-go/http/jfroghttpclient"
)

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
