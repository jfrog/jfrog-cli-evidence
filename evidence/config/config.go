package config

import (
	"os"
	"path/filepath"

	"github.com/jfrog/jfrog-cli-core/v2/utils/coreutils"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"github.com/jfrog/jfrog-client-go/utils/io/fileutils"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

const (
	jfrogDir         = ".jfrog"
	evidenceDir      = "evidence"
	evidenceFileYml  = "evidence.yml"
	evidenceFileYaml = "evidence.yaml"

	keySonarReportTaskFile    = "sonar.reportTaskFile"
	keySonarURL               = "sonar.url"
	keyPollingMaxRetries      = "sonar.pollingMaxRetries"
	keyPollingRetryIntervalMs = "sonar.pollingRetryIntervalMs"
	keyAttachmentTempTarget   = "attachment.tempTarget"

	envReportTaskFile         = "SONAR_REPORT_TASK_FILE"
	envSonarURL               = "SONAR_URL"
	envPollingMaxRetries      = "SONAR_POLLING_MAX_RETRIES"
	envPollingRetryIntervalMs = "SONAR_POLLING_RETRY_INTERVAL_MS"
	envAttachmentTempTarget   = "EVIDENCE_ATTACHMENT_TEMP_TARGET"
)

type SonarConfig struct {
	URL                    string `yaml:"url"`
	ReportTaskFile         string `yaml:"reportTaskFile"`
	PollingMaxRetries      *int   `yaml:"pollingMaxRetries"`
	PollingRetryIntervalMs *int   `yaml:"pollingRetryIntervalMs"`
}

type EvidenceConfig struct {
	Sonar      *SonarConfig      `yaml:"sonar"`
	Attachment *AttachmentConfig `yaml:"attachment"`
}

type AttachmentConfig struct {
	TempTarget string `yaml:"tempTarget"`
}

func LoadEvidenceConfig() *EvidenceConfig {
	// 1) Upstream .jfrog root
	if root, exists, _ := fileutils.FindUpstream(jfrogDir, fileutils.Dir); exists {
		if cfg := readConfigWithEnv(filepath.Join(root, jfrogDir, evidenceDir, evidenceFileYml)); cfg != nil {
			return cfg
		}
		if cfg := readConfigWithEnv(filepath.Join(root, jfrogDir, evidenceDir, evidenceFileYaml)); cfg != nil {
			return cfg
		}
	}

	// 2) Home fallback: ~/.jfrog/evidence/...
	if home, err := coreutils.GetJfrogHomeDir(); err == nil && home != "" {
		if cfg := readConfigWithEnv(filepath.Join(home, evidenceDir, evidenceFileYml)); cfg != nil {
			return cfg
		}
		if cfg := readConfigWithEnv(filepath.Join(home, evidenceDir, evidenceFileYaml)); cfg != nil {
			return cfg
		}
	}

	// 3) Env-only (no file)
	if cfg := readConfigWithEnv(""); cfg != nil {
		return cfg
	}

	return nil
}

func readConfigWithEnv(path string) *EvidenceConfig {
	v := viper.New()

	_ = v.BindEnv(keySonarReportTaskFile, envReportTaskFile)
	_ = v.BindEnv(keySonarURL, envSonarURL)
	_ = v.BindEnv(keyPollingMaxRetries, envPollingMaxRetries)
	_ = v.BindEnv(keyPollingRetryIntervalMs, envPollingRetryIntervalMs)
	_ = v.BindEnv(keyAttachmentTempTarget, envAttachmentTempTarget)
	v.AutomaticEnv()

	if path != "" {
		v.SetConfigFile(path)
		_ = v.ReadInConfig()
	}

	cfg := new(EvidenceConfig)
	if err := v.Unmarshal(&cfg); err != nil {
		_ = errorutils.CheckError(err)
		return nil
	}
	if (cfg.Sonar == nil || (*cfg.Sonar == (SonarConfig{}))) &&
		(cfg.Attachment == nil || (*cfg.Attachment == (AttachmentConfig{}))) {
		return nil
	}
	return cfg
}

func ResolveAttachmentTempTarget() string {
	if envValue := os.Getenv(envAttachmentTempTarget); envValue != "" {
		return envValue
	}
	cfg := LoadEvidenceConfig()
	if cfg != nil && cfg.Attachment != nil && cfg.Attachment.TempTarget != "" {
		return cfg.Attachment.TempTarget
	}
	return ""
}

func PersistAttachmentTempTarget(tempTarget string) error {
	path, err := resolveWritableConfigPath()
	if err != nil {
		return err
	}
	cfg := LoadEvidenceConfig()
	if cfg == nil {
		cfg = &EvidenceConfig{}
	}
	if cfg.Attachment == nil {
		cfg.Attachment = &AttachmentConfig{}
	}
	cfg.Attachment.TempTarget = tempTarget
	content, err := yaml.Marshal(cfg)
	if err != nil {
		return errorutils.CheckError(err)
	}
	if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return errorutils.CheckError(err)
	}
	if err = os.WriteFile(path, content, 0600); err != nil {
		return errorutils.CheckError(err)
	}
	return nil
}

func resolveWritableConfigPath() (string, error) {
	if root, exists, _ := fileutils.FindUpstream(jfrogDir, fileutils.Dir); exists {
		return filepath.Join(root, jfrogDir, evidenceDir, evidenceFileYml), nil
	}
	home, err := coreutils.GetJfrogHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, evidenceDir, evidenceFileYml), nil
}
