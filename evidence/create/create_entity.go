package create

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jfrog/jfrog-cli-core/v2/artifactory/utils/commandsummary"
	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli-evidence/evidence"
	"github.com/jfrog/jfrog-cli-evidence/evidence/client"
	"github.com/jfrog/jfrog-cli-evidence/evidence/intoto"
	"github.com/jfrog/jfrog-cli-evidence/evidence/model"
	"github.com/jfrog/jfrog-cli-evidence/evidence/sigstore"
	evidenceUtils "github.com/jfrog/jfrog-cli-evidence/evidence/utils"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"github.com/jfrog/jfrog-client-go/utils/log"
)

type createEvidenceEntity struct {
	createEvidenceBase
	model.EntitySubject
}

func NewCreateEvidenceEntity(serverDetails *config.ServerDetails, predicateFilePath, predicateType, markdownFilePath, key, keyId,
	entityType, entityID, entityRepo, projectKey, applicationKey, providerId, sigstoreBundlePath,
	attachLocalPath, attachArtifactoryTempPath, attachArtifactoryPath string) evidence.Command {
	return &createEvidenceEntity{
		createEvidenceBase: createEvidenceBase{
			serverDetails:             serverDetails,
			predicateFilePath:         predicateFilePath,
			predicateType:             predicateType,
			markdownFilePath:          markdownFilePath,
			key:                       key,
			keyId:                     keyId,
			providerId:                providerId,
			sigstoreBundlePath:        sigstoreBundlePath,
			attachLocalPath:           attachLocalPath,
			attachArtifactoryTempPath: attachArtifactoryTempPath,
			attachArtifactoryPath:     attachArtifactoryPath,
		},
		EntitySubject: model.EntitySubject{
			EntityType:     entityType,
			EntityID:       entityID,
			EntityRepo:     entityRepo,
			ProjectKey:     projectKey,
			ApplicationKey: applicationKey,
		},
	}
}

func (c *createEvidenceEntity) CommandName() string {
	return "create-entity-evidence"
}

func (c *createEvidenceEntity) ServerDetails() (*config.ServerDetails, error) {
	return c.serverDetails, nil
}

func (c *createEvidenceEntity) Run() error {
	if err := evidenceUtils.ResolveApplicationEntityProjectKey(c.serverDetails, &c.EntitySubject); err != nil {
		return err
	}

	if c.sigstoreBundlePath != "" {
		return c.runWithSigstoreBundle()
	}
	return c.runWithClientSignedEnvelope()
}

func (c *createEvidenceEntity) runWithSigstoreBundle() error {
	log.Info("Reading sigstore bundle from path:", c.sigstoreBundlePath)
	if _, err := sigstore.ParseBundle(c.sigstoreBundlePath); err != nil {
		return errorutils.CheckErrorf("failed to read sigstore bundle: %s", err.Error())
	}
	payload, err := os.ReadFile(c.sigstoreBundlePath)
	if err != nil {
		return errorutils.CheckError(err)
	}

	response, err := c.createEntityEvidence(payload)
	if err != nil {
		return err
	}
	c.recordSummary(response)
	return nil
}

func (c *createEvidenceEntity) runWithClientSignedEnvelope() error {
	attachment, cleanup, err := c.resolveEntityAttachment()
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}

	log.Info("Creating DSSE envelope for entity:", c.EntityType, c.EntityID)
	envelopeBytes, err := c.createEntityEnvelope(attachment)
	if err != nil {
		return err
	}
	envelopeBytes, err = c.wrapCreatePayloadWithAttachments(envelopeBytes, attachment)
	if err != nil {
		return err
	}

	response, err := c.createEntityEvidence(envelopeBytes)
	if err != nil {
		return err
	}
	c.recordSummary(response)
	return nil
}

func (c *createEvidenceEntity) resolveEntityAttachment() (*statementAttachment, func(), error) {
	if c.attachLocalPath == "" && c.attachArtifactoryPath == "" {
		return nil, nil, nil
	}
	artifactoryClient, err := c.createArtifactoryClient()
	if err != nil {
		log.Error("failed to create Artifactory client", err)
		return nil, nil, err
	}
	return c.resolveAttachment(artifactoryClient)
}

func (c *createEvidenceEntity) createEntityEnvelope(attachment *statementAttachment) ([]byte, error) {
	statementJson, err := c.buildEntityStatement(attachment)
	if err != nil {
		return nil, err
	}
	signedEnvelope, err := createAndSignEnvelope(statementJson, c.key, c.keyId)
	if err != nil {
		return nil, err
	}
	return json.Marshal(signedEnvelope)
}

func (c *createEvidenceEntity) buildEntityStatement(attachment *statementAttachment) ([]byte, error) {
	predicate, err := os.ReadFile(c.predicateFilePath)
	if err != nil {
		return nil, errorutils.CheckError(err)
	}

	user := EvdDefaultUser
	if c.serverDetails != nil && c.serverDetails.User != "" {
		user = c.serverDetails.User
	}

	statement := intoto.NewStatement(predicate, c.predicateType, user)
	if err = c.setMarkdown(statement); err != nil {
		return nil, err
	}
	if err = statement.SetEntitySubject(c.EntityType, c.EntityID); err != nil {
		return nil, err
	}
	statement.SetAttachments(toStatementAttachmentMeta(attachment))
	statementJson, err := statement.Marshal()
	if err != nil {
		log.Error("failed marshaling statement json file", err)
		return nil, err
	}
	return statementJson, nil
}

func (c *createEvidenceEntity) createEntityEvidence(payload []byte) (*model.CreateResponse, error) {
	if c.evidenceServiceClient == nil {
		evidenceServiceClient, err := client.NewEvidenceClient(c.serverDetails)
		if err != nil {
			return nil, err
		}
		c.evidenceServiceClient = evidenceServiceClient
	}

	body, err := c.evidenceServiceClient.CreateEntityEvidence(client.CreateEntityEvidenceRequest{
		EntityType:     c.EntityType,
		EntityID:       c.EntityID,
		EntityRepo:     c.EntityRepo,
		ProjectKey:     c.ProjectKey,
		ApplicationKey: c.ApplicationKey,
		ProviderID:     c.providerId,
	}, payload)
	if err != nil {
		return nil, err
	}
	return c.collectCreateResponse(body)
}

func (c *createEvidenceEntity) recordSummary(response *model.CreateResponse) {
	if !evidenceUtils.IsRunningUnderGitHubAction() {
		return
	}
	applicationKey := c.ApplicationKey
	subjectType := commandsummary.SubjectTypeArtifact
	if c.EntityType == "application" {
		applicationKey = c.EntityID
		subjectType = commandsummary.SubjectTypeApplication
	}
	err := c.recordEvidenceSummary(commandsummary.EvidenceSummaryData{
		Subject:        fmt.Sprintf("%s/%s", c.EntityType, c.EntityID),
		PredicateType:  c.predicateType,
		PredicateSlug:  response.PredicateSlug,
		Verified:       response.Verified,
		DisplayName:    fmt.Sprintf("%s %s", c.EntityType, c.EntityID),
		SubjectType:    subjectType,
		RepoKey:        c.EntityRepo,
		ApplicationKey: applicationKey,
	})
	if err != nil {
		log.Warn("Failed to record evidence summary:", err.Error())
	}
}
