package get

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

func TestGetUsageExamples(t *testing.T) {
	examples := GetUsageExamples()
	assert.NotEmpty(t, examples)
	assert.Contains(t, examples[0], "evd get --entity-type")
}
