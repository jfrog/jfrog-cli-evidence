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

func TestTUFRootCertificateProvider_ImplementsInterface(t *testing.T) {
	var _ TUFRootCertificateProvider = (*tufRootCertificateProvider)(nil)
	// This test ensures the struct implements the interface
}
