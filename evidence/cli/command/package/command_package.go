package _package

import (
	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"
	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/command"
	"github.com/jfrog/jfrog-cli-evidence/evidence/create"
	"github.com/jfrog/jfrog-cli-evidence/evidence/verify"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
)

type evidencePackageCommand struct {
	ctx     *components.Context
	execute command.ExecCommandFunc
}

func NewEvidencePackageCommand(ctx *components.Context, execute command.ExecCommandFunc) command.EvidenceCommands {
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
		epc.ctx.GetStringFlagValue(command.Predicate),
		epc.ctx.GetStringFlagValue(command.PredicateType),
		epc.ctx.GetStringFlagValue(command.Markdown),
		epc.ctx.GetStringFlagValue(command.Key),
		epc.ctx.GetStringFlagValue(command.KeyAlias),
		epc.ctx.GetStringFlagValue(command.PackageName),
		epc.ctx.GetStringFlagValue(command.PackageVersion),
		epc.ctx.GetStringFlagValue(command.PackageRepoName),
		epc.ctx.GetStringFlagValue(command.ProviderId),
		epc.ctx.GetStringFlagValue(command.Integration))
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
		epc.ctx.GetStringFlagValue(command.Format),
		epc.ctx.GetStringFlagValue(command.PackageName),
		epc.ctx.GetStringFlagValue(command.PackageVersion),
		epc.ctx.GetStringFlagValue(command.PackageRepoName),
		epc.ctx.GetStringsArrFlagValue(command.PublicKeys),
		epc.ctx.GetBoolFlagValue(command.UseArtifactoryKeys),
	)
	return epc.execute(verifyCmd)
}

func (epc *evidencePackageCommand) validateEvidencePackageContext(ctx *components.Context) error {
	if epc.ctx.GetStringFlagValue(command.SigstoreBundle) != "" {
		return errorutils.CheckErrorf("--%s is not supported for package evidence.", command.SigstoreBundle)
	}
	if command.AssertValueProvided(ctx, command.PackageVersion) != nil {
		return errorutils.CheckErrorf("--%s is a mandatory field for creating a Package evidence", command.PackageVersion)
	}
	if command.AssertValueProvided(ctx, command.PackageRepoName) != nil {
		return errorutils.CheckErrorf("--%s is a mandatory field for creating a Package evidence", command.PackageRepoName)
	}
	return nil
}
