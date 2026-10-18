package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jfrog/jfrog-client-go/onemodel"
)

const evidenceTypeFieldsIntrospectionQuery = `{"query":"{ __type(name: \"Evidence\") { fields { name } } }"}`

var ErrEvidenceRevocationNotSupported = errors.New(
	"--include-revoked requires an Evidence service version that supports revocation " +
		"(the GraphQL Evidence type has no 'revoked' field)")

func IsAttachmentsFieldNotFound(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "Cannot query field") && strings.Contains(errStr, "attachments")
}

// EnsureEvidenceRevocationSupported introspects the One-Model schema and returns
// ErrEvidenceRevocationNotSupported when the Evidence type has no revoked field.
func EnsureEvidenceRevocationSupported(client onemodel.Manager) error {
	response, err := client.GraphqlQuery([]byte(evidenceTypeFieldsIntrospectionQuery))
	if err != nil {
		return fmt.Errorf("failed to detect Evidence revocation support: %w", err)
	}
	var introspection struct {
		Data struct {
			Type *struct {
				Fields []struct {
					Name string `json:"name"`
				} `json:"fields"`
			} `json:"__type"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(response, &introspection); err != nil {
		return fmt.Errorf("failed to detect Evidence revocation support: %w", err)
	}
	if introspection.Data.Type == nil {
		if len(introspection.Errors) > 0 {
			return fmt.Errorf("failed to detect Evidence revocation support: %s", introspection.Errors[0].Message)
		}
		return ErrEvidenceRevocationNotSupported
	}
	for _, field := range introspection.Data.Type.Fields {
		if field.Name == FieldRevoked {
			return nil
		}
	}
	return ErrEvidenceRevocationNotSupported
}
