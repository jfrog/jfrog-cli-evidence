package sonar

import (
	"net/url"
	"os"
	"strings"

	conf "github.com/jfrog/jfrog-cli-evidence/evidence/config"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"github.com/jfrog/jfrog-client-go/utils/log"
)

// StatementResolver resolves a full in-toto statement.
type StatementResolver interface {
	ResolveStatement() (statement []byte, err error)
}

type statementResolver struct {
	providerFactory providerFactory
}

func NewStatementResolver() StatementResolver {
	return &statementResolver{
		providerFactory: defaultProviderFactory{},
	}
}

func (d *statementResolver) ResolveStatement() ([]byte, error) {
	log.Info("Fetching SonarQube in-toto statement")
	cfg := conf.LoadEvidenceConfig()
	return d.resolveStatementWithConfig(cfg)
}

func (d *statementResolver) resolveStatementWithConfig(cfg *conf.EvidenceConfig) ([]byte, error) {
	var reportPath string
	if cfg != nil && cfg.Sonar != nil {
		reportPath = cfg.Sonar.ReportTaskFile
	}
	reportPath = detectReportTaskPath(reportPath)
	if reportPath == "" {
		return nil, errorutils.CheckErrorf("no report-task.txt file found and no custom path configured")
	}

	rt, err := parseReportTask(reportPath)
	if err != nil {
		return nil, errorutils.CheckErrorf("failed to parse report-task file at %s: %v", reportPath, err)
	}

	log.Info("Parsed report-task file at", reportPath, "with ceTaskID:", rt.CeTaskID)

	sonarBaseURL, err := GetTrustedSonarURL()
	if err != nil {
		return nil, err
	}

	token := os.Getenv("SONAR_TOKEN")
	if token == "" {
		token = os.Getenv("SONARQUBE_TOKEN")
	}

	provider, err := d.providerFactory.New(sonarBaseURL, token)
	if err != nil {
		return nil, err
	}

	var pollingMaxRetries, pollingRetryIntervalMs *int
	if cfg != nil && cfg.Sonar != nil {
		pollingMaxRetries = cfg.Sonar.PollingMaxRetries
		pollingRetryIntervalMs = cfg.Sonar.PollingRetryIntervalMs
	}

	return provider.GetStatement(rt.CeTaskID, pollingMaxRetries, pollingRetryIntervalMs)
}

func GetTrustedSonarURL() (string, error) {
	sonarURL := strings.TrimSpace(conf.ResolveTrustedSonarURL())
	if sonarURL == "" {
		return "", errorutils.CheckErrorf("trusted SonarQube URL is required; set SONAR_URL, SONAR_HOST_URL, or configure sonar.url in the user-level JFrog evidence config")
	}
	if err := validateTrustedSonarURL(sonarURL); err != nil {
		return "", err
	}
	return sonarURL, nil
}

func validateTrustedSonarURL(sonarURL string) error {
	parsedURL, err := url.Parse(strings.TrimSpace(sonarURL))
	if err != nil || parsedURL.Host == "" {
		return errorutils.CheckErrorf("invalid trusted SonarQube URL")
	}
	if !strings.EqualFold(parsedURL.Scheme, "https") {
		return errorutils.CheckErrorf("trusted SonarQube URL must use HTTPS")
	}
	return nil
}

type providerFactory interface {
	New(sonarURL, token string) (StatementProvider, error)
}

type defaultProviderFactory struct{}

func (defaultProviderFactory) New(sonarURL, token string) (StatementProvider, error) {
	return NewSonarProviderWithCredentials(sonarURL, token)
}
