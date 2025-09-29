package command

import (
	pluginsCommon "github.com/jfrog/jfrog-cli-core/v2/plugins/common"
	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"
)

const (
	CreateEvidence  = "create-evidence"
	GetEvidence     = "get-evidence"
	VerifyEvidence  = "verify-evidence"
	GenerateKeyPair = "generate-key-pair"
)

const (
	ServerId    = "server-id"
	Url         = "url"
	User        = "user"
	AccessToken = "access-token"
	Project     = "project"
	Format      = "format"
	Output      = "output"

	ReleaseBundle        = "release-bundle"
	ReleaseBundleVersion = "release-bundle-version"
	BuildName            = "build-name"
	BuildNumber          = "build-number"
	PackageName          = "package-name"
	PackageVersion       = "package-version"
	PackageRepoName      = "package-repo-name"
	TypeFlag             = "type"
	ApplicationKey       = "application-key"
	ApplicationVersion   = "application-version"

	Predicate          = "predicate"
	PredicateType      = "predicate-type"
	IncludePredicate   = "include-Predicate"
	Markdown           = "markdown"
	SubjectRepoPath    = "subject-repo-path"
	SubjectSha256      = "subject-sha256"
	Key                = "key"
	KeyAlias           = "key-alias"
	ProviderId         = "provider-id"
	PublicKeys         = "public-keys"
	UseArtifactoryKeys = "use-artifactory-keys"
	Integration        = "integration"
	SigstoreBundle     = "sigstore-bundle"
	ArtifactsLimit     = "artifacts-limit"
	UploadPublicKey    = "upload-public-key"
	Force              = "force"
	OutputDir          = "Output-dir"
	KeyFileName        = "key-file-name"
)

// Flag keys mapped to their corresponding components.Flag definition.
var flagsMap = map[string]components.Flag{
	// Common command flags
	ServerId:    components.NewStringFlag(ServerId, "Server ID configured using the config command.", func(f *components.StringFlag) { f.Mandatory = false }),
	Url:         components.NewStringFlag(Url, "JFrog Platform URL.", func(f *components.StringFlag) { f.Mandatory = false }),
	User:        components.NewStringFlag(User, "JFrog username.", func(f *components.StringFlag) { f.Mandatory = false }),
	AccessToken: components.NewStringFlag(AccessToken, "JFrog access token.", func(f *components.StringFlag) { f.Mandatory = false }),
	Project:     components.NewStringFlag(Project, "Project key associated with the created evidence.", func(f *components.StringFlag) { f.Mandatory = false }),
	Format:      components.NewStringFlag(Format, "Output Format. Supported formats: 'json'. For 'jf evd get' command you can additionally choose 'jsonl' Format", func(f *components.StringFlag) { f.Mandatory = false }),
	Output:      components.NewStringFlag(Output, "Output file path, should be in the Format of 'path/to/file.json'. If not provided, Output will be printed to the console.", func(f *components.StringFlag) { f.Mandatory = false }),

	ReleaseBundle:        components.NewStringFlag(ReleaseBundle, "Release Bundle name.", func(f *components.StringFlag) { f.Mandatory = false }),
	ReleaseBundleVersion: components.NewStringFlag(ReleaseBundleVersion, "Release Bundle version.", func(f *components.StringFlag) { f.Mandatory = false }),
	BuildName:            components.NewStringFlag(BuildName, "Build name.", func(f *components.StringFlag) { f.Mandatory = false }),
	BuildNumber:          components.NewStringFlag(BuildNumber, "Build number.", func(f *components.StringFlag) { f.Mandatory = false }),
	PackageName:          components.NewStringFlag(PackageName, "Package name.", func(f *components.StringFlag) { f.Mandatory = false }),
	PackageVersion:       components.NewStringFlag(PackageVersion, "Package version.", func(f *components.StringFlag) { f.Mandatory = false }),
	PackageRepoName:      components.NewStringFlag(PackageRepoName, "Package repository Name.", func(f *components.StringFlag) { f.Mandatory = false }),
	TypeFlag:             components.NewStringFlag(TypeFlag, "Type can contain 'gh-commiter' value.", func(f *components.StringFlag) { f.Mandatory = false }),
	ApplicationKey:       components.NewStringFlag(ApplicationKey, "Application key.", func(f *components.StringFlag) { f.Mandatory = false }),
	ApplicationVersion:   components.NewStringFlag(ApplicationVersion, "Application version.", func(f *components.StringFlag) { f.Mandatory = false }),

	Predicate:        components.NewStringFlag(Predicate, "Path to the Predicate, arbitrary JSON. Mandatory unless --"+SigstoreBundle+" is used", func(f *components.StringFlag) { f.Mandatory = false }),
	PredicateType:    components.NewStringFlag(PredicateType, "Type of the Predicate. Mandatory unless --"+SigstoreBundle+" is used", func(f *components.StringFlag) { f.Mandatory = false }),
	IncludePredicate: components.NewBoolFlag(IncludePredicate, "Include the Predicate data in the get evidence Output.", components.WithBoolDefaultValueFalse()),
	Markdown:         components.NewStringFlag(Markdown, "Markdown of the Predicate.", func(f *components.StringFlag) { f.Mandatory = false }),
	SubjectRepoPath:  components.NewStringFlag(SubjectRepoPath, "Full path to some subject location.", func(f *components.StringFlag) { f.Mandatory = false }),
	SubjectSha256:    components.NewStringFlag(SubjectSha256, "Subject checksum sha256.", func(f *components.StringFlag) { f.Mandatory = false }),
	Key:              components.NewStringFlag(Key, "Path to a private key that will sign the DSSE. Supported keys: 'ecdsa','rsa' and 'ed25519'.", func(f *components.StringFlag) { f.Mandatory = false }),
	KeyAlias:         components.NewStringFlag(KeyAlias, "Key alias", func(f *components.StringFlag) { f.Mandatory = false }),

	ProviderId:         components.NewStringFlag(ProviderId, "Provider ID for the evidence.", func(f *components.StringFlag) { f.Mandatory = false }),
	PublicKeys:         components.NewStringFlag(PublicKeys, "Array of paths to public keys for signatures verification with \";\" separator. Supported keys: 'ecdsa','rsa' and 'ed25519'.", func(f *components.StringFlag) { f.Mandatory = false }),
	SigstoreBundle:     components.NewStringFlag(SigstoreBundle, "Path to a Sigstore bundle file with a pre-signed DSSE envelope. Incompatible with --"+Key+", --"+KeyAlias+", --"+Predicate+", --"+PredicateType+" and --"+SubjectSha256+".", func(f *components.StringFlag) { f.Mandatory = false }),
	UseArtifactoryKeys: components.NewBoolFlag(UseArtifactoryKeys, "Use Artifactory keys for verification. When enabled, the verify command retrieves keys from Artifactory.", components.WithBoolDefaultValueFalse()),
	ArtifactsLimit:     components.NewStringFlag(ArtifactsLimit, "The number of artifacts in a release bundle to be included in the evidences file. The default value is 1000 artifacts", func(f *components.StringFlag) { f.Mandatory = false }),
	Integration:        components.NewStringFlag(Integration, "Specify an integration to automatically generate the Predicate. Supported: 'sonar'. When using 'sonar', the 'SONAR_TOKEN' or 'SONARQUBE_TOKEN' environment variable must be set.", func(f *components.StringFlag) { f.Mandatory = false }),
	UploadPublicKey:    components.NewBoolFlag(UploadPublicKey, "Upload the generated public key to JFrog platform trusted keys. Requires server connection.", components.WithBoolDefaultValueFalse()),
	Force:              components.NewBoolFlag(Force, "Overwrite existing key files if they exist.", components.WithBoolDefaultValueFalse()),
	OutputDir:          components.NewStringFlag(OutputDir, "Output directory for key files. Creates the directory if it doesn't exist. Defaults to current directory.", func(f *components.StringFlag) { f.Mandatory = false }),
	KeyFileName:        components.NewStringFlag(KeyFileName, "Base name for key files (without extension). Private key will be saved as <name>.key and public key as <name>.pub. Defaults to 'evidence'.", func(f *components.StringFlag) { f.Mandatory = false }),
}

var commandFlags = map[string][]string{
	CreateEvidence: {
		Url,
		User,
		AccessToken,
		ServerId,
		Project,
		ReleaseBundle,
		ReleaseBundleVersion,
		BuildName,
		BuildNumber,
		PackageName,
		PackageVersion,
		PackageRepoName,
		TypeFlag,
		ApplicationKey,
		ApplicationVersion,
		Predicate,
		PredicateType,
		Markdown,
		SubjectRepoPath,
		SubjectSha256,
		Key,
		KeyAlias,
		ProviderId,
		Integration,
		SigstoreBundle,
	},
	VerifyEvidence: {
		Url,
		User,
		AccessToken,
		ServerId,
		PublicKeys,
		Format,
		Project,
		ReleaseBundle,
		ReleaseBundleVersion,
		SubjectRepoPath,
		BuildName,
		BuildNumber,
		PackageName,
		PackageVersion,
		PackageRepoName,
		UseArtifactoryKeys,
	},
	GetEvidence: {
		Url,
		User,
		AccessToken,
		ServerId,
		Format,
		Output,
		Project,
		ReleaseBundle,
		ReleaseBundleVersion,
		SubjectRepoPath,
		IncludePredicate,
		ArtifactsLimit,
	},
	GenerateKeyPair: {
		Url,
		User,
		AccessToken,
		ServerId,
		UploadPublicKey,
		KeyAlias,
		Force,
		OutputDir,
		KeyFileName,
	},
}

func GetCommandFlags(cmdKey string) []components.Flag {
	return pluginsCommon.GetCommandFlags(cmdKey, commandFlags, flagsMap)
}
