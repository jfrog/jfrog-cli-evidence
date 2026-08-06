package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsNonReleaseEvidenceVersion(t *testing.T) {
	tests := []struct {
		version string
		want    bool
	}{
		{version: "7.1285.0", want: false},
		{version: "7.1269.0", want: false},
		{version: "7.1269.1", want: false},
		{version: "7.1268.0", want: false},
		{version: "7.1269.0-1", want: true},
		{version: "development", want: true},
		{version: "Development", want: true},
		{version: "3.x-dev", want: true},
		{version: "7.x-SNAPSHOT", want: true},
		{version: "7.1269.0-SNAPSHOT", want: true},
		{version: "7.x-SNAPSHOT-master-1354", want: true},
		{version: "not-a-version", want: true},
		{version: "7.1269", want: true},
		{version: "7.1269.0.1", want: true},
		{version: "", want: true},
		{version: "  ", want: true},
	}
	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			assert.Equal(t, tt.want, IsNonReleaseEvidenceVersion(tt.version))
		})
	}
}
