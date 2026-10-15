package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEvidenceConfig_Upstream(t *testing.T) {
	dir := t.TempDir()
	jf := filepath.Join(dir, ".jfrog", "evidence")
	if err := os.MkdirAll(jf, 0755); err != nil {
		t.Fatal(err)
	}
	yml := filepath.Join(jf, "evidence.yml")
	if err := os.WriteFile(yml, []byte("sonar:\n  url: https://sonar\n  reportTaskFile: rpt.txt\n"), 0644); err != nil {
		t.Fatal(err)
	}
	old, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer func() { _ = os.Chdir(old) }()

	cfg := LoadEvidenceConfig()
	if cfg == nil || cfg.Sonar == nil || cfg.Sonar.URL != "https://sonar" || cfg.Sonar.ReportTaskFile != "rpt.txt" {
		t.Fatalf("unexpected cfg: %+v", cfg)
	}
}

func TestLoadEvidenceConfig_EnvOverride(t *testing.T) {
	dir := t.TempDir()
	jf := filepath.Join(dir, ".jfrog", "evidence")
	if err := os.MkdirAll(jf, 0755); err != nil {
		t.Fatal(err)
	}
	yml := filepath.Join(jf, "evidence.yaml")
	if err := os.WriteFile(yml, []byte("sonar:\n  url: https://file-url\n  reportTaskFile: file.txt\n"), 0644); err != nil {
		t.Fatal(err)
	}
	old, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer func() { _ = os.Chdir(old) }()

	_ = os.Setenv("SONAR_REPORT_TASK_FILE", "env.txt")
	_ = os.Setenv("SONAR_URL", "https://env-url")
	defer func() {
		_ = os.Unsetenv("SONAR_REPORT_TASK_FILE")
		_ = os.Unsetenv("SONAR_URL")
	}()

	cfg := LoadEvidenceConfig()

	if cfg == nil || cfg.Sonar == nil || cfg.Sonar.URL != "https://env-url" || cfg.Sonar.ReportTaskFile != "env.txt" {
		t.Fatalf("env overrides not applied: %+v", cfg)
	}
}

func TestLoadEvidenceConfig_EnvOnly(t *testing.T) {
	old, _ := os.Getwd()
	tmp := t.TempDir()
	_ = os.Chdir(tmp)
	defer func() { _ = os.Chdir(old) }()

	_ = os.Setenv("SONAR_REPORT_TASK_FILE", "only-env.txt")
	defer func() { _ = os.Unsetenv("SONAR_REPORT_TASK_FILE") }()

	cfg := LoadEvidenceConfig()
	if cfg == nil || cfg.Sonar == nil || cfg.Sonar.ReportTaskFile != "only-env.txt" {
		t.Fatalf("expected env-only cfg, got: %+v", cfg)
	}
}

func TestResolveTrustedSonarURL_EnvironmentTakesPrecedence(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, evidenceDir)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, evidenceFileYml), []byte("sonar:\n  url: https://home-sonar\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("JFROG_CLI_HOME_DIR", home)
	t.Setenv(envSonarURL, "https://env-sonar")
	t.Setenv(envSonarHostURL, "https://scanner-sonar")

	if got := ResolveTrustedSonarURL(); got != "https://env-sonar" {
		t.Fatalf("expected environment URL, got %q", got)
	}
}

func TestResolveTrustedSonarURL_UsesOfficialScannerEnvironment(t *testing.T) {
	t.Setenv(envSonarURL, "")
	t.Setenv(envSonarHostURL, "https://scanner-sonar")

	if got := ResolveTrustedSonarURL(); got != "https://scanner-sonar" {
		t.Fatalf("expected scanner environment URL, got %q", got)
	}
}

func TestResolveTrustedSonarURL_UsesHomeConfig(t *testing.T) {
	workspace := t.TempDir()
	repositoryConfigDir := filepath.Join(workspace, jfrogDir, evidenceDir)
	if err := os.MkdirAll(repositoryConfigDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repositoryConfigDir, evidenceFileYml), []byte("sonar:\n  url: https://repository-controlled.example\n"), 0644); err != nil {
		t.Fatal(err)
	}
	old, _ := os.Getwd()
	_ = os.Chdir(workspace)
	t.Cleanup(func() { _ = os.Chdir(old) })

	home := t.TempDir()
	homeConfigDir := filepath.Join(home, evidenceDir)
	if err := os.MkdirAll(homeConfigDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(homeConfigDir, evidenceFileYml), []byte("sonar:\n  url: https://home-sonar\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("JFROG_CLI_HOME_DIR", home)
	t.Setenv(envSonarURL, "")
	t.Setenv(envSonarHostURL, "")

	if got := ResolveTrustedSonarURL(); got != "https://home-sonar" {
		t.Fatalf("expected home-level URL, got %q", got)
	}
}

func TestResolveTrustedSonarURL_IgnoresRepositoryConfig(t *testing.T) {
	workspace := t.TempDir()
	repositoryConfigDir := filepath.Join(workspace, jfrogDir, evidenceDir)
	if err := os.MkdirAll(repositoryConfigDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repositoryConfigDir, evidenceFileYml), []byte("sonar:\n  url: https://repository-controlled.example\n"), 0644); err != nil {
		t.Fatal(err)
	}
	old, _ := os.Getwd()
	_ = os.Chdir(workspace)
	t.Cleanup(func() { _ = os.Chdir(old) })

	t.Setenv("JFROG_CLI_HOME_DIR", t.TempDir())
	t.Setenv(envSonarURL, "")
	t.Setenv(envSonarHostURL, "")

	if got := ResolveTrustedSonarURL(); got != "" {
		t.Fatalf("expected repository URL to be ignored, got %q", got)
	}
}

func TestResolveAttachmentArtifactoryTempPath_EnvPrecedence(t *testing.T) {
	dir := t.TempDir()
	jf := filepath.Join(dir, ".jfrog", "evidence")
	if err := os.MkdirAll(jf, 0755); err != nil {
		t.Fatal(err)
	}
	yml := filepath.Join(jf, "evidence.yml")
	if err := os.WriteFile(yml, []byte("attachment:\n  artifactoryTempPath: cfg-repo/cfg-path/\n"), 0644); err != nil {
		t.Fatal(err)
	}
	old, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer func() { _ = os.Chdir(old) }()

	_ = os.Setenv(EnvAttachmentArtifactoryTempPath, "env-repo/env-path/")
	defer func() { _ = os.Unsetenv(EnvAttachmentArtifactoryTempPath) }()

	target := ResolveAttachmentArtifactoryTempPath()
	if target != "env-repo/env-path/" {
		t.Fatalf("unexpected resolved target/source: %s", target)
	}
}

func TestPersistAttachmentArtifactoryTempPath(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".jfrog"), 0755); err != nil {
		t.Fatal(err)
	}
	old, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer func() { _ = os.Chdir(old) }()

	if err := PersistAttachmentArtifactoryTempPath("repo/path/"); err != nil {
		t.Fatal(err)
	}
	cfg := LoadEvidenceConfig()
	if cfg == nil || cfg.Attachment == nil || cfg.Attachment.ArtifactoryTempPath != "repo/path/" {
		t.Fatalf("persisted artifactory temp path mismatch: %+v", cfg)
	}
}
