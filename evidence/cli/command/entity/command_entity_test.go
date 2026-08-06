package entity

import (
	"flag"
	"testing"

	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/command/flags"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli"
)

func TestValidateEvidenceEntityContext(t *testing.T) {
	tests := []struct {
		name          string
		flags         []components.Flag
		errorContains string
	}{
		{
			name: "valid project scope",
			flags: []components.Flag{
				test.SetDefaultValue(flags.EntityType, "gitCommit"),
				test.SetDefaultValue(flags.EntityId, "abc123"),
				test.SetDefaultValue(flags.Project, "proj"),
			},
		},
		{
			name: "missing entity id",
			flags: []components.Flag{
				test.SetDefaultValue(flags.EntityType, "gitCommit"),
			},
			errorContains: "--entity-id is a mandatory field",
		},
		{
			name: "multiple scopes",
			flags: []components.Flag{
				test.SetDefaultValue(flags.EntityType, "gitCommit"),
				test.SetDefaultValue(flags.EntityId, "abc123"),
				test.SetDefaultValue(flags.Project, "proj"),
				test.SetDefaultValue(flags.EntityRepo, "gitCommit-entity"),
			},
			errorContains: "only one entity scope",
		},
		{
			name: "application version conflict",
			flags: []components.Flag{
				test.SetDefaultValue(flags.EntityType, "application"),
				test.SetDefaultValue(flags.EntityId, "my-app"),
				test.SetDefaultValue(flags.ApplicationVersion, "1.0.0"),
			},
			errorContains: "--application-version cannot be combined",
		},
		{
			name: "sigstore bundle allowed",
			flags: []components.Flag{
				test.SetDefaultValue(flags.EntityType, "gitCommit"),
				test.SetDefaultValue(flags.EntityId, "abc123"),
				test.SetDefaultValue(flags.SigstoreBundle, "/path/to/bundle.json"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := cli.NewApp()
			app.Commands = []cli.Command{{Name: "create"}}
			set := flag.NewFlagSet("test", 0)
			cliCtx := cli.NewContext(app, set, nil)

			ctx, err := components.ConvertContext(cliCtx, tt.flags...)
			require.NoError(t, err)

			cmd, ok := NewEvidenceEntityCommand(ctx, nil).(*evidenceEntityCommand)
			require.True(t, ok)
			err = cmd.validateEvidenceEntityContext(ctx)
			if tt.errorContains == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			}
		})
	}
}
