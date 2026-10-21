package utils

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestEscapeGraphQLString(t *testing.T) {
	testCases := []struct {
		value    string
		expected string
	}{
		{value: "my-repo", expected: "my-repo"},
		{value: `a"b`, expected: `a\\\"b`},
		{value: `a\b`, expected: `a\\\\b`},
		{value: `a\"b`, expected: `a\\\\\\\"b`},
		{value: "a&b<c>", expected: "a&b<c>"},
		{value: "line\nbreak", expected: `line\\nbreak`},
		{value: "a\x01b", expected: `a\\u0001b`},
		{value: "", expected: ""},
	}
	for _, tc := range testCases {
		assert.Equal(t, tc.expected, EscapeGraphQLString(tc.value))
	}
}

// TestEscapeGraphQLString_RoundTrip decodes the request the way the server does, first the JSON body
// and then the GraphQL string literal, and checks that the original value comes back.
func TestEscapeGraphQLString_RoundTrip(t *testing.T) {
	values := []string{
		"my-repo",
		`a"b`,
		`a\b`,
		`a\"b`,
		`x", path: "y`,
		`x\", path: \"y`,
		"line\nbreak\ttab\rreturn",
		"control\x00\x01\b\f\x1f",
		"unicode é 日本  ",
		"a&b<c>",
		"",
	}
	for _, value := range values {
		body := `{"query":"{ f(arg: \"` + EscapeGraphQLString(value) + `\") }"}`
		var request struct {
			Query string `json:"query"`
		}
		require.NoError(t, json.Unmarshal([]byte(body), &request), "request body for %q is not valid JSON", value)

		literal := strings.TrimSuffix(strings.TrimPrefix(request.Query, "{ f(arg: "), ") }")
		var decoded string
		require.NoError(t, json.Unmarshal([]byte(literal), &decoded), "GraphQL string literal %s for %q is not valid", literal, value)
		assert.Equal(t, value, decoded)
	}
}
