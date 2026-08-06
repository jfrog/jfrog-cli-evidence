package verify

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli-evidence/evidence"
	"github.com/jfrog/jfrog-cli-evidence/evidence/model"
	evidenceutils "github.com/jfrog/jfrog-cli-evidence/evidence/utils"
	"github.com/jfrog/jfrog-client-go/utils/log"
)

const searchEvidenceByEntityQueryTemplate = `{"query":"{ evidence { searchEvidence( where: { hasSubjectWith: { %s }} ) { edges { cursor node { ` + evidenceutils.NodeFieldsPlaceholder + ` } } } } }"}`

type verifyEvidenceEntity struct {
	verifyEvidenceBase
	entityType     string
	entityID       string
	entityRepo     string
	projectKey     string
	applicationKey string
}

// NewVerifyEvidenceEntity creates a command for verifying evidence attached to an entity subject.
func NewVerifyEvidenceEntity(serverDetails *config.ServerDetails, entityType, entityID, entityRepo, projectKey, applicationKey, format string, keys []string, useArtifactoryKeys bool) evidence.Command {
	return &verifyEvidenceEntity{
		verifyEvidenceBase: newVerifyEvidenceBase(serverDetails, format, keys, useArtifactoryKeys),
		entityType:         entityType,
		entityID:           entityID,
		entityRepo:         entityRepo,
		projectKey:         projectKey,
		applicationKey:     applicationKey,
	}
}

func (v *verifyEvidenceEntity) Run() error {
	defer v.quitProgress()

	if err := v.resolveApplicationEntityProject(); err != nil {
		return err
	}

	client, err := v.createArtifactoryClient()
	if err != nil {
		return fmt.Errorf("failed to create Artifactory client: %w", err)
	}

	metadata, err := v.queryEvidenceMetadataByEntity()
	if err != nil {
		return err
	}

	// Entity subjects have no content checksum. They are verified through the entity digest
	// carried in the signed in-toto statement.
	expectedSubject := model.SubjectDigest{Type: v.entityType, Value: v.entityID}
	subjectPath := fmt.Sprintf("%s/%s", v.entityType, v.entityID)
	return v.verifyEvidence(client, metadata, expectedSubject, subjectPath)
}

func (v *verifyEvidenceEntity) resolveApplicationEntityProject() error {
	if !strings.EqualFold(v.entityType, "application") || v.projectKey != "" || v.entityRepo != "" || v.applicationKey != "" {
		return nil
	}
	projectKey, err := evidenceutils.ResolveApplicationProjectKey(v.serverDetails, v.entityID)
	if err != nil {
		return err
	}
	v.projectKey = projectKey
	log.Debug("Resolved project key for application entity:", v.projectKey)
	return nil
}

func (v *verifyEvidenceEntity) queryEvidenceMetadataByEntity() (*[]model.SearchEvidenceEdge, error) {
	v.setHeadline("Searching evidence")

	if err := createOneModelService(&v.verifyEvidenceBase); err != nil {
		return nil, err
	}

	response, usedFallbackWithoutAttachments, err := v.fetchSearchEvidenceByEntityResponse()
	if err != nil {
		return nil, err
	}

	evidence := model.ResponseSearchEvidence{}
	if err = json.Unmarshal(response, &evidence); err != nil {
		return nil, fmt.Errorf("failed to unmarshal evidence metadata: %w", err)
	}
	edges := evidence.Data.Evidence.SearchEvidence.Edges
	if len(edges) == 0 {
		return nil, fmt.Errorf("no evidence found for the given subject")
	}
	if usedFallbackWithoutAttachments {
		for i := range edges {
			edges[i].Node.AttachmentsUnavailable = true
		}
	}
	return &edges, nil
}

func (v *verifyEvidenceEntity) fetchSearchEvidenceByEntityResponse() ([]byte, bool, error) {
	queryWithAttachments := v.buildSearchEvidenceByEntityQuery(true)
	log.Debug("Fetch evidence metadata using query:", queryWithAttachments)
	response, err := v.oneModelClient.GraphqlQuery([]byte(queryWithAttachments))
	if err == nil {
		return response, false, nil
	}

	if evidenceutils.IsAttachmentsFieldNotFound(err) {
		log.Debug("GraphQL schema does not support attachments field. Falling back to verify query without attachments.")
		queryWithoutAttachments := v.buildSearchEvidenceByEntityQuery(false)
		log.Debug("Fetch evidence metadata using query without attachments:", queryWithoutAttachments)
		response, err = v.oneModelClient.GraphqlQuery([]byte(queryWithoutAttachments))
		if err != nil {
			return nil, false, mapGraphqlQueryError(err)
		}
		return response, true, nil
	}
	return nil, false, mapGraphqlQueryError(err)
}

func (v *verifyEvidenceEntity) buildSearchEvidenceByEntityQuery(includeAttachments bool) string {
	nodeFields := evidenceutils.NewNodeFieldsBuilder(
		evidenceutils.FieldDownloadPath,
		evidenceutils.FieldPredicateType,
		evidenceutils.FieldCreatedAt,
		evidenceutils.FieldCreatedBy,
		evidenceutils.FieldSubjectSha256,
	).
		WithIf(includeAttachments, evidenceutils.AttachmentsFragment).
		WithIf(v.useArtifactoryKeys, evidenceutils.FieldSigningKeyWithPublicKey).
		Build()
	whereClause := evidenceutils.BuildGraphQLEntityHasSubjectWith(v.entityType, v.entityID, v.entityRepo, v.projectKey, v.applicationKey)
	return fmt.Sprintf(evidenceutils.BuildQuery(searchEvidenceByEntityQueryTemplate, nodeFields), whereClause)
}

func (v *verifyEvidenceEntity) ServerDetails() (*config.ServerDetails, error) {
	return v.serverDetails, nil
}

func (v *verifyEvidenceEntity) CommandName() string {
	return "verify-evidence-entity"
}
