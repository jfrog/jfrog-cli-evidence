package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildApplicationVersionRepoKey(t *testing.T) {
	tests := []struct {
		name     string
		project  string
		expected string
	}{
		{
			name:     "Normal project",
			project:  "my-project",
			expected: "my-project-application-versions",
		},
		{
			name:     "Default project",
			project:  "default",
			expected: "application-versions",
		},
		{
			name:     "Empty project",
			project:  "",
			expected: "application-versions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildApplicationVersionRepoKey(tt.project)
			assert.Equal(t, tt.expected, result)
		})
	}
}
