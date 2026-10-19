package utils

import (
	"bytes"
	"encoding/json"
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

// QuoteAqlString returns value as a quoted, escaped AQL string literal, so it can be interpolated into an AQL query.
func QuoteAqlString(value string) string {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	// AQL only needs JSON string escaping; keep '&', '<' and '>' as they are.
	encoder.SetEscapeHTML(false)
	// Encoding a string cannot fail.
	_ = encoder.Encode(value)
	return strings.TrimSuffix(buf.String(), "\n")
}

type AqlResult struct {
	Results []*servicesUtils.ResultItem `json:"results,omitempty"`
}
