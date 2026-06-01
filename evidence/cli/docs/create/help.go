package create

import "github.com/jfrog/jfrog-cli-core/v2/plugins/components"

func GetDescription() string {
	return " Create a custom evidence and save it to a repository. Add a predicate, predicate-type, repo-path, key, key-name and attachments."
}

func GetAIDescription() string {
	return `Sign a predicate and upload it as an in-toto DSSE evidence attached to a subject in Artifactory. Use this when an agent or CI step needs to record a verifiable attestation (SLSA provenance, SBOM, test results, scanner reports, custom claims) tied to a specific artifact, build, package, application or release bundle.

When to use:
- Attach signed provenance or scan results to an artifact, build, package, application or release bundle.
- Re-upload a pre-signed Sigstore bundle via --sigstore-bundle.
- Generate a predicate automatically from a SonarQube scan via --integration sonar.

Prerequisites:
- A configured JFrog Platform server (jf c add or jf login). Basic auth is rejected; use --access-token or a configured server-id.
- A private signing key supplied via --key (path or PEM body) or the JFROG_CLI_SIGNING_KEY env variable. Supported: ecdsa, rsa, ed25519.
- Exactly one subject: --subject-repo-path, --build-name/--build-number (or JFROG_CLI_BUILD_NAME/_NUMBER), --package-name/--package-version/--package-repo-name, --release-bundle/--release-bundle-version, or --application-key/--application-version.
- For attachments: --attach-artifactory-temp-path (or EVIDENCE_ATTACHMENT_ARTIFACTORY_TEMP_PATH config) when using --attach-local.
- For sonar integration: SONAR_TOKEN or SONARQUBE_TOKEN env var plus a report-task.txt from a completed scan.

Common patterns:
  $ jf evd create --subject-repo-path generic-local/app.tgz --predicate ./provenance.json --predicate-type https://slsa.dev/provenance/v1 --key ./evidence.key --key-alias my-signer
  $ jf evd create --build-name my-build --build-number 42 --predicate ./sbom.json --predicate-type https://cyclonedx.org/bom --key-alias my-signer
  $ jf evd create --package-name my-npm-pkg --package-version 1.2.3 --package-repo-name npm-local --predicate ./scan.json --predicate-type https://example.com/scan/v1
  $ jf evd create --release-bundle my-rb --release-bundle-version 1.0.0 --predicate ./attest.json --predicate-type https://example.com/attest/v1
  $ jf evd create --subject-repo-path generic-local/app.tgz --sigstore-bundle ./app.sigstore.json
  $ jf evd create --build-name my-build --build-number 42 --integration sonar

Gotchas:
- --sigstore-bundle is mutually exclusive with --key, --key-alias, --predicate, --predicate-type, --subject-sha256 and all --attach-* flags (values are extracted from the bundle).
- Specifying multiple subjects in one invocation is an error, except the documented --type + --build-name (gh-committer) combination.
- --attach-local uploads the file to --attach-artifactory-temp-path first; once set, the temp path is persisted in the evidence config for subsequent runs.
- Evidence services reject basic authentication; only access tokens work.
- When --integration sonar is used, --predicate and --predicate-type must be omitted; the predicate is generated from the SonarQube report.
- Output formatting (--format json|table) only renders after a successful create call.

Related: jf evd verify, jf evd get, jf evd gen-keys`
}

func GetArguments() []components.Argument {
	return []components.Argument{}
}
