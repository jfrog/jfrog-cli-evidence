package create

import (
	"fmt"

	"github.com/jfrog/jfrog-cli-core/v2/artifactory/utils/commandsummary"
	"github.com/jfrog/jfrog-client-go/metadata"

	"github.com/jfrog/jfrog-cli-core/v2/artifactory/utils"
	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli-evidence/evidence"
	"github.com/jfrog/jfrog-cli-evidence/evidence/model"
	"github.com/jfrog/jfrog-client-go/utils/log"
)

type createEvidencePackage struct {
	createEvidenceBase
	packageService evidence.PackageService
	metadataClient metadata.Manager
}

func NewCreateEvidencePackage(serverDetails *config.ServerDetails, predicateFilePath, predicateType, markdownFilePath, key, keyId, packageName,
	packageVersion, packageRepoName, providerId, integration string) evidence.Command {
	return &createEvidencePackage{
		createEvidenceBase: createEvidenceBase{
			serverDetails:     serverDetails,
			predicateFilePath: predicateFilePath,
			predicateType:     predicateType,
			markdownFilePath:  markdownFilePath,
			providerId:        providerId,
			key:               key,
			keyId:             keyId,
			integration:       integration,
		},
		packageService: evidence.NewPackageService(packageName, packageVersion, packageRepoName),
	}
}

func (c *createEvidencePackage) CommandName() string {
	return "create-package-evidence"
}

func (c *createEvidencePackage) ServerDetails() (*config.ServerDetails, error) {
	return c.serverDetails, nil
}

func (c *createEvidencePackage) Run() error {
	artifactoryClient, err := c.createArtifactoryClient()
	if err != nil {
		log.Error("failed to create Artifactory client", err)
		return err
	}
	if c.metadataClient == nil {
		c.metadataClient, err = utils.CreateMetadataServiceManager(c.serverDetails, false)
		if err != nil {
			return err
		}
	}

	packageType, err := c.packageService.GetPackageType(artifactoryClient)
	if err != nil {
		return err
	}

	leadArtifactPath, err := c.packageService.GetPackageVersionLeadArtifact(packageType, c.metadataClient, artifactoryClient)
	if err != nil {
		return err
	}

	leadArtifactChecksum, err := c.getFileChecksum(leadArtifactPath, artifactoryClient)
	if err != nil {
		return err
	}
	envelope, err := c.createEnvelope(leadArtifactPath, leadArtifactChecksum)
	if err != nil {
		return err
	}
	response, err := c.uploadEvidence(envelope, leadArtifactPath)
	if err != nil {
		return err
	}
	c.recordSummary(response, leadArtifactPath, leadArtifactChecksum)

	return nil
}

func (c *createEvidencePackage) recordSummary(response *model.CreateResponse, leadArtifactPath string, leadArtifactChecksum string) {
	displayName := fmt.Sprintf("%s %s", c.packageService.GetPackageName(), c.packageService.GetPackageVersion())
	commandSummary := commandsummary.EvidenceSummaryData{
		Subject:       leadArtifactPath,
		SubjectSha256: leadArtifactChecksum,
		PredicateType: c.predicateType,
		PredicateSlug: response.PredicateSlug,
		Verified:      response.Verified,
		DisplayName:   displayName,
		SubjectType:   commandsummary.SubjectTypePackage,
		RepoKey:       c.packageService.GetPackageRepoName(),
	}
	err := c.recordEvidenceSummary(commandSummary)
	if err != nil {
		log.Warn("Failed to record evidence summary:", err.Error())
	}
}
