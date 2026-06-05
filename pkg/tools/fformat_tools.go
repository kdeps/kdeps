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
			input, ok := args["input"].(string)
			if !ok {
				return "", errInputNotString
			}
			fmtStr, ok := args["format"].(string)
			if !ok {
				return "", errFormatNotString
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
			input, ok := args["input"].(string)
			if !ok {
				return "", errInputNotString
			}
			fmtStr, ok := args["format"].(string)
			if !ok {
				return "", errFormatNotString
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
			input, ok := args["input"].(string)
			if !ok {
				return "", errInputNotString
			}
			fromStr, ok := args["from"].(string)
			if !ok {
				return "", errFromNotString
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
			input, ok := args["input"].(string)
			if !ok {
				return "", errInputNotString
			}
			toStr, ok := args["to"].(string)
			if !ok {
				return "", errToNotString
			}
			return marshalFFormatResult(fformat.ConvertFromJSON(fformat.Format(toStr), input))
		},
	}
}
