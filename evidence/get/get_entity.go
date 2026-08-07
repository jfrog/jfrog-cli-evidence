package get

import (
	"encoding/json"
	"fmt"

	"github.com/jfrog/gofrog/log"
	"github.com/jfrog/jfrog-cli-core/v2/artifactory/utils"
	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli-evidence/evidence"
	evidenceutils "github.com/jfrog/jfrog-cli-evidence/evidence/utils"
	"github.com/jfrog/jfrog-client-go/onemodel"
)

const getEntityEvidenceQueryTemplate = `{"query":"{ evidence { searchEvidence( where: { hasSubjectWith: { %s }} ) { totalCount edges { node { ` + evidenceutils.NodeFieldsPlaceholder + ` } } } } }"}`

type getEvidenceEntity struct {
	getEvidenceBase
	entityType     string
	entityID       string
	entityRepo     string
	projectKey     string
	applicationKey string
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
		entityType:     entityType,
		entityID:       entityID,
		entityRepo:     entityRepo,
		projectKey:     projectKey,
		applicationKey: applicationKey,
	}
}

func (g *getEvidenceEntity) CommandName() string {
	return "get-entity-evidence"
}

func (g *getEvidenceEntity) ServerDetails() (*config.ServerDetails, error) {
	return g.serverDetails, nil
}

func (g *getEvidenceEntity) Run() error {
	if err := g.resolveApplicationEntityProject(); err != nil {
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

func (g *getEvidenceEntity) resolveApplicationEntityProject() error {
	if g.entityType != "application" || g.projectKey != "" || g.entityRepo != "" || g.applicationKey != "" {
		return nil
	}
	projectKey, err := evidenceutils.ResolveApplicationProjectKey(g.serverDetails, g.entityID)
	if err != nil {
		return err
	}
	g.projectKey = projectKey
	log.Debug("Resolved project key for application entity:", g.projectKey)
	return nil
}

func (g *getEvidenceEntity) getEvidence(onemodelClient onemodel.Manager) ([]byte, error) {
	evidenceBytes, err := onemodelClient.GraphqlQuery(g.buildGraphqlQuery(true))
	if err != nil {
		if !evidenceutils.IsAttachmentsFieldNotFound(err) {
			return nil, err
		}
		log.Debug("GraphQL schema does not support attachments field. Falling back to query without attachments.")
		evidenceBytes, err = onemodelClient.GraphqlQuery(g.buildGraphqlQuery(false))
		if err != nil {
			return nil, err
		}
	}
	return g.transformGraphQLOutput(evidenceBytes)
}

func (g *getEvidenceEntity) transformGraphQLOutput(rawEvidence []byte) ([]byte, error) {
	var graphqlResponse map[string]any
	if err := json.Unmarshal(rawEvidence, &graphqlResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal GraphQL response: %w", err)
	}

	evidenceData, ok := graphqlResponse["data"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid GraphQL response structure: missing data field")
	}

	searchEvidence, ok := evidenceData["evidence"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid GraphQL response structure: missing evidence field")
	}

	searchEvidenceData, ok := searchEvidence["searchEvidence"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("no evidence found for entity %s/%s", g.entityType, g.entityID)
	}

	edges, ok := searchEvidenceData["edges"].([]any)
	if !ok {
		return nil, fmt.Errorf("no evidence found for entity %s/%s", g.entityType, g.entityID)
	}

	evidenceArray := make([]EvidenceEntry, 0, len(edges))
	for _, edge := range edges {
		if edgeMap, ok := edge.(map[string]any); ok {
			if node, ok := edgeMap["node"].(map[string]any); ok {
				evidenceArray = append(evidenceArray, createOrderedEvidenceEntry(node, g.includePredicate))
			}
		}
	}

	output := EntityEvidenceOutput{
		SchemaVersion: SchemaVersion,
		Type:          EntityType,
		Result: EntityEvidenceResult{
			EntityType:     g.entityType,
			EntityId:       g.entityID,
			Project:        g.projectKey,
			ApplicationKey: g.applicationKey,
			EntityRepo:     g.entityRepo,
			Evidence:       evidenceArray,
		},
	}

	transformed, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transformed response: %w", err)
	}
	return transformed, nil
}

func (g *getEvidenceEntity) buildGraphqlQuery(includeAttachments bool) []byte {
	nodeFields := evidenceutils.NewNodeFieldsBuilder(
		evidenceutils.FieldPredicateSlug,
		evidenceutils.FieldPredicateType,
		evidenceutils.FieldDownloadPath,
		evidenceutils.FieldVerified,
		evidenceutils.FieldSigningKeyAlias,
		evidenceutils.FieldCreatedBy,
		evidenceutils.FieldCreatedAt,
		evidenceutils.FieldSubjectWithPath,
	).
		WithIf(includeAttachments, evidenceutils.AttachmentsFragment).
		WithIf(g.includePredicate, evidenceutils.FieldPredicate).
		Build()
	whereClause := evidenceutils.BuildGraphQLEntityHasSubjectWith(g.entityType, g.entityID, g.entityRepo, g.projectKey, g.applicationKey)
	queryTemplate := evidenceutils.BuildQuery(getEntityEvidenceQueryTemplate, nodeFields)
	graphqlQuery := fmt.Sprintf(queryTemplate, whereClause)
	log.Debug("GraphQL query: ", graphqlQuery)
	return []byte(graphqlQuery)
}
