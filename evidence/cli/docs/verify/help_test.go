package verify

import (
	"strings"
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
	assert.True(t, strings.HasPrefix(examples[0], "jf evd verify"))
	assert.Contains(t, strings.Join(examples, "\n"), "jf evd verify --entity-type")
}
