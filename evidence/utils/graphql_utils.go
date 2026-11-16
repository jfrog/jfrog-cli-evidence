package utils

import (
	"fmt"
	"strings"
)

func IsAttachmentsFieldNotFound(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "Cannot query field") && strings.Contains(errStr, "attachments")
}

// EscapeGraphQLString escapes value for use between the quotes of a GraphQL string argument
// (\"%s\") in a query that is sent as a JSON string ({"query":"..."}).
// The value is escaped twice: first for the GraphQL string, then for the JSON string that carries the query.
func EscapeGraphQLString(value string) string {
	return escapeJSONStringContent(escapeJSONStringContent(value))
}

// escapeJSONStringContent returns value escaped as the contents of a JSON string, without the quotes.
// GraphQL strings use the same escape sequences as JSON strings.
func escapeJSONStringContent(value string) string {
	var escaped strings.Builder
	for _, r := range value {
		switch {
		case r == '"' || r == '\\':
			escaped.WriteByte('\\')
			escaped.WriteRune(r)
		case r == '\n':
			escaped.WriteString(`\n`)
		case r == '\r':
			escaped.WriteString(`\r`)
		case r == '\t':
			escaped.WriteString(`\t`)
		case r < 0x20:
			_, _ = fmt.Fprintf(&escaped, `\u%04x`, r)
		default:
			escaped.WriteRune(r)
		}
	}
	return escaped.String()
}
