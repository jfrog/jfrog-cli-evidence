package utils

import "strings"

// Same logic as encodeForBuildInfoRepository() in jfrog-client-go (that is unexported)
func EncodeBuildInfoPathSegment(segment string) string {
	return strings.NewReplacer(
		"%", "%25",
		"\"", "%22",
		"/", "%2F",
		":", "%3A",
		";", "%3B",
		"?", "%3F",
		"*", "%2A",
		"\\", "%5C",
		"|", "%7C",
	).Replace(segment)
}
