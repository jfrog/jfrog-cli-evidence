package get

import (
	"errors"
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
	queryWithAttachments, err := g.buildGraphqlQuery(g.subjectRepoPath, true)
	if err != nil {
		return nil, err
	}
	queryWithoutAttachments, err := g.buildGraphqlQuery(g.subjectRepoPath, false)
	if err != nil {
		return nil, err
	}
	evidenceArray, err := g.searchEvidence(onemodelClient, searchEvidenceQueries{
		withAttachments:    queryWithAttachments,
		withoutAttachments: queryWithoutAttachments,
	})
	if err != nil {
		return nil, g.customSearchError(err)
	}
	return g.marshalEvidence(evidenceArray)
}

func (g *getEvidenceCustom) transformGraphQLOutput(rawEvidence []byte) ([]byte, error) {
	evidenceArray, err := evidenceEntriesFromSearchResponse(rawEvidence, g.includePredicate)
	if err != nil {
		return nil, g.customSearchError(err)
	}
	return g.marshalEvidence(evidenceArray)
}

func (g *getEvidenceCustom) customSearchError(err error) error {
	if errors.Is(err, errSearchEvidenceMissing) {
		return fmt.Errorf("repository does not exist for subject repository path: %s", g.subjectRepoPath)
	}
	if errors.Is(err, errSearchEvidenceEdgesMissing) {
		return fmt.Errorf("artifact was not found in subject repository path: %s", g.subjectRepoPath)
	}
	return err
}

func (g *getEvidenceCustom) marshalEvidence(evidenceArray []EvidenceEntry) ([]byte, error) {
	return marshalSearchEvidenceOutput(ArtifactType, CustomEvidenceResult{
		RepoPath: g.subjectRepoPath,
		Evidence: evidenceArray,
	})
}

func (g *getEvidenceCustom) buildGraphqlQuery(subjectRepoPath string, includeAttachments bool) ([]byte, error) {
	repoKey, pathVal, name, err := g.getRepoKeyAndPath(subjectRepoPath)
	if err != nil {
		return nil, err
	}
	nodeFields := g.buildSearchEvidenceNodeFields(evidenceutils.FieldSubjectSha256, includeAttachments)
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
