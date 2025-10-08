package releasebundle

import (
	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"
	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/command/flags"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/command/interface"
	utils2 "github.com/jfrog/jfrog-cli-evidence/evidence/cli/command/utils"
	"github.com/jfrog/jfrog-cli-evidence/evidence/create"
	"github.com/jfrog/jfrog-cli-evidence/evidence/get"
	"github.com/jfrog/jfrog-cli-evidence/evidence/utils"
	"github.com/jfrog/jfrog-cli-evidence/evidence/verify"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
)

type evidenceReleaseBundleCommand struct {
	ctx     *components.Context
	execute utils2.ExecCommandFunc
}

func NewEvidenceReleaseBundleCommand(ctx *components.Context, execute utils2.ExecCommandFunc) _interface.EvidenceCommands {
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
		erc.ctx.GetStringFlagValue(flags.Predicate),
		erc.ctx.GetStringFlagValue(flags.PredicateType),
		erc.ctx.GetStringFlagValue(flags.Markdown),
		erc.ctx.GetStringFlagValue(flags.Key),
		erc.ctx.GetStringFlagValue(flags.KeyAlias),
		erc.ctx.GetStringFlagValue(flags.Project),
		erc.ctx.GetStringFlagValue(flags.ReleaseBundle),
		erc.ctx.GetStringFlagValue(flags.ReleaseBundleVersion),
		erc.ctx.GetStringFlagValue(flags.ProviderId),
		erc.ctx.GetStringFlagValue(flags.Integration))
	return erc.execute(createCmd)
}

func (erc *evidenceReleaseBundleCommand) GetEvidence(ctx *components.Context, serverDetails *config.ServerDetails) error {
	err := erc.validateEvidenceReleaseBundleContext(ctx)
	if err != nil {
		return err
	}

	getCmd := get.NewGetEvidenceReleaseBundle(
		serverDetails,
		erc.ctx.GetStringFlagValue(flags.ReleaseBundle),
		erc.ctx.GetStringFlagValue(flags.ReleaseBundleVersion),
		erc.ctx.GetStringFlagValue(flags.Project),
		erc.ctx.GetStringFlagValue(flags.Format),
		erc.ctx.GetStringFlagValue(flags.Output),
		erc.ctx.GetStringFlagValue(flags.ArtifactsLimit),
		erc.ctx.GetBoolFlagValue(flags.IncludePredicate),
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
		erc.ctx.GetStringFlagValue(flags.Format),
		erc.ctx.GetStringFlagValue(flags.Project),
		erc.ctx.GetStringFlagValue(flags.ReleaseBundle),
		erc.ctx.GetStringFlagValue(flags.ReleaseBundleVersion),
		erc.ctx.GetStringsArrFlagValue(flags.PublicKeys),
		erc.ctx.GetBoolFlagValue(flags.UseArtifactoryKeys),
	)
	return erc.execute(verifyCmd)
}

func (erc *evidenceReleaseBundleCommand) validateEvidenceReleaseBundleContext(ctx *components.Context) error {
	// releaseBundleName is not validated since it is required for the evd context
	if erc.ctx.GetStringFlagValue(flags.SigstoreBundle) != "" {
		return errorutils.CheckErrorf("--%s is not supported for release bundle evidence.", flags.SigstoreBundle)
	}
	if utils2.AssertValueProvided(ctx, flags.ReleaseBundleVersion) != nil {
		return errorutils.CheckErrorf("--%s is a mandatory field for creating a Release Bundle evidence", flags.ReleaseBundleVersion)
	}
	if ctx.IsFlagSet(flags.ArtifactsLimit) && !utils.IsFlagPositiveNumber(ctx.GetStringFlagValue(flags.ArtifactsLimit)) {
		return errorutils.CheckErrorf("--%s must be a positive number", flags.ArtifactsLimit)
	}
	return nil
}
