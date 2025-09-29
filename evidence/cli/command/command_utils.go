package command

import (
	"github.com/jfrog/jfrog-cli-core/v2/common/commands"
	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
)

type ExecCommandFunc func(command commands.Command) error

func Exec(command commands.Command) error {
	return commands.Exec(command)
}

var subjectTypes = []string{
	SubjectRepoPath,
	ReleaseBundle,
	BuildName,
	PackageName,
	TypeFlag,
	ApplicationKey,
}

func AssertValueProvided(c *components.Context, fieldName string) error {
	if !c.IsFlagSet(fieldName) || c.GetStringFlagValue(fieldName) == "" {
		return errorutils.CheckErrorf("the argument --%s can not be empty", fieldName)
	}
	return nil
}
