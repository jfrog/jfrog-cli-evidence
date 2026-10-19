package create

import (
	"fmt"

	"github.com/jfrog/jfrog-cli-evidence/evidence/model"

	"github.com/jfrog/jfrog-cli-core/v2/artifactory/utils/commandsummary"

	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli-evidence/evidence"
	"github.com/jfrog/jfrog-cli-evidence/evidence/utils"
	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/utils/log"
)

type createEvidenceBuild struct {
	createEvidenceBase
	project     string
	buildName   string
	buildNumber string
}

func NewCreateEvidenceBuild(serverDetails *config.ServerDetails,
	predicateFilePath, predicateType, markdownFilePath, key, keyId, project, buildName, buildNumber, providerId, integration, attachLocalPath, attachArtifactoryTempPath, attachArtifactoryPath string) evidence.Command {
	return &createEvidenceBuild{
		createEvidenceBase: createEvidenceBase{
			serverDetails:             serverDetails,
			predicateFilePath:         predicateFilePath,
			predicateType:             predicateType,
			markdownFilePath:          markdownFilePath,
			key:                       key,
			keyId:                     keyId,
			providerId:                providerId,
			integration:               integration,
			attachLocalPath:           attachLocalPath,
			attachArtifactoryTempPath: attachArtifactoryTempPath,
			attachArtifactoryPath:     attachArtifactoryPath,
		},
		project:     project,
		buildName:   buildName,
		buildNumber: buildNumber,
	}
}

func (c *createEvidenceBuild) CommandName() string {
	return "create-buildName-evidence"
}

func (c *createEvidenceBuild) ServerDetails() (*config.ServerDetails, error) {
	return c.serverDetails, nil
}

func (c *createEvidenceBuild) Run() error {
	artifactoryClient, err := c.createArtifactoryClient()
	if err != nil {
		log.Error("failed to create Artifactory client", err)
		return err
	}

	attachment, cleanup, err := c.resolveAttachment(artifactoryClient)
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}

	timestamp, err := utils.GetBuildLatestTimestamp(c.buildName, c.buildNumber, c.project, artifactoryClient)
	if err != nil {
		return err
	}

	subject, sha256, err := c.buildBuildInfoSubjectPath(artifactoryClient, timestamp)
	if err != nil {
		return err
	}
	envelope, err := c.createEnvelope(subject, sha256, attachment)
	if err != nil {
		return err
	}

	response, err := c.uploadEvidence(envelope, subject, attachment)
	if err != nil {
		return err
	}
	c.recordSummary(subject, sha256, response, timestamp)

	return nil
}

func (c *createEvidenceBuild) buildBuildInfoSubjectPath(artifactoryClient artifactory.ArtifactoryServicesManager, timestamp string) (string, string, error) {
	repoKey := utils.BuildBuildInfoRepoKey(c.project)
	buildInfoPath := buildBuildInfoPath(repoKey, c.buildName, c.buildNumber, timestamp)
	buildInfoChecksum, err := getBuildInfoPathChecksum(buildInfoPath, artifactoryClient)
	if err != nil {
		return "", "", err
	}
	return buildInfoPath, buildInfoChecksum, nil
}

func (c *createEvidenceBuild) recordSummary(subject string, sha256 string, response *model.CreateResponse, timestamp string) {
	displayName := fmt.Sprintf("%s %s", c.buildName, c.buildNumber)
	commandSummary := commandsummary.EvidenceSummaryData{
		Subject:        subject,
		SubjectSha256:  sha256,
		PredicateType:  c.predicateType,
		PredicateSlug:  response.PredicateSlug,
		Verified:       response.Verified,
		DisplayName:    displayName,
		SubjectType:    commandsummary.SubjectTypeBuild,
		BuildName:      c.buildName,
		BuildNumber:    c.buildNumber,
		BuildTimestamp: timestamp,
		RepoKey:        utils.BuildBuildInfoRepoKey(c.project),
	}

	err := c.recordEvidenceSummary(commandSummary)
	if err != nil {
		log.Warn("Failed to record evidence summary:", err.Error())
	}
}

func buildBuildInfoPath(repoKey string, name string, number string, timestamp string) string {
	jsonFile := utils.BuildInfoFileName(number, timestamp)
	return fmt.Sprintf("%s/%s/%s", repoKey, name, jsonFile)
}

func getBuildInfoPathChecksum(buildInfoPath string, artifactoryClient artifactory.ArtifactoryServicesManager) (string, error) {
	res, err := artifactoryClient.FileInfo(buildInfoPath)
	if err != nil {
		log.Warn(fmt.Sprintf("buildName info json path '%s' does not exist.", buildInfoPath))
		return "", err
	}
	return res.Checksums.Sha256, nil
}
