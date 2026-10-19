package utils

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsRevocationNotSupported(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "unknown where input field", err: errors.New(`Field "includeRevoked" is not defined by type "EvidenceWhereInput".`), want: true},
		{name: "unknown argument", err: errors.New(`Unknown argument "includeRevoked" on field "EvidenceQueries.getEvidence".`), want: true},
		{name: "unknown evidenceConnection where", err: errors.New(`Unknown argument "where" on field "ReleaseBundleVersion.evidenceConnection".`), want: true},
		{name: "unknown revoked field", err: errors.New(`Cannot query field "revoked" on type "Evidence".`), want: true},
		{name: "unknown attachments field only", err: errors.New(`Cannot query field "attachments" on type "Evidence".`), want: false},
		{name: "unknown where on another field", err: errors.New(`Unknown argument "where" on field "Query.other".`), want: false},
		{name: "transport error", err: errors.New("HTTP 503: Service Unavailable"), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsRevocationNotSupported(tt.err))
		})
	}
}
