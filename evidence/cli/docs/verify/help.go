package verify

import "github.com/jfrog/jfrog-cli-core/v2/plugins/components"

func GetDescription() string {
	return `Verify all evidence associated with the specified subject. Provide the subject's path and relevant keys.
	Keys can be supplied using the --keys flag, the JFROG_CLI_SIGNING_KEY environment variable, or retrieved from Artifactory using the --use-artifactory-keys option.`
}

func GetAIDescription() string {
	return `Verify every DSSE evidence attached to a subject by checking signatures against supplied or Artifactory-stored public keys, plus any attachment integrity. Use this when an agent needs to confirm that evidence on an artifact, build, package or release bundle is signed by a trusted key before continuing a pipeline.

When to use:
- Gate a release on signed provenance/SBOM/scan evidence being present and valid.
- Re-verify evidence after a key rotation by re-running with the new --public-keys.
- Validate evidence with trust roots managed in Artifactory via --use-artifactory-keys.

Prerequisites:
- A configured JFrog Platform server (jf c add or jf login) using access-token auth.
- One or more public keys provided via --public-keys (semicolon-separated paths or PEM bodies), JFROG_CLI_SIGNING_KEY, or --use-artifactory-keys.
- Exactly one subject: --subject-repo-path, --build-name/--build-number, --package-name/--package-version/--package-repo-name, or --release-bundle/--release-bundle-version.
- Supported key algorithms: ecdsa, rsa, ed25519.

Common patterns:
  $ jf evd verify --subject-repo-path generic-local/app.tgz --public-keys ./evidence.pub
  $ jf evd verify --subject-repo-path generic-local/app.tgz --use-artifactory-keys --format json
  $ jf evd verify --build-name my-build --build-number 42 --public-keys ./key1.pub;./key2.pub
  $ jf evd verify --release-bundle my-rb --release-bundle-version 1.0.0 --public-keys ./evidence.pub
  $ jf evd verify --package-name my-npm-pkg --package-version 1.2.3 --package-repo-name npm-local --use-artifactory-keys

Gotchas:
- JFROG_CLI_SIGNING_KEY is appended to whatever is passed via --public-keys; ensure the env var is unset if you only want explicit keys.
- --public-keys uses ";" as the separator, not "," or whitespace.
- --application-key subjects are not supported by verify (only create/get cover them).
- Failures from the verifier are wrapped as "evidence verification failed: ..."; check the wrapped cause for the specific signature, key or attachment mismatch.
- --use-artifactory-keys still requires platform credentials with read access to the trusted-keys store.
- Attachments referenced by evidence are also verified; mismatched or missing attachment files cause the whole verify to fail.

Related: jf evd create, jf evd get, jf evd gen-keys`
}

func GetArguments() []components.Argument {
	return []components.Argument{}
}
