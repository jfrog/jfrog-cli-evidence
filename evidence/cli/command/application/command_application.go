package application

import (
	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"
	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/command"
	"github.com/jfrog/jfrog-cli-evidence/evidence/create"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
)

type evidenceApplicationCommand struct {
	ctx     *components.Context
	execute command.ExecCommandFunc
}

func NewEvidenceApplicationCommand(ctx *components.Context, execute command.ExecCommandFunc) command.EvidenceCommands {
	return &evidenceApplicationCommand{
		ctx:     ctx,
		execute: execute,
	}
}

func (eac *evidenceApplicationCommand) CreateEvidence(ctx *components.Context, serverDetails *config.ServerDetails) error {
	err := eac.validateEvidenceApplicationContext(ctx)
	if err != nil {
		return err
	}

	createCmd := create.NewCreateEvidenceApplication(
		serverDetails,
		eac.ctx.GetStringFlagValue(command.Predicate),
		eac.ctx.GetStringFlagValue(command.PredicateType),
		eac.ctx.GetStringFlagValue(command.Markdown),
		eac.ctx.GetStringFlagValue(command.Key),
		eac.ctx.GetStringFlagValue(command.KeyAlias),
		eac.ctx.GetStringFlagValue(command.ApplicationKey),
		eac.ctx.GetStringFlagValue(command.ApplicationVersion),
		eac.ctx.GetStringFlagValue(command.ProviderId),
		eac.ctx.GetStringFlagValue(command.Integration))
	return eac.execute(createCmd)
}

func (eac *evidenceApplicationCommand) GetEvidence(_ *components.Context, _ *config.ServerDetails) error {
	return errorutils.CheckErrorf("get evidence is not supported for application evidence yet")
}

func (eac *evidenceApplicationCommand) VerifyEvidence(_ *components.Context, _ *config.ServerDetails) error {
	return errorutils.CheckErrorf("verify evidence is not supported for application evidence yet")
}

func (eac *evidenceApplicationCommand) validateEvidenceApplicationContext(ctx *components.Context) error {
	if eac.ctx.GetStringFlagValue(command.SigstoreBundle) != "" {
		return errorutils.CheckErrorf("--%s is not supported for application evidence.", command.SigstoreBundle)
	}
	if command.AssertValueProvided(ctx, command.ApplicationKey) != nil {
		return errorutils.CheckErrorf("--%s is a mandatory field for creating an Application evidence", command.ApplicationKey)
	}
	if command.AssertValueProvided(ctx, command.ApplicationVersion) != nil {
		return errorutils.CheckErrorf("--%s is a mandatory field for creating an Application evidence", command.ApplicationVersion)
	}
	if ctx.IsFlagSet(command.Project) {
		return errorutils.CheckErrorf("--%s flag is not allowed when using application-based evidence creation. The project will be automatically determined from the application details", command.Project)
	}
	return nil
}
