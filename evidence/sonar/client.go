package sonar

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/jfrog/jfrog-cli-evidence/evidence/utils"
	"github.com/jfrog/jfrog-client-go/http/jfroghttpclient"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"github.com/jfrog/jfrog-client-go/utils/io/httputils"
	"github.com/jfrog/jfrog-client-go/utils/log"
)

type TaskDetails struct {
	Task struct {
		ID                 string      `json:"id"`
		Type               string      `json:"type"`
		ComponentID        string      `json:"componentId"`
		ComponentKey       string      `json:"componentKey"`
		ComponentName      string      `json:"componentName"`
		ComponentQualifier string      `json:"componentQualifier"`
		AnalysisID         string      `json:"analysisId"`
		Status             string      `json:"status"`
		SubmittedAt        string      `json:"submittedAt"`
		StartedAt          string      `json:"startedAt"`
		ExecutedAt         string      `json:"executedAt"`
		ExecutionTimeMs    int         `json:"executionTimeMs"`
		Logs               interface{} `json:"logs"`
		HasScannerContext  bool        `json:"hasScannerContext"`
		Organization       string      `json:"organization"`
	} `json:"task"`
}

type Client interface {
	GetTaskDetails(ceTaskID string) (*TaskDetails, error)
	GetSonarIntotoStatement(ceTaskID string) ([]byte, error)
}

type httpClient struct {
	baseURL string
	token   string
	client  *jfroghttpclient.JfrogHttpClient
}

func NewClient(sonarURL, token string) (Client, error) {
	base := strings.TrimRight(sonarURL, "/")
	cli, err := jfroghttpclient.JfrogClientBuilder().Build()
	if err != nil {
		return nil, errorutils.CheckError(err)
	}
	return &httpClient{baseURL: base, token: token, client: cli}, nil
}

func (c *httpClient) authHeader() string {
	if c.token != "" {
		return "Bearer " + c.token
	}
	return ""
}

func (c *httpClient) doGET(urlStr string) ([]byte, int, error) {
	details := httputils.HttpClientDetails{Headers: map[string]string{}}
	if h := c.authHeader(); h != "" {
		details.Headers["Authorization"] = h
	}
	resp, body, _, err := c.client.SendGet(urlStr, true, &details)
	if err != nil {
		log.Debug("HTTP GET error for", urlStr, "error:", err.Error())
		return nil, 0, err
	}
	log.Debug("HTTP GET response for", urlStr, "status:", resp.StatusCode, "body:", string(body))
	return body, resp.StatusCode, nil
}

func (c *httpClient) GetSonarIntotoStatement(ceTaskID string) ([]byte, error) {
	if ceTaskID == "" {
		return nil, errorutils.CheckError(fmt.Errorf("missing ce task id for enterprise endpoint"))
	}
	u, _ := url.Parse(c.baseURL)
	hostname := u.Hostname()

	cloudUrl := c.prepareCloudFormatUrl(ceTaskID, hostname)
	log.Debug(fmt.Sprintf("Getting intoto statement using cloud format sonar endpoint %s", cloudUrl))
	body, statusCode, err := c.doGET(cloudUrl)

	if utils.IsHttpStatusSuccessful(statusCode) {
		serverURL := c.prepareServerFormatUrl(ceTaskID)
		log.Debug(fmt.Sprintf("Getting intoto statement using server format sonar endpoint %s", serverURL))
		body, statusCode, err = c.doGET(serverURL)
	}

	if err != nil {
		return nil, errorutils.CheckErrorf("enterprise endpoint failed with status %d and response %s %v", statusCode, string(body), err)
	}
	if !utils.IsHttpStatusSuccessful(statusCode) {
		return nil, errorutils.CheckErrorf("enterprise endpoint returned status %d: %s", statusCode, string(body))
	}
	return body, nil
}

func (c *httpClient) GetTaskDetails(ceTaskID string) (*TaskDetails, error) {
	if ceTaskID == "" {
		return nil, nil
	}
	taskURL := fmt.Sprintf("%s/api/ce/task?id=%s", c.baseURL, url.QueryEscape(ceTaskID))
	body, statusCode, err := c.doGET(taskURL)
	if err != nil {
		return nil, err
	}
	if utils.IsHttpStatusSuccessful(statusCode) {
		return nil, errorutils.CheckErrorf("task endpoint returned status %d: %s", statusCode, string(body))
	}
	var response TaskDetails
	if err = json.Unmarshal(body, &response); err != nil {
		return nil, errorutils.CheckErrorf("failed to parse task response: %v", err)
	}
	return &response, nil
}

func (c *httpClient) prepareCloudFormatUrl(ceTaskID string, hostname string) string {
	cloudBaseURL := c.baseURL
	if !strings.HasPrefix(hostname, "api.") {
		cloudBaseURL = strings.Replace(c.baseURL, "://", "://api.", 1)
	}
	cloudURL := fmt.Sprintf("%s/dop-translation/jfrog-evidence/%s", cloudBaseURL, url.QueryEscape(ceTaskID))
	return cloudURL
}

func (c *httpClient) prepareServerFormatUrl(ceTaskID string) string {
	return fmt.Sprintf("%s/api/v2/dop-translation/jfrog-evidence/%s", c.baseURL, url.QueryEscape(ceTaskID))
}
