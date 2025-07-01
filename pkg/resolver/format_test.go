package resolver

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/apple/pkl-go/pkl"
	"github.com/google/uuid"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/kdeps/pkg/utils"
	apiserverresponse "github.com/kdeps/schema/gen/api_server_response"
	"github.com/kdeps/schema/gen/exec"
	pklHTTP "github.com/kdeps/schema/gen/http"
	pklLLM "github.com/kdeps/schema/gen/llm"
	"github.com/kdeps/schema/gen/python"
	pklPython "github.com/kdeps/schema/gen/python"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
)

/*
Filesystem Strategy for Tests:

This file establishes a standardized approach for choosing filesystem types in tests:

1. **Use Real Filesystem (setupTestResolverWithRealFS)** when:
   - Tests involve PKL operations (evaluation, loading, processing)
   - Tests call functions that use evaluator.EvalPkl()
   - Tests work with PKL schema loading (e.g., LoadFromPath, LoadResource)
   - Tests append entries to PKL files (AppendExecEntry, AppendPythonEntry, AppendChatEntry)
   - Tests create and process PKL files (CreateAndProcessPklFile)
   - Tests evaluate PKL formatted response files

   **Why**: PKL requires real file paths on disk to load modules and resolve imports.
   afero.NewMemMapFs() creates virtual files that PKL cannot access.

2. **Use In-Memory Filesystem (setupTestResolverWithMemFS)** when:
   - Tests only work with file I/O (reading/writing without PKL)
   - Tests involve simple string manipulation or formatting
   - Tests work with non-PKL configuration or data files
   - Tests don't involve external PKL binary execution

   **Why**: In-memory filesystem is faster and doesn't require cleanup.

3. **Helper Functions**:
   - setupTestResolverWithRealFS(t) - Creates resolver with afero.NewOsFs() + t.TempDir()
   - setupTestResolverWithMemFS(t) - Creates resolver with afero.NewMemMapFs()

4. **Examples**:
   ✅ Real FS: TestAppendExecEntry, TestEvalPklFormattedResponseFile, TestHandlePython
   ✅ Memory FS: TestWriteResponseToFile_EncodedAndPlain, TestFormatValue tests

This pattern ensures PKL tests work correctly while maintaining performance for non-PKL tests.
*/

// setupTestResolverWithRealFS creates a DependencyResolver with real filesystem
// using temporary directories. This is needed for PKL-related tests since PKL
// cannot work with afero's in-memory filesystem.
func setupTestResolverWithRealFS(t *testing.T) *DependencyResolver {
	tmpDir := t.TempDir()

	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	filesDir := filepath.Join(tmpDir, "files")
	actionDir := filepath.Join(tmpDir, "action")
	_ = fs.MkdirAll(filepath.Join(actionDir, "exec"), 0o755)
	_ = fs.MkdirAll(filepath.Join(actionDir, "python"), 0o755)
	_ = fs.MkdirAll(filepath.Join(actionDir, "llm"), 0o755)
	_ = fs.MkdirAll(filesDir, 0o755)

	return &DependencyResolver{
		Fs:        fs,
		Logger:    logger,
		Context:   ctx,
		FilesDir:  filesDir,
		ActionDir: actionDir,
		RequestID: "test-request",
	}
}

// setupTestResolverWithMemFS creates a DependencyResolver with in-memory filesystem
// for tests that don't need PKL functionality.
func setupTestResolverWithMemFS(t *testing.T) *DependencyResolver {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	filesDir := "/files"
	actionDir := "/action"
	_ = fs.MkdirAll(filepath.Join(actionDir, "exec"), 0o755)
	_ = fs.MkdirAll(filepath.Join(actionDir, "python"), 0o755)
	_ = fs.MkdirAll(filepath.Join(actionDir, "llm"), 0o755)
	_ = fs.MkdirAll(filesDir, 0o755)

	return &DependencyResolver{
		Fs:        fs,
		Logger:    logger,
		Context:   ctx,
		FilesDir:  filesDir,
		ActionDir: actionDir,
		RequestID: "test-request",
	}
}

func TestFormatMapSimple(t *testing.T) {
	m := map[interface{}]interface{}{
		"foo": "bar",
		1:     2,
	}
	out := formatMap(m)
	if !containsAll(out, []string{"new Mapping {", "[\"foo\"]", "[\"1\"] ="}) {
		t.Errorf("unexpected mapping output: %s", out)
	}
}

// Helper to check substring presence
func containsAll(s string, subs []string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}

func TestFormatValueVariants(t *testing.T) {
	// Case 1: nil interface -> "null"
	var v interface{} = nil
	if out := formatValue(v); out != "null" {
		t.Errorf("expected 'null' for nil, got %s", out)
	}

	// Case 2: map[string]interface{}
	mp := map[string]interface{}{"k": "v"}
	mv := formatValue(mp)
	if !strings.Contains(mv, "new Mapping {") || !strings.Contains(mv, "[\"k\"]") {
		t.Errorf("unexpected map formatting: %s", mv)
	}

	// Case 3: pointer to struct -> should format struct fields via Mapping
	type sample struct{ Field string }
	s := &sample{Field: "data"}
	sv := formatValue(s)
	if !strings.Contains(sv, "Field") || !strings.Contains(sv, "data") {
		t.Errorf("struct pointer formatting missing content: %s", sv)
	}

	// Case 4: direct struct (non-pointer)
	sp := sample{Field: "x"}
	st := formatValue(sp)
	if !strings.Contains(st, "Field") {
		t.Errorf("struct formatting unexpected: %s", st)
	}

	// Ensure default path returns triple-quoted string for primitive
	prim := formatValue("plain")
	if !strings.Contains(prim, "\"\"\"") {
		t.Errorf("primitive formatting not triple-quoted: %s", prim)
	}

	// Sanity: reflect-based call shouldn't panic for pointer nil
	var nilPtr *sample
	_ = formatValue(nilPtr)
	// the return is acceptable, we just ensure no panic
	_ = reflect.TypeOf(nilPtr)
}

func TestGeneratePklContent_Minimal(t *testing.T) {
	ctx := context.Background()
	logger := logging.NewTestLogger()

	prompt := "Hello"
	role := RoleHuman
	jsonResp := true
	res := &pklLLM.ResourceChat{
		Model:           "llama2",
		Prompt:          &prompt,
		Role:            &role,
		JSONResponse:    &jsonResp,
		TimeoutDuration: &pkl.Duration{Value: 5, Unit: pkl.Second},
	}
	m := map[string]*pklLLM.ResourceChat{"id1": res}

	pklStr := generatePklContent(m, ctx, logger)

	// Basic sanity checks
	if !strings.Contains(pklStr, "resources {") || !strings.Contains(pklStr, "\"id1\"") {
		t.Errorf("generated PKL missing expected identifiers: %s", pklStr)
	}
	if !strings.Contains(pklStr, "model = \"llama2\"") {
		t.Errorf("model field not serialized correctly: %s", pklStr)
	}
	if !strings.Contains(pklStr, "prompt = \"Hello\"") {
		t.Errorf("prompt field not serialized correctly: %s", pklStr)
	}
	if !strings.Contains(pklStr, "JSONResponse = true") {
		t.Errorf("JSONResponse field not serialized: %s", pklStr)
	}
}

func TestWriteResponseToFile_EncodedAndPlain(t *testing.T) {
	dr := setupTestResolverWithMemFS(t)
	dr.RequestID = "req123"

	resp := "this is the content"
	encoded := base64.StdEncoding.EncodeToString([]byte(resp))

	// Base64 encoded input
	path, err := dr.WriteResponseToFile("resID", &encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := afero.ReadFile(dr.Fs, path)
	if string(data) != resp {
		t.Errorf("decoded content mismatch: got %q, want %q", string(data), resp)
	}

	// Plain text input
	path2, err := dr.WriteResponseToFile("resID2", &resp)
	if err != nil {
		t.Fatalf("unexpected error (plain): %v", err)
	}
	data2, _ := afero.ReadFile(dr.Fs, path2)
	if string(data2) != resp {
		t.Errorf("plain content mismatch: got %q, want %q", string(data2), resp)
	}
}

func TestSummarizeMessageHistory(t *testing.T) {
	tests := []struct {
		name     string
		history  []llms.MessageContent
		expected string
	}{
		{
			name:     "empty history",
			history:  []llms.MessageContent{},
			expected: "",
		},
		{
			name: "single message",
			history: []llms.MessageContent{
				{
					Role:  llms.ChatMessageTypeHuman,
					Parts: []llms.ContentPart{llms.TextContent{Text: "Hello"}},
				},
			},
			expected: "Role:human Parts:Hello",
		},
		{
			name: "multiple messages",
			history: []llms.MessageContent{
				{
					Role:  llms.ChatMessageTypeHuman,
					Parts: []llms.ContentPart{llms.TextContent{Text: "Hello"}},
				},
				{
					Role:  llms.ChatMessageTypeAI,
					Parts: []llms.ContentPart{llms.TextContent{Text: "Hi there"}},
				},
			},
			expected: "Role:human Parts:Hello; Role:ai Parts:Hi there",
		},
		{
			name: "message with multiple parts",
			history: []llms.MessageContent{
				{
					Role: llms.ChatMessageTypeHuman,
					Parts: []llms.ContentPart{
						llms.TextContent{Text: "Part 1"},
						llms.TextContent{Text: "Part 2"},
					},
				},
			},
			expected: "Role:human Parts:Part 1|Part 2",
		},
		{
			name: "long message truncated",
			history: []llms.MessageContent{
				{
					Role:  llms.ChatMessageTypeHuman,
					Parts: []llms.ContentPart{llms.TextContent{Text: "This is a very long message that should be truncated to 50 characters"}},
				},
			},
			expected: "Role:human Parts:This is a very long message that should be trun...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := summarizeMessageHistory(tt.history)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildSystemPrompt(t *testing.T) {
	tests := []struct {
		name             string
		jsonResponse     *bool
		jsonResponseKeys *[]string
		tools            []llms.Tool
		expected         string
	}{
		{
			name:         "no tools",
			jsonResponse: nil,
			tools:        []llms.Tool{},
			expected:     "No tools are available. Respond with the final result as a string.\n",
		},
		{
			name:         "with JSON response",
			jsonResponse: boolPtr(true),
			tools:        []llms.Tool{},
			expected:     "Respond in JSON format. No tools are available. Respond with the final result as a string.\n",
		},
		{
			name:             "with JSON response and keys",
			jsonResponse:     boolPtr(true),
			jsonResponseKeys: &[]string{"key1", "key2"},
			tools:            []llms.Tool{},
			expected:         "Respond in JSON format, include `key1`, `key2` in response keys. No tools are available. Respond with the final result as a string.\n",
		},
		{
			name:         "with tools",
			jsonResponse: nil,
			tools: []llms.Tool{
				{
					Function: &llms.FunctionDefinition{
						Name:        "test_tool",
						Description: "A test tool",
						Parameters: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"param1": map[string]interface{}{
									"type": "string",
								},
							},
						},
					},
				},
			},
			expected: "\n\nYou have access to the following tools. Use tools only when necessary to fulfill the request. Consider all previous tool outputs when deciding which tools to use next. After tool execution, you will receive the results in the conversation history. Do NOT suggest the same tool with identical parameters unless explicitly required by new user input. Once all necessary tools are executed, return the final result as a string (e.g., '12345', 'joel').\n\nWhen using tools, respond with a JSON array of tool call objects, each containing 'name' and 'arguments' fields, even for a single tool:\n[\n  {\n    \"name\": \"tool1\",\n    \"arguments\": {\n      \"param1\": \"value1\"\n    }\n  }\n]\n\nRules:\n- Return a JSON array for tool calls, even for one tool.\n- Include all required parameters.\n- Execute tools in the specified order, using previous tool outputs to inform parameters.\n- After tool execution, return the final result as a string without tool calls unless new tools are needed.\n- Do NOT include explanatory text with tool call JSON.\n\nAvailable tools:\n- test_tool: A test tool\n  - param1: \n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildSystemPrompt(tt.jsonResponse, tt.jsonResponseKeys, tt.tools)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetRoleAndType(t *testing.T) {
	tests := []struct {
		name         string
		rolePtr      *string
		expectedRole string
		expectedType llms.ChatMessageType
	}{
		{
			name:         "nil role",
			rolePtr:      nil,
			expectedRole: RoleHuman,
			expectedType: llms.ChatMessageTypeHuman,
		},
		{
			name:         "empty role",
			rolePtr:      stringPtr(""),
			expectedRole: RoleHuman,
			expectedType: llms.ChatMessageTypeHuman,
		},
		{
			name:         "human role",
			rolePtr:      stringPtr("human"),
			expectedRole: "human",
			expectedType: llms.ChatMessageTypeHuman,
		},
		{
			name:         "system role",
			rolePtr:      stringPtr("system"),
			expectedRole: "system",
			expectedType: llms.ChatMessageTypeSystem,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			role, msgType := getRoleAndType(tt.rolePtr)
			assert.Equal(t, tt.expectedRole, role)
			assert.Equal(t, tt.expectedType, msgType)
		})
	}
}

func TestProcessScenarioMessages(t *testing.T) {
	tests := []struct {
		name     string
		scenario *[]*pklLLM.MultiChat
		expected []llms.MessageContent
	}{
		{
			name:     "nil scenario",
			scenario: nil,
			expected: []llms.MessageContent{},
		},
		{
			name:     "empty scenario",
			scenario: &[]*pklLLM.MultiChat{},
			expected: []llms.MessageContent{},
		},
		{
			name: "single message",
			scenario: &[]*pklLLM.MultiChat{
				{
					Role:   stringPtr("human"),
					Prompt: stringPtr("Hello"),
				},
			},
			expected: []llms.MessageContent{
				{
					Role:  llms.ChatMessageTypeHuman,
					Parts: []llms.ContentPart{llms.TextContent{Text: "Hello"}},
				},
			},
		},
		{
			name: "multiple messages",
			scenario: &[]*pklLLM.MultiChat{
				{
					Role:   stringPtr("human"),
					Prompt: stringPtr("Hello"),
				},
				{
					Role:   stringPtr("ai"),
					Prompt: stringPtr("Hi there"),
				},
			},
			expected: []llms.MessageContent{
				{
					Role:  llms.ChatMessageTypeHuman,
					Parts: []llms.ContentPart{llms.TextContent{Text: "Hello"}},
				},
				{
					Role:  llms.ChatMessageTypeAI,
					Parts: []llms.ContentPart{llms.TextContent{Text: "Hi there"}},
				},
			},
		},
		{
			name: "generic role",
			scenario: &[]*pklLLM.MultiChat{
				{
					Role:   stringPtr("custom"),
					Prompt: stringPtr("Custom message"),
				},
			},
			expected: []llms.MessageContent{
				{
					Role:  llms.ChatMessageTypeGeneric,
					Parts: []llms.ContentPart{llms.TextContent{Text: "[custom]: Custom message"}},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logging.NewTestLogger()
			result := processScenarioMessages(tt.scenario, logger)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMapRoleToLLMMessageType(t *testing.T) {
	tests := []struct {
		name     string
		role     string
		expected llms.ChatMessageType
	}{
		{"human role", "human", llms.ChatMessageTypeHuman},
		{"user role", "user", llms.ChatMessageTypeHuman},
		{"person role", "person", llms.ChatMessageTypeHuman},
		{"client role", "client", llms.ChatMessageTypeHuman},
		{"system role", "system", llms.ChatMessageTypeSystem},
		{"ai role", "ai", llms.ChatMessageTypeAI},
		{"assistant role", "assistant", llms.ChatMessageTypeAI},
		{"bot role", "bot", llms.ChatMessageTypeAI},
		{"chatbot role", "chatbot", llms.ChatMessageTypeAI},
		{"llm role", "llm", llms.ChatMessageTypeAI},
		{"function role", "function", llms.ChatMessageTypeFunction},
		{"action role", "action", llms.ChatMessageTypeFunction},
		{"tool role", "tool", llms.ChatMessageTypeTool},
		{"unknown role", "unknown", llms.ChatMessageTypeGeneric},
		{"empty role", "", llms.ChatMessageTypeGeneric},
		{"whitespace role", "   ", llms.ChatMessageTypeGeneric},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapRoleToLLMMessageType(tt.role)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper functions
func boolPtr(b bool) *bool {
	return &b
}

func stringPtr(s string) *string {
	return &s
}

func setupTestExecResolver(t *testing.T) *DependencyResolver {
	// Use the standardized real filesystem helper for exec tests
	return setupTestResolverWithRealFS(t)
}

func TestHandleExec(t *testing.T) {
	dr := setupTestExecResolver(t)

	t.Run("SuccessfulExecution", func(t *testing.T) {
		execBlock := &exec.ResourceExec{
			Command: "echo 'Hello, World!'",
		}

		err := dr.HandleExec("test-action", execBlock)
		assert.NoError(t, err)
	})

	t.Run("DecodeError", func(t *testing.T) {
		execBlock := &exec.ResourceExec{
			Command: "invalid base64",
		}

		err := dr.HandleExec("test-action", execBlock)
		assert.NoError(t, err)
	})
}

func TestDecodeExecBlock(t *testing.T) {
	dr := setupTestExecResolver(t)

	t.Run("ValidBase64Command", func(t *testing.T) {
		encodedCommand := "ZWNobyAnSGVsbG8sIFdvcmxkISc=" // "echo 'Hello, World!'"
		execBlock := &exec.ResourceExec{
			Command: encodedCommand,
		}

		err := dr.decodeExecBlock(execBlock)
		assert.NoError(t, err)
		assert.Equal(t, "echo 'Hello, World!'", execBlock.Command)
	})

	t.Run("ValidBase64Env", func(t *testing.T) {
		env := map[string]string{
			"TEST_KEY": "dGVzdF92YWx1ZQ==", // "test_value"
		}
		execBlock := &exec.ResourceExec{
			Command: "echo 'test'",
			Env:     &env,
		}

		err := dr.decodeExecBlock(execBlock)
		assert.NoError(t, err)
		assert.Equal(t, "test_value", (*execBlock.Env)["TEST_KEY"])
	})

	t.Run("InvalidBase64Command", func(t *testing.T) {
		execBlock := &exec.ResourceExec{
			Command: "invalid base64",
		}

		err := dr.decodeExecBlock(execBlock)
		assert.NoError(t, err)
	})
}

func TestWriteStdoutToFile(t *testing.T) {
	dr := setupTestExecResolver(t)

	t.Run("ValidStdout", func(t *testing.T) {
		encodedStdout := "SGVsbG8sIFdvcmxkIQ==" // "Hello, World!"
		resourceID := "test-resource"

		filePath, err := dr.WriteStdoutToFile(resourceID, &encodedStdout)
		assert.NoError(t, err)
		assert.NotEmpty(t, filePath)

		// Verify file contents
		content, err := afero.ReadFile(dr.Fs, filePath)
		assert.NoError(t, err)
		assert.NotEmpty(t, content)
	})

	t.Run("NilStdout", func(t *testing.T) {
		filePath, err := dr.WriteStdoutToFile("test-resource", nil)
		assert.NoError(t, err)
		assert.Empty(t, filePath)
	})

	t.Run("InvalidBase64", func(t *testing.T) {
		invalidStdout := "invalid base64"
		_, err := dr.WriteStdoutToFile("test-resource", &invalidStdout)
		assert.NoError(t, err)
	})
}

// skipIfPKLError skips the test when the provided error is non-nil and indicates
// that the PKL binary / registry is not available in the current CI
// environment. That allows us to exercise all pre-PKL logic while remaining
// green when the external dependency is missing.
func skipIfPKLError(t *testing.T, err error) {
	if err == nil {
		return
	}
	msg := err.Error()
	if strings.Contains(msg, "Cannot find module") ||
		strings.Contains(msg, "Received unexpected status code") ||
		strings.Contains(msg, "apple PKL not found") ||
		strings.Contains(msg, "Invalid token") {
		t.Skipf("Skipping test because PKL is unavailable: %v", err)
	}
}

func TestAppendExecEntry(t *testing.T) {
	// Helper to create fresh resolver inside each sub-test
	newResolver := func(t *testing.T) (*DependencyResolver, string) {
		dr := setupTestExecResolver(t)
		pklPath := filepath.Join(dr.ActionDir, "exec/"+dr.RequestID+"__exec_output.pkl")
		return dr, pklPath
	}

	t.Run("NewEntry", func(t *testing.T) {
		dr, pklPath := newResolver(t)

		initialContent := fmt.Sprintf(`extends "package://schema.kdeps.com/core@%s#/Exec.pkl"

resources {
}`, schema.SchemaVersion(dr.Context))
		require.NoError(t, afero.WriteFile(dr.Fs, pklPath, []byte(initialContent), 0o644))

		newExec := &exec.ResourceExec{
			Command:   "echo 'test'",
			Stdout:    utils.StringPtr("test output"),
			Timestamp: &pkl.Duration{Value: float64(time.Now().Unix()), Unit: pkl.Nanosecond},
		}

		err := dr.AppendExecEntry("test-resource", newExec)
		skipIfPKLError(t, err)
		assert.NoError(t, err)

		content, err := afero.ReadFile(dr.Fs, pklPath)
		skipIfPKLError(t, err)
		require.NoError(t, err)
		assert.Contains(t, string(content), "test-resource")
		assert.Contains(t, string(content), "ZWNobyAndGVzdCc=")
	})

	t.Run("ExistingEntry", func(t *testing.T) {
		dr, pklPath := newResolver(t)

		initialContent := fmt.Sprintf(`extends "package://schema.kdeps.com/core@%s#/Exec.pkl"

resources {
  ["existing-resource"] {
    command = "echo 'old'"
    timestamp = 1234567890.ns
  }
}`, schema.SchemaVersion(dr.Context))
		require.NoError(t, afero.WriteFile(dr.Fs, pklPath, []byte(initialContent), 0o644))

		newExec := &exec.ResourceExec{
			Command:   "echo 'new'",
			Stdout:    utils.StringPtr("new output"),
			Timestamp: &pkl.Duration{Value: float64(time.Now().Unix()), Unit: pkl.Nanosecond},
		}

		err := dr.AppendExecEntry("existing-resource", newExec)
		skipIfPKLError(t, err)
		assert.NoError(t, err)

		content, err := afero.ReadFile(dr.Fs, pklPath)
		skipIfPKLError(t, err)
		require.NoError(t, err)
		assert.Contains(t, string(content), "existing-resource")
		assert.Contains(t, string(content), "ZWNobyAnbmV3Jw==")
		assert.NotContains(t, string(content), "echo 'old'")
	})
}

func TestEncodeExecEnv(t *testing.T) {
	dr := setupTestExecResolver(t)

	t.Run("ValidEnv", func(t *testing.T) {
		env := map[string]string{
			"KEY1": "value1",
			"KEY2": "value2",
		}

		encoded := dr.encodeExecEnv(&env)
		assert.NotNil(t, encoded)
		assert.Equal(t, "dmFsdWUx", (*encoded)["KEY1"])
		assert.Equal(t, "dmFsdWUy", (*encoded)["KEY2"])
	})

	t.Run("NilEnv", func(t *testing.T) {
		encoded := dr.encodeExecEnv(nil)
		assert.Nil(t, encoded)
	})

	t.Run("EmptyEnv", func(t *testing.T) {
		env := map[string]string{}
		encoded := dr.encodeExecEnv(&env)
		assert.NotNil(t, encoded)
		assert.Empty(t, *encoded)
	})
}

func TestEncodeExecOutputs(t *testing.T) {
	dr := setupTestExecResolver(t)

	t.Run("ValidOutputs", func(t *testing.T) {
		stdout := "test output"
		stderr := "test error"

		encodedStdout, encodedStderr := dr.encodeExecOutputs(&stderr, &stdout)
		assert.NotNil(t, encodedStdout)
		assert.NotNil(t, encodedStderr)
		assert.Equal(t, "dGVzdCBlcnJvcg==", *encodedStdout)
		assert.Equal(t, "dGVzdCBvdXRwdXQ=", *encodedStderr)
	})

	t.Run("NilOutputs", func(t *testing.T) {
		encodedStdout, encodedStderr := dr.encodeExecOutputs(nil, nil)
		assert.Nil(t, encodedStdout)
		assert.Nil(t, encodedStderr)
	})
}

func newHTTPTestResolver(t *testing.T) *DependencyResolver {
	dr := setupTestResolverWithRealFS(t)
	dr.RequestID = "rid"
	// Create additional client directory for HTTP tests
	_ = dr.Fs.MkdirAll(filepath.Join(dr.ActionDir, "client"), 0o755)
	return dr
}

func TestWriteResponseBodyToFile(t *testing.T) {
	dr := newHTTPTestResolver(t)

	// happy path – encoded body should be decoded and written to file
	body := "hello world"
	enc := utils.EncodeValue(body)
	path, err := dr.WriteResponseBodyToFile("res1", &enc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path == "" {
		t.Fatalf("expected non-empty path")
	}
	// Verify file exists and content matches (decoded value)
	data, err := afero.ReadFile(dr.Fs, path)
	if err != nil {
		t.Fatalf("read file error: %v", err)
	}
	if string(data) != body {
		t.Errorf("file content mismatch: got %s want %s", string(data), body)
	}

	// nil body pointer should return empty path, nil error
	empty, err := dr.WriteResponseBodyToFile("res2", nil)
	if err != nil {
		t.Fatalf("unexpected error for nil input: %v", err)
	}
	if empty != "" {
		t.Errorf("expected empty path for nil input, got %s", empty)
	}

	// Ensure filename generation is as expected
	expectedFile := filepath.Join(dr.FilesDir, utils.GenerateResourceIDFilename("res1", dr.RequestID))
	if path != expectedFile {
		t.Errorf("unexpected file path: %s", path)
	}
}

func TestIsMethodWithBody_Cases(t *testing.T) {
	positive := []string{"POST", "put", "Patch", "DELETE"}
	for _, m := range positive {
		if !isMethodWithBody(m) {
			t.Errorf("expected %s to allow body", m)
		}
	}
	negative := []string{"GET", "HEAD", "OPTIONS"}
	for _, m := range negative {
		if isMethodWithBody(m) {
			t.Errorf("expected %s to not allow body", m)
		}
	}
}

func TestDecodeHTTPBlock_Base64(t *testing.T) {
	url := "https://example.com"
	urlEnc := base64.StdEncoding.EncodeToString([]byte(url))
	headerVal := utils.EncodeValue("application/json")
	paramVal := utils.EncodeValue("q")
	dataVal := utils.EncodeValue("body")

	client := &pklHTTP.ResourceHTTPClient{
		Url:     urlEnc,
		Headers: &map[string]string{"Content-Type": headerVal},
		Params:  &map[string]string{"search": paramVal},
		Data:    &[]string{dataVal},
	}

	dr := &DependencyResolver{Logger: logging.GetLogger()}
	if err := dr.decodeHTTPBlock(client); err != nil {
		t.Fatalf("decodeHTTPBlock returned error: %v", err)
	}

	if client.Url != url {
		t.Errorf("URL not decoded: %s", client.Url)
	}
	if (*client.Headers)["Content-Type"] != "application/json" {
		t.Errorf("header not decoded: %v", client.Headers)
	}
	if (*client.Params)["search"] != "q" {
		t.Errorf("param not decoded: %v", client.Params)
	}
	if (*client.Data)[0] != "body" {
		t.Errorf("data not decoded: %v", client.Data)
	}
}

func TestEncodeResponseHelpers(t *testing.T) {
	tmp := t.TempDir()
	fs := afero.NewOsFs()
	dr := &DependencyResolver{
		Fs:        fs,
		FilesDir:  tmp,
		RequestID: "rid",
		Logger:    logging.GetLogger(),
	}
	body := "hello world"
	headers := map[string]string{"X-Test": "val"}
	resp := &pklHTTP.ResponseBlock{Body: &body, Headers: &headers}

	encodedHeaders := encodeResponseHeaders(resp)
	if !strings.Contains(encodedHeaders, "X-Test") || !strings.Contains(encodedHeaders, utils.EncodeValue("val")) {
		t.Errorf("encoded headers missing values: %s", encodedHeaders)
	}

	resourceID := "res1"
	encodedBody := encodeResponseBody(resp, dr, resourceID)
	if !strings.Contains(encodedBody, utils.EncodeValue(body)) {
		t.Errorf("encoded body missing: %s", encodedBody)
	}

	// ensure file was created
	expectedFile := filepath.Join(tmp, utils.GenerateResourceIDFilename(resourceID, dr.RequestID))
	if exists, _ := afero.Exists(fs, expectedFile); !exists {
		t.Errorf("expected file not written: %s", expectedFile)
	}

	// Nil cases
	emptyHeaders := encodeResponseHeaders(nil)
	if emptyHeaders != "    headers {[\"\"] = \"\"}\n" {
		t.Errorf("unexpected default headers: %s", emptyHeaders)
	}
	emptyBody := encodeResponseBody(nil, dr, resourceID)
	if emptyBody != "    body=\"\"\n" {
		t.Errorf("unexpected default body: %s", emptyBody)
	}
}

func TestIsMethodWithBody(t *testing.T) {
	if !isMethodWithBody("POST") || !isMethodWithBody("put") {
		t.Errorf("expected POST/PUT to allow body")
	}
	if isMethodWithBody("GET") || isMethodWithBody("HEAD") {
		t.Errorf("expected GET/HEAD to not allow body")
	}
}

func TestDoRequest_GET(t *testing.T) {
	// Spin up a lightweight HTTP server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("q") != "test" {
			t.Errorf("missing query param")
		}
		w.Header().Set("X-Custom", "val")
		_, _ = w.Write([]byte("hello"))
	}))
	defer srv.Close()

	client := &pklHTTP.ResourceHTTPClient{
		Method: "GET",
		Url:    srv.URL,
		Params: &map[string]string{"q": "test"},
	}

	dr := &DependencyResolver{
		Fs:      afero.NewMemMapFs(),
		Context: context.Background(),
		Logger:  logging.GetLogger(),
	}

	if err := dr.DoRequest(client); err != nil {
		t.Fatalf("DoRequest returned error: %v", err)
	}
	if client.Response == nil || client.Response.Body == nil {
		t.Fatalf("response body not set")
	}
	if *client.Response.Body != "hello" {
		t.Errorf("unexpected response body: %s", *client.Response.Body)
	}
	if (*client.Response.Headers)["X-Custom"] != "val" {
		t.Errorf("header missing: %v", client.Response.Headers)
	}
	if client.Timestamp == nil || client.Timestamp.Unit != pkl.Nanosecond {
		t.Errorf("timestamp not set properly: %+v", client.Timestamp)
	}
}

// skipIfPKLError replicates helper from exec tests so we can ignore environments
// where the PKL binary / registry is not available.
func skipIfPKLErrorPy(t *testing.T, err error) {
	if err == nil {
		return
	}
	msg := err.Error()
	if strings.Contains(msg, "Cannot find module") ||
		strings.Contains(msg, "unexpected status code") ||
		strings.Contains(msg, "apple PKL not found") {
		t.Skipf("Skipping due to missing PKL: %v", err)
	}
}

func setupTestPyResolver(t *testing.T) *DependencyResolver {
	// Use real filesystem for Python tests since they may need PKL
	return setupTestResolverWithRealFS(t)
}

func TestAppendPythonEntryExtra(t *testing.T) {
	t.Parallel()

	newResolver := func(t *testing.T) (*DependencyResolver, string) {
		dr := setupTestPyResolver(t)
		pklPath := filepath.Join(dr.ActionDir, "python/"+dr.RequestID+"__python_output.pkl")
		return dr, pklPath
	}

	t.Run("NewEntry", func(t *testing.T) {
		dr, pklPath := newResolver(t)

		initial := fmt.Sprintf(`extends "package://schema.kdeps.com/core@%s#/Python.pkl"

resources {
}`,
			schema.SchemaVersion(dr.Context))
		require.NoError(t, afero.WriteFile(dr.Fs, pklPath, []byte(initial), 0o644))

		py := &pklPython.ResourcePython{
			Script:    "print('hello')",
			Stdout:    utils.StringPtr("output"),
			Timestamp: &pkl.Duration{Value: float64(time.Now().Unix()), Unit: pkl.Nanosecond},
		}

		err := dr.AppendPythonEntry("res", py)
		skipIfPKLErrorPy(t, err)
		assert.NoError(t, err)

		content, err := afero.ReadFile(dr.Fs, pklPath)
		skipIfPKLErrorPy(t, err)
		require.NoError(t, err)
		assert.Contains(t, string(content), "res")
		// encoded script should appear
		assert.Contains(t, string(content), utils.EncodeValue("print('hello')"))
	})

	t.Run("ExistingEntry", func(t *testing.T) {
		dr, pklPath := newResolver(t)

		initial := fmt.Sprintf(`extends "package://schema.kdeps.com/core@%s#/Python.pkl"

resources {
  ["res"] {
    script = "cHJpbnQoJ29sZCc pyk="
    timestamp = 1.ns
  }
}`,
			schema.SchemaVersion(dr.Context))
		require.NoError(t, afero.WriteFile(dr.Fs, pklPath, []byte(initial), 0o644))

		py := &pklPython.ResourcePython{
			Script:    "print('new')",
			Stdout:    utils.StringPtr("new out"),
			Timestamp: &pkl.Duration{Value: float64(time.Now().Unix()), Unit: pkl.Nanosecond},
		}

		err := dr.AppendPythonEntry("res", py)
		skipIfPKLErrorPy(t, err)
		assert.NoError(t, err)

		content, err := afero.ReadFile(dr.Fs, pklPath)
		skipIfPKLErrorPy(t, err)
		require.NoError(t, err)
		assert.Contains(t, string(content), utils.EncodeValue("print('new')"))
		assert.NotContains(t, string(content), "cHJpbnQoJ29sZCc pyk=")
	})
}

type mockExecute struct {
	command     string
	args        []string
	env         []string
	shouldError bool
	stdout      string
	stderr      string
}

func (m *mockExecute) Execute(ctx context.Context) (struct {
	Stdout string
	Stderr string
}, error,
) {
	if m.shouldError {
		return struct {
			Stdout string
			Stderr string
		}{}, errors.New("mock execution error")
	}
	return struct {
		Stdout string
		Stderr string
	}{
		Stdout: m.stdout,
		Stderr: m.stderr,
	}, nil
}

func setupTestResolver(t *testing.T) *DependencyResolver {
	// Use real filesystem for tests that might need PKL
	dr := setupTestResolverWithRealFS(t)
	dr.AnacondaInstalled = false
	return dr
}

func TestHandlePython(t *testing.T) {
	dr := setupTestResolver(t)

	t.Run("SuccessfulExecution", func(t *testing.T) {
		pythonBlock := &python.ResourcePython{
			Script: "print('Hello, World!')",
		}

		err := dr.HandlePython("test-action", pythonBlock)
		assert.NoError(t, err)
	})

	t.Run("DecodeError", func(t *testing.T) {
		pythonBlock := &python.ResourcePython{
			Script: "invalid base64",
		}

		err := dr.HandlePython("test-action", pythonBlock)
		assert.NoError(t, err)
	})
}

func TestDecodePythonBlock(t *testing.T) {
	dr := setupTestResolver(t)

	t.Run("ValidBase64Script", func(t *testing.T) {
		encodedScript := "cHJpbnQoJ0hlbGxvLCBXb3JsZCEnKQ==" // "print('Hello, World!')"
		pythonBlock := &python.ResourcePython{
			Script: encodedScript,
		}

		err := dr.decodePythonBlock(pythonBlock)
		assert.NoError(t, err)
		assert.Equal(t, "print('Hello, World!')", pythonBlock.Script)
	})

	t.Run("ValidBase64Env", func(t *testing.T) {
		env := map[string]string{
			"TEST_KEY": "dGVzdF92YWx1ZQ==", // "test_value"
		}
		pythonBlock := &python.ResourcePython{
			Script: "print('test')",
			Env:    &env,
		}

		err := dr.decodePythonBlock(pythonBlock)
		assert.NoError(t, err)
		assert.Equal(t, "test_value", (*pythonBlock.Env)["TEST_KEY"])
	})

	t.Run("InvalidBase64Script", func(t *testing.T) {
		pythonBlock := &python.ResourcePython{
			Script: "invalid base64",
		}

		err := dr.decodePythonBlock(pythonBlock)
		assert.NoError(t, err)
	})
}

func TestWritePythonStdoutToFile(t *testing.T) {
	dr := setupTestResolver(t)

	t.Run("ValidStdout", func(t *testing.T) {
		encodedStdout := "SGVsbG8sIFdvcmxkIQ==" // "Hello, World!"
		resourceID := "test-resource-valid"

		filePath, err := dr.WritePythonStdoutToFile(resourceID, &encodedStdout)
		assert.NoError(t, err)
		assert.NotEmpty(t, filePath)

		// Verify file contents
		content, err := afero.ReadFile(dr.Fs, filePath)
		assert.NoError(t, err)
		assert.Contains(t, string(content), "Hello, World!")
	})

	t.Run("NilStdout", func(t *testing.T) {
		filePath, err := dr.WritePythonStdoutToFile("test-resource-nil", nil)
		assert.NoError(t, err)
		assert.Empty(t, filePath)
	})

	t.Run("InvalidBase64", func(t *testing.T) {
		invalidStdout := "invalid base64"
		_, err := dr.WritePythonStdoutToFile("test-resource-invalid", &invalidStdout)
		assert.NoError(t, err)
	})
}

func TestFormatPythonEnv(t *testing.T) {
	dr := setupTestResolver(t)

	t.Run("ValidEnv", func(t *testing.T) {
		env := map[string]string{
			"KEY1": "value1",
			"KEY2": "value2",
		}

		formatted := dr.formatPythonEnv(&env)
		assert.Len(t, formatted, 2)
		assert.Contains(t, formatted, "KEY1=value1")
		assert.Contains(t, formatted, "KEY2=value2")
	})

	t.Run("NilEnv", func(t *testing.T) {
		formatted := dr.formatPythonEnv(nil)
		assert.Empty(t, formatted)
	})

	t.Run("EmptyEnv", func(t *testing.T) {
		env := map[string]string{}
		formatted := dr.formatPythonEnv(&env)
		assert.Empty(t, formatted)
	})
}

func TestCreatePythonTempFile(t *testing.T) {
	dr := setupTestResolver(t)

	t.Run("ValidScript", func(t *testing.T) {
		script := "print('test')"

		file, err := dr.createPythonTempFile(script)
		assert.NoError(t, err)
		assert.NotNil(t, file)

		// Verify file contents
		content, err := afero.ReadFile(dr.Fs, file.Name())
		assert.NoError(t, err)
		assert.Equal(t, script, string(content))

		// Cleanup
		dr.cleanupTempFile(file.Name())
	})

	t.Run("EmptyScript", func(t *testing.T) {
		file, err := dr.createPythonTempFile("")
		assert.NoError(t, err)
		assert.NotNil(t, file)

		// Verify file contents
		content, err := afero.ReadFile(dr.Fs, file.Name())
		assert.NoError(t, err)
		assert.Empty(t, string(content))

		// Cleanup
		dr.cleanupTempFile(file.Name())
	})
}

func TestCleanupTempFile(t *testing.T) {
	dr := setupTestResolver(t)

	t.Run("ExistingFile", func(t *testing.T) {
		// Create a temporary file
		file, err := dr.Fs.Create("/tmp/test-file.txt")
		require.NoError(t, err)
		file.Close()

		// Cleanup the file
		dr.cleanupTempFile("/tmp/test-file.txt")

		// Verify file is deleted
		exists, err := afero.Exists(dr.Fs, "/tmp/test-file.txt")
		assert.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("NonExistentFile", func(t *testing.T) {
		// Attempt to cleanup non-existent file
		dr.cleanupTempFile("/tmp/non-existent.txt")
		// Should not panic or error
	})
}

func TestHandleAPIErrorResponse_Extra(t *testing.T) {
	// Case 1: APIServerMode disabled – function should return an actual error to fail processing
	dr := &DependencyResolver{APIServerMode: false}
	fatalRet, err := dr.HandleAPIErrorResponse(400, "bad", true)
	if err == nil {
		t.Fatalf("expected error when APIServerMode is false, got nil")
	}
	expectedErr := "validation failed (code 400): bad"
	if err.Error() != expectedErr {
		t.Errorf("expected error message %q, got %q", expectedErr, err.Error())
	}
	if !fatalRet {
		t.Errorf("expected fatal=true to passthrough when APIServerMode off")
	}

	// NOTE: paths where APIServerMode==true are exercised in resource_response_test.go; we only
	// verify the non-API path here to avoid external PKL dependencies.
}

// createStubPkl creates a dummy executable named `pkl` that prints JSON and exits 0.
func createStubPkl(t *testing.T) (stubDir string, cleanup func()) {
	t.Helper()
	dir := t.TempDir()
	exeName := "pkl"
	if runtime.GOOS == "windows" {
		exeName = "pkl.bat"
	}
	stubPath := filepath.Join(dir, exeName)
	script := `#!/bin/sh
output_path=
prev=
for arg in "$@"; do
  if [ "$prev" = "--output-path" ]; then
    output_path="$arg"
    break
  fi
  prev="$arg"
done
json='{"hello":"world"}'
# emit JSON to stdout
echo "$json"
# if --output-path was supplied, also write JSON to that file
if [ -n "$output_path" ]; then
  echo "$json" > "$output_path"
fi
`
	if runtime.GOOS == "windows" {
		script = "@echo {\"hello\":\"world\"}\r\n"
	}
	if err := os.WriteFile(stubPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write stub: %v", err)
	}
	// ensure executable bit on unix
	if runtime.GOOS != "windows" {
		_ = os.Chmod(stubPath, 0o755)
	}
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", dir+string(os.PathListSeparator)+oldPath)
	return dir, func() { os.Setenv("PATH", oldPath) }
}

func newEvalResolver(t *testing.T) *DependencyResolver {
	dr := setupTestResolverWithRealFS(t)
	tmp := t.TempDir()
	dr.ResponsePklFile = filepath.Join(tmp, "resp.pkl")
	dr.ResponseTargetFile = filepath.Join(tmp, "resp.json")
	return dr
}

func TestExecutePklEvalCommand(t *testing.T) {
	_, restore := createStubPkl(t)
	defer restore()

	dr := newEvalResolver(t)
	// create dummy pkl file so existence check passes
	if err := afero.WriteFile(dr.Fs, dr.ResponsePklFile, []byte("{}"), 0o644); err != nil {
		t.Fatalf("write pkl: %v", err)
	}
	res, err := dr.executePklEvalCommand()
	if err != nil {
		t.Fatalf("executePklEvalCommand error: %v", err)
	}
	if res.Stdout == "" {
		t.Errorf("expected stdout from stub pkl, got empty")
	}
}

func TestEvalPklFormattedResponseFile(t *testing.T) {
	_, restore := createStubPkl(t)
	defer restore()

	dr := newEvalResolver(t)
	// create dummy pkl file
	if err := afero.WriteFile(dr.Fs, dr.ResponsePklFile, []byte("{}"), 0o644); err != nil {
		t.Fatalf("write pkl: %v", err)
	}

	out, err := dr.EvalPklFormattedResponseFile()
	if err != nil {
		t.Fatalf("EvalPklFormattedResponseFile error: %v", err)
	}
	if out == "" {
		t.Errorf("expected non-empty JSON output")
	}
	// If stub created file, ensure it's non-empty; otherwise, that's acceptable
	if exists, _ := afero.Exists(dr.Fs, dr.ResponseTargetFile); exists {
		if data, _ := afero.ReadFile(dr.Fs, dr.ResponseTargetFile); len(data) == 0 {
			t.Errorf("target file exists but empty")
		}
	}
}

func TestValidateAndEnsureResponseFiles(t *testing.T) {
	fs := afero.NewMemMapFs()
	dr := &DependencyResolver{
		Fs:                 fs,
		ResponsePklFile:    "/tmp/response.pkl",
		ResponseTargetFile: "/tmp/response.json",
		Logger:             logging.NewTestLogger(),
		Context:            context.Background(),
	}

	t.Run("ValidatePKLExtension_Success", func(t *testing.T) {
		require.NoError(t, dr.validatePklFileExtension())
	})

	t.Run("ValidatePKLExtension_Error", func(t *testing.T) {
		bad := &DependencyResolver{ResponsePklFile: "/tmp/file.txt"}
		err := bad.validatePklFileExtension()
		require.Error(t, err)
	})

	t.Run("EnsureTargetFileRemoved", func(t *testing.T) {
		// create the target file
		require.NoError(t, afero.WriteFile(fs, dr.ResponseTargetFile, []byte("x"), 0o644))
		// file should exist
		exists, _ := afero.Exists(fs, dr.ResponseTargetFile)
		require.True(t, exists)
		// call
		require.NoError(t, dr.ensureResponseTargetFileNotExists())
		// after call file should be gone
		exists, _ = afero.Exists(fs, dr.ResponseTargetFile)
		require.False(t, exists)
	})
}

func TestValidatePklFileExtension_Response(t *testing.T) {
	dr := &DependencyResolver{ResponsePklFile: "resp.pkl"}
	if err := dr.validatePklFileExtension(); err != nil {
		t.Errorf("expected .pkl to validate, got %v", err)
	}
	dr.ResponsePklFile = "bad.txt"
	if err := dr.validatePklFileExtension(); err == nil {
		t.Errorf("expected error for non-pkl extension")
	}
}

func TestDecodeErrorMessage_Handler(t *testing.T) {
	logger := logging.GetLogger()
	plain := "hello"
	enc := utils.EncodeValue(plain)
	if got := decodeErrorMessage(enc, logger); got != plain {
		t.Errorf("expected decoded value, got %s", got)
	}
	// non-base64 string passes through
	if got := decodeErrorMessage("not-encoded", logger); got != "not-encoded" {
		t.Errorf("expected passthrough, got %s", got)
	}
}

type sampleStruct struct {
	FieldA string
	FieldB int
}

func TestFormatValue_MiscTypes(t *testing.T) {
	// Map[string]interface{}
	m := map[string]interface{}{"k": "v"}
	out := formatValue(m)
	if !strings.Contains(out, "[\"k\"]") || !strings.Contains(out, "v") {
		t.Errorf("formatValue map missing expected content: %s", out)
	}

	// Nil pointer should render textual <nil>
	var ptr *sampleStruct
	if got := formatValue(ptr); !strings.Contains(got, "<nil>") {
		t.Errorf("expected output to contain <nil> for nil pointer, got %s", got)
	}

	// Struct pointer
	s := &sampleStruct{FieldA: "foo", FieldB: 42}
	out2 := formatValue(s)
	if !strings.Contains(out2, "FieldA") || !strings.Contains(out2, "foo") || !strings.Contains(out2, "42") {
		t.Errorf("formatValue struct output unexpected: %s", out2)
	}
}

func TestDecodeErrorMessage_Extra(t *testing.T) {
	orig := "hello world"
	enc := base64.StdEncoding.EncodeToString([]byte(orig))

	// base64 encoded
	if got := decodeErrorMessage(enc, logging.NewTestLogger()); got != orig {
		t.Errorf("expected decoded message %q, got %q", orig, got)
	}

	// plain string remains unchanged
	if got := decodeErrorMessage(orig, logging.NewTestLogger()); got != orig {
		t.Errorf("plain string should remain unchanged: got %q", got)
	}

	// empty string returns empty
	if got := decodeErrorMessage("", logging.NewTestLogger()); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestCreateResponsePklFile(t *testing.T) {
	// Initialize mock dependencies
	mockDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to create mock database: %v", err)
	}
	defer mockDB.Close()

	resolver := &DependencyResolver{
		Logger:          logging.NewTestLogger(),
		Fs:              afero.NewMemMapFs(),
		DBs:             []*sql.DB{mockDB},
		ResponsePklFile: "response.pkl",
	}

	// Test cases
	t.Run("SuccessfulResponse", func(t *testing.T) {
		t.Skip("Skipping SuccessfulResponse due to external pkl binary dependency")
		response := utils.NewAPIServerResponse(true, []any{"data"}, 0, "", "test-request-1")
		err := resolver.CreateResponsePklFile(response)
		assert.NoError(t, err)

		// Verify file was created
		exists, err := afero.Exists(resolver.Fs, resolver.ResponsePklFile)
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("NilResolver", func(t *testing.T) {
		var nilResolver *DependencyResolver
		err := nilResolver.CreateResponsePklFile(utils.NewAPIServerResponse(true, nil, 0, "", "test-request-nil"))
		assert.ErrorContains(t, err, "dependency resolver or database is nil")
	})

	t.Run("NilDatabase", func(t *testing.T) {
		resolver := &DependencyResolver{
			Logger: logging.NewTestLogger(),
			Fs:     afero.NewMemMapFs(),
			DBs:    nil,
		}
		err := resolver.CreateResponsePklFile(utils.NewAPIServerResponse(true, nil, 0, "", "test-request-valid"))
		assert.ErrorContains(t, err, "dependency resolver or database is nil")
	})
}

func TestEnsureResponsePklFileNotExists(t *testing.T) {
	dr := &DependencyResolver{
		Fs:     afero.NewMemMapFs(),
		Logger: logging.NewTestLogger(),
	}

	t.Run("FileDoesNotExist", func(t *testing.T) {
		err := dr.ensureResponsePklFileNotExists()
		assert.NoError(t, err)
	})

	t.Run("FileExists", func(t *testing.T) {
		// Create a test file
		err := afero.WriteFile(dr.Fs, dr.ResponsePklFile, []byte("test"), 0o644)
		require.NoError(t, err)

		err = dr.ensureResponsePklFileNotExists()
		assert.NoError(t, err)

		exists, err := afero.Exists(dr.Fs, dr.ResponsePklFile)
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestBuildResponseSections(t *testing.T) {
	dr := &DependencyResolver{
		Fs:     afero.NewMemMapFs(),
		Logger: logging.NewTestLogger(),
	}

	t.Run("FullResponse", func(t *testing.T) {
		response := utils.NewAPIServerResponse(true, []any{"data1", "data2"}, 0, "", "test-request-data")
		sections := dr.buildResponseSections("test-id", response)
		assert.NotEmpty(t, sections)
		assert.Contains(t, sections[0], "import")
		assert.Contains(t, sections[5], "success = true")
	})

	t.Run("ResponseWithError", func(t *testing.T) {
		response := utils.NewAPIServerResponse(false, nil, 404, "Resource not found", "test-request-error")
		sections := dr.buildResponseSections("test-id", response)
		assert.NotEmpty(t, sections)
		assert.Contains(t, sections[0], "import")
		assert.Contains(t, sections[5], "success = false")
	})
}

func TestFormatResponseData(t *testing.T) {
	t.Run("NilResponse", func(t *testing.T) {
		result := formatResponseData(nil)
		assert.Empty(t, result)
	})

	t.Run("EmptyData", func(t *testing.T) {
		response := &apiserverresponse.APIServerResponseBlock{
			Data: []any{},
		}
		result := formatResponseData(response)
		assert.Empty(t, result)
	})

	t.Run("WithData", func(t *testing.T) {
		response := &apiserverresponse.APIServerResponseBlock{
			Data: []any{"test"},
		}
		result := formatResponseData(response)
		assert.Contains(t, result, "response")
		assert.Contains(t, result, "data")
	})
}

func TestFormatResponseMeta(t *testing.T) {
	t.Run("NilMeta", func(t *testing.T) {
		result := formatResponseMeta("test-id", nil)
		assert.Contains(t, result, "requestID = \"test-id\"")
	})

	t.Run("EmptyMeta", func(t *testing.T) {
		meta := &apiserverresponse.APIServerResponseMetaBlock{
			Headers:    &map[string]string{},
			Properties: &map[string]string{},
		}
		result := formatResponseMeta("test-id", meta)
		assert.Contains(t, result, "requestID = \"test-id\"")
	})

	t.Run("WithHeadersAndProperties", func(t *testing.T) {
		headers := map[string]string{"Content-Type": "application/json"}
		properties := map[string]string{"key": "value"}
		meta := &apiserverresponse.APIServerResponseMetaBlock{
			Headers:    &headers,
			Properties: &properties,
		}
		result := formatResponseMeta("test-id", meta)
		assert.Contains(t, result, "requestID = \"test-id\"")
		assert.Contains(t, result, "Content-Type")
		assert.Contains(t, result, "key")
	})
}

func TestFormatErrors(t *testing.T) {
	logger := logging.NewTestLogger()

	t.Run("NilErrors", func(t *testing.T) {
		result := formatErrors(nil, logger)
		assert.Empty(t, result)
	})

	t.Run("EmptyErrors", func(t *testing.T) {
		errors := &[]*apiserverresponse.APIServerErrorsBlock{}
		result := formatErrors(errors, logger)
		assert.Empty(t, result)
	})

	t.Run("WithErrors", func(t *testing.T) {
		errors := &[]*apiserverresponse.APIServerErrorsBlock{
			{
				Code:    404,
				Message: "Resource not found",
			},
		}
		result := formatErrors(errors, logger)
		assert.Contains(t, result, "errors")
		assert.Contains(t, result, "code = 404")
		assert.Contains(t, result, "Resource not found")
	})
}

func TestDecodeErrorMessage(t *testing.T) {
	logger := logging.NewTestLogger()

	t.Run("EmptyMessage", func(t *testing.T) {
		result := decodeErrorMessage("", logger)
		assert.Empty(t, result)
	})

	t.Run("PlainMessage", func(t *testing.T) {
		message := "test message"
		result := decodeErrorMessage(message, logger)
		assert.Equal(t, message, result)
	})

	t.Run("Base64Message", func(t *testing.T) {
		message := "dGVzdCBtZXNzYWdl"
		result := decodeErrorMessage(message, logger)
		assert.Equal(t, "test message", result)
	})
}

func TestHandleAPIErrorResponse(t *testing.T) {
	t.Skip("Skipping HandleAPIErrorResponse tests due to external PKL dependency")
	dr := &DependencyResolver{
		Fs:            afero.NewMemMapFs(),
		Logger:        logging.NewTestLogger(),
		APIServerMode: true,
	}

	t.Run("ErrorResponse", func(t *testing.T) {
		fatal, err := dr.HandleAPIErrorResponse(404, "Resource not found", true)
		assert.NoError(t, err)
		assert.True(t, fatal)

		exists, err := afero.Exists(dr.Fs, dr.ResponsePklFile)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("NonAPIServerMode", func(t *testing.T) {
		dr.APIServerMode = false
		fatal, err := dr.HandleAPIErrorResponse(404, "Resource not found", true)
		assert.NoError(t, err)
		assert.True(t, fatal)

		exists, err := afero.Exists(dr.Fs, dr.ResponsePklFile)
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestFormatMapAndValueHelpers(t *testing.T) {
	simpleMap := map[interface{}]interface{}{uuid.New().String(): "value"}
	formatted := formatMap(simpleMap)
	require.Contains(t, formatted, "new Mapping {")
	require.Contains(t, formatted, "value")

	// Value wrappers
	require.Equal(t, "null", formatValue(nil))

	// Map[string]interface{}
	m := map[string]interface{}{"key": "val"}
	formattedMap := formatValue(m)
	require.Contains(t, formattedMap, "\"key\"")
	require.Contains(t, formattedMap, "val")

	// Struct pointer should deref
	type sample struct{ A string }
	s := &sample{A: "x"}
	formattedStruct := formatValue(s)
	require.Contains(t, formattedStruct, "A")
	require.Contains(t, formattedStruct, "x")

	// structToMap should reflect fields
	stMap := structToMap(sample{A: "y"})
	require.Equal(t, "y", stMap["A"])
}

func TestDecodeErrorMessageExtra(t *testing.T) {
	logger := logging.NewTestLogger()
	src := "hello"
	encoded := base64.StdEncoding.EncodeToString([]byte(src))
	// Should decode base64
	out := decodeErrorMessage(encoded, logger)
	require.Equal(t, src, out)

	// Non-base64 should return original
	require.Equal(t, src, decodeErrorMessage(src, logger))
}

// Simple struct for structToMap / formatValue tests
type demo struct {
	FieldA string
	FieldB int
}

func TestFormatValueVariousTypes(t *testing.T) {
	// nil becomes "null"
	assert.Contains(t, formatValue(nil), "null")

	// map[string]interface{}
	m := map[string]interface{}{"k1": "v1"}
	out := formatValue(m)
	assert.Contains(t, out, "[\"k1\"]")
	assert.Contains(t, out, "v1")

	// pointer to struct
	d := &demo{FieldA: "abc", FieldB: 123}
	out2 := formatValue(d)
	assert.Contains(t, out2, "FieldA")
	assert.Contains(t, out2, "abc")
}

func TestValidatePklFileExtension(t *testing.T) {
	fs := afero.NewMemMapFs()
	dr := &DependencyResolver{Fs: fs, ResponsePklFile: "/file.pkl", ResponseTargetFile: "/out.json"}
	assert.NoError(t, dr.validatePklFileExtension())

	dr.ResponsePklFile = "/file.txt"
	assert.Error(t, dr.validatePklFileExtension())
}

func TestEnsureResponseTargetFileNotExists(t *testing.T) {
	fs := afero.NewMemMapFs()
	path := "/out.json"
	_ = afero.WriteFile(fs, path, []byte("x"), 0o644)

	dr := &DependencyResolver{Fs: fs, ResponseTargetFile: path}
	assert.NoError(t, dr.ensureResponseTargetFileNotExists())
	exists, _ := afero.Exists(fs, path)
	assert.False(t, exists)
}
