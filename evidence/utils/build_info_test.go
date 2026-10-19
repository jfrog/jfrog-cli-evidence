package utils

import (
	"errors"
	"testing"

	buildinfo "github.com/jfrog/build-info-go/entities"
	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/artifactory/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockBuildInfoServicesManager struct {
	artifactory.EmptyArtifactoryServicesManager
	buildInfo *buildinfo.PublishedBuildInfo
	found     bool
	err       error
}

func (m *mockBuildInfoServicesManager) GetBuildInfo(services.BuildInfoParams) (*buildinfo.PublishedBuildInfo, bool, error) {
	return m.buildInfo, m.found, m.err
}

func TestGetBuildLatestTimestamp(t *testing.T) {
	client := &mockBuildInfoServicesManager{
		buildInfo: &buildinfo.PublishedBuildInfo{BuildInfo: buildinfo.BuildInfo{Started: "2023-11-14T22:13:20.123+0000"}},
		found:     true,
	}

	timestamp, err := GetBuildLatestTimestamp("build", "4", "", client)

	require.NoError(t, err)
	assert.Equal(t, "1700000000123", timestamp)
}

func TestGetBuildLatestTimestamp_NotFound(t *testing.T) {
	client := &mockBuildInfoServicesManager{found: false}

	_, err := GetBuildLatestTimestamp("build", "4", "proj", client)

	var notFound *BuildNotFoundError
	require.ErrorAs(t, err, &notFound)
	assert.Equal(t, &BuildNotFoundError{Name: "build", Number: "4", Project: "proj"}, notFound)
	assert.EqualError(t, err, "failed to find buildName, name:build, number:4, project: proj")
}

func TestGetBuildLatestTimestamp_Error(t *testing.T) {
	client := &mockBuildInfoServicesManager{err: errors.New("connection refused")}

	_, err := GetBuildLatestTimestamp("build", "4", "", client)

	assert.ErrorContains(t, err, "connection refused")
	var notFound *BuildNotFoundError
	assert.False(t, errors.As(err, &notFound))
}

func TestBuildInfoFileName(t *testing.T) {
	assert.Equal(t, "4-1700000000000.json", BuildInfoFileName("4", "1700000000000"))
	assert.Equal(t, "4-rc-1700000000000.json", BuildInfoFileName("4-rc", "1700000000000"))
}

func TestFormatAqlQuery(t *testing.T) {
	testCases := []struct {
		value    string
		expected string
	}{
		{value: "build-name", expected: `{"name":"build-name"}`},
		{value: `a"b`, expected: `{"name":"a\"b"}`},
		{value: `a\b`, expected: `{"name":"a\\b"}`},
		{value: "a&b<c>", expected: `{"name":"a&b<c>"}`},
		{value: "line\nbreak", expected: `{"name":"line\nbreak"}`},
		{value: "", expected: `{"name":""}`},
	}
	for _, tc := range testCases {
		query, err := FormatAqlQuery(`{"name":%s}`, tc.value)
		require.NoError(t, err)
		assert.Equal(t, tc.expected, query)
	}
}

func TestFormatAqlQuery_MultipleValues(t *testing.T) {
	query, err := FormatAqlQuery(`items.find({"repo":%s,"path":%s})`, "repo", `pa"th`)

	require.NoError(t, err)
	assert.Equal(t, `items.find({"repo":"repo","path":"pa\"th"})`, query)
}
