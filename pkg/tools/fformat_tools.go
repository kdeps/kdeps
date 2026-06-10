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

// fformatToolSpec describes one fformat tool: a fixed "input" string param
// plus one format-selector param whose value is handed to run.
type fformatToolSpec struct {
	name      string
	desc      string
	inputDesc string
	argName   string
	argDesc   string
	enum      []string
	argErr    error
	run       func(input string, format fformat.Format) fformat.Result
}

func newFFormatTool(spec fformatToolSpec) *Tool {
	return &Tool{
		Name:        spec.name,
		Description: spec.desc,
		Parameters: map[string]domain.ToolParam{
			"input": {
				Type:        "string",
				Description: spec.inputDesc,
				Required:    true,
			},
			spec.argName: {
				Type:        "string",
				Description: spec.argDesc,
				Required:    true,
				Enum:        spec.enum,
			},
		},
		Execute: func(args map[string]interface{}) (string, error) {
			input, err := requireStringArg(args, "input", errInputNotString)
			if err != nil {
				return "", err
			}
			val, err := requireStringArg(args, spec.argName, spec.argErr)
			if err != nil {
				return "", err
			}
			return marshalFFormatResult(spec.run(input, fformat.Format(val)))
		},
	}
}

func fformatValidateTool() *Tool {
	return newFFormatTool(fformatToolSpec{
		name:      "fformat_validate",
		desc:      "Validate whether a string is valid for a given format (json, yaml, csv, xml, toml, markdown, sql, html).",
		inputDesc: "The string to validate.",
		argName:   "format",
		argDesc:   "The format to validate against.",
		enum:      allFormats(),
		argErr:    errFormatNotString,
		run:       fformat.ValidateString,
	})
}

func fformatFormatTool() *Tool {
	return newFFormatTool(fformatToolSpec{
		name:      "fformat_format",
		desc:      "Pretty-print / normalize a string in the given format (json, yaml, xml, toml, sql, html).",
		inputDesc: "The string to format.",
		argName:   "format",
		argDesc:   "The format to apply.",
		enum:      allFormats(),
		argErr:    errFormatNotString,
		run:       fformat.FormatString,
	})
}

func fformatConvertToJSONTool() *Tool {
	return newFFormatTool(fformatToolSpec{
		name:      "fformat_convert_to_json",
		desc:      "Convert a string from the given format to JSON (supports yaml, csv, xml, toml).",
		inputDesc: "The string to convert.",
		argName:   "from",
		argDesc:   "The source format.",
		enum:      []string{"yaml", "csv", "xml", "toml"},
		argErr:    errFromNotString,
		run:       func(input string, f fformat.Format) fformat.Result { return fformat.ConvertToJSON(f, input) },
	})
}

func fformatConvertFromJSONTool() *Tool {
	return newFFormatTool(fformatToolSpec{
		name:      "fformat_convert_from_json",
		desc:      "Convert a JSON string to another format (supports yaml, csv).",
		inputDesc: "The JSON string to convert.",
		argName:   "to",
		argDesc:   "The target format.",
		enum:      []string{"yaml", "csv"},
		argErr:    errToNotString,
		run:       func(input string, f fformat.Format) fformat.Result { return fformat.ConvertFromJSON(f, input) },
	})
}
