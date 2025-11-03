package evidence

import (
	"fmt"
	"strings"

	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/artifactory/services"
	"github.com/jfrog/jfrog-client-go/metadata"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"github.com/jfrog/jfrog-client-go/utils/log"
)

// PackageService defines the interface for package-related operations
type PackageService interface {
	// GetPackageType retrieves the package type for the given package
	GetPackageType(artifactoryClient artifactory.ArtifactoryServicesManager) (string, error)

	// GetPackageVersionLeadArtifact retrieves the lead artifact path for a package version
	// with fallback logic from Artifactory to Metadata service
	GetPackageVersionLeadArtifact(packageType string, metadataClient metadata.Manager, artifactoryClient artifactory.ArtifactoryServicesManager) (string, error)

	// GetPackageName returns the package name
	GetPackageName() string

	// GetPackageVersion returns the package version
	GetPackageVersion() string

	// GetPackageRepoName returns the package repository name
	GetPackageRepoName() string
}

// NewPackageService creates a new PackageService instance
// This factory function allows for easy creation and potential future extension
func NewPackageService(name, version, repoName string) PackageService {
	return &basePackage{
		PackageName:     name,
		PackageVersion:  version,
		PackageRepoName: repoName,
	}
}

// basePackage provides shared logic for package evidence command (create/verify)
// It implements the PackageService interface
type basePackage struct {
	PackageName     string
	PackageVersion  string
	PackageRepoName string
}

// Ensure basePackage implements PackageService interface
var _ PackageService = (*basePackage)(nil)

func (b *basePackage) GetPackageType(artifactoryClient artifactory.ArtifactoryServicesManager) (string, error) {
	if artifactoryClient == nil {
		return "", errorutils.CheckErrorf("Artifactory client is required")
	}

	var response services.RepositoryDetails
	err := artifactoryClient.GetRepository(b.PackageRepoName, &response)
	if err != nil {
		return "", errorutils.CheckErrorf("failed to get repository '%s': %w", b.PackageRepoName, err)
	}
	return response.PackageType, nil
}

func (b *basePackage) GetPackageVersionLeadArtifact(packageType string, metadataClient metadata.Manager, artifactoryClient artifactory.ArtifactoryServicesManager) (string, error) {
	if artifactoryClient == nil {
		return "", errorutils.CheckErrorf("Artifactory client is required")
	}
	if metadataClient == nil {
		return "", errorutils.CheckErrorf("Metadata client is required")
	}

	leadFileRequest := services.LeadFileParams{
		PackageType:     strings.ToUpper(packageType),
		PackageRepoName: b.PackageRepoName,
		PackageName:     b.PackageName,
		PackageVersion:  b.PackageVersion,
	}

	leadArtifact, err := artifactoryClient.GetPackageLeadFile(leadFileRequest)
	if err != nil {
		log.Debug(fmt.Sprintf("failed to get lead artifact for package repository '%s', package name '%s', package version '%s', package type '%s'", b.PackageRepoName, b.PackageName, b.PackageVersion, packageType))
		return "", fmt.Errorf("failed to get lead artifact: %w", err)
	}

	leadArtifactPath := strings.Replace(string(leadArtifact), ":", "/", 1)
	return leadArtifactPath, nil
}

// GetPackageName returns the package name
func (b *basePackage) GetPackageName() string {
	return b.PackageName
}

// GetPackageVersion returns the package version
func (b *basePackage) GetPackageVersion() string {
	return b.PackageVersion
}

// GetPackageRepoName returns the package repository name
func (b *basePackage) GetPackageRepoName() string {
	return b.PackageRepoName
}
