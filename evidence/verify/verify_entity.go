package verify

import (
	"fmt"

	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli-evidence/evidence"
	"github.com/jfrog/jfrog-cli-evidence/evidence/model"
	evidenceutils "github.com/jfrog/jfrog-cli-evidence/evidence/utils"
)

const searchEvidenceByEntityQueryTemplate = `{"query":"{ evidence { searchEvidence( where: { hasSubjectWith: { %s }} ) { edges { cursor node { ` + evidenceutils.NodeFieldsPlaceholder + ` } } } } }"}`

type verifyEvidenceEntity struct {
	verifyEvidenceBase
	model.EntitySubject
}

// NewVerifyEvidenceEntity creates a command for verifying evidence attached to an entity subject.
func NewVerifyEvidenceEntity(serverDetails *config.ServerDetails, entityType, entityID, entityRepo, projectKey, applicationKey, format string, keys []string, useArtifactoryKeys bool) evidence.Command {
	return &verifyEvidenceEntity{
		verifyEvidenceBase: newVerifyEvidenceBase(serverDetails, format, keys, useArtifactoryKeys),
		EntitySubject: model.EntitySubject{
			EntityType:     entityType,
			EntityID:       entityID,
			EntityRepo:     entityRepo,
			ProjectKey:     projectKey,
			ApplicationKey: applicationKey,
		},
	}
}

func (v *verifyEvidenceEntity) Run() error {
	defer v.quitProgress()

	if err := evidenceutils.ResolveApplicationEntityProjectKey(v.serverDetails, &v.EntitySubject); err != nil {
		return err
	}

	client, err := v.createArtifactoryClient()
	if err != nil {
		return fmt.Errorf("failed to create Artifactory client: %w", err)
	}

	metadata, err := v.queryEvidenceMetadataWithQueries(
		v.buildSearchEvidenceByEntityQuery(true),
		v.buildSearchEvidenceByEntityQuery(false),
	)
	if err != nil {
		return err
	}

	// Entity subjects have no content checksum. They are verified through the entity digest
	// carried in the signed in-toto statement.
	expectedSubject := model.SubjectDigest{Type: v.EntityType, Value: v.EntityID}
	subjectPath := fmt.Sprintf("%s/%s", v.EntityType, v.EntityID)
	return v.verifyEvidence(client, metadata, expectedSubject, subjectPath)
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
	whereClause := evidenceutils.BuildGraphQLEntityHasSubjectWith(v.EntitySubject)
	return fmt.Sprintf(evidenceutils.BuildQuery(searchEvidenceByEntityQueryTemplate, nodeFields), whereClause)
}

func (v *verifyEvidenceEntity) ServerDetails() (*config.ServerDetails, error) {
	return v.serverDetails, nil
}

func (v *verifyEvidenceEntity) CommandName() string {
	return "verify-evidence-entity"
}
