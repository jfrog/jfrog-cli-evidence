package flags

import (
	"testing"

	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func findFlag(cmdKey, name string) (components.Flag, bool) {
	for _, flag := range GetCommandFlags(cmdKey) {
		if flag.GetName() == name {
			return flag, true
		}
	}
	return nil, false
}

func TestIncludeRevokedFlag_RegisteredOnGetOnly(t *testing.T) {
	flag, found := findFlag(GetEvidence, IncludeRevoked)
	require.True(t, found, "get must expose --%s", IncludeRevoked)
	boolFlag, ok := flag.(components.BoolFlag)
	require.True(t, ok, "--%s must be a boolean flag", IncludeRevoked)
	assert.False(t, boolFlag.DefaultValue, "--%s must default to false", IncludeRevoked)

	for _, cmdKey := range []string{VerifyEvidence, CreateEvidence, GenerateKeyPair} {
		_, found := findFlag(cmdKey, IncludeRevoked)
		assert.False(t, found, "%s must not expose --%s", cmdKey, IncludeRevoked)
	}
}
