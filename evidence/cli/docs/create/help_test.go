package create

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDescription(t *testing.T) {
	description := GetDescription()
	assert.NotEmpty(t, description)
	assert.Contains(t, description, "evidence")
}

func TestGetArguments(t *testing.T) {
	args := GetArguments()
	assert.NotNil(t, args)
	assert.Empty(t, args) // Currently returns empty slice
}
