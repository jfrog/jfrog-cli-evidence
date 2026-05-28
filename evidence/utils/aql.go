package utils

import (
	"encoding/json"
	"io"
	"strings"

	gofrogio "github.com/jfrog/gofrog/io"
	"github.com/jfrog/jfrog-client-go/artifactory"
	servicesUtils "github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	"github.com/jfrog/jfrog-client-go/utils/log"
)

// EscapeAqlValue escapes a string so it can be safely interpolated inside an
// AQL "..." string literal.
func EscapeAqlValue(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\\' || c == '"':
			b.WriteByte('\\')
			b.WriteByte(c)
		case c < 0x20:
			continue
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

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

type AqlResult struct {
	Results []*servicesUtils.ResultItem `json:"results,omitempty"`
}
