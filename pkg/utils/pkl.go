package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/pkg/logging"
)

func EncodePklMap(m *map[string]string) string {
	if m == nil {
		return "{}"
	}
	var builder strings.Builder
	builder.WriteString("{")
	first := true
	for k, v := range *m {
		if !first {
			builder.WriteString(";")
		}
		builder.WriteString(fmt.Sprintf(`["%s"]="%s"`, k, EncodeValue(v)))
		first = false
	}
	builder.WriteString("}")
	return builder.String()
}

func EncodePklSlice(s *[]string) string {
	if s == nil {
		return "{}"
	}
	var builder strings.Builder
	builder.WriteString("{")
	first := true
	for _, v := range *s {
		if !first {
			builder.WriteString(";")
		}
		builder.WriteString(fmt.Sprintf(`"%s"`, EncodeValue(v)))
		first = false
	}
	builder.WriteString("}")
	return builder.String()
}

// EvaluateStringToJSON processes a string as JSON, returning a formatted JSON string.
// PKL evaluation has been removed for simplicity.
func EvaluateStringToJSON(input string, logger *logging.Logger, pklEvaluator pkl.Evaluator, ctx context.Context) (string, error) {
	// Process as JSON without complex PKL re-parsing
	fixedJSON := FixJSON(input)
	if IsJSON(fixedJSON) {
		// Use compact JSON instead of pretty-printed
		var compactJSON bytes.Buffer
		err := json.Compact(&compactJSON, []byte(fixedJSON))
		if err == nil {
			logger.Debug("processed as JSON and compacted")
			return compactJSON.String(), nil
		}
		logger.Warn("failed to compact JSON", "error", err)
		return fixedJSON, nil
	}

	logger.Debug("input is not valid JSON, returning raw input")
	return input, nil
}
