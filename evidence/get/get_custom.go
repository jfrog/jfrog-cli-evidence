package get

import (
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/jfrog/gofrog/log"
	"github.com/jfrog/jfrog-cli-core/v2/artifactory/utils"
	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli-evidence/evidence"
	evidenceutils "github.com/jfrog/jfrog-cli-evidence/evidence/utils"
	"github.com/jfrog/jfrog-client-go/onemodel"
)

const getCustomEvidenceQueryTemplate = `{"query":"{ evidence { searchEvidence( where: { hasSubjectWith: { repositoryKey: \"%s\", path: \"%s\", name: \"%s\"}} ) { totalCount edges { node { ` + evidenceutils.NodeFieldsPlaceholder + ` } } } } }"}`

type getEvidenceCustom struct {
	getEvidenceBase
	subjectRepoPath string
}

// CustomEvidenceOutput represents the structured output format for custom evidence
type CustomEvidenceOutput struct {
	SchemaVersion string               `json:"schemaVersion"`
	Type          SubjectType          `json:"type"`
	Result        CustomEvidenceResult `json:"result"`
}

func NewGetEvidenceCustom(serverDetails *config.ServerDetails, subjectRepoPath, format, outputFileName string, includePredicate bool) evidence.Command {
	return &getEvidenceCustom{
		getEvidenceBase: getEvidenceBase{
			serverDetails:    serverDetails,
			format:           format,
			outputFileName:   outputFileName,
			includePredicate: includePredicate,
		},
		subjectRepoPath: subjectRepoPath,
	}
}

func (g *getEvidenceCustom) CommandName() string {
	return "get-custom-evidence"
}

func (g *getEvidenceCustom) ServerDetails() (*config.ServerDetails, error) {
	return g.serverDetails, nil
}

func (g *getEvidenceCustom) Run() error {
	onemodelClient, err := utils.CreateOnemodelServiceManager(g.serverDetails, false)
	if err != nil {
		log.Error("failed to create onemodel client", err)
		return fmt.Errorf("onemodel client init failed: %w", err)

	}

	evidence, err := g.getEvidence(onemodelClient)
	if err != nil {
		log.Error("Failed to get evidence:", err)
		return fmt.Errorf("evidence retrieval failed: %w", err)
	}

	return g.exportEvidenceToFile(evidence, g.outputFileName, g.format)
}

func (g *getEvidenceCustom) getEvidence(onemodelClient onemodel.Manager) ([]byte, error) {
	query, err := g.buildGraphqlQuery(g.subjectRepoPath, true)
	if err != nil {
		return nil, err
	}
	evidence, err := onemodelClient.GraphqlQuery(query)
	if err != nil {
		if evidenceutils.IsAttachmentsFieldNotFound(err) {
			log.Debug("GraphQL schema does not support attachments field. Falling back to query without attachments.")
			queryWithoutAttachments, qErr := g.buildGraphqlQuery(g.subjectRepoPath, false)
			if qErr != nil {
				return nil, qErr
			}
			evidence, err = onemodelClient.GraphqlQuery(queryWithoutAttachments)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	transformedEvidence, err := g.transformGraphQLOutput(evidence)
	if err != nil {
		return nil, err
	}

	return transformedEvidence, nil
}

func (g *getEvidenceCustom) transformGraphQLOutput(rawEvidence []byte) ([]byte, error) {
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
		return nil, fmt.Errorf("repository does not exist for subject repository path: %s", g.subjectRepoPath)
	}

	edges, ok := searchEvidenceData["edges"].([]any)
	if !ok {
		return nil, fmt.Errorf("artifact was not found in subject repository path: %s", g.subjectRepoPath)
	}

	evidenceArray := make([]EvidenceEntry, 0, len(edges))
	for _, edge := range edges {
		if edgeMap, ok := edge.(map[string]any); ok {
			if node, ok := edgeMap["node"].(map[string]any); ok {
				evidenceEntry := createOrderedEvidenceEntry(node, g.includePredicate)
				evidenceArray = append(evidenceArray, evidenceEntry)
			}
		}
	}

	output := CustomEvidenceOutput{
		SchemaVersion: SchemaVersion,
		Type:          ArtifactType,
		Result: CustomEvidenceResult{
			RepoPath: g.subjectRepoPath,
			Evidence: evidenceArray,
		},
	}

	transformed, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transformed response: %w", err)
	}

	return transformed, nil
}

func (g *getEvidenceCustom) buildGraphqlQuery(subjectRepoPath string, includeAttachments bool) ([]byte, error) {
	repoKey, pathVal, name, err := g.getRepoKeyAndPath(subjectRepoPath)
	if err != nil {
		return nil, err
	}
	nodeFields := evidenceutils.NewNodeFieldsBuilder(
		evidenceutils.FieldPredicateSlug,
		evidenceutils.FieldPredicateType,
		evidenceutils.FieldDownloadPath,
		evidenceutils.FieldVerified,
		evidenceutils.FieldSigningKeyAlias,
		evidenceutils.FieldCreatedBy,
		evidenceutils.FieldCreatedAt,
		evidenceutils.FieldSubjectSha256,
	).
		WithIf(includeAttachments, evidenceutils.AttachmentsFragment).
		WithIf(g.includePredicate, evidenceutils.FieldPredicate).
		Build()
	queryTemplate := evidenceutils.BuildQuery(getCustomEvidenceQueryTemplate, nodeFields)
	graphqlQuery := fmt.Sprintf(queryTemplate, repoKey, pathVal, name)
	log.Debug("GraphQL query: ", graphqlQuery)
	return []byte(graphqlQuery), nil
}

func (g *getEvidenceCustom) getRepoKeyAndPath(subjectRepoPath string) (string, string, string, error) {
	firstSlashIndex := strings.Index(subjectRepoPath, "/")
	if firstSlashIndex <= 0 || firstSlashIndex == len(subjectRepoPath)-1 {
		return "", "", "", fmt.Errorf("invalid input: expected format 'repo/path', got '%s'", subjectRepoPath)
	}
	repo := subjectRepoPath[:firstSlashIndex]
	pathAndName := subjectRepoPath[firstSlashIndex+1:]

	pathVal := path.Dir(pathAndName)
	name := path.Base(pathAndName)
	if pathVal == "." {
		pathVal = ""
	}

	return repo, pathVal, name, nil
}
