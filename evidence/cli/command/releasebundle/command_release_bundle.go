package releasebundle

import (
	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"
	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/command"
	"github.com/jfrog/jfrog-cli-evidence/evidence/create"
	"github.com/jfrog/jfrog-cli-evidence/evidence/get"
	"github.com/jfrog/jfrog-cli-evidence/evidence/utils"
	"github.com/jfrog/jfrog-cli-evidence/evidence/verify"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
)

type evidenceReleaseBundleCommand struct {
	ctx     *components.Context
	execute command.ExecCommandFunc
}

func NewEvidenceReleaseBundleCommand(ctx *components.Context, execute command.ExecCommandFunc) command.EvidenceCommands {
	return &evidenceReleaseBundleCommand{
		ctx:     ctx,
		execute: execute,
	}
}

func (erc *evidenceReleaseBundleCommand) CreateEvidence(ctx *components.Context, serverDetails *config.ServerDetails) error {
	err := erc.validateEvidenceReleaseBundleContext(ctx)
	if err != nil {
		return err
	}

	createCmd := create.NewCreateEvidenceReleaseBundle(
		serverDetails,
		erc.ctx.GetStringFlagValue(command.Predicate),
		erc.ctx.GetStringFlagValue(command.PredicateType),
		erc.ctx.GetStringFlagValue(command.Markdown),
		erc.ctx.GetStringFlagValue(command.Key),
		erc.ctx.GetStringFlagValue(command.KeyAlias),
		erc.ctx.GetStringFlagValue(command.Project),
		erc.ctx.GetStringFlagValue(command.ReleaseBundle),
		erc.ctx.GetStringFlagValue(command.ReleaseBundleVersion),
		erc.ctx.GetStringFlagValue(command.ProviderId),
		erc.ctx.GetStringFlagValue(command.Integration))
	return erc.execute(createCmd)
}

func (erc *evidenceReleaseBundleCommand) GetEvidence(ctx *components.Context, serverDetails *config.ServerDetails) error {
	err := erc.validateEvidenceReleaseBundleContext(ctx)
	if err != nil {
		return err
	}

	getCmd := get.NewGetEvidenceReleaseBundle(
		serverDetails,
		erc.ctx.GetStringFlagValue(command.ReleaseBundle),
		erc.ctx.GetStringFlagValue(command.ReleaseBundleVersion),
		erc.ctx.GetStringFlagValue(command.Project),
		erc.ctx.GetStringFlagValue(command.Format),
		erc.ctx.GetStringFlagValue(command.Output),
		erc.ctx.GetStringFlagValue(command.ArtifactsLimit),
		erc.ctx.GetBoolFlagValue(command.IncludePredicate),
	)
	return erc.execute(getCmd)
}

func (erc *evidenceReleaseBundleCommand) VerifyEvidence(ctx *components.Context, serverDetails *config.ServerDetails) error {
	err := erc.validateEvidenceReleaseBundleContext(ctx)
	if err != nil {
		return err
	}

	verifyCmd := verify.NewVerifyEvidenceReleaseBundle(
		serverDetails,
		erc.ctx.GetStringFlagValue(command.Format),
		erc.ctx.GetStringFlagValue(command.Project),
		erc.ctx.GetStringFlagValue(command.ReleaseBundle),
		erc.ctx.GetStringFlagValue(command.ReleaseBundleVersion),
		erc.ctx.GetStringsArrFlagValue(command.PublicKeys),
		erc.ctx.GetBoolFlagValue(command.UseArtifactoryKeys),
	)
	return erc.execute(verifyCmd)
}

func (erc *evidenceReleaseBundleCommand) validateEvidenceReleaseBundleContext(ctx *components.Context) error {
	// releaseBundleName is not validated since it is required for the evd context
	if erc.ctx.GetStringFlagValue(command.SigstoreBundle) != "" {
		return errorutils.CheckErrorf("--%s is not supported for release bundle evidence.", command.SigstoreBundle)
	}
	if command.AssertValueProvided(ctx, command.ReleaseBundleVersion) != nil {
		return errorutils.CheckErrorf("--%s is a mandatory field for creating a Release Bundle evidence", command.ReleaseBundleVersion)
	}
	if ctx.IsFlagSet(command.ArtifactsLimit) && !utils.IsFlagPositiveNumber(ctx.GetStringFlagValue(command.ArtifactsLimit)) {
		return errorutils.CheckErrorf("--%s must be a positive number", command.ArtifactsLimit)
	}
	return nil
}
