package entity

import (
	"strings"

	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"
	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/command/flags"
	_interface "github.com/jfrog/jfrog-cli-evidence/evidence/cli/command/interface"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/command/utils"
	"github.com/jfrog/jfrog-cli-evidence/evidence/client"
	"github.com/jfrog/jfrog-cli-evidence/evidence/create"
	"github.com/jfrog/jfrog-cli-evidence/evidence/get"
	"github.com/jfrog/jfrog-cli-evidence/evidence/verify"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
)

type evidenceEntityCommand struct {
	ctx     *components.Context
	execute utils.ExecCommandFunc
}

func NewEvidenceEntityCommand(ctx *components.Context, execute utils.ExecCommandFunc) _interface.EvidenceCommands {
	return &evidenceEntityCommand{
		ctx:     ctx,
		execute: execute,
	}
}

func (eec *evidenceEntityCommand) CreateEvidence(ctx *components.Context, serverDetails *config.ServerDetails) error {
	if err := eec.validateEvidenceEntityContext(ctx); err != nil {
		return err
	}
	if err := client.EnsureEntityAPISupported(serverDetails); err != nil {
		return err
	}

	createCmd := create.NewCreateEvidenceEntity(
		serverDetails,
		eec.ctx.GetStringFlagValue(flags.Predicate),
		eec.ctx.GetStringFlagValue(flags.PredicateType),
		eec.ctx.GetStringFlagValue(flags.Markdown),
		eec.ctx.GetStringFlagValue(flags.Key),
		eec.ctx.GetStringFlagValue(flags.KeyAlias),
		eec.ctx.GetStringFlagValue(flags.EntityType),
		eec.ctx.GetStringFlagValue(flags.EntityId),
		eec.ctx.GetStringFlagValue(flags.EntityRepo),
		eec.ctx.GetStringFlagValue(flags.Project),
		eec.ctx.GetStringFlagValue(flags.ApplicationKey),
		eec.ctx.GetStringFlagValue(flags.ProviderId),
		eec.ctx.GetStringFlagValue(flags.SigstoreBundle),
		eec.ctx.GetStringFlagValue(flags.AttachLocal),
		eec.ctx.GetStringFlagValue(flags.AttachArtifactoryTempPath),
		eec.ctx.GetStringFlagValue(flags.AttachArtifactoryPath),
	)
	return eec.execute(createCmd)
}

func (eec *evidenceEntityCommand) GetEvidence(ctx *components.Context, serverDetails *config.ServerDetails) error {
	if err := eec.validateEvidenceEntityContext(ctx); err != nil {
		return err
	}
	if err := client.EnsureEntityAPISupported(serverDetails); err != nil {
		return err
	}

	getCmd := get.NewGetEvidenceEntity(
		serverDetails,
		eec.ctx.GetStringFlagValue(flags.EntityType),
		eec.ctx.GetStringFlagValue(flags.EntityId),
		eec.ctx.GetStringFlagValue(flags.EntityRepo),
		eec.ctx.GetStringFlagValue(flags.Project),
		eec.ctx.GetStringFlagValue(flags.ApplicationKey),
		eec.ctx.GetStringFlagValue(flags.Format),
		eec.ctx.GetStringFlagValue(flags.Output),
		eec.ctx.GetBoolFlagValue(flags.IncludePredicate),
	)
	return eec.execute(getCmd)
}

func (eec *evidenceEntityCommand) VerifyEvidence(ctx *components.Context, serverDetails *config.ServerDetails) error {
	if err := eec.validateEvidenceEntityContext(ctx); err != nil {
		return err
	}
	if err := client.EnsureEntityAPISupported(serverDetails); err != nil {
		return err
	}

	verifyCmd := verify.NewVerifyEvidenceEntity(
		serverDetails,
		eec.ctx.GetStringFlagValue(flags.EntityType),
		eec.ctx.GetStringFlagValue(flags.EntityId),
		eec.ctx.GetStringFlagValue(flags.EntityRepo),
		eec.ctx.GetStringFlagValue(flags.Project),
		eec.ctx.GetStringFlagValue(flags.ApplicationKey),
		eec.ctx.GetStringFlagValue(flags.Format),
		eec.ctx.GetStringsArrFlagValue(flags.PublicKeys),
		eec.ctx.GetBoolFlagValue(flags.UseArtifactoryKeys),
	)
	return eec.execute(verifyCmd)
}

func (eec *evidenceEntityCommand) validateEvidenceEntityContext(ctx *components.Context) error {
	if utils.AssertValueProvided(ctx, flags.EntityType) != nil {
		return errorutils.CheckErrorf("--%s is a mandatory field for entity evidence", flags.EntityType)
	}
	if utils.AssertValueProvided(ctx, flags.EntityId) != nil {
		return errorutils.CheckErrorf("--%s is a mandatory field for entity evidence", flags.EntityId)
	}
	if strings.TrimSpace(ctx.GetStringFlagValue(flags.Integration)) != "" {
		return errorutils.CheckErrorf("--%s is not supported for entity evidence", flags.Integration)
	}
	if ctx.IsFlagSet(flags.ApplicationVersion) && ctx.GetStringFlagValue(flags.ApplicationVersion) != "" {
		return errorutils.CheckErrorf("--%s cannot be combined with --%s; use --%s/--%s for AppTrust application version subjects",
			flags.ApplicationVersion, flags.EntityType, flags.ApplicationKey, flags.ApplicationVersion)
	}

	scopeCount := 0
	for _, flag := range []string{flags.Project, flags.ApplicationKey, flags.EntityRepo} {
		if strings.TrimSpace(ctx.GetStringFlagValue(flag)) != "" {
			scopeCount++
		}
	}
	if scopeCount > 1 {
		return errorutils.CheckErrorf("only one entity scope may be set: --%s, --%s, or --%s",
			flags.Project, flags.ApplicationKey, flags.EntityRepo)
	}
	return nil
}
