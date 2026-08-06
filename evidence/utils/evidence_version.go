package utils

import (
	"strings"
)

// MinEvidenceVersionForEntityAPI is the first Evidence release that exposes /entity APIs.
const MinEvidenceVersionForEntityAPI = "7.1269.0"

// IsNonReleaseEvidenceVersion reports whether version is anything other than a pure
// major.minor.patch release string (e.g. "7.1285.0"). Production Evidence returns
// that form only; every other string (dev, SNAPSHOT, empty, suffixes) is treated as
// development and should not fail entity API checks.
func IsNonReleaseEvidenceVersion(version string) bool {
	parts := strings.Split(strings.TrimSpace(version), ".")
	if len(parts) != 3 {
		return true
	}
	for _, part := range parts {
		if part == "" || !isAllDigits(part) {
			return true
		}
	}
	return false
}

func isAllDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
