package releasebundle

import (
	"flag"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/command/flags"
	testUtil "github.com/jfrog/jfrog-cli-evidence/evidence/cli/test"
	"testing"

	"github.com/jfrog/jfrog-cli-core/v2/common/commands"
	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"
	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
)

func TestEvidenceReleaseBundleCommand_CreateEvidence_SigstoreBundle(t *testing.T) {
	tests := []struct {
		name          string
		flags         []components.Flag
		expectError   bool
		errorContains string
	}{
		{
			name: "Invalid_SigstoreBundle_Not_Supported",
			flags: []components.Flag{
				testUtil.SetDefaultValue(flags.SigstoreBundle, "/path/to/bundle.json"),
				testUtil.SetDefaultValue(flags.ReleaseBundle, "test-release-bundle"),
				testUtil.SetDefaultValue(flags.ReleaseBundleVersion, "1.0.0"),
			},
			expectError:   true,
			errorContains: "--sigstore-bundle is not supported for release bundle evidence.",
		},
		{
			name: "Valid_Without_SigstoreBundle",
			flags: []components.Flag{
				testUtil.SetDefaultValue(flags.ReleaseBundle, "test-release-bundle"),
				testUtil.SetDefaultValue(flags.ReleaseBundleVersion, "1.0.0"),
				testUtil.SetDefaultValue(flags.Predicate, "/path/to/predicate.json"),
				testUtil.SetDefaultValue(flags.PredicateType, "test-type"),
				testUtil.SetDefaultValue(flags.Key, "/path/to/key.pem"),
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

			cmd := NewEvidenceReleaseBundleCommand(ctx, mockExec)
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
