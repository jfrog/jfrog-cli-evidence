package artifacts

import (
	"flag"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/command/flags"
	testUtils "github.com/jfrog/jfrog-cli-evidence/evidence/cli/test"

	"github.com/jfrog/jfrog-cli-core/v2/common/commands"
	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"
	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli"
)

func TestEvidenceCustomCommand_CreateEvidence_SigstoreBundle(t *testing.T) {
	tests := []struct {
		name          string
		flags         []components.Flag
		expectError   bool
		errorContains string
	}{
		{
			name: "Valid_SigstoreBundle_Without_SubjectSha256",
			flags: []components.Flag{
				testUtils.SetDefaultValue(flags.SigstoreBundle, "/path/to/bundle.json"),
				testUtils.SetDefaultValue(flags.SubjectRepoPath, "test-repo/test-artifact"),
			},
			expectError: false,
		},
		{
			name: "Invalid_SigstoreBundle_With_SubjectSha256",
			flags: []components.Flag{
				testUtils.SetDefaultValue(flags.SigstoreBundle, "/path/to/bundle.json"),
				testUtils.SetDefaultValue(flags.SubjectRepoPath, "test-repo/test-artifact"),
				testUtils.SetDefaultValue(flags.SubjectSha256, "abcd1234567890"),
			},
			expectError:   true,
			errorContains: "The parameter --subject-sha256 cannot be used with --sigstore-bundle",
		},
		{
			name: "Valid_No_SigstoreBundle_With_SubjectSha256",
			flags: []components.Flag{
				testUtils.SetDefaultValue(flags.SubjectRepoPath, "test-repo/test-artifact"),
				testUtils.SetDefaultValue(flags.SubjectSha256, "abcd1234567890"),
				testUtils.SetDefaultValue(flags.Predicate, "/path/to/predicate.json"),
				testUtils.SetDefaultValue(flags.PredicateType, "test-type"),
				testUtils.SetDefaultValue(flags.Key, "/path/to/key.pem"),
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := cli.NewApp()
			app.Commands = []cli.Command{{Name: "create"}}
			set := flag.NewFlagSet("test", 0)
			cliCtx := cli.NewContext(app, set, nil)

			ctx, err := components.ConvertContext(cliCtx, tt.flags...)
			assert.NoError(t, err)

			mockExec := func(cmd commands.Command) error {
				// Mock successful execution
				return nil
			}

			cmd := NewEvidenceCustomCommand(ctx, mockExec)
			serverDetails := &config.ServerDetails{}

			err = cmd.CreateEvidence(ctx, serverDetails)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEvidenceCustomCommand_GetEvidence_IncludeRevokedFlag(t *testing.T) {
	const search = `{"data":{"evidence":{"searchEvidence":{"totalCount":1,"edges":[{"node":{"predicateSlug":"slug","downloadPath":"p","verified":true,"revoked":true,"revocation":{"revokedAt":"2026-10-15T10:00:00Z","revokedBy":"admin"},"createdBy":"ci","createdAt":"2026-10-01T00:00:00Z","subject":{"sha256":"abc"}}}]}}}}`

	tests := []struct {
		name            string
		args            []string
		serverVersion   string
		expectedQueries int
		expectVersion   bool
		expectRevoked   bool
		errorContains   string
	}{
		{name: "flag set", args: []string{"--" + flags.IncludeRevoked}, serverVersion: "7.1695.0",
			expectedQueries: 1, expectVersion: true, expectRevoked: true},
		{name: "flag set on old server", args: []string{"--" + flags.IncludeRevoked}, serverVersion: "7.1694.0",
			expectVersion: true, errorContains: "requires version 7.1695.0 or higher"},
		{name: "flag not set", args: nil, serverVersion: "7.1694.0", expectedQueries: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("JFROG_CLI_HOME_DIR", t.TempDir())
			var queries []string
			versionChecked := false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/evidence/api/v1/system/version" {
					versionChecked = true
					_, _ = w.Write([]byte(tt.serverVersion))
					return
				}
				assert.Equal(t, "/onemodel/api/v1/graphql", r.URL.Path)
				body, err := io.ReadAll(r.Body)
				assert.NoError(t, err)
				queries = append(queries, string(body))
				_, _ = w.Write([]byte(search))
			}))
			defer server.Close()

			outputFile := filepath.Join(t.TempDir(), "evidence.json")
			set := flag.NewFlagSet("test", 0)
			set.Bool(flags.IncludeRevoked, false, "")
			require.NoError(t, set.Parse(tt.args))
			ctx, err := components.ConvertContext(cli.NewContext(cli.NewApp(), set, nil),
				testUtils.SetDefaultValue(flags.SubjectRepoPath, "generic-local/dir/app.tgz"),
				testUtils.SetDefaultValue(flags.Format, "json"),
				testUtils.SetDefaultValue(flags.Output, outputFile),
				components.NewBoolFlag(flags.IncludeRevoked, "", components.WithBoolDefaultValueFalse()),
			)
			require.NoError(t, err)

			runCommand := func(cmd commands.Command) error { return cmd.Run() }
			serverDetails := &config.ServerDetails{
				OnemodelUrl: server.URL + "/onemodel/", EvidenceUrl: server.URL + "/evidence/", AccessToken: "token",
			}
			err = NewEvidenceCustomCommand(ctx, runCommand).GetEvidence(ctx, serverDetails)

			assert.Equal(t, tt.expectVersion, versionChecked)
			require.Len(t, queries, tt.expectedQueries)
			if tt.errorContains != "" {
				assert.ErrorContains(t, err, tt.errorContains)
				return
			}
			require.NoError(t, err)
			searchQuery := queries[len(queries)-1]
			output, err := os.ReadFile(outputFile)
			require.NoError(t, err)
			if tt.expectRevoked {
				assert.Contains(t, searchQuery, "includeRevoked: true")
				assert.Contains(t, string(output), `"revokedBy": "admin"`)
			} else {
				assert.NotContains(t, searchQuery, "includeRevoked")
				assert.NotContains(t, searchQuery, "revoked")
			}
		})
	}
}
