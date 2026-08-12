package create

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/jfrog/jfrog-cli-core/v2/artifactory/utils/commandsummary"
	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli-evidence/evidence"
	"github.com/jfrog/jfrog-cli-evidence/evidence/client"
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
	return c.runWithPrepare()
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

func (c *createEvidenceEntity) runWithPrepare() error {
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

	prepareResponse, err := c.buildEvidenceStatement(attachment)
	if err != nil {
		return err
	}

	payloadBytes, err := base64.StdEncoding.DecodeString(prepareResponse.DSSEPayload)
	if err != nil {
		return errorutils.CheckErrorf("failed to decode prepared DSSE payload: %v", err)
	}

	signedEnvelope, err := createAndSignEnvelope(payloadBytes, c.key, c.keyId)
	if err != nil {
		return err
	}
	envelopeBytes, err := json.Marshal(signedEnvelope)
	if err != nil {
		return err
	}

	envelopeBytes, err = wrapEnvelopeWithAttachmentRefs(envelopeBytes, toAttachmentRefs(prepareResponse.Attachments))
	if err != nil {
		return err
	}

	response, err := c.uploadPreparedEvidence(prepareResponse.PostURL, envelopeBytes)
	if err != nil {
		return err
	}
	c.recordSummary(response)
	return nil
}

// createEntityEvidence POSTs a ready-made payload (for example a Sigstore bundle) to the
// entity create API. It does not use the prepare flow.
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

func (c *createEvidenceEntity) buildEvidenceStatement(attachment *statementAttachment) (*client.PrepareEvidenceResponse, error) {
	if c.evidenceServiceClient == nil {
		evidenceServiceClient, err := client.NewEvidenceClient(c.serverDetails)
		if err != nil {
			return nil, err
		}
		c.evidenceServiceClient = evidenceServiceClient
	}

	predicate, err := os.ReadFile(c.predicateFilePath)
	if err != nil {
		return nil, errorutils.CheckError(err)
	}

	request := client.PrepareEvidenceRequest{
		Predicate:     json.RawMessage(predicate),
		PredicateType: c.predicateType,
		ProviderID:    c.providerId,
		Subject: client.PrepareEvidenceSubject{
			SubjectType: client.SubjectTypeEntity,
			EntityType:  c.EntityType,
			EntityID:    c.EntityID,
		},
		EntityRepo:     c.EntityRepo,
		ProjectKey:     c.ProjectKey,
		ApplicationKey: c.ApplicationKey,
	}

	if attachment != nil {
		request.Attachments = []client.PrepareEvidenceAttachment{{
			Repository: attachment.Repository,
			Path:       attachment.Path,
			SHA256:     attachment.Sha256,
		}}
	}

	markdown, err := c.readMarkdown()
	if err != nil {
		return nil, err
	}
	request.Markdown = markdown

	log.Debug("Preparing entity evidence:", c.EntityType, c.EntityID)
	return c.evidenceServiceClient.PrepareEvidence(request, false)
}

func (c *createEvidenceEntity) readMarkdown() (string, error) {
	if c.markdownFilePath == "" {
		return "", nil
	}
	if !strings.HasSuffix(c.markdownFilePath, ".md") {
		return "", fmt.Errorf("file '%s' does not have a .md extension", c.markdownFilePath)
	}
	markdown, err := os.ReadFile(c.markdownFilePath)
	if err != nil {
		return "", err
	}
	return string(markdown), nil
}

func toAttachmentRefs(attachments []client.PrepareEvidenceAttachment) []attachmentRef {
	refs := make([]attachmentRef, 0, len(attachments))
	for _, att := range attachments {
		refs = append(refs, attachmentRef{
			Repository: att.Repository,
			Path:       att.Path,
			Sha256:     att.SHA256,
		})
	}
	return refs
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
