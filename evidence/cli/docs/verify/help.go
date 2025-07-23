package verify

import "github.com/jfrog/jfrog-cli-core/v2/plugins/components"

func GetDescription() string {
	return `Verify all evidence associated with the specified subject. Provide the subject's path and relevant keys.
	Keys can be supplied using the --keys flag, the JFROG_CLI_SIGNING_KEY environment variable, or retrieved from Artifactory using the --use-artifactory-keys option.`
}

func GetArguments() []components.Argument {
	return []components.Argument{}
}
