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

	keySonarReportTaskFile           = "sonar.reportTaskFile"
	keySonarURL                      = "sonar.url"
	keyPollingMaxRetries             = "sonar.pollingMaxRetries"
	keyPollingRetryIntervalMs        = "sonar.pollingRetryIntervalMs"
	keyAttachmentArtifactoryTempPath = "attachment.artifactoryTempPath"

	envReportTaskFile         = "SONAR_REPORT_TASK_FILE"
	envSonarURL               = "SONAR_URL"
	envPollingMaxRetries      = "SONAR_POLLING_MAX_RETRIES"
	envPollingRetryIntervalMs = "SONAR_POLLING_RETRY_INTERVAL_MS"

	EnvAttachmentArtifactoryTempPath = "EVIDENCE_ATTACHMENT_ARTIFACTORY_TEMP_PATH"
	KeyAttachmentArtifactoryTempPath = keyAttachmentArtifactoryTempPath
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
	ArtifactoryTempPath string `yaml:"artifactoryTempPath"`
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
	_ = v.BindEnv(keyAttachmentArtifactoryTempPath, EnvAttachmentArtifactoryTempPath)
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

// ResolveAttachmentArtifactoryTempPath returns an operator-controlled upload target.
// A repository .jfrog/evidence/evidence.yml is ignored: that file is part of the checkout,
// and this path is both uploaded and later deleted.
func ResolveAttachmentArtifactoryTempPath() string {
	if envValue := os.Getenv(EnvAttachmentArtifactoryTempPath); envValue != "" {
		return envValue
	}
	cfg, err := readHomeEvidenceConfig()
	if err != nil || cfg == nil || cfg.Attachment == nil {
		return ""
	}
	return cfg.Attachment.ArtifactoryTempPath
}

func PersistAttachmentArtifactoryTempPath(artifactoryTempPath string) error {
	path, err := homeEvidenceConfigPath()
	if err != nil {
		return err
	}
	cfg, err := readEvidenceConfigFile(path)
	if err != nil {
		return errorutils.CheckError(err)
	}
	if cfg == nil {
		cfg = &EvidenceConfig{}
	}
	if cfg.Attachment == nil {
		cfg.Attachment = &AttachmentConfig{}
	}
	cfg.Attachment.ArtifactoryTempPath = artifactoryTempPath
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

func homeEvidenceConfigPath() (string, error) {
	home, err := coreutils.GetJfrogHomeDir()
	if err != nil {
		return "", err
	}
	yml := filepath.Join(home, evidenceDir, evidenceFileYml)
	yamlPath := filepath.Join(home, evidenceDir, evidenceFileYaml)
	// Keep an existing evidence.yaml when evidence.yml was never created.
	if evidenceFileExists(yamlPath) && !evidenceFileExists(yml) {
		return yamlPath, nil
	}
	return yml, nil
}

func evidenceFileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func readHomeEvidenceConfig() (*EvidenceConfig, error) {
	path, err := homeEvidenceConfigPath()
	if err != nil {
		return nil, err
	}
	return readEvidenceConfigFile(path)
}

func readEvidenceConfigFile(path string) (*EvidenceConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &EvidenceConfig{}, nil
		}
		return nil, err
	}
	cfg := &EvidenceConfig{}
	if err = yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
