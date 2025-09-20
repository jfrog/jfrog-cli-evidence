package cli

import (
	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"
	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli-evidence/evidence/create"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
)

type evidenceApplicationCommand struct {
	ctx     *components.Context
	execute execCommandFunc
}

func NewEvidenceApplicationCommand(ctx *components.Context, execute execCommandFunc) EvidenceCommands {
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
		eac.ctx.GetStringFlagValue(predicate),
		eac.ctx.GetStringFlagValue(predicateType),
		eac.ctx.GetStringFlagValue(markdown),
		eac.ctx.GetStringFlagValue(key),
		eac.ctx.GetStringFlagValue(keyAlias),
		eac.ctx.GetStringFlagValue(applicationKey),
		eac.ctx.GetStringFlagValue(applicationVersion),
		eac.ctx.GetStringFlagValue(providerId),
		eac.ctx.GetStringFlagValue(integration))
	return eac.execute(createCmd)
}

func (eac *evidenceApplicationCommand) GetEvidence(ctx *components.Context, serverDetails *config.ServerDetails) error {
	return errorutils.CheckErrorf("get evidence is not supported for application evidence yet")
}

func (eac *evidenceApplicationCommand) VerifyEvidence(ctx *components.Context, serverDetails *config.ServerDetails) error {
	return errorutils.CheckErrorf("verify evidence is not supported for application evidence yet")
}

func (eac *evidenceApplicationCommand) validateEvidenceApplicationContext(ctx *components.Context) error {
	if eac.ctx.GetStringFlagValue(sigstoreBundle) != "" {
		return errorutils.CheckErrorf("--%s is not supported for application evidence.", sigstoreBundle)
	}
	if assertValueProvided(ctx, applicationKey) != nil {
		return errorutils.CheckErrorf("--%s is a mandatory field for creating an Application evidence", applicationKey)
	}
	if assertValueProvided(ctx, applicationVersion) != nil {
		return errorutils.CheckErrorf("--%s is a mandatory field for creating an Application evidence", applicationVersion)
	}
	if ctx.IsFlagSet(project) {
		return errorutils.CheckErrorf("--%s flag is not allowed when using application-based evidence creation. The project will be automatically determined from the application details", project)
	}
	return nil
}
