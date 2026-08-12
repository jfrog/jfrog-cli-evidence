package get

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	evidenceutils "github.com/jfrog/jfrog-cli-evidence/evidence/utils"
	"github.com/jfrog/jfrog-client-go/onemodel"
	"github.com/jfrog/jfrog-client-go/utils/log"
)

const SchemaVersion = "1.1"

type SubjectType string // Types in GetEvidence output

const (
	ArtifactType      SubjectType = "artifact"
	BuildType         SubjectType = "build"
	ReleaseBundleType SubjectType = "release-bundle"
	EntityType        SubjectType = "entity"
)

type getEvidenceBase struct {
	serverDetails    *config.ServerDetails
	outputFileName   string
	format           string
	includePredicate bool
}

type searchEvidenceQueries struct {
	withAttachments    []byte
	withoutAttachments []byte
}

type JsonlLine struct {
	SchemaVersion string      `json:"schemaVersion"`
	Type          SubjectType `json:"type"`
	Result        any         `json:"result"`
}

type EvidenceEntry struct {
	PredicateSlug string               `json:"predicateSlug"`
	PredicateType string               `json:"predicateType,omitempty"`
	DownloadPath  string               `json:"downloadPath"`
	Verified      bool                 `json:"verified"`
	SigningKey    map[string]any       `json:"signingKey,omitempty"`
	Subject       map[string]any       `json:"subject"`
	CreatedBy     string               `json:"createdBy"`
	CreatedAt     string               `json:"createdAt"`
	Predicate     map[string]any       `json:"predicate,omitempty"`
	Attachments   []EvidenceAttachment `json:"attachments,omitempty"`
}

type EvidenceAttachment struct {
	Name         string `json:"name"`
	Sha256       string `json:"sha256"`
	Type         string `json:"type,omitempty"`
	DownloadPath string `json:"downloadPath"`
}

type CustomEvidenceResult struct {
	RepoPath string          `json:"subjectRepoPath"`
	Evidence []EvidenceEntry `json:"evidence"`
}

type ArtifactEvidence struct {
	Evidence    EvidenceEntry `json:"evidence"`
	PackageType string        `json:"packageType"`
	RepoPath    string        `json:"subjectRepoPath"`
}

type BuildEvidence struct {
	Evidence    EvidenceEntry `json:"evidence"`
	BuildName   string        `json:"buildName"`
	BuildNumber string        `json:"buildNumber"`
	StartedAt   string        `json:"startedAt"`
}

type ReleaseBundleResult struct {
	ReleaseBundle        string             `json:"releaseBundle"`
	ReleaseBundleVersion string             `json:"releaseBundleVersion"`
	Evidence             []EvidenceEntry    `json:"evidence"`
	Artifacts            []ArtifactEvidence `json:"artifacts,omitempty"`
	Builds               []BuildEvidence    `json:"builds,omitempty"`
}

func (g *getEvidenceBase) exportEvidenceToFile(evidence []byte, outputFileName, format string) error {
	if format == "" {
		format = "json"
	}

	switch format {
	case "json":
		return exportEvidenceToJsonFile(evidence, outputFileName)
	case "jsonl":
		return exportEvidenceToJsonlFile(evidence, outputFileName)
	default:
		log.Error("Unsupported format. Supported formats are: json, jsonl")
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// graphqlQueryWithAttachmentsFallback runs queryWithAttachments first. If the server schema
// does not support the attachments field, it retries with queryWithoutAttachments.
func graphqlQueryWithAttachmentsFallback(client onemodel.Manager, queryWithAttachments, queryWithoutAttachments []byte) ([]byte, error) {
	response, err := client.GraphqlQuery(queryWithAttachments)
	if err == nil {
		return response, nil
	}
	if !evidenceutils.IsAttachmentsFieldNotFound(err) {
		return nil, err
	}
	log.Debug("GraphQL schema does not support attachments field. Falling back to query without attachments.")
	return client.GraphqlQuery(queryWithoutAttachments)
}

func (g *getEvidenceBase) searchEvidence(client onemodel.Manager, queries searchEvidenceQueries) ([]EvidenceEntry, error) {
	response, err := graphqlQueryWithAttachmentsFallback(client, queries.withAttachments, queries.withoutAttachments)
	if err != nil {
		return nil, err
	}
	return evidenceEntriesFromSearchResponse(response, g.includePredicate)
}

func (g *getEvidenceBase) buildSearchEvidenceNodeFields(subjectField string, includeAttachments bool) string {
	return evidenceutils.NewNodeFieldsBuilder(
		evidenceutils.FieldPredicateSlug,
		evidenceutils.FieldPredicateType,
		evidenceutils.FieldDownloadPath,
		evidenceutils.FieldVerified,
		evidenceutils.FieldSigningKeyAlias,
		evidenceutils.FieldCreatedBy,
		evidenceutils.FieldCreatedAt,
		subjectField,
	).
		WithIf(includeAttachments, evidenceutils.AttachmentsFragment).
		WithIf(g.includePredicate, evidenceutils.FieldPredicate).
		Build()
}

func marshalSearchEvidenceOutput(subjectType SubjectType, result any) ([]byte, error) {
	output := JsonlLine{
		SchemaVersion: SchemaVersion,
		Type:          subjectType,
		Result:        result,
	}
	transformed, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transformed response: %w", err)
	}
	return transformed, nil
}

func exportEvidenceToJsonFile(evidence []byte, outputFileName string) error {
	if outputFileName == "" {
		// Stream to console
		fmt.Println(string(evidence))
		return nil
	}

	file, err := os.Create(outputFileName)
	if err != nil {
		return err
	}

	defer func() {
		_ = file.Close()
	}()

	_, err = file.Write(evidence)
	if err != nil {
		return err
	}

	log.Info("Evidence successfully exported to file name: ", outputFileName)
	return nil
}

func exportEvidenceToJsonlFile(data []byte, outputFileName string) error {
	if outputFileName == "" {
		return writeEvidenceJsonl(data, os.Stdout)
	}

	file, err := os.Create(outputFileName)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	return writeEvidenceJsonl(data, file)
}

// writeEvidenceJsonl handles evidence output structures that have schemaVersion, type, and result fields
func writeEvidenceJsonl(data []byte, file *os.File) error {
	var evidenceOutput map[string]any
	if err := json.Unmarshal(data, &evidenceOutput); err != nil {
		return fmt.Errorf("failed to parse evidence output: %w", err)
	}

	schemaVersion, _ := evidenceOutput["schemaVersion"].(string)
	typeString, _ := evidenceOutput["type"].(string)

	typeField := SubjectType(typeString)

	log.Debug("Processing evidence with type:", typeField)

	switch typeField {
	case ReleaseBundleType:
		var releaseBundleOutput ReleaseBundleOutput
		if err := json.Unmarshal(data, &releaseBundleOutput); err != nil {
			return fmt.Errorf("failed to parse release bundle output: %w", err)
		}
		return writeReleaseBundleJsonlFromStruct(schemaVersion, typeField, releaseBundleOutput.Result, file)
	case EntityType:
		var entityEvidenceOutput EntityEvidenceOutput
		if err := json.Unmarshal(data, &entityEvidenceOutput); err != nil {
			return fmt.Errorf("failed to parse entity evidence output: %w", err)
		}
		return writeEvidenceEntriesJsonl(schemaVersion, typeField, entityEvidenceOutput.Result.Evidence, file)
	default:
		var customEvidenceOutput CustomEvidenceOutput
		if err := json.Unmarshal(data, &customEvidenceOutput); err != nil {
			return fmt.Errorf("failed to parse custom evidence output: %w", err)
		}
		return writeEvidenceEntriesJsonl(schemaVersion, typeField, customEvidenceOutput.Result.Evidence, file)
	}
}

func writeEvidenceEntriesJsonl(schemaVersion string, typeField SubjectType, evidence []EvidenceEntry, file *os.File) error {
	for _, entry := range evidence {
		lineWithMetadata := JsonlLine{
			SchemaVersion: schemaVersion,
			Type:          typeField,
			Result:        entry,
		}
		jsonLine, err := json.Marshal(lineWithMetadata)
		if err != nil {
			return fmt.Errorf("failed to marshal evidence line: %w", err)
		}
		if _, err := file.Write(append(jsonLine, '\n')); err != nil {
			return fmt.Errorf("failed to write evidence line: %w", err)
		}
	}

	if file != os.Stdout {
		log.Info("Evidence successfully exported to file name: ", file.Name())
	}
	return nil
}

var (
	errSearchEvidenceMissing      = errors.New("invalid GraphQL response structure: missing searchEvidence")
	errSearchEvidenceEdgesMissing = errors.New("invalid GraphQL response structure: missing edges")
)

// evidenceEntriesFromSearchResponse parses a One-Model searchEvidence GraphQL payload into
// ordered evidence entries. Callers map errSearchEvidenceMissing / errSearchEvidenceEdgesMissing
// to subject-specific errors when needed.
func evidenceEntriesFromSearchResponse(rawEvidence []byte, includePredicate bool) ([]EvidenceEntry, error) {
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
		return nil, errSearchEvidenceMissing
	}

	edges, ok := searchEvidenceData["edges"].([]any)
	if !ok {
		return nil, errSearchEvidenceEdgesMissing
	}

	evidenceArray := make([]EvidenceEntry, 0, len(edges))
	for _, edge := range edges {
		if edgeMap, ok := edge.(map[string]any); ok {
			if node, ok := edgeMap["node"].(map[string]any); ok {
				evidenceArray = append(evidenceArray, createOrderedEvidenceEntry(node, includePredicate))
			}
		}
	}
	return evidenceArray, nil
}

func writeReleaseBundleJsonlFromStruct(schemaVersion string, typeField SubjectType, result ReleaseBundleResult, file *os.File) error {
	for _, evidence := range result.Evidence {
		lineWithMetadata := JsonlLine{
			SchemaVersion: schemaVersion,
			Type:          typeField,
			Result:        evidence,
		}
		jsonLine, err := json.Marshal(lineWithMetadata)
		if err != nil {
			return fmt.Errorf("failed to marshal release bundle evidence line: %w", err)
		}
		if _, err := file.Write(append(jsonLine, '\n')); err != nil {
			return fmt.Errorf("failed to write evidence line: %w", err)
		}
	}

	for _, artifact := range result.Artifacts {
		lineWithMetadata := JsonlLine{
			SchemaVersion: schemaVersion,
			Type:          ArtifactType,
			Result:        artifact,
		}
		jsonLine, err := json.Marshal(lineWithMetadata)
		if err != nil {
			return fmt.Errorf("failed to marshal artifact evidence line: %w", err)
		}
		if _, err := file.Write(append(jsonLine, '\n')); err != nil {
			return fmt.Errorf("failed to write evidence line: %w", err)
		}
	}

	for _, build := range result.Builds {
		lineWithMetadata := JsonlLine{
			SchemaVersion: schemaVersion,
			Type:          BuildType,
			Result:        build,
		}
		jsonLine, err := json.Marshal(lineWithMetadata)
		if err != nil {
			return fmt.Errorf("failed to marshal build evidence line: %w", err)
		}
		if _, err := file.Write(append(jsonLine, '\n')); err != nil {
			return fmt.Errorf("failed to write evidence line: %w", err)
		}
	}

	if file != os.Stdout {
		log.Info("Evidence successfully exported to file name: ", file.Name())
	}

	return nil
}

func createOrderedEvidenceEntry(node map[string]any, includePredicate bool) EvidenceEntry {
	entry := EvidenceEntry{}

	if predicateSlug, ok := node["predicateSlug"].(string); ok {
		entry.PredicateSlug = predicateSlug
	}

	if predicateType, ok := node["predicateType"].(string); ok && predicateType != "" {
		entry.PredicateType = predicateType
	}

	if downloadPath, ok := node["downloadPath"].(string); ok {
		entry.DownloadPath = downloadPath
	}

	if verified, ok := node["verified"].(bool); ok {
		entry.Verified = verified
	}

	if signingKeyRaw, exists := node["signingKey"]; exists && signingKeyRaw != nil {
		if signingKey, ok := signingKeyRaw.(map[string]any); ok && len(signingKey) > 0 {
			entry.SigningKey = signingKey
		}
	}

	if subject, ok := node["subject"].(map[string]any); ok {
		entry.Subject = subject
	}

	if createdBy, ok := node["createdBy"].(string); ok {
		entry.CreatedBy = createdBy
	}

	if createdAt, ok := node["createdAt"].(string); ok {
		entry.CreatedAt = createdAt
	}

	if includePredicate {
		if predicate, ok := node["predicate"].(map[string]any); ok {
			entry.Predicate = predicate
		}
	}

	if attachments, ok := extractEvidenceAttachments(node); ok {
		entry.Attachments = attachments
	}

	return entry
}

func extractEvidenceAttachments(node map[string]any) ([]EvidenceAttachment, bool) {
	attachmentsRaw, exists := node["attachments"]
	if !exists || attachmentsRaw == nil {
		return nil, false
	}

	items, ok := attachmentsRaw.([]any)
	if !ok || len(items) == 0 {
		return nil, false
	}

	attachments := make([]EvidenceAttachment, 0, len(items))
	for _, item := range items {
		attMap, ok := item.(map[string]any)
		if !ok {
			continue
		}

		name, _ := attMap["name"].(string)
		sha256, _ := attMap["sha256"].(string)
		downloadPath, _ := attMap["downloadPath"].(string)

		mimeType, _ := attMap["type"].(string)

		attachments = append(attachments, EvidenceAttachment{
			Name:         name,
			Sha256:       sha256,
			Type:         mimeType,
			DownloadPath: downloadPath,
		})
	}

	if len(attachments) == 0 {
		return nil, false
	}
	return attachments, true
}
