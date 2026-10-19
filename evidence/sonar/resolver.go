package sonar

import (
	"net/url"
	"os"
	"strings"

	conf "github.com/jfrog/jfrog-cli-evidence/evidence/config"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"github.com/jfrog/jfrog-client-go/utils/log"
)

const defaultSonarURL = "https://sonarcloud.io"

type sonarURLSource int

const (
	sonarURLFromExplicit sonarURLSource = iota
	sonarURLFromEvidenceConfig
	sonarURLFromReportTask
	sonarURLFromDefault
)

// sonarURLFlag is set from --sonar-url before evidence creation.
var sonarURLFlag string

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

// SetSonarURLFlag records the --sonar-url value for the current command.
func SetSonarURLFlag(value string) {
	sonarURLFlag = strings.TrimSpace(value)
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

	sonarBaseURL, source, err := resolveSonarURL(cfg, rt)
	if err != nil {
		return nil, err
	}
	warnSonarURLSource(source)

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

func ValidateExplicitSonarURL() error {
	sonarURL, ok := explicitSonarURL()
	if !ok {
		return nil
	}
	return validateSonarURL(sonarURL)
}

func resolveSonarURL(cfg *conf.EvidenceConfig, rt ReportTask) (string, sonarURLSource, error) {
	sonarURL, source := selectSonarURL(cfg, rt)
	if err := validateSonarURL(sonarURL); err != nil {
		return "", source, err
	}
	return sonarURL, source, nil
}

func selectSonarURL(cfg *conf.EvidenceConfig, rt ReportTask) (string, sonarURLSource) {
	if sonarURL, ok := explicitSonarURL(); ok {
		return sonarURL, sonarURLFromExplicit
	}
	if cfg != nil && cfg.Sonar != nil && strings.TrimSpace(cfg.Sonar.URL) != "" {
		return strings.TrimSpace(cfg.Sonar.URL), sonarURLFromEvidenceConfig
	}
	if sonarURL, fromReport := reportSonarURL(rt.CeTaskURL, rt.ServerURL); fromReport {
		return sonarURL, sonarURLFromReportTask
	}
	return defaultSonarURL, sonarURLFromDefault
}

func explicitSonarURL() (string, bool) {
	if sonarURLFlag != "" {
		return sonarURLFlag, true
	}
	if sonarURL := strings.TrimSpace(os.Getenv("SONAR_URL")); sonarURL != "" {
		return sonarURL, true
	}
	if sonarURL := strings.TrimSpace(os.Getenv("SONAR_HOST_URL")); sonarURL != "" {
		return sonarURL, true
	}
	return "", false
}

func reportSonarURL(ceTaskURL, serverURL string) (string, bool) {
	if serverURL != "" {
		return serverURL, true
	}
	if ceTaskURL != "" {
		parsedURL, err := url.Parse(ceTaskURL)
		if err == nil && parsedURL.Host != "" {
			return parsedURL.Scheme + "://" + parsedURL.Host, true
		}
	}
	return "", false
}

func warnSonarURLSource(source sonarURLSource) {
	switch source {
	case sonarURLFromEvidenceConfig:
		log.Warn("SonarQube URL was read from evidence.yml. Provide it with --sonar-url or the SONAR_URL environment variable.")
	case sonarURLFromReportTask:
		log.Warn("SonarQube URL was read from report-task.txt. Provide it with --sonar-url or the SONAR_URL environment variable.")
	}
}

func validateSonarURL(sonarURL string) error {
	parsedURL, err := url.Parse(strings.TrimSpace(sonarURL))
	if err != nil || parsedURL.Host == "" {
		return errorutils.CheckErrorf("invalid SonarQube URL")
	}
	if !strings.EqualFold(parsedURL.Scheme, "https") {
		return errorutils.CheckErrorf("SonarQube URL must use HTTPS")
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
