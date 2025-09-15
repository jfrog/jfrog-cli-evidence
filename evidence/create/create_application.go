package create

import (
	"fmt"

	"github.com/jfrog/jfrog-cli-core/v2/artifactory/utils/commandsummary"

	artifactoryUtils "github.com/jfrog/jfrog-cli-core/v2/artifactory/utils"
	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli-evidence/evidence"
	"github.com/jfrog/jfrog-cli-evidence/evidence/model"
	"github.com/jfrog/jfrog-cli-evidence/evidence/utils"
	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/utils/log"
)

type createEvidenceApplication struct {
	createEvidenceBase
	applicationKey     string
	applicationVersion string
	projectKey         string
}

func NewCreateEvidenceApplication(serverDetails *config.ServerDetails, predicateFilePath, predicateType, markdownFilePath, key, keyId, applicationKey,
	applicationVersion, providerId, integration string) evidence.Command {
	return &createEvidenceApplication{
		createEvidenceBase: createEvidenceBase{
			serverDetails:     serverDetails,
			predicateFilePath: predicateFilePath,
			predicateType:     predicateType,
			markdownFilePath:  markdownFilePath,
			key:               key,
			keyId:             keyId,
			providerId:        providerId,
			stage:             getApplicationVersionStage(serverDetails, applicationKey, applicationVersion),
			integration:       integration,
		},
		applicationKey:     applicationKey,
		applicationVersion: applicationVersion,
	}
}

func (c *createEvidenceApplication) CommandName() string {
	return "create-application-evidence"
}

func (c *createEvidenceApplication) ServerDetails() (*config.ServerDetails, error) {
	return c.serverDetails, nil
}

func (c *createEvidenceApplication) Run() error {
	// Get project key from application details
	err := c.fetchProjectKey()
	if err != nil {
		return err
	}

	artifactoryClient, err := c.createArtifactoryClient()
	if err != nil {
		log.Error("failed to create Artifactory client", err)
		return err
	}
	subject, sha256, err := c.buildApplicationSubjectPath(artifactoryClient)
	if err != nil {
		return err
	}
	envelope, err := c.createEnvelope(subject, sha256)
	if err != nil {
		return err
	}
	response, err := c.uploadEvidence(envelope, subject)
	if err != nil {
		return err
	}
	c.recordSummary(response, subject, sha256)

	return nil
}

func (c *createEvidenceApplication) fetchProjectKey() error {
	apptrustServiceManager, err := artifactoryUtils.CreateApptrustServiceManager(c.serverDetails, false)
	if err != nil {
		return fmt.Errorf("failed to create apptrust service manager: %w", err)
	}

	applicationDetails, err := apptrustServiceManager.GetApplicationDetails(c.applicationKey)
	if err != nil {
		return fmt.Errorf("failed to get application details for %s: %w", c.applicationKey, err)
	}

	c.projectKey = applicationDetails.ProjectKey
	log.Debug("Retrieved project key from application:", c.projectKey)
	return nil
}

func (c *createEvidenceApplication) buildApplicationSubjectPath(artifactoryClient artifactory.ArtifactoryServicesManager) (string, string, error) {
	repoKey := utils.BuildApplicationVersionRepoKey(c.projectKey)
	manifestPath := buildApplicationManifestPath(repoKey, c.applicationKey, c.applicationVersion)

	manifestChecksum, err := c.getFileChecksum(manifestPath, artifactoryClient)
	if err != nil {
		return "", "", err
	}

	return manifestPath, manifestChecksum, nil
}

func (c *createEvidenceApplication) recordSummary(response *model.CreateResponse, subject string, sha256 string) {
	displayName := fmt.Sprintf("%s %s", c.applicationKey, c.applicationVersion)
	commandSummary := commandsummary.EvidenceSummaryData{
		Subject:       subject,
		SubjectSha256: sha256,
		PredicateType: c.predicateType,
		PredicateSlug: response.PredicateSlug,
		Verified:      response.Verified,
		DisplayName:   displayName,
		SubjectType:   commandsummary.SubjectTypeApplication,
		RepoKey:       utils.BuildApplicationVersionRepoKey(c.projectKey),
	}
	err := c.recordEvidenceSummary(commandSummary)
	if err != nil {
		log.Warn("Failed to record evidence summary:", err.Error())
	}
}

func buildApplicationManifestPath(repoKey, applicationKey, applicationVersion string) string {
	return fmt.Sprintf("%s/%s/%s/application-version.json.evd", repoKey, applicationKey, applicationVersion)
}

func getApplicationVersionStage(serverDetails *config.ServerDetails, applicationKey, applicationVersion string) string {
	log.Debug("fetching application version %s:%s stage", applicationKey, applicationVersion)
	apptrustServiceManager, err := artifactoryUtils.CreateApptrustServiceManager(serverDetails, false)
	if err != nil {
		log.Warn("Failed to create apptrust service manager:", err)
		return ""
	}

	queryParams := make(map[string]string)
	// Order by created descending to get the latest promotions first
	queryParams["order_by"] = "created"
	queryParams["order_asc"] = "false"

	promotionsResponse, err := apptrustServiceManager.GetApplicationVersionPromotions(applicationKey, applicationVersion, queryParams)
	if err != nil {
		log.Warn("Failed to get application version promotions:", err)
		return ""
	}

	if promotionsResponse != nil && len(promotionsResponse.Promotions) > 0 {
		for _, promotion := range promotionsResponse.Promotions {
			if promotion.Status == "COMPLETED" {
				return promotion.TargetStage
			}
		}
	}

	return ""
}
