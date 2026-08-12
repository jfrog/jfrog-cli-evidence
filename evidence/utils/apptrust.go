package utils

import (
	"fmt"

	artifactoryUtils "github.com/jfrog/jfrog-cli-core/v2/artifactory/utils"
	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli-evidence/evidence/model"
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

// ResolveApplicationEntityProjectKey resolves project scope for bare application-entity subjects
// (entity-type=application with no --project/--entity-repo/--application-key scope already set).
// When resolution is not needed, subject is left unchanged.
func ResolveApplicationEntityProjectKey(serverDetails *config.ServerDetails, subject *model.EntitySubject) error {
	if subject == nil {
		return nil
	}
	if subject.EntityType != "application" || subject.ProjectKey != "" || subject.EntityRepo != "" || subject.ApplicationKey != "" {
		return nil
	}
	projectKey, err := ResolveApplicationProjectKey(serverDetails, subject.EntityID)
	if err != nil {
		return err
	}
	subject.ProjectKey = projectKey
	return nil
}
