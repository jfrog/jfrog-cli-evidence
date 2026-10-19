package verify

import (
	"errors"
	"fmt"

	"github.com/jfrog/jfrog-cli-evidence/evidence"
	"github.com/jfrog/jfrog-cli-evidence/evidence/model"
	"github.com/jfrog/jfrog-cli-evidence/evidence/utils"

	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-client-go/artifactory"
)

const aqlBuildInfoFileQueryTemplate = `items.find({"repo":%s,"path":%s,"name":%s}).include("repo","path","name","sha256")`

// verifyEvidenceBuild verifies evidence for a build.
type verifyEvidenceBuild struct {
	verifyEvidenceBase
	project     string
	buildName   string
	buildNumber string
}

// NewVerifyEvidenceBuild creates a new command for verifying evidence for a build.
func NewVerifyEvidenceBuild(serverDetails *config.ServerDetails, project, buildName, buildNumber, format string, keys []string, useArtifactoryKeys bool) evidence.Command {
	return &verifyEvidenceBuild{
		verifyEvidenceBase: newVerifyEvidenceBase(serverDetails, format, keys, useArtifactoryKeys),
		project:            project,
		buildName:          buildName,
		buildNumber:        buildNumber,
	}
}

// Run executes the build evidence verification command.
func (v *verifyEvidenceBuild) Run() error {
	defer v.quitProgress()

	client, err := v.createArtifactoryClient()
	if err != nil {
		return fmt.Errorf("failed to create Artifactory client: %w", err)
	}

	repoKey := utils.BuildBuildInfoRepoKey(v.project)

	v.setHeadline("Searching build")
	subjectFileName, buildInfoSha256, err := v.findBuildInfoFile(client, repoKey)
	if err != nil {
		return err
	}

	metadata, err := v.queryEvidenceMetadata(repoKey, v.buildName, subjectFileName)
	if err != nil {
		return err
	}

	subjectPath := fmt.Sprintf("%s/%s/%s", repoKey, v.buildName, subjectFileName)
	return v.verifyEvidence(client, metadata, model.SubjectDigest{Type: model.Sha256DigestType, Value: buildInfoSha256}, subjectPath)
}

// findBuildInfoFile returns the name and sha256 of the requested build's build-info file.
// It resolves the file the same way evidence creation does: the build-info API returns the start time
// of the latest run, which names the file "<number>-<timestamp>.json". The file is then looked up by
// that exact name, so a build with a different number is never selected.
func (v *verifyEvidenceBuild) findBuildInfoFile(client *artifactory.ArtifactoryServicesManager, repoKey string) (string, string, error) {
	timestamp, err := utils.GetBuildLatestTimestamp(v.buildName, v.buildNumber, v.project, *client)
	if err != nil {
		var notFound *utils.BuildNotFoundError
		if errors.As(err, &notFound) {
			return "", "", fmt.Errorf("no build found for build name %q and build number %q", v.buildName, v.buildNumber)
		}
		return "", "", err
	}

	fileName := utils.BuildInfoFileName(v.buildNumber, timestamp)
	query := fmt.Sprintf(aqlBuildInfoFileQueryTemplate, utils.QuoteAqlString(repoKey), utils.QuoteAqlString(v.buildName), utils.QuoteAqlString(fileName))
	result, err := utils.ExecuteAqlQuery(query, client)
	if err != nil {
		return "", "", fmt.Errorf("failed to execute AQL query: %w", err)
	}
	for _, item := range result.Results {
		if item.Name == fileName {
			return item.Name, item.Sha256, nil
		}
	}
	return "", "", fmt.Errorf("no build found for build name %q and build number %q: build-info file %s/%s/%s does not exist", v.buildName, v.buildNumber, repoKey, v.buildName, fileName)
}

// ServerDetails returns the server details for the command.
func (v *verifyEvidenceBuild) ServerDetails() (*config.ServerDetails, error) {
	return v.serverDetails, nil
}

// CommandName returns the command name for build evidence verification.
func (v *verifyEvidenceBuild) CommandName() string {
	return "verify-evidence-build"
}
