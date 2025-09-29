package github

import (
	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"
	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/command"
	"github.com/jfrog/jfrog-cli-evidence/evidence/create"
	"github.com/jfrog/jfrog-cli-evidence/evidence/utils"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
)

type evidenceGitHubCommand struct {
	ctx     *components.Context
	execute command.ExecCommandFunc
}

func NewEvidenceGitHubCommand(ctx *components.Context, execute command.ExecCommandFunc) command.EvidenceCommands {
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
		ebc.ctx.GetStringFlagValue(command.Predicate),
		ebc.ctx.GetStringFlagValue(command.PredicateType),
		ebc.ctx.GetStringFlagValue(command.Markdown),
		ebc.ctx.GetStringFlagValue(command.Key),
		ebc.ctx.GetStringFlagValue(command.KeyAlias),
		ebc.ctx.GetStringFlagValue(command.Project),
		ebc.ctx.GetStringFlagValue(command.BuildName),
		ebc.ctx.GetStringFlagValue(command.BuildNumber),
		ebc.ctx.GetStringFlagValue(command.TypeFlag))
	return ebc.execute(createCmd)
}

func (ebc *evidenceGitHubCommand) VerifyEvidence(_ *components.Context, _ *config.ServerDetails) error {
	return errorutils.CheckErrorf("Verify evidence is not supported with github")

}

func (ebc *evidenceGitHubCommand) validateEvidenceGithubContext(ctx *components.Context) error {
	// buildName is not validated since it is required for the evd context
	if utils.IsSonarIntegration(ctx.GetStringFlagValue(command.Integration)) {
		return errorutils.CheckErrorf("--%s %s is not supported for GitHub evidence.", command.Integration, utils.SonarIntegration)
	}
	if ebc.ctx.GetStringFlagValue(command.SigstoreBundle) != "" {
		return errorutils.CheckErrorf("--%s is not supported for GitHub evidence.", command.SigstoreBundle)
	}
	if command.AssertValueProvided(ctx, command.BuildNumber) != nil {
		return errorutils.CheckErrorf("--%s is a mandatory field for creating a Release Bundle evidence", command.BuildNumber)
	}
	return nil
}
