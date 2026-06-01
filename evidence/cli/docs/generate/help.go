package generate

import "github.com/jfrog/jfrog-cli-core/v2/plugins/components"

func GetDescription() string {
	return "Generate an ECDSA P-256 key pair for evidence signing. Creates evidence.key (private) and evidence.pub (public) files in the specified output directory (current directory by default). Private keys are stored unencrypted with secure file permissions. Optionally uploads the public key to JFrog platform trusted keys."
}

func GetAIDescription() string {
	return `Generate an ECDSA P-256 key pair used to sign evidence and, by default, upload the public key to the JFrog Platform trusted-keys store under a chosen alias. Use this when bootstrapping a signer identity for jf evd create / jf evd verify in a project or pipeline.

When to use:
- Bootstrap a new signing identity for a project, environment or CI step.
- Rotate keys by generating a new pair with a fresh --key-alias.
- Produce local-only key files for self-hosted verification by setting --upload-public-key=false.

Prerequisites:
- A configured JFrog Platform server (jf c add or jf login) with permission to write trusted keys (when uploading).
- Write access to the output directory (--key-file-path).

Common patterns:
  $ jf evd gen-keys --key-alias my-signer
  $ jf evd gen-keys --key-alias my-signer --key-file-path ./keys --key-file-name release-signer
  $ jf evd gen-keys --upload-public-key=false --key-file-path ./local-keys

Gotchas:
- --upload-public-key defaults to true; passing no flag will contact the configured platform and upload the public key.
- The private key is written unencrypted (mode 0600); store it outside of source control and pipeline logs.
- Default file names are evidence.key and evidence.pub; existing files in the target directory will be overwritten without prompting.
- --key-file-path creates the directory if it does not exist, but only one level deep.
- Generated keys are ECDSA P-256 only; if you need RSA or ed25519 keys, produce them with your own tooling and skip this command.

Related: jf evd create, jf evd verify`
}

func GetArguments() []components.Argument {
	return []components.Argument{}
}
