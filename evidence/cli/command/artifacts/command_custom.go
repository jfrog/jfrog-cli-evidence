package artifacts

import (
	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"
	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/command"
	"github.com/jfrog/jfrog-cli-evidence/evidence/create"
	"github.com/jfrog/jfrog-cli-evidence/evidence/get"
	"github.com/jfrog/jfrog-cli-evidence/evidence/verify"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
)

type evidenceCustomCommand struct {
	ctx     *components.Context
	execute command.ExecCommandFunc
}

func NewEvidenceCustomCommand(ctx *components.Context, execute command.ExecCommandFunc) command.EvidenceCommands {
	return &evidenceCustomCommand{
		ctx:     ctx,
		execute: execute,
	}
}

func (ecc *evidenceCustomCommand) CreateEvidence(_ *components.Context, serverDetails *config.ServerDetails) error {
	err := ecc.validateEvidenceFlagUsage()
	if err != nil {
		return err
	}

	// Single command handles both regular evidence creation and sigstore bundles
	createCmd := create.NewCreateEvidenceCustom(
		serverDetails,
		ecc.ctx.GetStringFlagValue(command.Predicate),
		ecc.ctx.GetStringFlagValue(command.PredicateType),
		ecc.ctx.GetStringFlagValue(command.Markdown),
		ecc.ctx.GetStringFlagValue(command.Key),
		ecc.ctx.GetStringFlagValue(command.KeyAlias),
		ecc.ctx.GetStringFlagValue(command.SubjectRepoPath),
		ecc.ctx.GetStringFlagValue(command.SubjectSha256),
		ecc.ctx.GetStringFlagValue(command.SigstoreBundle),
		ecc.ctx.GetStringFlagValue(command.ProviderId),
		ecc.ctx.GetStringFlagValue(command.Integration))
	return ecc.execute(createCmd)
}

func (ecc *evidenceCustomCommand) GetEvidence(_ *components.Context, serverDetails *config.ServerDetails) error {
	getCmd := get.NewGetEvidenceCustom(
		serverDetails,
		ecc.ctx.GetStringFlagValue(command.SubjectRepoPath),
		ecc.ctx.GetStringFlagValue(command.Format),
		ecc.ctx.GetStringFlagValue(command.Output),
		ecc.ctx.GetBoolFlagValue(command.IncludePredicate),
	)

	return ecc.execute(getCmd)
}

func (ecc *evidenceCustomCommand) VerifyEvidence(_ *components.Context, serverDetails *config.ServerDetails) error {
	verifyCmd := verify.NewVerifyEvidenceCustom(
		serverDetails,
		ecc.ctx.GetStringFlagValue(command.SubjectRepoPath),
		ecc.ctx.GetStringFlagValue(command.Format),
		ecc.ctx.GetStringsArrFlagValue(command.PublicKeys),
		ecc.ctx.GetBoolFlagValue(command.UseArtifactoryKeys),
	)
	return ecc.execute(verifyCmd)
}

func (ecc *evidenceCustomCommand) validateEvidenceFlagUsage() error {
	if ecc.ctx.GetStringFlagValue(command.SigstoreBundle) != "" && ecc.ctx.GetStringFlagValue(command.SubjectSha256) != "" {
		return errorutils.CheckErrorf("The parameter --%s cannot be used with --%s. The subject hash is extracted from the bundle itself.", command.SubjectSha256, command.SigstoreBundle)
	}
	return nil
}
