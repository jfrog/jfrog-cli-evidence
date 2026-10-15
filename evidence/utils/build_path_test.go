package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodeBuildInfoPathSegment(t *testing.T) {
	tests := []struct {
		name     string
		segment  string
		expected string
	}{
		{name: "plain", segment: "SimpleBuild", expected: "SimpleBuild"},
		{name: "spaces and symbols artifactory keeps", segment: "Build & Test + Deploy", expected: "Build & Test + Deploy"},
		{name: "slash", segment: "abc/abc", expected: "abc%2Fabc"},
		{name: "colon", segment: "colon:name", expected: "colon%3Aname"},
		{name: "slash and colon", segment: "mix a/b:c", expected: "mix a%2Fb%3Ac"},
		{name: "percent", segment: "Build%20Name", expected: "Build%2520Name"},
		{name: "build number slash", segment: "1/2", expected: "1%2F2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, EncodeBuildInfoPathSegment(tt.segment))
		})
	}
}
