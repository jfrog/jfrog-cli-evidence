package ca

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewTUFRootCertificateProvider(t *testing.T) {
	provider := NewTUFRootCertificateProvider()
	assert.NotNil(t, provider)
	assert.IsType(t, &tufRootCertificateProvider{}, provider)
}

func TestLoadTUFRootGithubCertificate(t *testing.T) {
	provider := NewTUFRootCertificateProvider()
	assert.NotNil(t, provider)

	// Test loading GitHub certificate
	trustedRoot, err := provider.LoadTUFRootGithubCertificate()
	assert.NoError(t, err)
	assert.NotNil(t, trustedRoot)
}
