package get

import (
	"errors"
	"fmt"

	"github.com/jfrog/gofrog/log"
	"github.com/jfrog/jfrog-cli-core/v2/artifactory/utils"
	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli-evidence/evidence"
	"github.com/jfrog/jfrog-cli-evidence/evidence/model"
	evidenceutils "github.com/jfrog/jfrog-cli-evidence/evidence/utils"
	"github.com/jfrog/jfrog-client-go/onemodel"
)

const getEntityEvidenceQueryTemplate = `{"query":"{ evidence { searchEvidence( where: { hasSubjectWith: { %s }} ) { totalCount edges { node { ` + evidenceutils.NodeFieldsPlaceholder + ` } } } } }"}`

type getEvidenceEntity struct {
	getEvidenceBase
	model.EntitySubject
}

// EntityEvidenceOutput is the structured get output for entity subjects.
type EntityEvidenceOutput struct {
	SchemaVersion string               `json:"schemaVersion"`
	Type          SubjectType          `json:"type"`
	Result        EntityEvidenceResult `json:"result"`
}

// EntityEvidenceResult holds entity identity and attached evidence entries.
type EntityEvidenceResult struct {
	EntityType     string          `json:"entityType"`
	EntityId       string          `json:"entityId"`
	Project        string          `json:"project,omitempty"`
	ApplicationKey string          `json:"applicationKey,omitempty"`
	EntityRepo     string          `json:"entityRepo,omitempty"`
	Evidence       []EvidenceEntry `json:"evidence"`
}

func NewGetEvidenceEntity(serverDetails *config.ServerDetails, entityType, entityID, entityRepo, projectKey, applicationKey, format, outputFileName string, includePredicate bool) evidence.Command {
	return &getEvidenceEntity{
		getEvidenceBase: getEvidenceBase{
			serverDetails:    serverDetails,
			format:           format,
			outputFileName:   outputFileName,
			includePredicate: includePredicate,
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

func (g *getEvidenceEntity) CommandName() string {
	return "get-entity-evidence"
}

func (g *getEvidenceEntity) ServerDetails() (*config.ServerDetails, error) {
	return g.serverDetails, nil
}

func (g *getEvidenceEntity) Run() error {
	if err := evidenceutils.ResolveApplicationEntityProjectKey(g.serverDetails, &g.EntitySubject); err != nil {
		return err
	}

	onemodelClient, err := utils.CreateOnemodelServiceManager(g.serverDetails, false)
	if err != nil {
		log.Error("failed to create onemodel client", err)
		return fmt.Errorf("onemodel client init failed: %w", err)
	}

	evidenceBytes, err := g.getEvidence(onemodelClient)
	if err != nil {
		log.Error("Failed to get evidence:", err)
		return fmt.Errorf("evidence retrieval failed: %w", err)
	}

	return g.exportEvidenceToFile(evidenceBytes, g.outputFileName, g.format)
}

func (g *getEvidenceEntity) getEvidence(onemodelClient onemodel.Manager) ([]byte, error) {
	evidenceArray, err := g.searchEvidence(onemodelClient, searchEvidenceQueries{
		withAttachments:    g.buildGraphqlQuery(true),
		withoutAttachments: g.buildGraphqlQuery(false),
	})
	if err != nil {
		return nil, g.entitySearchError(err)
	}
	return g.marshalEvidence(evidenceArray)
}

func (g *getEvidenceEntity) entitySearchError(err error) error {
	if errors.Is(err, errSearchEvidenceMissing) || errors.Is(err, errSearchEvidenceEdgesMissing) {
		return fmt.Errorf("no evidence found for entity %s/%s", g.EntityType, g.EntityID)
	}
	return err
}

func (g *getEvidenceEntity) marshalEvidence(evidenceArray []EvidenceEntry) ([]byte, error) {
	return marshalSearchEvidenceOutput(EntityType, EntityEvidenceResult{
		EntityType:     g.EntityType,
		EntityId:       g.EntityID,
		Project:        g.ProjectKey,
		ApplicationKey: g.ApplicationKey,
		EntityRepo:     g.EntityRepo,
		Evidence:       evidenceArray,
	})
}

func (g *getEvidenceEntity) buildGraphqlQuery(includeAttachments bool) []byte {
	nodeFields := g.buildSearchEvidenceNodeFields(evidenceutils.FieldSubjectWithPath, includeAttachments)
	whereClause := evidenceutils.BuildGraphQLEntityHasSubjectWith(g.EntitySubject)
	queryTemplate := evidenceutils.BuildQuery(getEntityEvidenceQueryTemplate, nodeFields)
	graphqlQuery := fmt.Sprintf(queryTemplate, whereClause)
	log.Debug("GraphQL query: ", graphqlQuery)
	return []byte(graphqlQuery)
}
