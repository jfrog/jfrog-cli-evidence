package utils

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockIntrospectionManager struct {
	response []byte
	err      error
	queries  []string
}

func (m *mockIntrospectionManager) GraphqlQuery(query []byte) ([]byte, error) {
	m.queries = append(m.queries, string(query))
	return m.response, m.err
}

func TestEnsureEvidenceRevocationSupported_FieldPresent(t *testing.T) {
	manager := &mockIntrospectionManager{
		response: []byte(`{"data":{"__type":{"fields":[{"name":"id"},{"name":"verified"},{"name":"revoked"},{"name":"revocation"}]}}}`),
	}

	assert.NoError(t, EnsureEvidenceRevocationSupported(manager))
	if assert.Len(t, manager.queries, 1) {
		assert.Equal(t, `{"query":"{ __type(name: \"Evidence\") { fields { name } } }"}`, manager.queries[0])
	}
}

func TestEnsureEvidenceRevocationSupported_FieldMissing(t *testing.T) {
	manager := &mockIntrospectionManager{
		response: []byte(`{"data":{"__type":{"fields":[{"name":"id"},{"name":"verified"},{"name":"revocation"}]}}}`),
	}

	err := EnsureEvidenceRevocationSupported(manager)
	assert.ErrorIs(t, err, ErrEvidenceRevocationNotSupported)
	assert.Contains(t, err.Error(), "--include-revoked requires an Evidence service version that supports revocation")
}

func TestEnsureEvidenceRevocationSupported_TypeMissing(t *testing.T) {
	manager := &mockIntrospectionManager{response: []byte(`{"data":{"__type":null}}`)}

	assert.ErrorIs(t, EnsureEvidenceRevocationSupported(manager), ErrEvidenceRevocationNotSupported)
}

func TestEnsureEvidenceRevocationSupported_GraphqlErrors(t *testing.T) {
	manager := &mockIntrospectionManager{
		response: []byte(`{"data":null,"errors":[{"message":"introspection is disabled"}]}`),
	}

	err := EnsureEvidenceRevocationSupported(manager)
	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrEvidenceRevocationNotSupported)
	assert.Contains(t, err.Error(), "failed to detect Evidence revocation support: introspection is disabled")
}

func TestEnsureEvidenceRevocationSupported_TransportError(t *testing.T) {
	manager := &mockIntrospectionManager{err: errors.New("HTTP 503: Service Unavailable")}

	err := EnsureEvidenceRevocationSupported(manager)
	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrEvidenceRevocationNotSupported)
	assert.Contains(t, err.Error(), "HTTP 503")
}

func TestEnsureEvidenceRevocationSupported_InvalidJson(t *testing.T) {
	manager := &mockIntrospectionManager{response: []byte(`not json`)}

	err := EnsureEvidenceRevocationSupported(manager)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to detect Evidence revocation support")
}
