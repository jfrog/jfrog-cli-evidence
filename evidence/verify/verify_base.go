package verify

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jfrog/jfrog-cli-core/v2/artifactory/utils"
	coreProgress "github.com/jfrog/jfrog-cli-core/v2/common/progressbar"
	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli-core/v2/utils/coreutils"
	"github.com/jfrog/jfrog-cli-evidence/evidence/model"
	evidenceutils "github.com/jfrog/jfrog-cli-evidence/evidence/utils"
	"github.com/jfrog/jfrog-cli-evidence/evidence/verify/reports"
	"github.com/jfrog/jfrog-cli-evidence/evidence/verify/verifiers"

	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/onemodel"
	ioUtils "github.com/jfrog/jfrog-client-go/utils/io"
	"github.com/jfrog/jfrog-client-go/utils/log"
)

const searchEvidenceQueryTemplate = `{"query":"{ evidence { searchEvidence( where: { hasSubjectWith: { repositoryKey: \"%s\", path: \"%s\", name: \"%s\" }} ) { edges { cursor node { ` + evidenceutils.NodeFieldsPlaceholder + ` } } } } }"}`

// verifyEvidenceBase provides shared logic for evidence verification command.
type verifyEvidenceBase struct {
	serverDetails      *config.ServerDetails
	format             string
	keys               []string
	useArtifactoryKeys bool
	artifactoryClient  *artifactory.ArtifactoryServicesManager
	oneModelClient     onemodel.Manager
	verifier           verifiers.EvidenceVerifierInterface
	progressMgr        ioUtils.ProgressMgr
}

// newVerifyEvidenceBase builds a base with optional progress manager initialized.
func newVerifyEvidenceBase(serverDetails *config.ServerDetails, format string, keys []string, useArtifactoryKeys bool) verifyEvidenceBase {
	v := verifyEvidenceBase{
		serverDetails:      serverDetails,
		format:             format,
		keys:               keys,
		useArtifactoryKeys: useArtifactoryKeys,
	}
	// Initialize progress manager if possible. The progress manager is optional.
	if pm, _ := coreProgress.InitFilesProgressBarIfPossible(false); pm != nil {
		v.progressMgr = pm
	}
	return v
}

func (v *verifyEvidenceBase) setHeadline(msg string) {
	if v.progressMgr != nil {
		v.progressMgr.SetHeadlineMsg(msg)
	}
}

func (v *verifyEvidenceBase) quitProgress() {
	if v.progressMgr != nil {
		_ = v.progressMgr.Quit()
		v.progressMgr = nil
	}
}

// printVerifyResult prints the verification result in the requested format.
func (v *verifyEvidenceBase) printVerifyResult(result *model.VerificationResponse) error {
	switch v.format {
	case "markdown":
		return reports.MarkdownReportPrinter.Print(result)
	case "json":
		return reports.JsonReportPrinter.Print(result)
	default:
		return reports.PlaintextReportPrinter.Print(result)
	}
}

// verifyEvidence runs the verification process for the given evidence metadata and subject sha256.
func (v *verifyEvidenceBase) verifyEvidence(client *artifactory.ArtifactoryServicesManager, evidenceMetadata *[]model.SearchEvidenceEdge, sha256, subjectPath string) error {
	if v.verifier == nil {
		v.setHeadline("Verifying evidence")
		v.verifier = verifiers.NewEvidenceVerifier(v.keys, v.useArtifactoryKeys, client, v.progressMgr)
	}
	verify, err := v.verifier.Verify(sha256, evidenceMetadata, subjectPath)
	if err != nil {
		return err
	}

	v.quitProgress()
	err = v.printVerifyResult(verify)
	if verify.OverallVerificationStatus == model.Failed {
		return coreutils.CliError{ExitCode: coreutils.ExitCodeError}
	}
	return err
}

// createArtifactoryClient creates an Artifactory client for evidence operations.
func (v *verifyEvidenceBase) createArtifactoryClient() (*artifactory.ArtifactoryServicesManager, error) {
	if v.artifactoryClient != nil {
		return v.artifactoryClient, nil
	}
	artifactoryClient, err := utils.CreateUploadServiceManager(v.serverDetails, 1, 0, 0, false, nil)
	if err != nil {
		return nil, err
	}
	v.artifactoryClient = &artifactoryClient
	return v.artifactoryClient, nil
}

// queryEvidenceMetadata queries evidence metadata for a given repo, path, and name.
func (v *verifyEvidenceBase) queryEvidenceMetadata(repo string, path string, name string) (*[]model.SearchEvidenceEdge, error) {
	v.setHeadline("Searching evidence")

	err := createOneModelService(v)
	if err != nil {
		return nil, err
	}
	response, usedFallbackWithoutAttachments, err := v.fetchSearchEvidenceResponse(repo, path, name)
	if err != nil {
		return nil, err
	}
	evidence := model.ResponseSearchEvidence{}
	err = json.Unmarshal(response, &evidence)
	if err != nil {
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

func (v *verifyEvidenceBase) fetchSearchEvidenceResponse(repo, path, name string) ([]byte, bool, error) {
	queryWithAttachments := fmt.Sprintf(v.buildSearchEvidenceQuery(true), repo, path, name)
	log.Debug("Fetch evidence metadata using query:", queryWithAttachments)
	response, err := v.oneModelClient.GraphqlQuery([]byte(queryWithAttachments))
	if err == nil {
		return response, false, nil
	}

	if evidenceutils.IsAttachmentsFieldNotFound(err) {
		log.Debug("GraphQL schema does not support attachments field. Falling back to verify query without attachments.")
		queryWithoutAttachments := fmt.Sprintf(v.buildSearchEvidenceQuery(false), repo, path, name)
		log.Debug("Fetch evidence metadata using query without attachments:", queryWithoutAttachments)
		response, err = v.oneModelClient.GraphqlQuery([]byte(queryWithoutAttachments))
		if err != nil {
			return nil, false, mapGraphqlQueryError(err)
		}
		return response, true, nil
	}
	return nil, false, mapGraphqlQueryError(err)
}

func mapGraphqlQueryError(err error) error {
	if isPublicKeyFieldNotFound(err.Error()) {
		return fmt.Errorf("the evidence service version should be at least 7.125.0 and the onemodel version should be at least 1.55.0")
	}
	return fmt.Errorf("error querying evidence from One-Model service: %w", err)
}

func createOneModelService(v *verifyEvidenceBase) error {
	if v.oneModelClient != nil {
		return nil
	}
	manager, err := utils.CreateOnemodelServiceManager(v.serverDetails, false)
	if err != nil {
		return err
	}
	v.oneModelClient = manager
	return nil
}

func isPublicKeyFieldNotFound(errStr string) bool {
	return strings.Contains(errStr, "publicKey")
}

func (v *verifyEvidenceBase) buildSearchEvidenceQuery(includeAttachments bool) string {
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
	return evidenceutils.BuildQuery(searchEvidenceQueryTemplate, nodeFields)
}
