package generate

import "github.com/jfrog/jfrog-cli-core/v2/plugins/components"

func GetDescription() string {
	return "Generate an ECDSA P-256 key pair for evidence signing. Creates evidence.key (private) and evidence.pub (public) files in the specified output directory (current directory by default). Private keys are stored unencrypted with secure file permissions. Optionally uploads the public key to JFrog platform trusted keys."
}

func GetArguments() []components.Argument {
	return []components.Argument{}
}
