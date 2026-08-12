package utils

import (
	"strings"

	clientutils "github.com/jfrog/jfrog-client-go/utils"
)

// EvidenceFeature identifies an Evidence service capability gated by a minimum version.
type EvidenceFeature string

const (
	// FeatureEntityAPI is the Evidence /entity create, get, and verify APIs.
	FeatureEntityAPI EvidenceFeature = "entityAPI"
)

// featureMinVersions maps each Evidence feature to the first release that supports it.
var featureMinVersions = map[EvidenceFeature]string{
	FeatureEntityAPI: "7.1269.0",
}

// MinVersionForFeature returns the minimum Evidence release that supports feature,
// or an empty string when the feature is unknown.
func MinVersionForFeature(feature EvidenceFeature) string {
	return featureMinVersions[feature]
}

// IsFeatureSupported reports whether the given Evidence service version supports feature.
// Non-release / development version strings are treated as supported so local and
// snapshot builds are not blocked. Unknown features are not supported.
func IsFeatureSupported(feature EvidenceFeature, version string) bool {
	minVersion := MinVersionForFeature(feature)
	if minVersion == "" {
		return false
	}
	if IsNonReleaseEvidenceVersion(version) {
		return true
	}
	return clientutils.ValidateMinimumVersion("JFrog Evidence", version, minVersion) == nil
}

// IsNonReleaseEvidenceVersion reports whether version is anything other than a pure
// major.minor.patch release string (e.g. "7.1285.0"). Production Evidence returns
// that form only; every other string (dev, SNAPSHOT, empty, suffixes) is treated as
// development and should not fail feature checks.
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
