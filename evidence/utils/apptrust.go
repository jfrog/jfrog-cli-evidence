package utils

import (
	"fmt"

	artifactoryUtils "github.com/jfrog/jfrog-cli-core/v2/artifactory/utils"
	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
)

// ResolveApplicationProjectKey looks up an application in AppTrust and returns its project key.
func ResolveApplicationProjectKey(serverDetails *config.ServerDetails, applicationKey string) (string, error) {
	apptrustServiceManager, err := artifactoryUtils.CreateApptrustServiceManager(serverDetails, false)
	if err != nil {
		return "", fmt.Errorf("failed to create apptrust service manager: %w", err)
	}
	applicationDetails, err := apptrustServiceManager.GetApplicationDetails(applicationKey)
	if err != nil {
		return "", fmt.Errorf("failed to get application details for %s: %w", applicationKey, err)
	}
	return applicationDetails.ProjectKey, nil
}
