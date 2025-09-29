package command

import (
	"errors"
	"fmt"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/command/application"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/command/artifacts"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/command/build"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/command/github"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/command/package"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/command/releasebundle"
	"os"
	"slices"
	"strings"

	commonCliUtils "github.com/jfrog/jfrog-cli-core/v2/common/cliutils"
	"github.com/jfrog/jfrog-cli-core/v2/common/commands"
	pluginsCommon "github.com/jfrog/jfrog-cli-core/v2/plugins/common"
	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"
	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	coreUtils "github.com/jfrog/jfrog-cli-core/v2/utils/coreutils"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/docs/create"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/docs/generate"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/docs/get"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/docs/verify"
	sonarhelper "github.com/jfrog/jfrog-cli-evidence/evidence/sonar"
	evidenceUtils "github.com/jfrog/jfrog-cli-evidence/evidence/utils"
	"github.com/jfrog/jfrog-client-go/utils"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"github.com/jfrog/jfrog-client-go/utils/log"

	generateCmd "github.com/jfrog/jfrog-cli-evidence/evidence/generate"
)

func GetCommands() []components.Command {
	return []components.Command{
		{
			Name:        "create-evidence",
			Aliases:     []string{"create"},
			Flags:       GetCommandFlags(CreateEvidence),
			Description: create.GetDescription(),
			Arguments:   create.GetArguments(),
			Action:      createEvidence,
		},
		{
			Name:        "get-evidence",
			Aliases:     []string{"get"},
			Flags:       GetCommandFlags(GetEvidence),
			Description: get.GetDescription(),
			Arguments:   get.GetArguments(),
			Action:      getEvidence,
		},
		{
			Name:        "verify-evidence",
			Aliases:     []string{"verify"},
			Flags:       GetCommandFlags(VerifyEvidence),
			Description: verify.GetDescription(),
			Arguments:   verify.GetArguments(),
			Action:      verifyEvidence,
		},
		{
			Name:        "generate-key-pair",
			Aliases:     []string{"gen-keys"},
			Flags:       GetCommandFlags(GenerateKeyPair),
			Description: generate.GetDescription(),
			Arguments:   generate.GetArguments(),
			Action:      generateKeyPair,
		},
	}
}

var execFunc = commands.Exec
var ErrUnsupportedSubject = errors.New("unsupported subject")

func createEvidence(ctx *components.Context) error {
	if err := validateCreateEvidenceCommonContext(ctx); err != nil {
		return err
	}
	evidenceType, err := getAndValidateSubject(ctx)
	if err != nil {
		return err
	}
	serverDetails, err := evidenceDetailsByFlags(ctx)
	if err != nil {
		return err
	}

	if slices.Contains(evidenceType, TypeFlag) || (slices.Contains(evidenceType, BuildName) && slices.Contains(evidenceType, TypeFlag)) {
		return github.NewEvidenceGitHubCommand(ctx, execFunc).CreateEvidence(ctx, serverDetails)
	}

	evidenceCommands := map[string]func(*components.Context, ExecCommandFunc) EvidenceCommands{
		SubjectRepoPath: artifacts.NewEvidenceCustomCommand,
		ReleaseBundle:   releasebundle.NewEvidenceReleaseBundleCommand,
		BuildName:       build.NewEvidenceBuildCommand,
		PackageName:     _package.NewEvidencePackageCommand,
		ApplicationKey:  application.NewEvidenceApplicationCommand,
	}

	if commandFunc, exists := evidenceCommands[evidenceType[0]]; exists {
		return commandFunc(ctx, execFunc).CreateEvidence(ctx, serverDetails)
	}

	return ErrUnsupportedSubject
}

func getEvidence(ctx *components.Context) error {
	if err := validateGetEvidenceCommonContext(ctx); err != nil {
		return err
	}

	evidenceType, err := getAndValidateSubject(ctx)
	if err != nil {
		return err
	}

	serverDetails, err := evidenceDetailsByFlags(ctx)
	if err != nil {
		return err
	}

	evidenceCommands := map[string]func(*components.Context, ExecCommandFunc) EvidenceCommands{
		SubjectRepoPath: artifacts.NewEvidenceCustomCommand,
		ReleaseBundle:   releasebundle.NewEvidenceReleaseBundleCommand,
	}

	if commandFunc, exists := evidenceCommands[evidenceType[0]]; exists {
		return commandFunc(ctx, execFunc).GetEvidence(ctx, serverDetails)
	}

	return ErrUnsupportedSubject
}

func validateGetEvidenceCommonContext(ctx *components.Context) error {
	if show, err := pluginsCommon.ShowCmdHelpIfNeeded(ctx, ctx.Arguments); show || err != nil {
		return err
	}

	if len(ctx.Arguments) > 1 {
		return pluginsCommon.WrongNumberOfArgumentsHandler(ctx)
	}

	return nil
}

func verifyEvidence(ctx *components.Context) error {
	// validate common context
	serverDetails, err := evidenceDetailsByFlags(ctx)
	if err != nil {
		return err
	}
	subjectType, err := getAndValidateSubject(ctx)
	if err != nil {
		return err
	}
	err = validateKeys(ctx)
	if err != nil {
		return err
	}
	evidenceCommands := map[string]func(*components.Context, ExecCommandFunc) EvidenceCommands{
		SubjectRepoPath: artifacts.NewEvidenceCustomCommand,
		ReleaseBundle:   releasebundle.NewEvidenceReleaseBundleCommand,
		BuildName:       build.NewEvidenceBuildCommand,
		PackageName:     _package.NewEvidencePackageCommand,
	}
	if commandFunc, exists := evidenceCommands[subjectType[0]]; exists {
		err = commandFunc(ctx, execFunc).VerifyEvidence(ctx, serverDetails)
		if err != nil {
			if err.Error() != "" {
				return fmt.Errorf("evidence verification failed: %w", err)
			}
			return err
		}
		return nil
	}
	return errors.New("unsupported subject")
}

func validateCreateEvidenceCommonContext(ctx *components.Context) error {
	if show, err := pluginsCommon.ShowCmdHelpIfNeeded(ctx, ctx.Arguments); show || err != nil {
		return err
	}

	if len(ctx.Arguments) > 1 {
		return pluginsCommon.WrongNumberOfArgumentsHandler(ctx)
	}

	if AssertValueProvided(ctx, SigstoreBundle) == nil {
		if err := validateSigstoreBundleArgsConflicts(ctx); err != nil {
			return err
		}
		return nil
	}

	if AssertValueProvided(ctx, Integration) == nil {
		if err := evidenceUtils.ValidateIntegration(ctx.GetStringFlagValue(Integration)); err != nil {
			return err
		}
	}

	if AssertValueProvided(ctx, Predicate) != nil && !ctx.IsFlagSet(TypeFlag) {
		if !evidenceUtils.IsSonarIntegration(ctx.GetStringFlagValue(Integration)) {
			return errorutils.CheckErrorf("'Predicate' is a mandatory field for creating evidence: --%s", Predicate)
		}
	}

	if AssertValueProvided(ctx, PredicateType) != nil && !ctx.IsFlagSet(TypeFlag) {
		if !evidenceUtils.IsSonarIntegration(ctx.GetStringFlagValue(Integration)) {
			return errorutils.CheckErrorf("'Predicate-type' is a mandatory field for creating evidence: --%s", PredicateType)
		}
	}

	// Validate SonarQube requirements when sonar integration is set
	if evidenceUtils.IsSonarIntegration(ctx.GetStringFlagValue(Integration)) {
		if err := validateSonarQubeRequirements(); err != nil {
			return err
		}
		// Conflicting flags with sonar evidence type
		if ctx.IsFlagSet(Predicate) && ctx.GetStringFlagValue(Predicate) != "" {
			return errorutils.CheckErrorf("--%s cannot be used together with --%s %s", Predicate, Integration, evidenceUtils.SonarIntegration)
		}
		if ctx.IsFlagSet(PredicateType) && ctx.GetStringFlagValue(PredicateType) != "" {
			return errorutils.CheckErrorf("--%s cannot be used together with --%s %s", PredicateType, Integration, evidenceUtils.SonarIntegration)
		}
	}

	if err := resolveAndNormalizeKey(ctx, Key); err != nil {
		return err
	}

	if !ctx.IsFlagSet(KeyAlias) {
		setKeyAliasIfProvided(ctx, KeyAlias)
	}
	return nil
}

func validateSigstoreBundleArgsConflicts(ctx *components.Context) error {
	var conflictingParams []string

	if ctx.IsFlagSet(Key) && ctx.GetStringFlagValue(Key) != "" {
		conflictingParams = append(conflictingParams, "--"+Key)
	}
	if ctx.IsFlagSet(KeyAlias) && ctx.GetStringFlagValue(KeyAlias) != "" {
		conflictingParams = append(conflictingParams, "--"+KeyAlias)
	}
	if ctx.IsFlagSet(Predicate) && ctx.GetStringFlagValue(Predicate) != "" {
		conflictingParams = append(conflictingParams, "--"+Predicate)
	}
	if ctx.IsFlagSet(PredicateType) && ctx.GetStringFlagValue(PredicateType) != "" {
		conflictingParams = append(conflictingParams, "--"+PredicateType)
	}

	if len(conflictingParams) > 0 {
		return errorutils.CheckErrorf("The following parameters cannot be used with --%s: %s. These values are extracted from the bundle itself:", SigstoreBundle, strings.Join(conflictingParams, ", "))
	}

	return nil
}

func resolveAndNormalizeKey(ctx *components.Context, key string) error {
	if AssertValueProvided(ctx, key) == nil {
		// Trim whitespace and newlines from the flag value
		keyValue := ctx.GetStringFlagValue(key)
		log.Debug(fmt.Sprintf("Flag '%s' original value: %q (length: %d)", key, keyValue, len(keyValue)))

		trimmedKeyValue := strings.TrimSpace(keyValue)
		log.Debug(fmt.Sprintf("Flag '%s' trimmed value: %q (length: %d)", key, trimmedKeyValue, len(trimmedKeyValue)))

		// Always update the flag value with the trimmed version
		ctx.AddStringFlag(key, trimmedKeyValue)
		return nil
	}

	signingKeyValue, _ := evidenceUtils.GetEnvVariable(coreUtils.SigningKey)
	if signingKeyValue == "" {
		return errorutils.CheckErrorf("JFROG_CLI_SIGNING_KEY env variable or --%s flag must be provided when creating evidence", key)
	}

	log.Debug(fmt.Sprintf("Environment variable '%s' original value: %q (length: %d)", coreUtils.SigningKey, signingKeyValue, len(signingKeyValue)))

	// Trim whitespace and newlines from the environment variable
	signingKeyValue = strings.TrimSpace(signingKeyValue)
	log.Debug(fmt.Sprintf("Environment variable '%s' trimmed value: %q (length: %d)", coreUtils.SigningKey, signingKeyValue, len(signingKeyValue)))

	ctx.AddStringFlag(key, signingKeyValue)
	return nil
}

func setKeyAliasIfProvided(ctx *components.Context, keyAlias string) {
	evdKeyAliasValue, _ := evidenceUtils.GetEnvVariable(coreUtils.KeyAlias)
	if evdKeyAliasValue != "" {
		ctx.AddStringFlag(keyAlias, evdKeyAliasValue)
	}
}

func getAndValidateSubject(ctx *components.Context) ([]string, error) {
	var foundSubjects []string
	for _, key := range subjectTypes {
		if ctx.GetStringFlagValue(key) != "" {
			foundSubjects = append(foundSubjects, key)
		}
	}

	if len(foundSubjects) == 0 {
		if AssertValueProvided(ctx, SigstoreBundle) == nil {
			return []string{SubjectRepoPath}, nil // Return subjectRepoPath as the type for routing
		}
		// If we have no subject - we will try to create EVD on build
		if !attemptSetBuildNameAndNumber(ctx) {
			return nil, errorutils.CheckErrorf("subject must be one of the fields: [%s]", strings.Join(subjectTypes, ", "))
		}
		foundSubjects = append(foundSubjects, BuildName)
	}

	if err := validateFoundSubjects(ctx, foundSubjects); err != nil {
		return nil, err
	}

	return foundSubjects, nil
}

func attemptSetBuildNameAndNumber(ctx *components.Context) bool {
	buildNameAdded := setBuildValue(ctx, BuildName, coreUtils.BuildName)
	buildNumberAdded := setBuildValue(ctx, BuildNumber, coreUtils.BuildNumber)

	return buildNameAdded && buildNumberAdded
}

func setBuildValue(ctx *components.Context, flag, envVar string) bool {
	// Check if the flag is provided. If so, we use it.
	if ctx.IsFlagSet(flag) {
		return true
	}
	// If the flag is not set, then check the environment variable
	if currentValue := os.Getenv(envVar); currentValue != "" {
		ctx.AddStringFlag(flag, currentValue)
		return true
	}
	return false
}

func validateKeys(ctx *components.Context) error {
	signingKeyValue, _ := evidenceUtils.GetEnvVariable(coreUtils.SigningKey)
	providedKeys := ctx.GetStringsArrFlagValue(PublicKeys)
	if len(providedKeys) > 0 {
		joinedKeys := strings.Join(append(providedKeys, signingKeyValue), ";")
		ctx.SetStringFlagValue(PublicKeys, joinedKeys)
	} else {
		ctx.AddStringFlag(PublicKeys, signingKeyValue)
	}
	return nil
}

func validateFoundSubjects(ctx *components.Context, foundSubjects []string) error {
	if slices.Contains(foundSubjects, TypeFlag) && slices.Contains(foundSubjects, BuildName) {
		return nil
	}

	if slices.Contains(foundSubjects, TypeFlag) && attemptSetBuildNameAndNumber(ctx) {
		return nil
	}

	if len(foundSubjects) > 1 {
		return errorutils.CheckErrorf("multiple subjects found: [%s]", strings.Join(foundSubjects, ", "))
	}
	return nil
}

func evidenceDetailsByFlags(ctx *components.Context) (*config.ServerDetails, error) {
	serverDetails, err := pluginsCommon.CreateServerDetailsWithConfigOffer(ctx, true, commonCliUtils.Platform)
	if err != nil {
		return nil, err
	}
	if serverDetails.Url == "" {
		return nil, errors.New("platform URL is mandatory for evidence command")
	}
	platformToEvidenceUrls(serverDetails)

	if serverDetails.GetUser() != "" && serverDetails.GetPassword() != "" {
		return nil, errors.New("evidence service does not support basic authentication")
	}

	return serverDetails, nil
}

func platformToEvidenceUrls(rtDetails *config.ServerDetails) {
	rtDetails.ArtifactoryUrl = utils.AddTrailingSlashIfNeeded(rtDetails.Url) + "artifactory/"
	rtDetails.EvidenceUrl = utils.AddTrailingSlashIfNeeded(rtDetails.Url) + "evidence/"
	rtDetails.MetadataUrl = utils.AddTrailingSlashIfNeeded(rtDetails.Url) + "metadata/"
	rtDetails.OnemodelUrl = utils.AddTrailingSlashIfNeeded(rtDetails.Url) + "onemodel/"
	rtDetails.LifecycleUrl = utils.AddTrailingSlashIfNeeded(rtDetails.Url) + "lifecycle/"
	rtDetails.ApptrustUrl = utils.AddTrailingSlashIfNeeded(rtDetails.Url) + "apptrust/"
}

func validateSonarQubeRequirements() error {
	// Check if SonarQube token is present
	if os.Getenv("SONAR_TOKEN") == "" && os.Getenv("SONARQUBE_TOKEN") == "" {
		return errorutils.CheckErrorf("SonarQube token is required when using --%s %s. Please set SONAR_TOKEN or SONARQUBE_TOKEN environment variable", Integration, evidenceUtils.SonarIntegration)
	}

	// Check if report-task.txt exists using the detector or config
	reportPath := sonarhelper.GetReportTaskPath()
	if reportPath == "" {
		return errorutils.CheckErrorf("SonarQube report-task.txt file not found. Please ensure SonarQube analysis has been completed or configure a custom path in evidence config")
	}
	log.Info("Found SonarQube task report:", reportPath)

	return nil
}

func generateKeyPair(ctx *components.Context) error {
	if show, err := pluginsCommon.ShowCmdHelpIfNeeded(ctx, ctx.Arguments); show || err != nil {
		return err
	}

	if len(ctx.Arguments) > 0 {
		return pluginsCommon.WrongNumberOfArgumentsHandler(ctx)
	}

	// Get upload flag, key alias, force flag, Output directory, and key file name
	uploadKey := ctx.GetBoolFlagValue(UploadPublicKey)
	alias := ctx.GetStringFlagValue(KeyAlias)
	forceOverwrite := ctx.GetBoolFlagValue(Force)
	outputDirectory := ctx.GetStringFlagValue(OutputDir)
	fileName := ctx.GetStringFlagValue(KeyFileName)

	var serverDetails *config.ServerDetails
	var err error

	// Only get server details if upload is requested
	if uploadKey {
		serverDetails, err = evidenceDetailsByFlags(ctx)
		if err != nil {
			return err
		}
	}

	cmd := generateCmd.NewGenerateKeyPairCommand(serverDetails, uploadKey, alias, forceOverwrite, outputDirectory, fileName)
	return cmd.Run()
}
