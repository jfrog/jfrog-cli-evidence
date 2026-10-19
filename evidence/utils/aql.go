package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	gofrogio "github.com/jfrog/gofrog/io"
	"github.com/jfrog/jfrog-client-go/artifactory"
	servicesUtils "github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	"github.com/jfrog/jfrog-client-go/utils/log"
	"io"
	"strings"
)

// ExecuteAqlQuery executes an AQL query and returns the result.
func ExecuteAqlQuery(query string, client *artifactory.ArtifactoryServicesManager) (*AqlResult, error) {
	log.Debug("Getting artifactory sha256 using AQL query:", query)
	stream, err := (*client).Aql(query)
	if err != nil {
		return nil, err
	}
	defer gofrogio.Close(stream, &err)
	result, err := io.ReadAll(stream)
	if err != nil {
		return nil, err
	}
	parsedResult := new(AqlResult)
	if err = json.Unmarshal(result, parsedResult); err != nil {
		return nil, err
	}
	return parsedResult, nil
}

// FormatAqlQuery replaces each %s in template with the matching value, quoted and escaped as an AQL string literal.
func FormatAqlQuery(template string, values ...string) (string, error) {
	literals := make([]any, len(values))
	for i, value := range values {
		literal, err := quoteAqlString(value)
		if err != nil {
			return "", err
		}
		literals[i] = literal
	}
	return fmt.Sprintf(template, literals...), nil
}

func quoteAqlString(value string) (string, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	// AQL only needs JSON string escaping; keep '&', '<' and '>' as they are.
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return "", err
	}
	return strings.TrimSuffix(buf.String(), "\n"), nil
}

type AqlResult struct {
	Results []*servicesUtils.ResultItem `json:"results,omitempty"`
}
