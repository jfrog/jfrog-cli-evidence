package command

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/command/application"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/command/artifacts"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/command/build"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/command/flags"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/command/github"
	_interface "github.com/jfrog/jfrog-cli-evidence/evidence/cli/command/interface"
	_package "github.com/jfrog/jfrog-cli-evidence/evidence/cli/command/package"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/command/releasebundle"
	commandUtils "github.com/jfrog/jfrog-cli-evidence/evidence/cli/command/utils"
	evdConfig "github.com/jfrog/jfrog-cli-evidence/evidence/config"
	"github.com/jfrog/jfrog-cli-evidence/evidence/model"

	commonCliUtils "github.com/jfrog/jfrog-cli-core/v2/common/cliutils"
	"github.com/jfrog/jfrog-cli-core/v2/common/commands"
	"github.com/jfrog/jfrog-cli-core/v2/common/format"
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

// responseCollector is implemented by evidence create commands that accumulate
// CreateResponse values during Run(). The interface is checked via type assertion
// after command execution so that printCreateEvidenceResponse can format output.
type responseCollector interface {
	CollectedResponses() []*model.CreateResponse
}

func GetCommands() []components.Command {
	return []components.Command{
		{
			Name:             "create-evidence",
			Aliases:          []string{"create"},
			Flags:            flags.GetCommandFlags(flags.CreateEvidence),
			Description:      create.GetDescription(),
			AIDescription:    create.GetAIDescription(),
			Arguments:        create.GetArguments(),
			Action:           createEvidence,
			SupportedFormats: []format.OutputFormat{format.Json, format.Table},
		},
		{
			Name:          "get-evidence",
			Aliases:       []string{"get"},
			Flags:         flags.GetCommandFlags(flags.GetEvidence),
			Description:   get.GetDescription(),
			AIDescription: get.GetAIDescription(),
			Arguments:     get.GetArguments(),
			Action:        getEvidence,
		},
		{
			Name:          "verify-evidence",
			Aliases:       []string{"verify"},
			Flags:         flags.GetCommandFlags(flags.VerifyEvidence),
			Description:   verify.GetDescription(),
			AIDescription: verify.GetAIDescription(),
			Arguments:     verify.GetArguments(),
			Action:        verifyEvidence,
		},
		{
			Name:          "generate-key-pair",
			Aliases:       []string{"gen-keys"},
			Flags:         flags.GetCommandFlags(flags.GenerateKeyPair),
			Description:   generate.GetDescription(),
			AIDescription: generate.GetAIDescription(),
			Arguments:     generate.GetArguments(),
			Action:        generateKeyPair,
		},
	}
}

var (
	execFunc              = commands.Exec
	ErrUnsupportedSubject = errors.New("unsupported subject")
)

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

	outputFormat, err := ctx.GetOutputFormat()
	if err != nil {
		return err
	}

	// Wrap execFunc to capture collected responses after each command runs.
	var collectedResponses []*model.CreateResponse
	collectingExec := func(cmd commands.Command) error {
		if runErr := execFunc(cmd); runErr != nil {
			return runErr
		}
		if rc, ok := cmd.(responseCollector); ok {
			collectedResponses = append(collectedResponses, rc.CollectedResponses()...)
		}
		return nil
	}

	var createErr error
	if slices.Contains(evidenceType, flags.TypeFlag) || (slices.Contains(evidenceType, flags.BuildName) && slices.Contains(evidenceType, flags.TypeFlag)) {
		createErr = github.NewEvidenceGitHubCommand(ctx, collectingExec).CreateEvidence(ctx, serverDetails)
	} else {
		evidenceCommands := map[string]func(*components.Context, commandUtils.ExecCommandFunc) _interface.EvidenceCommands{
			flags.SubjectRepoPath: artifacts.NewEvidenceCustomCommand,
			flags.ReleaseBundle:   releasebundle.NewEvidenceReleaseBundleCommand,
			flags.BuildName:       build.NewEvidenceBuildCommand,
			flags.PackageName:     _package.NewEvidencePackageCommand,
			flags.ApplicationKey:  application.NewEvidenceApplicationCommand,
		}

		if commandFunc, exists := evidenceCommands[evidenceType[0]]; exists {
			createErr = commandFunc(ctx, collectingExec).CreateEvidence(ctx, serverDetails)
		} else {
			return ErrUnsupportedSubject
		}
	}

	if createErr != nil {
		return createErr
	}

	if outputFormat != "" {
		return printCreateEvidenceResponse(os.Stdout, outputFormat, collectedResponses)
	}
	return nil
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

	evidenceCommands := map[string]func(*components.Context, commandUtils.ExecCommandFunc) _interface.EvidenceCommands{
		flags.SubjectRepoPath: artifacts.NewEvidenceCustomCommand,
		flags.ReleaseBundle:   releasebundle.NewEvidenceReleaseBundleCommand,
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
	evidenceCommands := map[string]func(*components.Context, commandUtils.ExecCommandFunc) _interface.EvidenceCommands{
		flags.SubjectRepoPath: artifacts.NewEvidenceCustomCommand,
		flags.ReleaseBundle:   releasebundle.NewEvidenceReleaseBundleCommand,
		flags.BuildName:       build.NewEvidenceBuildCommand,
		flags.PackageName:     _package.NewEvidencePackageCommand,
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

	if commandUtils.AssertValueProvided(ctx, flags.SigstoreBundle) == nil {
		if err := validateSigstoreBundleArgsConflicts(ctx); err != nil {
			return err
		}
		if ctx.GetStringFlagValue(flags.AttachLocal) != "" || ctx.GetStringFlagValue(flags.AttachArtifactoryPath) != "" {
			return errorutils.CheckErrorf("attachments are supported only for in-toto flow and cannot be used with --%s", flags.SigstoreBundle)
		}
		return nil
	}

	if commandUtils.AssertValueProvided(ctx, flags.Integration) == nil {
		if err := evidenceUtils.ValidateIntegration(ctx.GetStringFlagValue(flags.Integration)); err != nil {
			return err
		}
	}

	if commandUtils.AssertValueProvided(ctx, flags.Predicate) != nil && !ctx.IsFlagSet(flags.TypeFlag) {
		if !evidenceUtils.IsSonarIntegration(ctx.GetStringFlagValue(flags.Integration)) {
			return errorutils.CheckErrorf("'Predicate' is a mandatory field for creating evidence: --%s", flags.Predicate)
		}
	}

	if commandUtils.AssertValueProvided(ctx, flags.PredicateType) != nil && !ctx.IsFlagSet(flags.TypeFlag) {
		if !evidenceUtils.IsSonarIntegration(ctx.GetStringFlagValue(flags.Integration)) {
			return errorutils.CheckErrorf("'Predicate-type' is a mandatory field for creating evidence: --%s", flags.PredicateType)
		}
	}

	// Validate SonarQube requirements when sonar integration is set
	if evidenceUtils.IsSonarIntegration(ctx.GetStringFlagValue(flags.Integration)) {
		if err := validateSonarQubeRequirements(); err != nil {
			return err
		}
		// Conflicting flags with sonar evidence type
		if ctx.IsFlagSet(flags.Predicate) && ctx.GetStringFlagValue(flags.Predicate) != "" {
			return errorutils.CheckErrorf("--%s cannot be used together with --%s %s", flags.Predicate, flags.Integration, evidenceUtils.SonarIntegration)
		}
		if ctx.IsFlagSet(flags.PredicateType) && ctx.GetStringFlagValue(flags.PredicateType) != "" {
			return errorutils.CheckErrorf("--%s cannot be used together with --%s %s", flags.PredicateType, flags.Integration, evidenceUtils.SonarIntegration)
		}
	}

	if err := resolveAndNormalizeKey(ctx, flags.Key); err != nil {
		return err
	}

	if !ctx.IsFlagSet(flags.KeyAlias) {
		setKeyAliasIfProvided(ctx, flags.KeyAlias)
	}
	if err := validateAttachmentFlags(ctx); err != nil {
		return err
	}
	return nil
}

func validateAttachmentFlags(ctx *components.Context) error {
	attachLocal := ctx.GetStringFlagValue(flags.AttachLocal)
	attachArtifactoryPath := ctx.GetStringFlagValue(flags.AttachArtifactoryPath)
	attachArtifactoryTempPath := ctx.GetStringFlagValue(flags.AttachArtifactoryTempPath)

	if attachLocal != "" && attachArtifactoryPath != "" {
		return errorutils.CheckErrorf("exactly one of --%s or --%s can be used", flags.AttachLocal, flags.AttachArtifactoryPath)
	}

	if attachArtifactoryTempPath != "" && attachLocal == "" {
		return errorutils.CheckErrorf("--%s can be used only with --%s", flags.AttachArtifactoryTempPath, flags.AttachLocal)
	}

	if attachLocal != "" && attachArtifactoryTempPath == "" {
		defaultTarget := evdConfig.ResolveAttachmentArtifactoryTempPath()
		if defaultTarget == "" {
			return errorutils.CheckErrorf("--%s is required with --%s (or set %s / %s)", flags.AttachArtifactoryTempPath, flags.AttachLocal, evdConfig.EnvAttachmentArtifactoryTempPath, evdConfig.KeyAttachmentArtifactoryTempPath)
		}
		ctx.AddStringFlag(flags.AttachArtifactoryTempPath, defaultTarget)
	}

	if attachLocal != "" && ctx.IsFlagSet(flags.AttachArtifactoryTempPath) {
		if err := evdConfig.PersistAttachmentArtifactoryTempPath(ctx.GetStringFlagValue(flags.AttachArtifactoryTempPath)); err != nil {
			log.Warn("error persisting attachment artifactory temp path:", err)
			return nil
		}
	}
	return nil
}

func validateSigstoreBundleArgsConflicts(ctx *components.Context) error {
	var conflictingParams []string

	if ctx.IsFlagSet(flags.Key) && ctx.GetStringFlagValue(flags.Key) != "" {
		conflictingParams = append(conflictingParams, "--"+flags.Key)
	}
	if ctx.IsFlagSet(flags.KeyAlias) && ctx.GetStringFlagValue(flags.KeyAlias) != "" {
		conflictingParams = append(conflictingParams, "--"+flags.KeyAlias)
	}
	if ctx.IsFlagSet(flags.Predicate) && ctx.GetStringFlagValue(flags.Predicate) != "" {
		conflictingParams = append(conflictingParams, "--"+flags.Predicate)
	}
	if ctx.IsFlagSet(flags.PredicateType) && ctx.GetStringFlagValue(flags.PredicateType) != "" {
		conflictingParams = append(conflictingParams, "--"+flags.PredicateType)
	}
	if ctx.IsFlagSet(flags.AttachLocal) && ctx.GetStringFlagValue(flags.AttachLocal) != "" {
		conflictingParams = append(conflictingParams, "--"+flags.AttachLocal)
	}
	if ctx.IsFlagSet(flags.AttachArtifactoryTempPath) && ctx.GetStringFlagValue(flags.AttachArtifactoryTempPath) != "" {
		conflictingParams = append(conflictingParams, "--"+flags.AttachArtifactoryTempPath)
	}
	if ctx.IsFlagSet(flags.AttachArtifactoryPath) && ctx.GetStringFlagValue(flags.AttachArtifactoryPath) != "" {
		conflictingParams = append(conflictingParams, "--"+flags.AttachArtifactoryPath)
	}

	if len(conflictingParams) > 0 {
		return errorutils.CheckErrorf("The following parameters cannot be used with --%s: %s. These values are extracted from the bundle itself:", flags.SigstoreBundle, strings.Join(conflictingParams, ", "))
	}

	return nil
}

func resolveAndNormalizeKey(ctx *components.Context, key string) error {
	if commandUtils.AssertValueProvided(ctx, key) == nil {
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
	for _, key := range commandUtils.SubjectTypes {
		if ctx.GetStringFlagValue(key) != "" {
			foundSubjects = append(foundSubjects, key)
		}
	}

	if len(foundSubjects) == 0 {
		if commandUtils.AssertValueProvided(ctx, flags.SigstoreBundle) == nil {
			return []string{flags.SubjectRepoPath}, nil // Return subjectRepoPath as the type for routing
		}
		// If we have no subject - we will try to create EVD on build
		if !attemptSetBuildNameAndNumber(ctx) {
			return nil, errorutils.CheckErrorf("subject must be one of the fields: [%s]", strings.Join(commandUtils.SubjectTypes, ", "))
		}
		foundSubjects = append(foundSubjects, flags.BuildName)
	}

	if err := validateFoundSubjects(ctx, foundSubjects); err != nil {
		return nil, err
	}

	return foundSubjects, nil
}

func attemptSetBuildNameAndNumber(ctx *components.Context) bool {
	buildNameAdded := setBuildValue(ctx, flags.BuildName, coreUtils.BuildName)
	buildNumberAdded := setBuildValue(ctx, flags.BuildNumber, coreUtils.BuildNumber)

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
	providedKeys := ctx.GetStringsArrFlagValue(flags.PublicKeys)
	if len(providedKeys) > 0 {
		joinedKeys := strings.Join(append(providedKeys, signingKeyValue), ";")
		ctx.SetStringFlagValue(flags.PublicKeys, joinedKeys)
	} else {
		ctx.AddStringFlag(flags.PublicKeys, signingKeyValue)
	}
	return nil
}

func validateFoundSubjects(ctx *components.Context, foundSubjects []string) error {
	if slices.Contains(foundSubjects, flags.TypeFlag) && slices.Contains(foundSubjects, flags.BuildName) {
		return nil
	}

	if slices.Contains(foundSubjects, flags.TypeFlag) && attemptSetBuildNameAndNumber(ctx) {
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
		return errorutils.CheckErrorf("SonarQube token is required when using --%s %s. Please set SONAR_TOKEN or SONARQUBE_TOKEN environment variable", flags.Integration, evidenceUtils.SonarIntegration)
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

	// Get upload flag, key alias, key file path, and key file name
	uploadKey := ctx.GetBoolFlagValue(flags.UploadPublicKey)
	alias := ctx.GetStringFlagValue(flags.KeyAlias)
	keyFilePath := ctx.GetStringFlagValue(flags.KeyFilePath)
	fileName := ctx.GetStringFlagValue(flags.KeyFileName)

	var serverDetails *config.ServerDetails
	var err error

	// Get server details for upload (default is true now)
	if uploadKey {
		serverDetails, err = evidenceDetailsByFlags(ctx)
		if err != nil {
			return err
		}
	}

	cmd := generateCmd.NewGenerateKeyPairCommand(serverDetails, uploadKey, alias, keyFilePath, fileName)
	return cmd.Run()
}

// tableFieldOrder defines the display order for create-evidence table output.
var tableFieldOrder = []struct {
	key   string
	value func(*model.CreateResponse) string
}{
	{"repository", func(r *model.CreateResponse) string { return r.Repository }},
	{"path", func(r *model.CreateResponse) string { return r.Path }},
	{"name", func(r *model.CreateResponse) string { return r.Name }},
	{"uri", func(r *model.CreateResponse) string { return r.Uri }},
	{"sha256", func(r *model.CreateResponse) string { return r.Sha256 }},
	{"predicate_type", func(r *model.CreateResponse) string { return r.PredicateType }},
	{"predicate_category", func(r *model.CreateResponse) string { return r.PredicateCategory }},
	{"predicate_slug", func(r *model.CreateResponse) string { return r.PredicateSlug }},
	{"created_at", func(r *model.CreateResponse) string { return r.CreatedAt }},
	{"created_by", func(r *model.CreateResponse) string { return r.CreatedBy }},
	{"verified", func(r *model.CreateResponse) string {
		if r.Verified {
			return "true"
		}
		return "false"
	}},
	{"provider_id", func(r *model.CreateResponse) string { return r.ProviderId }},
}

// printCreateEvidenceResponse writes formatted output for the given responses.
func printCreateEvidenceResponse(w io.Writer, outputFormat format.OutputFormat, responses []*model.CreateResponse) error {
	switch outputFormat {
	case format.Json:
		return printCreateEvidenceJSON(w, responses)
	case format.Table:
		return printCreateEvidenceTable(w, responses)
	default:
		return fmt.Errorf("unsupported format %q: accepted values are json, table", outputFormat)
	}
}

func printCreateEvidenceJSON(w io.Writer, responses []*model.CreateResponse) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if len(responses) == 1 {
		return enc.Encode(responses[0])
	}
	return enc.Encode(responses)
}

func printCreateEvidenceTable(w io.Writer, responses []*model.CreateResponse) error {
	for i, r := range responses {
		if i > 0 {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
		t := table.NewWriter()
		t.SetOutputMirror(w)
		t.SetStyle(table.StyleLight)
		t.Style().Options.SeparateRows = false
		t.AppendHeader(table.Row{"FIELD", "VALUE"})
		t.SetColumnConfigs([]table.ColumnConfig{
			{Number: 1, Align: text.AlignLeft, AlignHeader: text.AlignLeft},
			{Number: 2, Align: text.AlignLeft, AlignHeader: text.AlignLeft},
		})
		for _, field := range tableFieldOrder {
			val := field.value(r)
			if val != "" {
				t.AppendRow(table.Row{field.key, val})
			}
		}
		for _, row := range attachmentTableRows(r.Attachments) {
			t.AppendRow(row)
		}
		t.Render()
	}
	return nil
}

// attachmentTableRows builds FIELD/VALUE rows for an attachments slice.
// Single attachment uses "attachment.<field>", multiple use "attachments[i].<field>".
// download_path is intentionally omitted from table output (too long to render
// without distorting the layout); it is still emitted in JSON output.
func attachmentTableRows(attachments []model.CreateResponseAttachment) []table.Row {
	if len(attachments) == 0 {
		return nil
	}
	fields := []struct {
		key   string
		value func(model.CreateResponseAttachment) string
	}{
		{"name", func(a model.CreateResponseAttachment) string { return a.Name }},
		{"sha256", func(a model.CreateResponseAttachment) string { return a.Sha256 }},
		{"type", func(a model.CreateResponseAttachment) string { return a.Type }},
	}
	var rows []table.Row
	for i, att := range attachments {
		prefix := "attachment"
		if len(attachments) > 1 {
			prefix = fmt.Sprintf("attachments[%d]", i)
		}
		for _, f := range fields {
			val := f.value(att)
			if val == "" {
				continue
			}
			rows = append(rows, table.Row{fmt.Sprintf("%s.%s", prefix, f.key), val})
		}
	}
	return rows
}
