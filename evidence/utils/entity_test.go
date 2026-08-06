package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildGraphQLEntityHasSubjectWith(t *testing.T) {
	tests := []struct {
		name           string
		entityType     string
		entityID       string
		entityRepo     string
		projectKey     string
		applicationKey string
		want           string
	}{
		{
			name:       "project scope",
			entityType: "gitCommit",
			entityID:   "abc123",
			projectKey: "proj",
			want:       `entityType: \"gitCommit\", entityId: \"abc123\", projectKey: \"proj\"`,
		},
		{
			name:       "entity repo scope",
			entityType: "languageModel",
			entityID:   "model-1",
			entityRepo: "models-entity",
			want:       `entityType: \"languageModel\", entityId: \"model-1\", repositoryKey: \"models-entity\"`,
		},
		{
			name:           "application scope",
			entityType:     "gitCommit",
			entityID:       "abc123",
			applicationKey: "my-app",
			want:           `entityType: \"gitCommit\", entityId: \"abc123\", applicationKey: \"my-app\"`,
		},
		{
			name:       "no scope",
			entityType: "gitCommit",
			entityID:   "abc123",
			want:       `entityType: \"gitCommit\", entityId: \"abc123\"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, BuildGraphQLEntityHasSubjectWith(tt.entityType, tt.entityID, tt.entityRepo, tt.projectKey, tt.applicationKey))
		})
	}
}

func TestEscapeGraphQLString(t *testing.T) {
	assert.Equal(t, `a\\\"b`, escapeGraphQLString(`a\"b`))
}
