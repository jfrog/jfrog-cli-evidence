package artifacts

import (
	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"
	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/command/flags"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/command/interface"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/command/utils"
	"github.com/jfrog/jfrog-cli-evidence/evidence/create"
	"github.com/jfrog/jfrog-cli-evidence/evidence/get"
	"github.com/jfrog/jfrog-cli-evidence/evidence/verify"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
)

type evidenceCustomCommand struct {
	ctx     *components.Context
	execute utils.ExecCommandFunc
}

func NewEvidenceCustomCommand(ctx *components.Context, execute utils.ExecCommandFunc) _interface.EvidenceCommands {
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
		ecc.ctx.GetStringFlagValue(flags.Predicate),
		ecc.ctx.GetStringFlagValue(flags.PredicateType),
		ecc.ctx.GetStringFlagValue(flags.Markdown),
		ecc.ctx.GetStringFlagValue(flags.Key),
		ecc.ctx.GetStringFlagValue(flags.KeyAlias),
		ecc.ctx.GetStringFlagValue(flags.SubjectRepoPath),
		ecc.ctx.GetStringFlagValue(flags.SubjectSha256),
		ecc.ctx.GetStringFlagValue(flags.SigstoreBundle),
		ecc.ctx.GetStringFlagValue(flags.ProviderId),
		ecc.ctx.GetStringFlagValue(flags.Integration))
	return ecc.execute(createCmd)
}

func (ecc *evidenceCustomCommand) GetEvidence(_ *components.Context, serverDetails *config.ServerDetails) error {
	getCmd := get.NewGetEvidenceCustom(
		serverDetails,
		ecc.ctx.GetStringFlagValue(flags.SubjectRepoPath),
		ecc.ctx.GetStringFlagValue(flags.Format),
		ecc.ctx.GetStringFlagValue(flags.Output),
		ecc.ctx.GetBoolFlagValue(flags.IncludePredicate),
	)

	return ecc.execute(getCmd)
}

func (ecc *evidenceCustomCommand) VerifyEvidence(_ *components.Context, serverDetails *config.ServerDetails) error {
	verifyCmd := verify.NewVerifyEvidenceCustom(
		serverDetails,
		ecc.ctx.GetStringFlagValue(flags.SubjectRepoPath),
		ecc.ctx.GetStringFlagValue(flags.Format),
		ecc.ctx.GetStringsArrFlagValue(flags.PublicKeys),
		ecc.ctx.GetBoolFlagValue(flags.UseArtifactoryKeys),
	)
	return ecc.execute(verifyCmd)
}

func (ecc *evidenceCustomCommand) validateEvidenceFlagUsage() error {
	if ecc.ctx.GetStringFlagValue(flags.SigstoreBundle) != "" && ecc.ctx.GetStringFlagValue(flags.SubjectSha256) != "" {
		return errorutils.CheckErrorf("The parameter --%s cannot be used with --%s. The subject hash is extracted from the bundle itself.", flags.SubjectSha256, flags.SigstoreBundle)
	}
	return nil
}
