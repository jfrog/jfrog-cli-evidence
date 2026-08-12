package utils

import (
	"testing"

	"github.com/jfrog/jfrog-cli-evidence/evidence/model"
	"github.com/stretchr/testify/assert"
)

func TestBuildGraphQLEntityHasSubjectWith(t *testing.T) {
	tests := []struct {
		name    string
		subject model.EntitySubject
		want    string
	}{
		{
			name: "project scope",
			subject: model.EntitySubject{
				EntityType: "gitCommit",
				EntityID:   "abc123",
				ProjectKey: "proj",
			},
			want: `entityType: \"gitCommit\", entityId: \"abc123\", projectKey: \"proj\"`,
		},
		{
			name: "entity repo scope",
			subject: model.EntitySubject{
				EntityType: "languageModel",
				EntityID:   "model-1",
				EntityRepo: "models-entity",
			},
			want: `entityType: \"languageModel\", entityId: \"model-1\", repositoryKey: \"models-entity\"`,
		},
		{
			name: "application scope",
			subject: model.EntitySubject{
				EntityType:     "gitCommit",
				EntityID:       "abc123",
				ApplicationKey: "my-app",
			},
			want: `entityType: \"gitCommit\", entityId: \"abc123\", applicationKey: \"my-app\"`,
		},
		{
			name: "no scope",
			subject: model.EntitySubject{
				EntityType: "gitCommit",
				EntityID:   "abc123",
			},
			want: `entityType: \"gitCommit\", entityId: \"abc123\"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, BuildGraphQLEntityHasSubjectWith(tt.subject))
		})
	}
}

func TestEscapeGraphQLString(t *testing.T) {
	assert.Equal(t, `a\\\"b`, escapeGraphQLString(`a\"b`))
}

func TestResolveApplicationEntityProjectKey_SkipsWhenNotNeeded(t *testing.T) {
	subject := &model.EntitySubject{
		EntityType: "gitCommit",
		EntityID:   "abc123",
	}
	assert.NoError(t, ResolveApplicationEntityProjectKey(nil, subject))
	assert.Empty(t, subject.ProjectKey)

	subject = &model.EntitySubject{
		EntityType: "application",
		EntityID:   "my-app",
		ProjectKey: "already-set",
	}
	assert.NoError(t, ResolveApplicationEntityProjectKey(nil, subject))
	assert.Equal(t, "already-set", subject.ProjectKey)

	assert.NoError(t, ResolveApplicationEntityProjectKey(nil, nil))
}
