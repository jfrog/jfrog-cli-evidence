package utils

import (
	"fmt"

	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/artifactory/services"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"github.com/jfrog/jfrog-client-go/utils/log"
)

// BuildNotFoundError is returned when Artifactory has no build-info for the requested build name and number.
type BuildNotFoundError struct {
	Name    string
	Number  string
	Project string
}

func (e *BuildNotFoundError) Error() string {
	return fmt.Sprintf("failed to find buildName, name:%s, number:%s, project: %s", e.Name, e.Number, e.Project)
}

// GetBuildLatestTimestamp returns the start time, in Unix milliseconds, of the latest run of the given build.
func GetBuildLatestTimestamp(name string, number string, project string, artifactoryClient artifactory.ArtifactoryServicesManager) (string, error) {
	buildInfo := services.BuildInfoParams{
		BuildName:   name,
		BuildNumber: number,
		ProjectKey:  project,
	}
	log.Debug("Getting build info for buildName:", name, "buildNumber:", number, "project:", project)
	res, ok, err := artifactoryClient.GetBuildInfo(buildInfo)
	if err != nil {
		return "", fmt.Errorf("failed to get build info for buildName: %s, buildNumber: %s, project: %s, error: %w", name, number, project, err)
	}
	if !ok {
		return "", errorutils.CheckError(&BuildNotFoundError{Name: name, Number: number, Project: project})
	}
	timestamp, err := ParseIsoTimestamp(res.BuildInfo.Started)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d", timestamp.UnixMilli()), nil
}

// BuildInfoFileName returns the name of a build run's build-info file in the build-info repository.
func BuildInfoFileName(number string, timestamp string) string {
	return fmt.Sprintf("%s-%s.json", number, timestamp)
}
