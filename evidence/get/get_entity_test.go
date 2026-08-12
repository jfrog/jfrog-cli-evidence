package get

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli-evidence/evidence/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockOnemodelManagerEntitySuccess struct {
	lastQuery []byte
}

func (m *mockOnemodelManagerEntitySuccess) GraphqlQuery(query []byte) ([]byte, error) {
	m.lastQuery = append([]byte{}, query...)
	response := `{"data":{"evidence":{"searchEvidence":{"totalCount":1,"edges":[{"cursor":"1","node":{"predicateSlug":"test-slug","downloadPath":"gitCommit-entity/.evidence/abc/e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855/evd.json","verified":true,"signingKey":{"alias":"test-alias"},"subject":{"sha256":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855","fullPath":"gitCommit-entity/.entities/gitCommit/abc1/abc123","path":".entities/gitCommit/abc1","name":"abc123","repositoryKey":"gitCommit-entity"},"createdBy":"test-user","createdAt":"2024-01-01T00:00:00Z"}}]}}}}`
	return []byte(response), nil
}

type mockOnemodelManagerEntityError struct{}

func (m *mockOnemodelManagerEntityError) GraphqlQuery(_ []byte) ([]byte, error) {
	return nil, fmt.Errorf("HTTP %d: Not Found", http.StatusNotFound)
}

func TestGetEvidenceEntity_Transform(t *testing.T) {
	cmd := &getEvidenceEntity{
		getEvidenceBase: getEvidenceBase{includePredicate: false},
		EntitySubject: model.EntitySubject{
			EntityType: "gitCommit",
			EntityID:   "abc123",
			ProjectKey: "proj",
		},
	}
	mock := &mockOnemodelManagerEntitySuccess{}
	out, err := cmd.getEvidence(mock)
	require.NoError(t, err)
	assert.Contains(t, string(mock.lastQuery), `entityType: \"gitCommit\"`)
	assert.Contains(t, string(mock.lastQuery), `entityId: \"abc123\"`)
	assert.Contains(t, string(mock.lastQuery), `projectKey: \"proj\"`)
	assert.Contains(t, string(mock.lastQuery), "fullPath")

	var parsed EntityEvidenceOutput
	require.NoError(t, json.Unmarshal(out, &parsed))
	assert.Equal(t, SchemaVersion, parsed.SchemaVersion)
	assert.Equal(t, EntityType, parsed.Type)
	assert.Equal(t, "gitCommit", parsed.Result.EntityType)
	assert.Equal(t, "abc123", parsed.Result.EntityId)
	assert.Equal(t, "proj", parsed.Result.Project)
	require.Len(t, parsed.Result.Evidence, 1)
	assert.Equal(t, "test-slug", parsed.Result.Evidence[0].PredicateSlug)
	assert.Equal(t, "gitCommit-entity/.entities/gitCommit/abc1/abc123", parsed.Result.Evidence[0].Subject["fullPath"])
}

func TestGetEvidenceEntity_QueryError(t *testing.T) {
	cmd := &getEvidenceEntity{
		getEvidenceBase: getEvidenceBase{serverDetails: &config.ServerDetails{}},
		EntitySubject: model.EntitySubject{
			EntityType: "gitCommit",
			EntityID:   "abc123",
		},
	}
	_, err := cmd.getEvidence(&mockOnemodelManagerEntityError{})
	require.Error(t, err)
}

func TestNewGetEvidenceEntity(t *testing.T) {
	cmd := NewGetEvidenceEntity(&config.ServerDetails{Url: "https://example.jfrog.io/"}, "gitCommit", "abc", "", "proj", "", "json", "", false)
	entityCmd, ok := cmd.(*getEvidenceEntity)
	require.True(t, ok)
	assert.Equal(t, "get-entity-evidence", entityCmd.CommandName())
	assert.Equal(t, "gitCommit", entityCmd.EntityType)
	assert.Equal(t, "abc", entityCmd.EntityID)
	assert.Equal(t, "proj", entityCmd.ProjectKey)
}
