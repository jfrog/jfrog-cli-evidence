package cli

import (
	"flag"
	"testing"

	"github.com/jfrog/jfrog-cli-core/v2/common/commands"
	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"
	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
)

func TestEvidenceApplicationCommand_CreateEvidence_SigstoreBundle(t *testing.T) {
	tests := []struct {
		name          string
		flags         []components.Flag
		expectError   bool
		errorContains string
	}{
		{
			name: "Invalid_SigstoreBundle_Not_Supported",
			flags: []components.Flag{
				setDefaultValue(sigstoreBundle, "/path/to/bundle.json"),
				setDefaultValue(applicationKey, "test-app"),
				setDefaultValue(applicationVersion, "1.0.0"),
			},
			expectError:   true,
			errorContains: "--sigstore-bundle is not supported for application evidence.",
		},
		{
			name: "Valid_Without_SigstoreBundle",
			flags: []components.Flag{
				setDefaultValue(applicationKey, "test-app"),
				setDefaultValue(applicationVersion, "1.0.0"),
				setDefaultValue(predicate, "/path/to/predicate.json"),
				setDefaultValue(predicateType, "test-type"),
				setDefaultValue(key, "/path/to/key.pem"),
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

			cmd, ok := NewEvidenceApplicationCommand(ctx, mockExec).(*evidenceApplicationCommand)
			if !ok {
				t.Fatalf("NewEvidenceApplicationCommand returned a non-evidenceApplicationCommand")
			}
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

func TestEvidenceApplicationCommand_ValidateEvidenceApplicationContext(t *testing.T) {
	tests := []struct {
		name          string
		flags         []components.Flag
		expectError   bool
		errorContains string
	}{
		{
			name: "Valid_All_Required_Fields",
			flags: []components.Flag{
				setDefaultValue(applicationKey, "test-app"),
				setDefaultValue(applicationVersion, "1.0.0"),
			},
			expectError: false,
		},
		{
			name: "Invalid_Missing_ApplicationKey",
			flags: []components.Flag{
				setDefaultValue(applicationVersion, "1.0.0"),
			},
			expectError:   true,
			errorContains: "--application-key is a mandatory field for creating an Application evidence",
		},
		{
			name: "Invalid_Missing_ApplicationVersion",
			flags: []components.Flag{
				setDefaultValue(applicationKey, "test-app"),
			},
			expectError:   true,
			errorContains: "--application-version is a mandatory field for creating an Application evidence",
		},
		{
			name: "Invalid_Empty_ApplicationKey",
			flags: []components.Flag{
				setDefaultValue(applicationKey, ""),
				setDefaultValue(applicationVersion, "1.0.0"),
			},
			expectError:   true,
			errorContains: "--application-key is a mandatory field for creating an Application evidence",
		},
		{
			name: "Invalid_Empty_ApplicationVersion",
			flags: []components.Flag{
				setDefaultValue(applicationKey, "test-app"),
				setDefaultValue(applicationVersion, ""),
			},
			expectError:   true,
			errorContains: "--application-version is a mandatory field for creating an Application evidence",
		},
		{
			name: "Invalid_Project_Flag_Not_Allowed",
			flags: []components.Flag{
				setDefaultValue(applicationKey, "test-app"),
				setDefaultValue(applicationVersion, "1.0.0"),
				setDefaultValue(project, "test-project"),
			},
			expectError:   true,
			errorContains: "--project flag is not allowed when using application-based evidence creation",
		},
		{
			name: "Invalid_SigstoreBundle_Not_Supported",
			flags: []components.Flag{
				setDefaultValue(applicationKey, "test-app"),
				setDefaultValue(applicationVersion, "1.0.0"),
				setDefaultValue(sigstoreBundle, "/path/to/bundle.json"),
			},
			expectError:   true,
			errorContains: "--sigstore-bundle is not supported for application evidence.",
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
				return nil
			}

			cmd, ok := NewEvidenceApplicationCommand(ctx, mockExec).(*evidenceApplicationCommand)
			if !ok {
				t.Fatalf("NewEvidenceApplicationCommand returned a non-evidenceApplicationCommand")
			}

			err = cmd.validateEvidenceApplicationContext(ctx)

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

func TestEvidenceApplicationCommand_GetEvidence(t *testing.T) {
	app := cli.NewApp()
	app.Commands = []cli.Command{{Name: "get"}}
	set := flag.NewFlagSet("test", 0)
	cliCtx := cli.NewContext(app, set, nil)

	ctx, err := components.ConvertContext(cliCtx)
	assert.NoError(t, err)

	mockExec := func(cmd commands.Command) error {
		return nil
	}

	cmd, ok := NewEvidenceApplicationCommand(ctx, mockExec).(*evidenceApplicationCommand)
	if !ok {
		t.Fatalf("NewEvidenceApplicationCommand returned a non-evidenceApplicationCommand")
	}
	serverDetails := &config.ServerDetails{}

	err = cmd.GetEvidence(ctx, serverDetails)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get evidence is not supported for application evidence yet")
}

func TestEvidenceApplicationCommand_VerifyEvidence(t *testing.T) {
	app := cli.NewApp()
	app.Commands = []cli.Command{{Name: "verify"}}
	set := flag.NewFlagSet("test", 0)
	cliCtx := cli.NewContext(app, set, nil)

	ctx, err := components.ConvertContext(cliCtx)
	assert.NoError(t, err)

	mockExec := func(cmd commands.Command) error {
		return nil
	}

	cmd := NewEvidenceApplicationCommand(ctx, mockExec)
	serverDetails := &config.ServerDetails{}

	err = cmd.VerifyEvidence(ctx, serverDetails)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "verify evidence is not supported for application evidence yet")
}

func TestNewEvidenceApplicationCommand(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	cliCtx := cli.NewContext(app, set, nil)

	ctx, err := components.ConvertContext(cliCtx)
	assert.NoError(t, err)

	mockExec := func(cmd commands.Command) error {
		return nil
	}

	appCmd, ok := NewEvidenceApplicationCommand(ctx, mockExec).(*evidenceApplicationCommand)
	if !ok {
		t.Fatalf("NewEvidenceApplicationCommand returned a non-evidenceApplicationCommand")
	}

	assert.NotNil(t, appCmd)

	assert.Equal(t, ctx, appCmd.ctx)
	assert.NotNil(t, appCmd.execute)
}
