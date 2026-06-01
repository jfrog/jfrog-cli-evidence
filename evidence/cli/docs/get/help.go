package get

import "github.com/jfrog/jfrog-cli-core/v2/plugins/components"

func GetDescription() string {
	return ` Fetch evidence based on a specified subject, which can be either an artifact or a release bundle.
                             When retrieving evidence from a release bundle, you will obtain information about the builds contained within it,
                             as well as the artifacts associated with those builds.
                             Supports JSON and JSONL formats.`
}

func GetAIDescription() string {
	return `Retrieve evidence (DSSE envelopes and metadata) attached to a subject from Artifactory. Use this when an agent needs to inspect or ingest existing evidence for an artifact or release bundle, optionally including the predicate payload.

When to use:
- List all evidence attached to a single artifact (--subject-repo-path).
- Enumerate evidence across every build and artifact of a release bundle (--release-bundle / --release-bundle-version).
- Pipe machine-readable output into downstream tooling via --format json or --format jsonl.

Prerequisites:
- A configured JFrog Platform server (jf c add or jf login) with access-token auth; basic auth is rejected.
- Read permissions on the subject path and the evidence repository.
- Exactly one of --subject-repo-path or --release-bundle (with --release-bundle-version). Other subject types are not supported by get.

Common patterns:
  $ jf evd get --subject-repo-path generic-local/app.tgz
  $ jf evd get --subject-repo-path generic-local/app.tgz --include-predicate --format json
  $ jf evd get --release-bundle my-rb --release-bundle-version 1.0.0 --format jsonl --output ./rb-evidence.jsonl
  $ jf evd get --release-bundle my-rb --release-bundle-version 1.0.0 --artifacts-limit 5000

Gotchas:
- --include-predicate is off by default; without it the predicate body is omitted from results.
- --artifacts-limit defaults to 1000 for release bundles; larger bundles need an explicit higher value.
- Only --subject-repo-path and --release-bundle subjects are accepted; passing --build-name, --package-name or --application-key returns "unsupported subject".
- --output writes to a file; without it results go to stdout.
- jsonl is only useful with --format; the default human format ignores --output formatting for streaming.

Related: jf evd create, jf evd verify`
}

func GetArguments() []components.Argument {
	return []components.Argument{}
}
