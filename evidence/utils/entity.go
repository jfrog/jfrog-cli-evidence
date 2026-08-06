package utils

import (
	"fmt"
	"strings"
)

// EmptySubjectSha256 is the SHA-256 of an empty payload used as the placeholder
// subject checksum for entity evidence subjects.
const EmptySubjectSha256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

// BuildGraphQLEntityHasSubjectWith builds the GraphQL hasSubjectWith fields for an entity subject.
// Exactly one of entityRepo, projectKey, and applicationKey may be set (or none).
func BuildGraphQLEntityHasSubjectWith(entityType, entityID, entityRepo, projectKey, applicationKey string) string {
	fields := []string{fmt.Sprintf(`entityType: \"%s\"`, escapeGraphQLString(entityType))}
	if entityID != "" {
		fields = append(fields, fmt.Sprintf(`entityId: \"%s\"`, escapeGraphQLString(entityID)))
	}
	switch {
	case entityRepo != "":
		fields = append(fields, fmt.Sprintf(`repositoryKey: \"%s\"`, escapeGraphQLString(entityRepo)))
	case projectKey != "":
		fields = append(fields, fmt.Sprintf(`projectKey: \"%s\"`, escapeGraphQLString(projectKey)))
	case applicationKey != "":
		fields = append(fields, fmt.Sprintf(`applicationKey: \"%s\"`, escapeGraphQLString(applicationKey)))
	}
	return strings.Join(fields, ", ")
}

func escapeGraphQLString(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	return replacer.Replace(value)
}
