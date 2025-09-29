package application

import (
	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"
	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/command/flags"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/command/interface"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/command/utils"
	"github.com/jfrog/jfrog-cli-evidence/evidence/create"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
)

type evidenceApplicationCommand struct {
	ctx     *components.Context
	execute utils.ExecCommandFunc
}

func NewEvidenceApplicationCommand(ctx *components.Context, execute utils.ExecCommandFunc) _interface.EvidenceCommands {
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
		eac.ctx.GetStringFlagValue(flags.Predicate),
		eac.ctx.GetStringFlagValue(flags.PredicateType),
		eac.ctx.GetStringFlagValue(flags.Markdown),
		eac.ctx.GetStringFlagValue(flags.Key),
		eac.ctx.GetStringFlagValue(flags.KeyAlias),
		eac.ctx.GetStringFlagValue(flags.ApplicationKey),
		eac.ctx.GetStringFlagValue(flags.ApplicationVersion),
		eac.ctx.GetStringFlagValue(flags.ProviderId),
		eac.ctx.GetStringFlagValue(flags.Integration))
	return eac.execute(createCmd)
}

func (eac *evidenceApplicationCommand) GetEvidence(_ *components.Context, _ *config.ServerDetails) error {
	return errorutils.CheckErrorf("get evidence is not supported for application evidence yet")
}

func (eac *evidenceApplicationCommand) VerifyEvidence(_ *components.Context, _ *config.ServerDetails) error {
	return errorutils.CheckErrorf("verify evidence is not supported for application evidence yet")
}

func (eac *evidenceApplicationCommand) validateEvidenceApplicationContext(ctx *components.Context) error {
	if eac.ctx.GetStringFlagValue(flags.SigstoreBundle) != "" {
		return errorutils.CheckErrorf("--%s is not supported for application evidence.", flags.SigstoreBundle)
	}
	if utils.AssertValueProvided(ctx, flags.ApplicationKey) != nil {
		return errorutils.CheckErrorf("--%s is a mandatory field for creating an Application evidence", flags.ApplicationKey)
	}
	if utils.AssertValueProvided(ctx, flags.ApplicationVersion) != nil {
		return errorutils.CheckErrorf("--%s is a mandatory field for creating an Application evidence", flags.ApplicationVersion)
	}
	if ctx.IsFlagSet(flags.Project) {
		return errorutils.CheckErrorf("--%s flag is not allowed when using application-based evidence creation. The project will be automatically determined from the application details", flags.Project)
	}
	return nil
}
