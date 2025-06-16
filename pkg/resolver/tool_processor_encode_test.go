package resolver

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/utils"
	pklLLM "github.com/kdeps/schema/gen/llm"
	"github.com/stretchr/testify/assert"
)

func TestEncodeToolsAndParamsUnit(t *testing.T) {
	logger := logging.NewTestLogger()

	name := "mytool"
	script := "echo hi"
	desc := "sample tool"
	req := true
	ptype := "string"

	params := map[string]*pklLLM.ToolProperties{
		"arg1": {
			Required:    &req,
			Type:        &ptype,
			Description: &desc,
		},
	}

	tool := &pklLLM.Tool{
		Name:        &name,
		Script:      &script,
		Description: &desc,
		Parameters:  &params,
	}
	tools := []*pklLLM.Tool{tool}

	encoded := encodeTools(&tools)
	assert.Len(t, encoded, 1)
	// ensure values are encoded (base64) via utils.EncodeValue helper
	assert.Equal(t, utils.EncodeValue(name), *encoded[0].Name)
	assert.Equal(t, utils.EncodeValue(script), *encoded[0].Script)

	// verify encodeToolParameters encodes nested map
	encodedParams := encodeToolParameters(&params)
	assert.NotNil(t, encodedParams)
	assert.Contains(t, *encodedParams, "arg1")
	encType := *(*encodedParams)["arg1"].Type
	assert.Equal(t, utils.EncodeValue(ptype), encType)

	// convertToolParamsToString with various types
	logger.Debug("testing convertToolParamsToString")
	assert.Equal(t, "hello", convertToolParamsToString("hello", "p", "t", logger))
	assert.Equal(t, "3.5", convertToolParamsToString(3.5, "p", "t", logger))
	assert.Equal(t, "true", convertToolParamsToString(true, "p", "t", logger))

	obj := map[string]int{"x": 1}
	str := convertToolParamsToString(obj, "p", "t", logger)
	var recovered map[string]int
	assert.NoError(t, json.Unmarshal([]byte(str), &recovered))
	assert.Equal(t, obj["x"], recovered["x"])

	var sb strings.Builder
	serializeTools(&sb, &tools)
	serialized := sb.String()
	assert.Contains(t, serialized, "name = \"mytool\"")
}
