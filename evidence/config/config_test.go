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

func TestResolveAttachmentTempTarget_EnvPrecedence(t *testing.T) {
	dir := t.TempDir()
	jf := filepath.Join(dir, ".jfrog", "evidence")
	if err := os.MkdirAll(jf, 0755); err != nil {
		t.Fatal(err)
	}
	yml := filepath.Join(jf, "evidence.yml")
	if err := os.WriteFile(yml, []byte("attachment:\n  tempTarget: cfg-repo/cfg-path/\n"), 0644); err != nil {
		t.Fatal(err)
	}
	old, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer func() { _ = os.Chdir(old) }()

	_ = os.Setenv("EVIDENCE_ATTACHMENT_TEMP_TARGET", "env-repo/env-path/")
	defer func() { _ = os.Unsetenv("EVIDENCE_ATTACHMENT_TEMP_TARGET") }()

	target := ResolveAttachmentTempTarget()
	if target != "env-repo/env-path/" {
		t.Fatalf("unexpected resolved target/source: %s", target)
	}
}

func TestPersistAttachmentTempTarget(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".jfrog"), 0755); err != nil {
		t.Fatal(err)
	}
	old, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer func() { _ = os.Chdir(old) }()

	if err := PersistAttachmentTempTarget("repo/path/"); err != nil {
		t.Fatal(err)
	}
	cfg := LoadEvidenceConfig()
	if cfg == nil || cfg.Attachment == nil || cfg.Attachment.TempTarget != "repo/path/" {
		t.Fatalf("persisted temp target mismatch: %+v", cfg)
	}
}
