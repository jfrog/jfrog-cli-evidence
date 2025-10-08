package command

import (
	"flag"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/command/flags"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/command/utils"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/test"
	"os"
	"testing"

	"github.com/jfrog/jfrog-cli-core/v2/common/commands"
	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"
	coreUtils "github.com/jfrog/jfrog-cli-core/v2/utils/coreutils"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
	"go.uber.org/mock/gomock"
)

func TestCreateEvidence_Context(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	assert.NoError(t, os.Setenv(coreUtils.SigningKey, "PGP"), "Failed to set env: "+coreUtils.SigningKey)
	assert.NoError(t, os.Setenv(coreUtils.BuildName, flags.BuildName), "Failed to set env: JFROG_CLI_BUILD_NAME")
	defer func() {
		assert.NoError(t, os.Unsetenv(coreUtils.SigningKey), "Failed to unset env: "+coreUtils.SigningKey)
	}()
	defer func() {
		assert.NoError(t, os.Unsetenv(coreUtils.BuildName), "Failed to unset env: JFROG_CLI_BUILD_NAME")
	}()

	app := cli.NewApp()
	app.Commands = []cli.Command{
		{
			Name: "create",
		},
	}
	set := flag.NewFlagSet(flags.Predicate, 0)
	ctx := cli.NewContext(app, set, nil)

	tests := []struct {
		name      string
		flags     []components.Flag
		expectErr bool
	}{
		{
			name: "InvalidContext - Missing Subject",
			flags: []components.Flag{
				test.SetDefaultValue(flags.Predicate, flags.Predicate),
				test.SetDefaultValue(flags.PredicateType, flags.PredicateType),
				test.SetDefaultValue(flags.Key, flags.Key),
			},
			expectErr: true,
		},
		{
			name: "InvalidContext - Missing Predicate",
			flags: []components.Flag{
				test.SetDefaultValue("", ""),
				test.SetDefaultValue(flags.PredicateType, "InToto"),
				test.SetDefaultValue(flags.Key, "PGP"),
			},
			expectErr: true,
		},
		{
			name: "InvalidContext - Subject Duplication",
			flags: []components.Flag{
				test.SetDefaultValue(flags.Predicate, flags.Predicate),
				test.SetDefaultValue(flags.PredicateType, "InToto"),
				test.SetDefaultValue(flags.Key, "PGP"),
				test.SetDefaultValue(flags.SubjectRepoPath, flags.SubjectRepoPath),
				test.SetDefaultValue(flags.ReleaseBundle, flags.ReleaseBundle),
				test.SetDefaultValue(flags.ReleaseBundleVersion, flags.ReleaseBundleVersion),
			},
			expectErr: true,
		},
		{
			name: "ValidContext - ReleaseBundle",
			flags: []components.Flag{
				test.SetDefaultValue(flags.Predicate, flags.Predicate),
				test.SetDefaultValue(flags.PredicateType, "InToto"),
				test.SetDefaultValue(flags.Key, "PGP"),
				test.SetDefaultValue(flags.ReleaseBundle, flags.ReleaseBundle),
				test.SetDefaultValue(flags.ReleaseBundleVersion, flags.ReleaseBundleVersion),
				test.SetDefaultValue("Url", "Url"),
			},
			expectErr: false,
		},
		{
			name: "ValidContext - RepoPath",
			flags: []components.Flag{
				test.SetDefaultValue(flags.Predicate, flags.Predicate),
				test.SetDefaultValue(flags.PredicateType, "InToto"),
				test.SetDefaultValue(flags.Key, "PGP"),
				test.SetDefaultValue(flags.SubjectRepoPath, flags.SubjectRepoPath),
				test.SetDefaultValue("Url", "Url"),
			},
			expectErr: false,
		},
		{
			name: "ValidContext - Build",
			flags: []components.Flag{
				test.SetDefaultValue(flags.Predicate, flags.Predicate),
				test.SetDefaultValue(flags.PredicateType, "InToto"),
				test.SetDefaultValue(flags.Key, "PGP"),
				test.SetDefaultValue(flags.BuildName, flags.BuildName),
				test.SetDefaultValue(flags.BuildNumber, flags.BuildNumber),
				test.SetDefaultValue("Url", "Url"),
			},
			expectErr: false,
		},
		{
			name: "ValidContext - Build With BuildNumber As Env Var",
			flags: []components.Flag{
				test.SetDefaultValue(flags.Predicate, flags.Predicate),
				test.SetDefaultValue(flags.PredicateType, "InToto"),
				test.SetDefaultValue(flags.Key, "PGP"),
				test.SetDefaultValue(flags.BuildNumber, flags.BuildNumber),
				test.SetDefaultValue("Url", "Url"),
			},
			expectErr: false,
		},
		{
			name: "InvalidContext - Build",
			flags: []components.Flag{
				test.SetDefaultValue(flags.Predicate, flags.Predicate),
				test.SetDefaultValue(flags.PredicateType, "InToto"),
				test.SetDefaultValue(flags.Key, "PGP"),
				test.SetDefaultValue(flags.BuildName, flags.BuildName),
				test.SetDefaultValue("Url", "Url"),
			},
			expectErr: true,
		},
		{
			name: "ValidContext - Package",
			flags: []components.Flag{
				test.SetDefaultValue(flags.Predicate, flags.Predicate),
				test.SetDefaultValue(flags.PredicateType, "InToto"),
				test.SetDefaultValue(flags.Key, "PGP"),
				test.SetDefaultValue(flags.PackageName, flags.PackageName),
				test.SetDefaultValue(flags.PackageVersion, flags.PackageVersion),
				test.SetDefaultValue(flags.PackageRepoName, flags.PackageRepoName),
				test.SetDefaultValue("Url", "Url"),
			},
			expectErr: false,
		},
		{
			name: "ValidContext With Key As Env Var- Package",
			flags: []components.Flag{
				test.SetDefaultValue(flags.Predicate, flags.Predicate),
				test.SetDefaultValue(flags.PredicateType, "InToto"),
				test.SetDefaultValue(flags.PackageName, flags.PackageName),
				test.SetDefaultValue(flags.PackageVersion, flags.PackageVersion),
				test.SetDefaultValue(flags.PackageRepoName, flags.PackageRepoName),
				test.SetDefaultValue("Url", "Url"),
			},
			expectErr: false,
		},
		{
			name: "InvalidContext - Missing package version",
			flags: []components.Flag{
				test.SetDefaultValue(flags.Predicate, flags.Predicate),
				test.SetDefaultValue(flags.PredicateType, "InToto"),
				test.SetDefaultValue(flags.Key, "PGP"),
				test.SetDefaultValue(flags.PackageName, flags.PackageName),
				test.SetDefaultValue(flags.PackageRepoName, flags.PackageRepoName),
				test.SetDefaultValue("Url", "Url"),
			},
			expectErr: true,
		},
		{
			name: "InvalidContext - Missing package repository key",
			flags: []components.Flag{
				test.SetDefaultValue(flags.Predicate, flags.Predicate),
				test.SetDefaultValue(flags.PredicateType, "InToto"),
				test.SetDefaultValue(flags.Key, "PGP"),
				test.SetDefaultValue(flags.PackageName, flags.PackageName),
				test.SetDefaultValue(flags.PackageVersion, flags.PackageVersion),
				test.SetDefaultValue("Url", "Url"),
			},
			expectErr: true,
		},
		{
			name: "InvalidContext - Unsupported Basic Auth",
			flags: []components.Flag{
				test.SetDefaultValue(flags.Predicate, flags.Predicate),
				test.SetDefaultValue(flags.PredicateType, "InToto"),
				test.SetDefaultValue(flags.Key, "PGP"),
				test.SetDefaultValue(flags.ReleaseBundle, flags.ReleaseBundle),
				test.SetDefaultValue("Url", "Url"),
				test.SetDefaultValue("User", "testUser"),
				test.SetDefaultValue("password", "testPassword"),
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			context, err1 := components.ConvertContext(ctx, tt.flags...)
			if err1 != nil {
				return
			}

			execFunc = func(command commands.Command) error {
				return nil
			}
			// Replace execFunc with the mockExec function
			defer func() { execFunc = utils.Exec }() // Restore original execFunc after test

			err := createEvidence(context)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestVerifyEvidence_Context(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	assert.NoError(t, os.Setenv(coreUtils.SigningKey, "PGP"), "Failed to set env: "+coreUtils.SigningKey)
	assert.NoError(t, os.Setenv(coreUtils.BuildName, flags.BuildName), "Failed to set env: JFROG_CLI_BUILD_NAME")
	defer func() {
		assert.NoError(t, os.Unsetenv(coreUtils.SigningKey), "Failed to unset env: "+coreUtils.SigningKey)
	}()
	defer func() {
		assert.NoError(t, os.Unsetenv(coreUtils.BuildName), "Failed to unset env: JFROG_CLI_BUILD_NAME")
	}()

	app := cli.NewApp()
	app.Commands = []cli.Command{
		{
			Name: "verify",
		},
	}
	set := flag.NewFlagSet(flags.Predicate, 0)
	ctx := cli.NewContext(app, set, nil)

	tests := []struct {
		name      string
		flags     []components.Flag
		expectErr bool
	}{
		{
			name: "InvalidContext - Missing Subject",
			flags: []components.Flag{
				test.SetDefaultValue(flags.PublicKeys, "PGP"),
				test.SetDefaultValue(flags.Format, "json"),
			},
			expectErr: true,
		},
		{
			name: "InvalidContext - Subject Duplication",
			flags: []components.Flag{
				test.SetDefaultValue(flags.PublicKeys, "PGP"),
				test.SetDefaultValue(flags.SubjectRepoPath, flags.SubjectRepoPath),
				test.SetDefaultValue(flags.ReleaseBundle, flags.ReleaseBundle),
				test.SetDefaultValue(flags.ReleaseBundleVersion, flags.ReleaseBundleVersion),
			},
			expectErr: true,
		},
		{
			name: "ValidContext - ReleaseBundle",
			flags: []components.Flag{
				test.SetDefaultValue(flags.ReleaseBundle, flags.ReleaseBundle),
				test.SetDefaultValue(flags.ReleaseBundleVersion, flags.ReleaseBundleVersion),
				test.SetDefaultValue("Url", "Url"),
			},
			expectErr: false,
		},
		{
			name: "ValidContext - RepoPath",
			flags: []components.Flag{
				test.SetDefaultValue(flags.SubjectRepoPath, flags.SubjectRepoPath),
				test.SetDefaultValue("Url", "Url"),
			},
			expectErr: false,
		},
		{
			name: "ValidContext - Build",
			flags: []components.Flag{
				test.SetDefaultValue(flags.PublicKeys, "PGP"),
				test.SetDefaultValue(flags.Format, "full"),
				test.SetDefaultValue(flags.BuildName, flags.BuildName),
				test.SetDefaultValue(flags.BuildNumber, flags.BuildNumber),
				test.SetDefaultValue("Url", "Url"),
			},
			expectErr: false,
		},
		{
			name: "InvalidContext - Build",
			flags: []components.Flag{
				test.SetDefaultValue(flags.BuildName, flags.BuildName),
				test.SetDefaultValue("Url", "Url"),
			},
			expectErr: true,
		},
		{
			name: "ValidContext - Package",
			flags: []components.Flag{
				test.SetDefaultValue(flags.PackageName, flags.PackageName),
				test.SetDefaultValue(flags.PackageVersion, flags.PackageVersion),
				test.SetDefaultValue(flags.PackageRepoName, flags.PackageRepoName),
				test.SetDefaultValue("Url", "Url"),
			},
			expectErr: false,
		},
		{
			name: "InvalidContext - Missing package version",
			flags: []components.Flag{
				test.SetDefaultValue(flags.PackageName, flags.PackageName),
				test.SetDefaultValue(flags.PackageRepoName, flags.PackageRepoName),
				test.SetDefaultValue("Url", "Url"),
			},
			expectErr: true,
		},
		{
			name: "InvalidContext - Missing package repository key",
			flags: []components.Flag{
				test.SetDefaultValue(flags.PackageName, flags.PackageName),
				test.SetDefaultValue(flags.PackageVersion, flags.PackageVersion),
				test.SetDefaultValue("Url", "Url"),
			},
			expectErr: true,
		},
		{
			name: "InvalidContext - Unsupported Basic Auth",
			flags: []components.Flag{
				test.SetDefaultValue(flags.ReleaseBundle, flags.ReleaseBundle),
				test.SetDefaultValue("Url", "Url"),
				test.SetDefaultValue("User", "testUser"),
				test.SetDefaultValue("password", "testPassword"),
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			context, err1 := components.ConvertContext(ctx, tt.flags...)
			if err1 != nil {
				return
			}

			execFunc = func(command commands.Command) error {
				return nil
			}
			// Replace execFunc with the mockExec function
			defer func() { execFunc = utils.Exec }() // Restore original execFunc after test

			err := verifyEvidence(context)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCreateEvidenceValidation_SigstoreBundle(t *testing.T) {
	app := cli.NewApp()
	app.Commands = []cli.Command{
		{
			Name: "create",
		},
	}
	ctx := cli.NewContext(app, &flag.FlagSet{}, nil)

	tests := []struct {
		name          string
		flags         []components.Flag
		expectError   bool
		errorContains string
	}{
		{
			name: "ValidContext_-_SigstoreBundle_Without_Predicate",
			flags: []components.Flag{
				test.SetDefaultValue(flags.SigstoreBundle, "/path/to/bundle.json"),
				test.SetDefaultValue(flags.SubjectRepoPath, "test-repo/test-artifact"),
			},
			expectError: false,
		},
		{
			name: "ValidContext_-_SigstoreBundle_Without_Any_Subject",
			flags: []components.Flag{
				test.SetDefaultValue(flags.SigstoreBundle, "/path/to/bundle.json"),
				// No subject fields provided - should still pass since subject is extracted from bundle
			},
			expectError: false,
		},
		{
			name: "InvalidContext_-_Missing_Predicate_Without_SigstoreBundle",
			flags: []components.Flag{
				test.SetDefaultValue(flags.SubjectRepoPath, "test-repo/test-artifact"),
				test.SetDefaultValue(flags.Key, "/path/to/key.pem"),
			},
			expectError:   true,
			errorContains: "'Predicate' is a mandatory field",
		},
		{
			name: "InvalidContext_-_Missing_PredicateType_Without_SigstoreBundle",
			flags: []components.Flag{
				test.SetDefaultValue(flags.SubjectRepoPath, "test-repo/test-artifact"),
				test.SetDefaultValue(flags.Predicate, "/path/to/Predicate.json"),
				test.SetDefaultValue(flags.Key, "/path/to/key.pem"),
			},
			expectError:   true,
			errorContains: "'Predicate-type' is a mandatory field",
		},
		{
			name: "InvalidContext_-_SigstoreBundle_With_Key",
			flags: []components.Flag{
				test.SetDefaultValue(flags.SigstoreBundle, "/path/to/bundle.json"),
				test.SetDefaultValue(flags.Key, "/path/to/key.pem"),
			},
			expectError:   true,
			errorContains: "The following parameters cannot be used with --sigstore-bundle: --key",
		},
		{
			name: "InvalidContext_-_SigstoreBundle_With_KeyAlias",
			flags: []components.Flag{
				test.SetDefaultValue(flags.SigstoreBundle, "/path/to/bundle.json"),
				test.SetDefaultValue(flags.KeyAlias, "my-key-alias"),
			},
			expectError:   true,
			errorContains: "The following parameters cannot be used with --sigstore-bundle: --key-alias",
		},
		{
			name: "InvalidContext_-_SigstoreBundle_With_Predicate",
			flags: []components.Flag{
				test.SetDefaultValue(flags.SigstoreBundle, "/path/to/bundle.json"),
				test.SetDefaultValue(flags.Predicate, "/path/to/Predicate.json"),
			},
			expectError:   true,
			errorContains: "The following parameters cannot be used with --sigstore-bundle: --predicate",
		},
		{
			name: "InvalidContext_-_SigstoreBundle_With_PredicateType",
			flags: []components.Flag{
				test.SetDefaultValue(flags.SigstoreBundle, "/path/to/bundle.json"),
				test.SetDefaultValue(flags.PredicateType, "test-type"),
			},
			expectError:   true,
			errorContains: "The following parameters cannot be used with --sigstore-bundle: --predicate-type",
		},
		{
			name: "InvalidContext_-_SigstoreBundle_With_Multiple_Conflicting_Params",
			flags: []components.Flag{
				test.SetDefaultValue(flags.SigstoreBundle, "/path/to/bundle.json"),
				test.SetDefaultValue(flags.Key, "/path/to/key.pem"),
				test.SetDefaultValue(flags.KeyAlias, "my-key-alias"),
				test.SetDefaultValue(flags.Predicate, "/path/to/Predicate.json"),
				test.SetDefaultValue(flags.PredicateType, "test-type"),
			},
			expectError:   true,
			errorContains: "The following parameters cannot be used with --sigstore-bundle: --key, --key-alias, --predicate, --predicate-type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			context, err := components.ConvertContext(ctx, tt.flags...)
			assert.NoError(t, err)

			err = validateCreateEvidenceCommonContext(context)
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

func TestGetAndValidateSubject_SigstoreBundle(t *testing.T) {
	app := cli.NewApp()
	app.Commands = []cli.Command{
		{
			Name: "create",
		},
	}
	ctx := cli.NewContext(app, &flag.FlagSet{}, nil)

	tests := []struct {
		name            string
		flags           []components.Flag
		expectError     bool
		expectedSubject []string
	}{
		{
			name: "SigstoreBundle_NoSubjectFields",
			flags: []components.Flag{
				test.SetDefaultValue(flags.SigstoreBundle, "/path/to/bundle.json"),
			},
			expectError:     false,
			expectedSubject: []string{flags.SubjectRepoPath},
		},
		{
			name: "SigstoreBundle_WithSubjectRepoPath",
			flags: []components.Flag{
				test.SetDefaultValue(flags.SigstoreBundle, "/path/to/bundle.json"),
				test.SetDefaultValue(flags.SubjectRepoPath, "test-repo/test-artifact"),
			},
			expectError:     false,
			expectedSubject: []string{flags.SubjectRepoPath},
		},
		{
			name:  "NoSigstoreBundle_NoSubject_ShouldFail",
			flags: []components.Flag{
				// No sigstore bundle and no subject fields
			},
			expectError: true,
		},
		{
			name: "NoSigstoreBundle_WithSubject_ShouldPass",
			flags: []components.Flag{
				test.SetDefaultValue(flags.SubjectRepoPath, "test-repo/test-artifact"),
			},
			expectError:     false,
			expectedSubject: []string{flags.SubjectRepoPath},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			context, err := components.ConvertContext(ctx, tt.flags...)
			assert.NoError(t, err)

			subjects, err := getAndValidateSubject(context)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedSubject, subjects)
			}
		})
	}
}

func TestValidateSigstoreBundleConflicts(t *testing.T) {
	app := cli.NewApp()
	app.Commands = []cli.Command{
		{
			Name: "create",
		},
	}
	set := flag.NewFlagSet("create", 0)
	ctx := cli.NewContext(app, set, nil)

	tests := []struct {
		name          string
		flags         []components.Flag
		expectError   bool
		errorContains string
	}{
		{
			name: "No_Conflicts",
			flags: []components.Flag{
				test.SetDefaultValue(flags.SigstoreBundle, "/path/to/bundle.json"),
				test.SetDefaultValue(flags.SubjectRepoPath, "test-repo/test-artifact"),
			},
			expectError: false,
		},
		{
			name: "Conflict_With_Key",
			flags: []components.Flag{
				test.SetDefaultValue(flags.SigstoreBundle, "/path/to/bundle.json"),
				test.SetDefaultValue(flags.Key, "/path/to/key"),
			},
			expectError:   true,
			errorContains: "--key",
		},
		{
			name: "Conflict_With_Multiple_Params",
			flags: []components.Flag{
				test.SetDefaultValue(flags.SigstoreBundle, "/path/to/bundle.json"),
				test.SetDefaultValue(flags.Key, "/path/to/key"),
				test.SetDefaultValue(flags.KeyAlias, "my-key"),
				test.SetDefaultValue(flags.Predicate, "/path/to/Predicate"),
			},
			expectError:   true,
			errorContains: "--key, --key-alias, --predicate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			context, err := components.ConvertContext(ctx, tt.flags...)
			if err != nil {
				t.Fatal(err)
			}

			err = validateSigstoreBundleArgsConflicts(context)

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

func TestResolveAndNormalizeKey_TrimsWhitespace(t *testing.T) {
	tests := []struct {
		name           string
		envKeyValue    string
		flagKeyValue   string
		setFlag        bool
		expectedResult string
	}{
		{
			name:           "Env key with trailing newline",
			envKeyValue:    "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQE\n-----END PRIVATE KEY-----\n",
			setFlag:        false,
			expectedResult: "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQE\n-----END PRIVATE KEY-----",
		},
		{
			name:           "Env key with trailing spaces and newlines",
			envKeyValue:    "  -----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQE\n-----END PRIVATE KEY-----  \n\n",
			setFlag:        false,
			expectedResult: "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQE\n-----END PRIVATE KEY-----",
		},
		{
			name:           "Flag key with trailing newline",
			flagKeyValue:   "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQE\n-----END PRIVATE KEY-----\n",
			setFlag:        true,
			expectedResult: "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQE\n-----END PRIVATE KEY-----",
		},
		{
			name:           "Flag key with carriage return and newline",
			flagKeyValue:   "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQE\n-----END PRIVATE KEY-----\r\n",
			setFlag:        true,
			expectedResult: "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQE\n-----END PRIVATE KEY-----",
		},
		{
			name:           "Flag vs Env behavior difference - flag takes precedence and gets trimmed",
			envKeyValue:    "  env-key-value  \n",
			flagKeyValue:   "  flag-key-value  \n",
			setFlag:        true,
			expectedResult: "flag-key-value",
		},
		{
			name:           "Env fallback when no flag - env key gets trimmed",
			envKeyValue:    "  env-key-value  \n",
			setFlag:        false,
			expectedResult: "env-key-value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			if tt.envKeyValue != "" {
				assert.NoError(t, os.Setenv(coreUtils.SigningKey, tt.envKeyValue))
				defer func() {
					if err := os.Unsetenv(coreUtils.SigningKey); err != nil {
						t.Errorf("failed to unset env %q: %v", coreUtils.SigningKey, err)
					}
				}()
			}

			// Create context
			app := cli.NewApp()
			set := flag.NewFlagSet("test", 0)
			cliCtx := cli.NewContext(app, set, nil)

			var componentFlag []components.Flag
			if tt.setFlag {
				componentFlag = append(componentFlag, test.SetDefaultValue(flags.Key, tt.flagKeyValue))
			}
			componentFlag = append(componentFlag, test.SetDefaultValue(flags.Predicate, "test"))

			ctx, err := components.ConvertContext(cliCtx, componentFlag...)
			assert.NoError(t, err)

			// Execute
			err = resolveAndNormalizeKey(ctx, flags.Key)
			assert.NoError(t, err)

			// Verify
			actualValue := ctx.GetStringFlagValue(flags.Key)

			// Debug Output to help understand the difference in trimming behavior
			if tt.setFlag {
				t.Logf("Flag input: %q (len=%d)", tt.flagKeyValue, len(tt.flagKeyValue))
			} else {
				t.Logf("Env input: %q (len=%d)", tt.envKeyValue, len(tt.envKeyValue))
			}
			t.Logf("Expected: %q (len=%d)", tt.expectedResult, len(tt.expectedResult))
			t.Logf("Actual:   %q (len=%d)", actualValue, len(actualValue))

			assert.Equal(t, tt.expectedResult, actualValue)
		})
	}
}

func TestValidateSonarQubeRequirements(t *testing.T) {
	// Save original environment variables
	originalSonarToken := os.Getenv("SONAR_TOKEN")
	originalSonarQubeToken := os.Getenv("SONARQUBE_TOKEN")
	defer func() {
		err := os.Setenv("SONAR_TOKEN", originalSonarToken)
		if err != nil {
			assert.FailNow(t, err.Error())
		}
		err = os.Setenv("SONARQUBE_TOKEN", originalSonarQubeToken)
		if err != nil {
			assert.FailNow(t, err.Error())
		}
	}()

	tests := []struct {
		name           string
		sonarToken     string
		sonarQubeToken string
		expectError    bool
		errorContains  string
	}{
		{
			name:           "Valid_With_SONAR_TOKEN",
			sonarToken:     "test-token",
			sonarQubeToken: "",
			expectError:    false,
		},
		{
			name:           "Valid_With_SONARQUBE_TOKEN",
			sonarToken:     "",
			sonarQubeToken: "test-token",
			expectError:    false,
		},
		{
			name:           "Valid_With_Both_Tokens",
			sonarToken:     "test-token-1",
			sonarQubeToken: "test-token-2",
			expectError:    false,
		},
		{
			name:           "Invalid_No_Token",
			sonarToken:     "",
			sonarQubeToken: "",
			expectError:    true,
			errorContains:  "SonarQube token is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables for test
			err := os.Setenv("SONAR_TOKEN", tt.sonarToken)
			if err != nil {
				assert.FailNow(t, err.Error())
			}
			err = os.Setenv("SONARQUBE_TOKEN", tt.sonarQubeToken)
			if err != nil {
				assert.FailNow(t, err.Error())
			}

			err = validateSonarQubeRequirements()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else if err != nil {
				// Note: This test will fail if report-task.txt doesn't exist
				// In a real test environment, you might want to mock the file system
				// or create a temporary file for testing.
				// If an error occurs, it's expected to be about the missing report-task.txt file.
				assert.Contains(t, err.Error(), "report-task.txt file not found")
			}
		})
	}
}
