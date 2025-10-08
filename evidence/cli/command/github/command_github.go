package github

import (
	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"
	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/command/flags"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/command/interface"
	utils2 "github.com/jfrog/jfrog-cli-evidence/evidence/cli/command/utils"
	"github.com/jfrog/jfrog-cli-evidence/evidence/create"
	"github.com/jfrog/jfrog-cli-evidence/evidence/utils"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
)

type evidenceGitHubCommand struct {
	ctx     *components.Context
	execute utils2.ExecCommandFunc
}

func NewEvidenceGitHubCommand(ctx *components.Context, execute utils2.ExecCommandFunc) _interface.EvidenceCommands {
	return &evidenceGitHubCommand{
		ctx:     ctx,
		execute: execute,
	}
}

func (ebc *evidenceGitHubCommand) GetEvidence(_ *components.Context, _ *config.ServerDetails) error {
	return errorutils.CheckErrorf("Get evidence is not supported with github")
}

func (ebc *evidenceGitHubCommand) CreateEvidence(ctx *components.Context, serverDetails *config.ServerDetails) error {
	err := ebc.validateEvidenceGithubContext(ctx)
	if err != nil {
		return err
	}

	createCmd := create.NewCreateGithub(
		serverDetails,
		ebc.ctx.GetStringFlagValue(flags.Predicate),
		ebc.ctx.GetStringFlagValue(flags.PredicateType),
		ebc.ctx.GetStringFlagValue(flags.Markdown),
		ebc.ctx.GetStringFlagValue(flags.Key),
		ebc.ctx.GetStringFlagValue(flags.KeyAlias),
		ebc.ctx.GetStringFlagValue(flags.Project),
		ebc.ctx.GetStringFlagValue(flags.BuildName),
		ebc.ctx.GetStringFlagValue(flags.BuildNumber),
		ebc.ctx.GetStringFlagValue(flags.TypeFlag))
	return ebc.execute(createCmd)
}

func (ebc *evidenceGitHubCommand) VerifyEvidence(_ *components.Context, _ *config.ServerDetails) error {
	return errorutils.CheckErrorf("Verify evidence is not supported with github")

}

func (ebc *evidenceGitHubCommand) validateEvidenceGithubContext(ctx *components.Context) error {
	// buildName is not validated since it is required for the evd context
	if utils.IsSonarIntegration(ctx.GetStringFlagValue(flags.Integration)) {
		return errorutils.CheckErrorf("--%s %s is not supported for GitHub evidence.", flags.Integration, utils.SonarIntegration)
	}
	if ebc.ctx.GetStringFlagValue(flags.SigstoreBundle) != "" {
		return errorutils.CheckErrorf("--%s is not supported for GitHub evidence.", flags.SigstoreBundle)
	}
	if utils2.AssertValueProvided(ctx, flags.BuildNumber) != nil {
		return errorutils.CheckErrorf("--%s is a mandatory field for creating a Release Bundle evidence", flags.BuildNumber)
	}
	return nil
}
