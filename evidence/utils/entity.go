package utils

import (
	"fmt"
	"strings"

	"github.com/jfrog/jfrog-cli-evidence/evidence/model"
)

// BuildGraphQLEntityHasSubjectWith builds the GraphQL hasSubjectWith fields for an entity subject.
// Exactly one of EntityRepo, ProjectKey, and ApplicationKey may be set (or none).
func BuildGraphQLEntityHasSubjectWith(subject model.EntitySubject) string {
	fields := []string{fmt.Sprintf(`entityType: \"%s\"`, escapeGraphQLString(subject.EntityType))}
	if subject.EntityID != "" {
		fields = append(fields, fmt.Sprintf(`entityId: \"%s\"`, escapeGraphQLString(subject.EntityID)))
	}
	switch {
	case subject.EntityRepo != "":
		fields = append(fields, fmt.Sprintf(`repositoryKey: \"%s\"`, escapeGraphQLString(subject.EntityRepo)))
	case subject.ProjectKey != "":
		fields = append(fields, fmt.Sprintf(`projectKey: \"%s\"`, escapeGraphQLString(subject.ProjectKey)))
	case subject.ApplicationKey != "":
		fields = append(fields, fmt.Sprintf(`applicationKey: \"%s\"`, escapeGraphQLString(subject.ApplicationKey)))
	}
	return strings.Join(fields, ", ")
}

func escapeGraphQLString(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	return replacer.Replace(value)
}
