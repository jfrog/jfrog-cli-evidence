package ca

//go:generate ${PROJECT_DIR}/scripts/mockgen.sh ${GOFILE}

import (
	_ "embed"
	"path/filepath"

	"github.com/jfrog/jfrog-cli-core/v2/utils/coreutils"
	"github.com/pkg/errors"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/tuf"
)

//go:embed embed/tuf-repo.github.com/root.json
var githubRootBytes []byte

type TUFRootCertificateProvider interface {
	LoadTUFRootCertificate() (root.TrustedMaterial, error)
	LoadTUFRootGithubCertificate() (root.TrustedMaterial, error)
}

type tufRootCertificateProvider struct {
}

func NewTUFRootCertificateProvider() TUFRootCertificateProvider {
	return &tufRootCertificateProvider{}
}

func (t *tufRootCertificateProvider) LoadTUFRootCertificate() (root.TrustedMaterial, error) {
	jfrogHomeDir, err := coreutils.GetJfrogHomeDir()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get JFrog home directory")
	}
	opts := tuf.DefaultOptions().WithCachePath(filepath.Join(jfrogHomeDir, "evidence/security/certs"))
	trustedRoot, err := root.FetchTrustedRootWithOptions(opts)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch trusted root")
	}

	return trustedRoot, nil
}

func (t *tufRootCertificateProvider) LoadTUFRootGithubCertificate() (root.TrustedMaterial, error) {
	jfrogHomeDir, err := coreutils.GetJfrogHomeDir()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get JFrog home directory")
	}
	opts := tuf.DefaultOptions().
		WithCachePath(filepath.Join(jfrogHomeDir, "evidence/security/certs/github")).
		WithRepositoryBaseURL("https://tuf-repo.github.com").
		WithRoot(githubRootBytes)

	trustedRoot, err := root.FetchTrustedRootWithOptions(opts)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create trusted root from JSON")
	}

	return trustedRoot, nil
}
