package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEscapeAqlValue(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"plain ascii", "test-repo", "test-repo"},
		{"path with slashes", "some/path/file.txt", "some/path/file.txt"},
		{"sha256 with colon", "sha256:1234567890abcdef", "sha256:1234567890abcdef"},
		{"empty", "", ""},
		{"double quote escaped", `bad"repo`, `bad\"repo`},
		{"backslash escaped", `bad\repo`, `bad\\repo`},
		{"both backslash and quote", `a"b\c`, `a\"b\\c`},
		{"injection payload", `x","extra":"y`, `x\",\"extra\":\"y`},
		{"newline dropped", "line1\nline2", "line1line2"},
		{"tab dropped", "a\tb", "ab"},
		{"carriage return dropped", "a\rb", "ab"},
		{"null byte dropped", "a\x00b", "ab"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, EscapeAqlValue(tc.input))
		})
	}
}
