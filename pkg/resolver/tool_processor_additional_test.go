package resolver

import (
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	pklLLM "github.com/kdeps/schema/gen/llm"
	"github.com/stretchr/testify/assert"
)

// helper to construct pointer of string
func strPtr(s string) *string { return &s }

func TestGenerateAvailableToolsAndRelatedHelpers(t *testing.T) {
	logger := logging.NewTestLogger()

	// Build a ResourceChat with two tools – one duplicate to hit duplicate filtering.
	desc := "echo something"
	req := true
	// Parameters definition
	params := map[string]*pklLLM.ToolProperties{
		"msg": {
			Required:    &req,
			Type:        strPtr("string"),
			Description: strPtr("message to echo"),
		},
	}

	tool1 := &pklLLM.Tool{
		Name:        strPtr("echo"),
		Description: &desc,
		Script:      strPtr("echo $msg"),
		Parameters:  &params,
	}
	// Duplicate with same name (should be skipped by generator)
	toolDup := &pklLLM.Tool{
		Name:       strPtr("echo"),
		Script:     strPtr("echo dup"),
		Parameters: &params,
	}
	tool2 := &pklLLM.Tool{
		Name:        strPtr("sum"),
		Description: strPtr("add numbers"),
		Script:      strPtr("expr $a + $b"),
	}

	toolsSlice := []*pklLLM.Tool{tool1, toolDup, tool2}
	chat := &pklLLM.ResourceChat{Tools: &toolsSlice}

	available := generateAvailableTools(chat, logger)
	assert.Len(t, available, 2, "duplicate tool should have been filtered out")
	// ensure function metadata is copied.
	names := []string{available[0].Function.Name, available[1].Function.Name}
	assert.ElementsMatch(t, []string{"echo", "sum"}, names)

	// exercise formatToolParameters using first available tool
	var sb strings.Builder
	formatToolParameters(available[0], &sb)
	formatted := sb.String()
	assert.Contains(t, formatted, "msg", "expected parameter name in formatted output")
}

func TestBuildToolURIAndExtractParams(t *testing.T) {
	logger := logging.NewTestLogger()

	// Build chatBlock for extractToolParams
	req := true
	script := "echo $msg"
	toolProps := map[string]*pklLLM.ToolProperties{
		"msg": {Required: &req, Type: strPtr("string"), Description: strPtr("m")},
	}
	toolEntry := &pklLLM.Tool{
		Name:       strPtr("echo"),
		Script:     &script,
		Parameters: &toolProps,
	}
	tools := []*pklLLM.Tool{toolEntry}
	chat := &pklLLM.ResourceChat{Tools: &tools}

	// Arguments map simulating parsed JSON args
	args := map[string]interface{}{"msg": "hello"}

	name, gotScript, paramsStr, err := extractToolParams(args, chat, "echo", logger)
	assert.NoError(t, err)
	assert.Equal(t, "echo", name)
	assert.Equal(t, script, gotScript)
	assert.Equal(t, "hello", paramsStr)

	// Build the tool URI
	uri, err := buildToolURI("id123", gotScript, paramsStr)
	assert.NoError(t, err)
	// Should encode params as query param
	assert.Contains(t, uri.String(), "params=")

	// We omit executing through tool reader to keep the test lightweight.
}
