package utils

import "fmt"

const DefaultProject = "default"

func BuildBuildInfoRepoKey(project string) string {
	if project == "" || project == DefaultProject {
		return "artifactory-build-info"
	}
	return fmt.Sprintf("%s-build-info", project)
}

func BuildReleaseBundleRepoKey(project string) string {
	if project == "" || project == DefaultProject {
		return "release-bundles-v2"
	}
	return fmt.Sprintf("%s-release-bundles-v2", project)
}

func BuildApplicationVersionRepoKey(project string) string {
	if project == "" || project == DefaultProject {
		return "application-versions"
	}
	return fmt.Sprintf("%s-application-versions", project)
}
