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
		return "{}\n"
	}
	var builder strings.Builder
	builder.WriteString("{\n")
	for k, v := range *m {
		builder.WriteString(fmt.Sprintf("      [\"%s\"] = \"%s\"\n", k, EncodeValue(v)))
	}
	builder.WriteString("    }\n")
	return builder.String()
}

func EncodePklSlice(s *[]string) string {
	if s == nil {
		return "{}\n"
	}
	var builder strings.Builder
	builder.WriteString("{\n")
	for _, v := range *s {
		builder.WriteString(fmt.Sprintf("      \"%s\"\n", EncodeValue(v)))
	}
	builder.WriteString("    }\n")
	return builder.String()
}

// EvaluateString evaluates a string as PKL or JSON, returning a JSON string.
func EvaluateStringToJSON(input string, logger *logging.Logger, pklEvaluator pkl.Evaluator, ctx context.Context) (string, error) {
	// Try to parse as PKL first
	var pklResult interface{}
	err := pklEvaluator.EvaluateExpression(ctx, pkl.TextSource(input), "", &pklResult)
	if err == nil {
		// Successfully parsed as PKL, convert to JSON
		jsonBytes, err := json.MarshalIndent(pklResult, "", "  ")
		if err == nil {
			logger.Info("parsed PKL and converted to JSON")
			return string(jsonBytes), nil
		}
		logger.Error(err, "failed to convert PKL to JSON, using raw input")
		return input, nil // Fallback to raw input
	}
	logger.Info("input is not a valid PKL expression, trying JSON", "error", err)

	// Fallback to JSON processing
	fixedJSON := FixJSON(input)
	if IsJSON(fixedJSON) {
		var prettyJSON bytes.Buffer
		err := json.Indent(&prettyJSON, []byte(fixedJSON), "", "  ")
		if err == nil {
			logger.Info("processed as JSON and pretty-printed")
			return prettyJSON.String(), nil
		}
		logger.Error(err, "failed to pretty-print JSON")
		return fixedJSON, nil
	}

	logger.Error(nil, "input is neither valid PKL nor JSON, returning raw input")
	return input, nil
}
