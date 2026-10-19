package sonar

import (
	"os"
	"path/filepath"
	"testing"

	conf "github.com/jfrog/jfrog-cli-evidence/evidence/config"
	"github.com/stretchr/testify/assert"
)

func withChdir(t *testing.T, dir string) {
	old, _ := os.Getwd()
	assert.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(old) })
}

func TestResolveStatement_NoReportTask(t *testing.T) {
	dir := t.TempDir()
	withChdir(t, dir)
	resolver := NewStatementResolver()
	_, err := resolver.ResolveStatement()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no report-task.txt file found")
}

// Helpers and factory-based tests

type fakeProvider struct {
	nextResult   []byte
	nextErr      error
	lastCeTaskID string
}

func (f *fakeProvider) GetStatement(ceTaskID string, pollingMaxRetries *int, pollingRetryIntervalMs *int) ([]byte, error) {
	f.lastCeTaskID = ceTaskID
	return f.nextResult, f.nextErr
}

type capturingFactory struct {
	lastURL   string
	lastToken string
	provider  StatementProvider
	retErr    error
}

func (c *capturingFactory) New(sonarURL, token string) (StatementProvider, error) {
	c.lastURL = sonarURL
	c.lastToken = token
	return c.provider, c.retErr
}

func writeReportTask(t *testing.T, dir, serverURL, ceTaskID string) string {
	reportDir := filepath.Join(dir, "target", "sonar")
	assert.NoError(t, os.MkdirAll(reportDir, 0o755))
	path := filepath.Join(reportDir, "report-task.txt")
	content := "serverUrl=" + serverURL + "\nceTaskId=" + ceTaskID + "\n"
	assert.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func TestResolver_ResolveStatementWithConfig_Success(t *testing.T) {
	SetSonarURLFlag("")
	dir := t.TempDir()
	serverURL := "https://sonar.local"
	ceTaskID := "task-xyz"
	reportPath := writeReportTask(t, dir, serverURL, ceTaskID)

	fprov := &fakeProvider{nextResult: []byte("ok"), nextErr: nil}
	cf := &capturingFactory{provider: fprov}
	res := &statementResolver{providerFactory: cf}

	t.Setenv("SONAR_TOKEN", "token-123")
	t.Setenv("SONAR_URL", serverURL)

	cfg := &conf.EvidenceConfig{Sonar: &conf.SonarConfig{ReportTaskFile: reportPath}}
	stmt, err := res.resolveStatementWithConfig(cfg)
	assert.NoError(t, err)
	assert.Equal(t, []byte("ok"), stmt)
	assert.Equal(t, serverURL, cf.lastURL)
	assert.Equal(t, "token-123", cf.lastToken)
	assert.Equal(t, ceTaskID, fprov.lastCeTaskID)
}

func TestResolver_EnvironmentURLOverridesReportAndEvidenceConfig(t *testing.T) {
	SetSonarURLFlag("")
	dir := t.TempDir()
	reportURL := "https://attacker.example"
	trustedURL := "https://sonar.local"
	ceTaskID := "task-abc"
	reportPath := writeReportTask(t, dir, reportURL, ceTaskID)

	fprov := &fakeProvider{nextResult: []byte("ok2"), nextErr: nil}
	cf := &capturingFactory{provider: fprov}
	res := &statementResolver{providerFactory: cf}

	t.Setenv("SONARQUBE_TOKEN", "fallback-token")
	// Ensure SONAR_TOKEN is empty so fallback is used
	t.Setenv("SONAR_TOKEN", "")
	t.Setenv("SONAR_URL", trustedURL)

	cfg := &conf.EvidenceConfig{Sonar: &conf.SonarConfig{
		ReportTaskFile: reportPath,
		URL:            "https://repository-controlled.example",
	}}
	stmt, err := res.resolveStatementWithConfig(cfg)
	assert.NoError(t, err)
	assert.Equal(t, []byte("ok2"), stmt)
	assert.Equal(t, trustedURL, cf.lastURL)
	assert.Equal(t, "fallback-token", cf.lastToken)
	assert.Equal(t, ceTaskID, fprov.lastCeTaskID)
}

func TestResolver_UsesEvidenceConfigURLWhenEnvironmentIsUnset(t *testing.T) {
	SetSonarURLFlag("")
	dir := t.TempDir()
	reportPath := writeReportTask(t, dir, "https://report.example", "task-abc")
	fprov := &fakeProvider{nextResult: []byte("ok"), nextErr: nil}
	cf := &capturingFactory{provider: fprov}
	res := &statementResolver{providerFactory: cf}

	t.Setenv("SONAR_TOKEN", "token-123")
	t.Setenv("SONAR_URL", "")
	t.Setenv("SONAR_HOST_URL", "")

	cfg := &conf.EvidenceConfig{Sonar: &conf.SonarConfig{
		ReportTaskFile: reportPath,
		URL:            "https://evidence-config.example",
	}}
	stmt, err := res.resolveStatementWithConfig(cfg)

	assert.NoError(t, err)
	assert.Equal(t, []byte("ok"), stmt)
	assert.Equal(t, "https://evidence-config.example", cf.lastURL)
}

func TestResolver_UsesReportURLWhenNoConfiguredURL(t *testing.T) {
	SetSonarURLFlag("")
	dir := t.TempDir()
	reportPath := writeReportTask(t, dir, "https://report.example", "task-abc")
	fprov := &fakeProvider{nextResult: []byte("ok"), nextErr: nil}
	cf := &capturingFactory{provider: fprov}
	res := &statementResolver{providerFactory: cf}

	t.Setenv("SONAR_TOKEN", "token-123")
	t.Setenv("SONAR_URL", "")
	t.Setenv("SONAR_HOST_URL", "")

	cfg := &conf.EvidenceConfig{Sonar: &conf.SonarConfig{ReportTaskFile: reportPath}}
	stmt, err := res.resolveStatementWithConfig(cfg)

	assert.NoError(t, err)
	assert.Equal(t, []byte("ok"), stmt)
	assert.Equal(t, "https://report.example", cf.lastURL)
}

func TestResolver_FlagOverridesEnvironmentAndReport(t *testing.T) {
	dir := t.TempDir()
	reportPath := writeReportTask(t, dir, "https://report.example", "task-abc")
	fprov := &fakeProvider{nextResult: []byte("ok"), nextErr: nil}
	cf := &capturingFactory{provider: fprov}
	res := &statementResolver{providerFactory: cf}
	t.Cleanup(func() { SetSonarURLFlag("") })

	t.Setenv("SONAR_TOKEN", "token-123")
	t.Setenv("SONAR_URL", "https://env.example")
	SetSonarURLFlag("https://flag.example")

	cfg := &conf.EvidenceConfig{Sonar: &conf.SonarConfig{ReportTaskFile: reportPath}}
	_, err := res.resolveStatementWithConfig(cfg)

	assert.NoError(t, err)
	assert.Equal(t, "https://flag.example", cf.lastURL)
}

func TestResolver_RejectsHTTPFromReport(t *testing.T) {
	SetSonarURLFlag("")
	dir := t.TempDir()
	reportPath := writeReportTask(t, dir, "http://sonar.local", "task-abc")
	cf := &capturingFactory{provider: &fakeProvider{}}
	res := &statementResolver{providerFactory: cf}

	t.Setenv("SONAR_URL", "")
	t.Setenv("SONAR_HOST_URL", "")

	cfg := &conf.EvidenceConfig{Sonar: &conf.SonarConfig{ReportTaskFile: reportPath}}
	_, err := res.resolveStatementWithConfig(cfg)

	assert.ErrorContains(t, err, "must use HTTPS")
	assert.Empty(t, cf.lastURL)
}

func TestResolver_RejectsInsecureExplicitURL(t *testing.T) {
	SetSonarURLFlag("")
	dir := t.TempDir()
	reportPath := writeReportTask(t, dir, "http://sonar.local", "task-abc")
	cf := &capturingFactory{provider: &fakeProvider{}}
	res := &statementResolver{providerFactory: cf}

	t.Setenv("SONAR_URL", "http://sonar.local")

	cfg := &conf.EvidenceConfig{Sonar: &conf.SonarConfig{ReportTaskFile: reportPath}}
	_, err := res.resolveStatementWithConfig(cfg)

	assert.ErrorContains(t, err, "must use HTTPS")
	assert.Empty(t, cf.lastURL)
}
