package create

import "github.com/jfrog/jfrog-cli-core/v2/plugins/components"

func GetDescription() string {
	return " Create a custom evidence and save it to a repository. Add a predicate, predicate-type, repo-path, key, key-name and attachments."
}

func GetAIDescription() string {
	return `Sign a predicate and upload it as an in-toto DSSE evidence attached to a subject in Artifactory. Use this when an agent or CI step needs to record a verifiable attestation (SLSA provenance, SBOM, test results, scanner reports, custom claims) tied to a specific artifact, build, package, application, release bundle, or entity.

When to use:
- Attach signed provenance or scan results to an artifact, build, package, application or release bundle.
- Attach signed evidence to an entity via --entity-type/--entity-id (for example a git commit).
- Re-upload a pre-signed Sigstore bundle via --sigstore-bundle.
- Generate a predicate automatically from a SonarQube scan via --integration sonar.

Prerequisites:
- A configured JFrog Platform server (jf c add or jf login). Basic auth is rejected; use --access-token or a configured server-id.
- A private signing key supplied via --key (path or PEM body) or the JFROG_CLI_SIGNING_KEY env variable. Supported: ecdsa, rsa, ed25519.
- Exactly one subject: --subject-repo-path, --build-name/--build-number (or JFROG_CLI_BUILD_NAME/_NUMBER), --package-name/--package-version/--package-repo-name, --release-bundle/--release-bundle-version, --application-key/--application-version, --entity-type/--entity-id, or bare --application-key (application entity).
- For entity subjects, exactly one of --project, --application-key, or --entity-repo may be set to calculate the Artifactory repository name: --entity-repo uses that key as-is; --project → {project}-{entity-type}-entity; --application-key → {application-key}-{entity-type}-entity; omit all three to default to {entity-type}-entity.
- For attachments: --attach-artifactory-temp-path (or EVIDENCE_ATTACHMENT_ARTIFACTORY_TEMP_PATH config) when using --attach-local.
- For sonar integration: SONAR_TOKEN or SONARQUBE_TOKEN env var plus a report-task.txt from a completed scan.

Common patterns:
  $ jf evd create --subject-repo-path generic-local/app.tgz --predicate ./provenance.json --predicate-type https://slsa.dev/provenance/v1 --key ./evidence.key --key-alias my-signer
  $ jf evd create --build-name my-build --build-number 42 --predicate ./sbom.json --predicate-type https://cyclonedx.org/bom --key-alias my-signer
  $ jf evd create --package-name my-npm-pkg --package-version 1.2.3 --package-repo-name npm-local --predicate ./scan.json --predicate-type https://example.com/scan/v1
  $ jf evd create --release-bundle my-rb --release-bundle-version 1.0.0 --predicate ./attest.json --predicate-type https://example.com/attest/v1
  $ jf evd create --entity-type gitCommit --entity-id 57bb812f3733b80e270272ba063274e52c34bd23 --project my-proj --predicate ./approval.json --predicate-type https://jfrog.com/evidence/commit-approval/v1 --key ./evidence.key
  $ jf evd create --entity-type gitCommit --entity-id 57bb812f3733b80e270272ba063274e52c34bd23 --project my-proj --sigstore-bundle ./commit.sigstore.json
  $ jf evd create --application-key my-app --predicate ./app.json --predicate-type https://example.com/app/v1 --key ./evidence.key
  $ jf evd create --subject-repo-path generic-local/app.tgz --sigstore-bundle ./app.sigstore.json
  $ jf evd create --build-name my-build --build-number 42 --integration sonar

Gotchas:
- --sigstore-bundle is mutually exclusive with --key, --key-alias, --predicate, --predicate-type, --subject-sha256 and all --attach-* flags (values are extracted from the bundle). With --entity-type/--entity-id the bundle is uploaded to the entity endpoint; the statement subject digest must use the entity type key (for example gitCommit), not sha256.
- Specifying multiple subjects in one invocation is an error, except the documented --type + --build-name (gh-committer) combination.
- Bare --application-key (without --application-version) creates entity evidence for the application and auto-resolves its project.
- --attach-local uploads the file to --attach-artifactory-temp-path first; once set, the temp path is persisted in the evidence config for subsequent runs.
- Evidence services reject basic authentication; only access tokens work.
- When --integration sonar is used, --predicate and --predicate-type must be omitted; the predicate is generated from the SonarQube report.
- Output formatting (--format json|table) only renders after a successful create call.

Related: jf evd verify, jf evd get, jf evd gen-keys`
}

func GetArguments() []components.Argument {
	return []components.Argument{}
}

// GetUsageExamples returns entity-focused create usage lines shown under the command's Usage help.
func GetUsageExamples() []string {
	return []string{
		"evd create --entity-type gitCommit --entity-id 57bb812f3733b80e270272ba063274e52c34bd23 --project my-proj --predicate ./approval.json --predicate-type https://jfrog.com/evidence/commit-approval/v1 --key ./evidence.key",
		"evd create --entity-type gitCommit --entity-id 57bb812f3733b80e270272ba063274e52c34bd23 --project my-proj --sigstore-bundle ./commit.sigstore.json",
		"evd create --application-key my-app --predicate ./app.json --predicate-type https://example.com/app/v1 --key ./evidence.key",
	}
}
