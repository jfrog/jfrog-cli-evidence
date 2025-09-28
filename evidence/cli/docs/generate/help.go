package generate

import "github.com/jfrog/jfrog-cli-core/v2/plugins/components"

func GetDescription() string {
	return "Generate an ECDSA P-256 key pair for evidence signing. Creates evidence.key (private) and evidence.pub (public) files in the specified output directory (current directory by default). The private key is stored unencrypted and protected by file permissions (0600). Optionally uploads the public key to JFrog platform trusted keys."
}

func GetArguments() []components.Argument {
	return []components.Argument{}
}
