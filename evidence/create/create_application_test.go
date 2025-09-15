package create

import (
	"testing"

	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/stretchr/testify/assert"
)

func TestNewCreateEvidenceApplication(t *testing.T) {
	serverDetails := &config.ServerDetails{
		Url:         "http://localhost:8081/",
		AccessToken: "test-token",
	}

	cmd := NewCreateEvidenceApplication(
		serverDetails,
		"/path/to/predicate.json",
		"test-predicate-type",
		"/path/to/markdown.md",
		"/path/to/key",
		"test-key-id",
		"test-app-key",
		"1.0.0",
		"test-provider-id",
		"",
	)

	assert.NotNil(t, cmd)
	
	appCmd, ok := cmd.(*createEvidenceApplication)
	assert.True(t, ok)
	assert.Equal(t, "test-app-key", appCmd.applicationKey)
	assert.Equal(t, "1.0.0", appCmd.applicationVersion)
	assert.Equal(t, "create-application-evidence", appCmd.CommandName())
	
	retrievedServerDetails, err := appCmd.ServerDetails()
	assert.NoError(t, err)
	assert.Equal(t, serverDetails, retrievedServerDetails)
}

func TestBuildApplicationManifestPath(t *testing.T) {
	repoKey := "test-project-application-versions"
	applicationKey := "my-app"
	applicationVersion := "1.2.3"
	
	expected := "test-project-application-versions/my-app/1.2.3/application-version.json.evd"
	actual := buildApplicationManifestPath(repoKey, applicationKey, applicationVersion)
	
	assert.Equal(t, expected, actual)
}
