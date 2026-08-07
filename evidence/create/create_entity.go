package create

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
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

type prepareEvidenceClient interface {
	PrepareEvidence(request client.PrepareEvidenceRequest, includePAE bool) (*client.PrepareEvidenceResponse, error)
	UploadPreparedSignedEvidence(postURL string, signedEnvelope []byte) ([]byte, error)
}

type createEvidenceEntity struct {
	createEvidenceBase
	entityType     string
	entityID       string
	entityRepo     string
	projectKey     string
	applicationKey string
	prepareClient  prepareEvidenceClient
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
		entityType:     entityType,
		entityID:       entityID,
		entityRepo:     entityRepo,
		projectKey:     projectKey,
		applicationKey: applicationKey,
	}
}

func (c *createEvidenceEntity) CommandName() string {
	return "create-entity-evidence"
}

func (c *createEvidenceEntity) ServerDetails() (*config.ServerDetails, error) {
	return c.serverDetails, nil
}

func (c *createEvidenceEntity) Run() error {
	if err := c.resolveApplicationEntityProject(); err != nil {
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

	response, err := c.uploadPreparedEvidence(c.buildEntityPostURL(), payload)
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

	prepareResponse, err := c.prepareSignedStatement(attachment)
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

func (c *createEvidenceEntity) buildEntityPostURL() string {
	postURL := "/evidence/api/v1/entity/" + c.entityType + "/" + c.entityID
	query := url.Values{}
	switch {
	case c.entityRepo != "":
		query.Set("repo", c.entityRepo)
	case c.projectKey != "":
		query.Set("project", c.projectKey)
	case c.applicationKey != "":
		query.Set("application", c.applicationKey)
	}
	if c.providerId != "" {
		query.Set("providerId", c.providerId)
	}
	if encoded := query.Encode(); encoded != "" {
		postURL += "?" + encoded
	}
	return postURL
}

func (c *createEvidenceEntity) resolveApplicationEntityProject() error {
	// Bare --application-key shorthand uses entity-type=application and resolves project scope via AppTrust.
	if c.entityType != "application" || c.projectKey != "" || c.entityRepo != "" || c.applicationKey != "" {
		return nil
	}
	var err error
	c.projectKey, err = evidenceUtils.ResolveApplicationProjectKey(c.serverDetails, c.entityID)
	if err != nil {
		return err
	}
	log.Debug("Resolved project key for application entity:", c.projectKey)
	return nil
}

func (c *createEvidenceEntity) prepareSignedStatement(attachment *statementAttachment) (*client.PrepareEvidenceResponse, error) {
	if c.prepareClient == nil {
		prepareClient, err := client.NewEvidenceClient(c.serverDetails)
		if err != nil {
			return nil, err
		}
		c.prepareClient = prepareClient
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
			EntityType:  c.entityType,
			EntityID:    c.entityID,
		},
		EntityRepo:     c.entityRepo,
		ProjectKey:     c.projectKey,
		ApplicationKey: c.applicationKey,
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

	log.Debug("Preparing entity evidence:", c.entityType, c.entityID)
	return c.prepareClient.PrepareEvidence(request, false)
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

func (c *createEvidenceEntity) uploadPreparedEvidence(postURL string, envelopeBytes []byte) (*model.CreateResponse, error) {
	if c.prepareClient == nil {
		prepareClient, err := client.NewEvidenceClient(c.serverDetails)
		if err != nil {
			return nil, err
		}
		c.prepareClient = prepareClient
	}

	log.Debug("Uploading entity evidence to:", postURL)
	body, err := c.prepareClient.UploadPreparedSignedEvidence(postURL, envelopeBytes)
	if err != nil {
		return nil, err
	}
	return c.collectCreateResponse(body)
}

func (c *createEvidenceEntity) recordSummary(response *model.CreateResponse) {
	if !evidenceUtils.IsRunningUnderGitHubAction() {
		return
	}
	applicationKey := c.applicationKey
	subjectType := commandsummary.SubjectTypeArtifact
	if c.entityType == "application" {
		applicationKey = c.entityID
		subjectType = commandsummary.SubjectTypeApplication
	}
	err := c.recordEvidenceSummary(commandsummary.EvidenceSummaryData{
		Subject:        fmt.Sprintf("%s/%s", c.entityType, c.entityID),
		PredicateType:  c.predicateType,
		PredicateSlug:  response.PredicateSlug,
		Verified:       response.Verified,
		DisplayName:    fmt.Sprintf("%s %s", c.entityType, c.entityID),
		SubjectType:    subjectType,
		RepoKey:        c.entityRepo,
		ApplicationKey: applicationKey,
	})
	if err != nil {
		log.Warn("Failed to record evidence summary:", err.Error())
	}
}
