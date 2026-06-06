// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tools

import (
	"encoding/json"
	"errors"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/tools/fformat"
)

var (
	errInputNotString  = errors.New("input must be a string")
	errFormatNotString = errors.New("format must be a string")
	errFromNotString   = errors.New("from must be a string")
	errToNotString     = errors.New("to must be a string")
)

func allFormats() []string {
	return []string{"json", "yaml", "csv", "xml", "toml", "markdown", "sql", "html"}
}

// jsonMarshalIndent is json.Marshal, overridable for testing.
//
//nolint:gochecknoglobals // test-replaceable
var jsonMarshal = json.Marshal

func marshalFFormatResult(result fformat.Result) (string, error) {
	out, err := jsonMarshal(result)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func requireStringArg(args map[string]interface{}, key string, typeErr error) (string, error) {
	val, ok := args[key].(string)
	if !ok {
		return "", typeErr
	}
	return val, nil
}

// RegisterFFormatTools registers format validation, formatting, and conversion tools.
func RegisterFFormatTools(r *Registry) {
	r.Register(fformatValidateTool())
	r.Register(fformatFormatTool())
	r.Register(fformatConvertToJSONTool())
	r.Register(fformatConvertFromJSONTool())
}

func fformatValidateTool() *Tool {
	return &Tool{
		Name:        "fformat_validate",
		Description: "Validate whether a string is valid for a given format (json, yaml, csv, xml, toml, markdown, sql, html).",
		Parameters: map[string]domain.ToolParam{
			"input": {
				Type:        "string",
				Description: "The string to validate.",
				Required:    true,
			},
			"format": {
				Type:        "string",
				Description: "The format to validate against.",
				Required:    true,
				Enum:        allFormats(),
			},
		},
		Execute: func(args map[string]interface{}) (string, error) {
			input, err := requireStringArg(args, "input", errInputNotString)
			if err != nil {
				return "", err
			}
			fmtStr, err := requireStringArg(args, "format", errFormatNotString)
			if err != nil {
				return "", err
			}
			return marshalFFormatResult(fformat.ValidateString(input, fformat.Format(fmtStr)))
		},
	}
}

func fformatFormatTool() *Tool {
	return &Tool{
		Name:        "fformat_format",
		Description: "Pretty-print / normalize a string in the given format (json, yaml, xml, toml, sql, html).",
		Parameters: map[string]domain.ToolParam{
			"input": {
				Type:        "string",
				Description: "The string to format.",
				Required:    true,
			},
			"format": {
				Type:        "string",
				Description: "The format to apply.",
				Required:    true,
				Enum:        allFormats(),
			},
		},
		Execute: func(args map[string]interface{}) (string, error) {
			input, err := requireStringArg(args, "input", errInputNotString)
			if err != nil {
				return "", err
			}
			fmtStr, err := requireStringArg(args, "format", errFormatNotString)
			if err != nil {
				return "", err
			}
			return marshalFFormatResult(fformat.FormatString(input, fformat.Format(fmtStr)))
		},
	}
}

func fformatConvertToJSONTool() *Tool {
	return &Tool{
		Name:        "fformat_convert_to_json",
		Description: "Convert a string from the given format to JSON (supports yaml, csv, xml, toml).",
		Parameters: map[string]domain.ToolParam{
			"input": {
				Type:        "string",
				Description: "The string to convert.",
				Required:    true,
			},
			"from": {
				Type:        "string",
				Description: "The source format.",
				Required:    true,
				Enum:        []string{"yaml", "csv", "xml", "toml"},
			},
		},
		Execute: func(args map[string]interface{}) (string, error) {
			input, err := requireStringArg(args, "input", errInputNotString)
			if err != nil {
				return "", err
			}
			fromStr, err := requireStringArg(args, "from", errFromNotString)
			if err != nil {
				return "", err
			}
			return marshalFFormatResult(fformat.ConvertToJSON(fformat.Format(fromStr), input))
		},
	}
}

func fformatConvertFromJSONTool() *Tool {
	return &Tool{
		Name:        "fformat_convert_from_json",
		Description: "Convert a JSON string to another format (supports yaml, csv).",
		Parameters: map[string]domain.ToolParam{
			"input": {
				Type:        "string",
				Description: "The JSON string to convert.",
				Required:    true,
			},
			"to": {
				Type:        "string",
				Description: "The target format.",
				Required:    true,
				Enum:        []string{"yaml", "csv"},
			},
		},
		Execute: func(args map[string]interface{}) (string, error) {
			input, err := requireStringArg(args, "input", errInputNotString)
			if err != nil {
				return "", err
			}
			toStr, err := requireStringArg(args, "to", errToNotString)
			if err != nil {
				return "", err
			}
			return marshalFFormatResult(fformat.ConvertFromJSON(fformat.Format(toStr), input))
		},
	}
}
