package get

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jfrog/jfrog-cli-evidence/evidence/model"
	evidenceutils "github.com/jfrog/jfrog-cli-evidence/evidence/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Query templates as they were before --include-revoked. Without the flag the CLI must send
// exactly these queries so older Evidence servers keep working.
const (
	legacyCustomQueryTemplate        = `{"query":"{ evidence { searchEvidence( where: { hasSubjectWith: { repositoryKey: \"%s\", path: \"%s\", name: \"%s\"}} ) { totalCount edges { node { ` + evidenceutils.NodeFieldsPlaceholder + ` } } } } }"}`
	legacyEntityQueryTemplate        = `{"query":"{ evidence { searchEvidence( where: { hasSubjectWith: { %s }} ) { totalCount edges { node { ` + evidenceutils.NodeFieldsPlaceholder + ` } } } } }"}`
	legacyReleaseBundleQueryTemplate = `{"query":"{ releaseBundleVersion { getVersion(repositoryKey: \"%s\", name: \"%s\", version: \"%s\") { evidenceConnection { edges { node { ` + evidenceutils.NodeFieldsPlaceholder + ` } } } artifactsConnection(first: %s, after: \"YXJ0aWZhY3Q6MA==\", where: { hasEvidence: true }) { totalCount edges { node { sourceRepositoryPath packageType evidenceConnection(first: 0) { totalCount edges { node { ` + evidenceutils.NodeFieldsPlaceholder + ` } } } } } } fromBuilds { name number startedAt evidenceConnection { edges { node { ` + evidenceutils.NodeFieldsPlaceholder + ` } } } } } } }"}`

	legacyNodeFieldsWithAttachments = "predicateSlug predicateType downloadPath verified signingKey { alias } createdBy createdAt subject { sha256 } attachments { name sha256 type downloadPath }"
	revokedNodeFields               = "revoked revocation { revokedAt revokedBy }"

	introspectionWithRevoked    = `{"data":{"__type":{"fields":[{"name":"id"},{"name":"verified"},{"name":"revoked"},{"name":"revocation"}]}}}`
	introspectionWithoutRevoked = `{"data":{"__type":{"fields":[{"name":"id"},{"name":"verified"}]}}}`

	revokedSearchResponse = `{"data":{"evidence":{"searchEvidence":{"totalCount":2,"edges":[` +
		`{"node":{"predicateSlug":"active-slug","downloadPath":"repo/.evidence/active.json","verified":true,"revoked":false,"revocation":null,"createdBy":"ci","createdAt":"2026-10-01T00:00:00Z","subject":{"sha256":"abc"}}},` +
		`{"node":{"predicateSlug":"revoked-slug","downloadPath":"repo/.evidence/revoked.json","verified":true,"revoked":true,"revocation":{"revokedAt":"2026-10-15T10:00:00Z","revokedBy":"admin"},"createdBy":"ci","createdAt":"2026-10-02T00:00:00Z","subject":{"sha256":"abc"}}}` +
		`]}}}}`

	revokedReleaseBundleResponse = `{"data":{"releaseBundleVersion":{"getVersion":{` +
		`"evidenceConnection":{"edges":[{"node":{"predicateSlug":"rb-revoked","downloadPath":"rb/.evidence/rb.json","verified":true,"revoked":true,"revocation":{"revokedAt":"2026-10-15T10:00:00Z","revokedBy":"admin"},"createdBy":"ci","createdAt":"2026-10-01T00:00:00Z","subject":{"sha256":"rb"}}}]},` +
		`"artifactsConnection":{"totalCount":1,"edges":[{"node":{"sourceRepositoryPath":"generic-local/app.tgz","packageType":"generic","evidenceConnection":{"totalCount":1,"edges":[{"node":{"predicateSlug":"artifact-revoked","downloadPath":"generic-local/.evidence/a.json","verified":false,"revoked":true,"revocation":{"revokedAt":"2026-10-16T10:00:00Z","revokedBy":"alice"},"createdBy":"ci","createdAt":"2026-10-01T00:00:00Z","subject":{"sha256":"art"}}}]}}}]},` +
		`"fromBuilds":[{"name":"my-build","number":"7","startedAt":"2026-10-01T00:00:00Z","evidenceConnection":{"edges":[{"node":{"predicateSlug":"build-active","downloadPath":"build-info/.evidence/b.json","verified":true,"revoked":false,"revocation":null,"createdBy":"ci","createdAt":"2026-10-01T00:00:00Z","subject":{"sha256":"bld"}}}]}}]` +
		`}}}}`
)

type mockOnemodelManagerRevocation struct {
	supportsRevocation bool
	searchResponse     string
	queries            []string
}

func (m *mockOnemodelManagerRevocation) GraphqlQuery(query []byte) ([]byte, error) {
	m.queries = append(m.queries, string(query))
	if strings.Contains(string(query), "__type") {
		if m.supportsRevocation {
			return []byte(introspectionWithRevoked), nil
		}
		return []byte(introspectionWithoutRevoked), nil
	}
	return []byte(m.searchResponse), nil
}

func newCustomGetter(includeRevoked bool) *getEvidenceCustom {
	return &getEvidenceCustom{
		subjectRepoPath: "generic-local/dir/app.tgz",
		getEvidenceBase: getEvidenceBase{format: "json", includeRevoked: includeRevoked},
	}
}

func newEntityGetter(includeRevoked bool) *getEvidenceEntity {
	return &getEvidenceEntity{
		getEvidenceBase: getEvidenceBase{format: "json", includeRevoked: includeRevoked},
		EntitySubject:   model.EntitySubject{EntityType: "gitCommit", EntityID: "abc123", ProjectKey: "proj"},
	}
}

func newReleaseBundleGetter(includeRevoked bool) *getEvidenceReleaseBundle {
	return &getEvidenceReleaseBundle{
		getEvidenceBase:      getEvidenceBase{format: "json", includeRevoked: includeRevoked},
		project:              "proj",
		releaseBundle:        "my-rb",
		releaseBundleVersion: "1.0.0",
		artifactsLimit:       "5",
	}
}

func TestBuildQuery_WithoutIncludeRevoked_MatchesLegacyQuery(t *testing.T) {
	custom, err := newCustomGetter(false).buildGraphqlQuery("generic-local/dir/app.tgz", true)
	require.NoError(t, err)
	expectedCustom := fmt.Sprintf(evidenceutils.BuildQuery(legacyCustomQueryTemplate, legacyNodeFieldsWithAttachments), "generic-local", "dir", "app.tgz")
	assert.Equal(t, expectedCustom, string(custom))

	entity := newEntityGetter(false)
	expectedEntity := fmt.Sprintf(evidenceutils.BuildQuery(legacyEntityQueryTemplate,
		strings.Replace(legacyNodeFieldsWithAttachments, "subject { sha256 }", evidenceutils.FieldSubjectWithPath, 1)),
		evidenceutils.BuildGraphQLEntityHasSubjectWith(entity.EntitySubject))
	assert.Equal(t, expectedEntity, string(entity.buildGraphqlQuery(true)))

	releaseBundle := newReleaseBundleGetter(false)
	expectedReleaseBundle := fmt.Sprintf(evidenceutils.BuildQuery(legacyReleaseBundleQueryTemplate, legacyNodeFieldsWithAttachments),
		"proj-release-bundles-v2", "my-rb", "1.0.0", "5")
	assert.Equal(t, expectedReleaseBundle, string(releaseBundle.buildGraphqlQuery("my-rb", "1.0.0", true)))
}

func TestBuildQuery_WithIncludeRevoked_SearchEvidence(t *testing.T) {
	custom, err := newCustomGetter(true).buildGraphqlQuery("generic-local/dir/app.tgz", false)
	require.NoError(t, err)
	entity := newEntityGetter(true).buildGraphqlQuery(false)

	for name, query := range map[string]string{"custom": string(custom), "entity": string(entity)} {
		t.Run(name, func(t *testing.T) {
			assert.True(t, json.Valid([]byte(query)), "query must stay valid JSON: %s", query)
			assert.Contains(t, query, "}, includeRevoked: true} ) {")
			assert.Equal(t, 1, strings.Count(query, evidenceutils.IncludeRevokedArgument))
			assert.Contains(t, query, revokedNodeFields)
			assert.Contains(t, query, "verified")
			assert.NotContains(t, query, "{{")
		})
	}
	assert.Contains(t, string(custom), `name: \"app.tgz\"}, includeRevoked: true}`)
}

func TestBuildQuery_WithIncludeRevoked_ReleaseBundle(t *testing.T) {
	query := string(newReleaseBundleGetter(true).buildGraphqlQuery("my-rb", "1.0.0", true))

	assert.True(t, json.Valid([]byte(query)), "query must stay valid JSON: %s", query)
	assert.Equal(t, 3, strings.Count(query, evidenceutils.IncludeRevokedArgument))
	assert.Equal(t, 2, strings.Count(query, "evidenceConnection(includeRevoked: true) { edges"))
	assert.Equal(t, 1, strings.Count(query, "evidenceConnection(first: 0, includeRevoked: true) { totalCount"))
	assert.Equal(t, 3, strings.Count(query, revokedNodeFields))
	assert.Contains(t, query, "where: { hasEvidence: true }")
	assert.NotContains(t, query, "{{")
}

func TestGetEvidence_WithoutIncludeRevoked_SkipsFeatureDetection(t *testing.T) {
	manager := &mockOnemodelManagerRevocation{searchResponse: revokedSearchResponse}

	_, err := newCustomGetter(false).getEvidence(manager)
	require.NoError(t, err)

	require.Len(t, manager.queries, 1)
	assert.NotContains(t, manager.queries[0], "__type")
	assert.NotContains(t, manager.queries[0], "includeRevoked")
	assert.NotContains(t, manager.queries[0], "revoked")
}

func TestGetEvidence_IncludeRevoked_UnsupportedServer(t *testing.T) {
	getters := map[string]func(*mockOnemodelManagerRevocation) error{
		"custom": func(m *mockOnemodelManagerRevocation) error {
			_, err := newCustomGetter(true).getEvidence(m)
			return err
		},
		"entity": func(m *mockOnemodelManagerRevocation) error {
			_, err := newEntityGetter(true).getEvidence(m)
			return err
		},
		"release-bundle": func(m *mockOnemodelManagerRevocation) error {
			_, err := newReleaseBundleGetter(true).getEvidence(m)
			return err
		},
	}
	for name, run := range getters {
		t.Run(name, func(t *testing.T) {
			manager := &mockOnemodelManagerRevocation{supportsRevocation: false, searchResponse: revokedSearchResponse}

			err := run(manager)

			assert.ErrorIs(t, err, evidenceutils.ErrEvidenceRevocationNotSupported)
			assert.Contains(t, err.Error(), "--include-revoked requires an Evidence service version that supports revocation")
			require.Len(t, manager.queries, 1, "no evidence query may be sent to a server without revocation support")
			assert.Contains(t, manager.queries[0], "__type")
		})
	}
}

func TestGetCustomEvidence_IncludeRevoked_PrintsRevokedEntries(t *testing.T) {
	manager := &mockOnemodelManagerRevocation{supportsRevocation: true, searchResponse: revokedSearchResponse}

	result, err := newCustomGetter(true).getEvidence(manager)
	require.NoError(t, err)

	require.Len(t, manager.queries, 2)
	assert.Contains(t, manager.queries[0], "__type")
	assert.Contains(t, manager.queries[1], "includeRevoked: true")

	var output CustomEvidenceOutput
	require.NoError(t, json.Unmarshal(result, &output))
	require.Len(t, output.Result.Evidence, 2)

	active, revoked := output.Result.Evidence[0], output.Result.Evidence[1]
	require.NotNil(t, active.Revoked)
	assert.False(t, *active.Revoked)
	assert.Empty(t, active.RevokedAt)
	assert.Empty(t, active.RevokedBy)

	require.NotNil(t, revoked.Revoked)
	assert.True(t, *revoked.Revoked)
	assert.True(t, revoked.Verified, "verified is the stored value and is not changed by revoke")
	assert.Equal(t, "2026-10-15T10:00:00Z", revoked.RevokedAt)
	assert.Equal(t, "admin", revoked.RevokedBy)

	assert.Contains(t, string(result), `"revoked": false`)
	assert.Contains(t, string(result), `"revokedBy": "admin"`)
}

func TestGetEntityEvidence_IncludeRevoked_PrintsRevokedEntries(t *testing.T) {
	manager := &mockOnemodelManagerRevocation{supportsRevocation: true, searchResponse: revokedSearchResponse}

	result, err := newEntityGetter(true).getEvidence(manager)
	require.NoError(t, err)

	var output EntityEvidenceOutput
	require.NoError(t, json.Unmarshal(result, &output))
	require.Len(t, output.Result.Evidence, 2)
	assert.Equal(t, "admin", output.Result.Evidence[1].RevokedBy)
	assert.True(t, *output.Result.Evidence[1].Revoked)
}

func TestGetReleaseBundleEvidence_IncludeRevoked_PrintsRevokedEntries(t *testing.T) {
	manager := &mockOnemodelManagerRevocation{supportsRevocation: true, searchResponse: revokedReleaseBundleResponse}

	result, err := newReleaseBundleGetter(true).getEvidence(manager)
	require.NoError(t, err)
	require.Len(t, manager.queries, 2)
	assert.Equal(t, 3, strings.Count(manager.queries[1], evidenceutils.IncludeRevokedArgument))

	var output ReleaseBundleOutput
	require.NoError(t, json.Unmarshal(result, &output))

	require.Len(t, output.Result.Evidence, 1)
	assert.True(t, *output.Result.Evidence[0].Revoked)
	assert.Equal(t, "admin", output.Result.Evidence[0].RevokedBy)

	require.Len(t, output.Result.Artifacts, 1)
	artifact := output.Result.Artifacts[0].Evidence
	assert.True(t, *artifact.Revoked)
	assert.False(t, artifact.Verified)
	assert.Equal(t, "2026-10-16T10:00:00Z", artifact.RevokedAt)
	assert.Equal(t, "alice", artifact.RevokedBy)

	require.Len(t, output.Result.Builds, 1)
	assert.False(t, *output.Result.Builds[0].Evidence.Revoked)
}

func TestEvidenceEntry_WithoutRevokedFields_OmitsThemFromJson(t *testing.T) {
	entry := createOrderedEvidenceEntry(map[string]any{"predicateSlug": "slug", "verified": true}, false)

	assert.Nil(t, entry.Revoked)
	encoded, err := json.Marshal(entry)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "revoked")
}

func TestEvidenceEntry_NullRevocationFields(t *testing.T) {
	entry := createOrderedEvidenceEntry(map[string]any{
		"revoked":    true,
		"revocation": map[string]any{"revokedAt": nil, "revokedBy": nil},
	}, false)

	require.NotNil(t, entry.Revoked)
	assert.True(t, *entry.Revoked)
	assert.Empty(t, entry.RevokedAt)
	assert.Empty(t, entry.RevokedBy)
}

func TestWriteEvidenceJsonl_IncludesRevokedFields(t *testing.T) {
	customResult, err := newCustomGetter(true).getEvidence(&mockOnemodelManagerRevocation{supportsRevocation: true, searchResponse: revokedSearchResponse})
	require.NoError(t, err)
	releaseBundleResult, err := newReleaseBundleGetter(true).getEvidence(&mockOnemodelManagerRevocation{supportsRevocation: true, searchResponse: revokedReleaseBundleResponse})
	require.NoError(t, err)

	tests := []struct {
		name          string
		data          []byte
		expectedLines int
		revokedBy     []string
	}{
		{name: "custom", data: customResult, expectedLines: 2, revokedBy: []string{"", "admin"}},
		{name: "release-bundle", data: releaseBundleResult, expectedLines: 3, revokedBy: []string{"admin", "alice", ""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputFile := filepath.Join(t.TempDir(), "evidence.jsonl")
			require.NoError(t, exportEvidenceToJsonlFile(tt.data, outputFile))

			lines := readJsonlLines(t, outputFile)
			require.Len(t, lines, tt.expectedLines)
			for i, line := range lines {
				evidenceEntry := line
				if nested, ok := line["evidence"].(map[string]any); ok {
					evidenceEntry = nested
				}
				_, hasRevoked := evidenceEntry["revoked"]
				assert.True(t, hasRevoked, "line %d must carry revoked", i)
				revokedBy, _ := evidenceEntry["revokedBy"].(string)
				assert.Equal(t, tt.revokedBy[i], revokedBy, "line %d", i)
			}
		})
	}
}

func readJsonlLines(t *testing.T, path string) []map[string]any {
	file, err := os.Open(path)
	require.NoError(t, err)
	defer func() { _ = file.Close() }()

	var results []map[string]any
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var line JsonlLine
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &line))
		result, ok := line.Result.(map[string]any)
		require.True(t, ok)
		results = append(results, result)
	}
	require.NoError(t, scanner.Err())
	return results
}
