package utils

import (
	"errors"
	"strings"
)

var ErrEvidenceRevocationNotSupported = errors.New(
	"--include-revoked requires an Evidence service version that supports revocation " +
		"(the GraphQL schema has no 'includeRevoked' argument or 'revoked' field)")

func IsAttachmentsFieldNotFound(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "Cannot query field") && strings.Contains(errStr, "attachments")
}

// IsRevocationNotSupported reports whether a GraphQL validation error comes from a schema that
// predates evidence revocation. Introspection is disabled in production, so the query itself is the probe.
func IsRevocationNotSupported(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	if strings.Contains(errStr, "includeRevoked") {
		return true
	}
	if strings.Contains(errStr, "Unknown argument") && strings.Contains(errStr, "where") &&
		strings.Contains(errStr, "evidenceConnection") {
		return true
	}
	return strings.Contains(errStr, "Cannot query field") && strings.Contains(errStr, "revoked")
}
