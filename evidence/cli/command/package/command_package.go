package _package

import (
	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"
	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/command/flags"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/command/interface"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/command/utils"
	"github.com/jfrog/jfrog-cli-evidence/evidence/create"
	"github.com/jfrog/jfrog-cli-evidence/evidence/verify"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
)

type evidencePackageCommand struct {
	ctx     *components.Context
	execute utils.ExecCommandFunc
}

func NewEvidencePackageCommand(ctx *components.Context, execute utils.ExecCommandFunc) _interface.EvidenceCommands {
	return &evidencePackageCommand{
		ctx:     ctx,
		execute: execute,
	}
}

func (epc *evidencePackageCommand) CreateEvidence(ctx *components.Context, serverDetails *config.ServerDetails) error {
	err := epc.validateEvidencePackageContext(ctx)
	if err != nil {
		return err
	}

	createCmd := create.NewCreateEvidencePackage(
		serverDetails,
		epc.ctx.GetStringFlagValue(flags.Predicate),
		epc.ctx.GetStringFlagValue(flags.PredicateType),
		epc.ctx.GetStringFlagValue(flags.Markdown),
		epc.ctx.GetStringFlagValue(flags.Key),
		epc.ctx.GetStringFlagValue(flags.KeyAlias),
		epc.ctx.GetStringFlagValue(flags.PackageName),
		epc.ctx.GetStringFlagValue(flags.PackageVersion),
		epc.ctx.GetStringFlagValue(flags.PackageRepoName),
		epc.ctx.GetStringFlagValue(flags.ProviderId),
		epc.ctx.GetStringFlagValue(flags.Integration))
	return epc.execute(createCmd)
}

func (epc *evidencePackageCommand) GetEvidence(_ *components.Context, _ *config.ServerDetails) error {
	return errorutils.CheckErrorf("Get evidence is not supported with packages")
}

func (epc *evidencePackageCommand) VerifyEvidence(ctx *components.Context, serverDetails *config.ServerDetails) error {
	err := epc.validateEvidencePackageContext(ctx)
	if err != nil {
		return err
	}

	verifyCmd := verify.NewVerifyEvidencePackage(
		serverDetails,
		epc.ctx.GetStringFlagValue(flags.Format),
		epc.ctx.GetStringFlagValue(flags.PackageName),
		epc.ctx.GetStringFlagValue(flags.PackageVersion),
		epc.ctx.GetStringFlagValue(flags.PackageRepoName),
		epc.ctx.GetStringsArrFlagValue(flags.PublicKeys),
		epc.ctx.GetBoolFlagValue(flags.UseArtifactoryKeys),
	)
	return epc.execute(verifyCmd)
}

func (epc *evidencePackageCommand) validateEvidencePackageContext(ctx *components.Context) error {
	if epc.ctx.GetStringFlagValue(flags.SigstoreBundle) != "" {
		return errorutils.CheckErrorf("--%s is not supported for package evidence.", flags.SigstoreBundle)
	}
	if utils.AssertValueProvided(ctx, flags.PackageVersion) != nil {
		return errorutils.CheckErrorf("--%s is a mandatory field for creating a Package evidence", flags.PackageVersion)
	}
	if utils.AssertValueProvided(ctx, flags.PackageRepoName) != nil {
		return errorutils.CheckErrorf("--%s is a mandatory field for creating a Package evidence", flags.PackageRepoName)
	}
	return nil
}
