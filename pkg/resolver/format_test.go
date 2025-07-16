package resolver_test

import (
	"context"
	"database/sql"
	"encoding/base64"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/apple/pkl-go/pkl"
	"github.com/google/uuid"
	"github.com/kdeps/kdeps/pkg/evaluator"
	"github.com/kdeps/kdeps/pkg/logging"
	resolverpkg "github.com/kdeps/kdeps/pkg/resolver"
	"github.com/kdeps/kdeps/pkg/utils"
	apiserverresponse "github.com/kdeps/schema/gen/api_server_response"
	"github.com/kdeps/schema/gen/exec"
	pklLLM "github.com/kdeps/schema/gen/llm"
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
func setupTestResolverWithRealFS(t *testing.T) *resolverpkg.DependencyResolver {
	// Initialize evaluator for tests that need PKL functionality
	evaluator.TestSetup(t)

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

	dr := &resolverpkg.DependencyResolver{
		Fs:        fs,
		Logger:    logger,
		Context:   ctx,
		FilesDir:  filesDir,
		ActionDir: actionDir,
		RequestID: "test-request",
	}

	// Initialize PklresHelper for tests that need it
	dr.PklresHelper = resolverpkg.NewPklresHelper(dr)

	return dr
}

// setupTestResolverWithMemFS creates a DependencyResolver with in-memory filesystem
// for tests that don't need PKL functionality.
func setupTestResolverWithMemFS(_ *testing.T) *resolverpkg.DependencyResolver {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	filesDir := "/files"
	actionDir := "/action"
	_ = fs.MkdirAll(filepath.Join(actionDir, "exec"), 0o755)
	_ = fs.MkdirAll(filepath.Join(actionDir, "python"), 0o755)
	_ = fs.MkdirAll(filepath.Join(actionDir, "llm"), 0o755)
	_ = fs.MkdirAll(filesDir, 0o755)

	dr := &resolverpkg.DependencyResolver{
		Fs:        fs,
		Logger:    logger,
		Context:   ctx,
		FilesDir:  filesDir,
		ActionDir: actionDir,
		RequestID: "test-request",
	}

	// Initialize PklresHelper for tests that need it
	dr.PklresHelper = resolverpkg.NewPklresHelper(dr)

	return dr
}

func TestFormatMapSimple(t *testing.T) {
	m := map[interface{}]interface{}{
		"foo": "bar",
		1:     2,
	}
	out := resolverpkg.FormatMap(m)
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
	var v interface{}
	if out := resolverpkg.FormatValue(v); out != "null" {
		t.Errorf("expected 'null' for nil, got %s", out)
	}

	// Case 2: map[string]interface{}
	mp := map[string]interface{}{"k": "v"}
	mv := resolverpkg.FormatValue(mp)
	if !strings.Contains(mv, "new Mapping {") || !strings.Contains(mv, "[\"k\"]") {
		t.Errorf("unexpected map formatting: %s", mv)
	}

	// Case 3: pointer to struct -> should format struct fields via Mapping
	type sample struct{ Field string }
	s := &sample{Field: "data"}
	sv := resolverpkg.FormatValue(s)
	if !strings.Contains(sv, "Field") || !strings.Contains(sv, "data") {
		t.Errorf("struct pointer formatting missing content: %s", sv)
	}

	// Case 4: direct struct (non-pointer)
	sp := sample{Field: "x"}
	st := resolverpkg.FormatValue(sp)
	if !strings.Contains(st, "Field") {
		t.Errorf("struct formatting unexpected: %s", st)
	}

	// Ensure default path returns triple-quoted string for primitive
	prim := resolverpkg.FormatValue("plain")
	if !strings.Contains(prim, "\"\"\"") {
		t.Errorf("primitive formatting not triple-quoted: %s", prim)
	}

	// Sanity: reflect-based call shouldn't panic for pointer nil
	var nilPtr *sample
	_ = resolverpkg.FormatValue(nilPtr)
	// the return is acceptable, we just ensure no panic
	_ = reflect.TypeOf(nilPtr)
}

func TestGeneratePklContent_Minimal(t *testing.T) {
	ctx := context.Background()
	logger := logging.NewTestLogger()

	prompt := "Hello"
	role := resolverpkg.RoleHuman
	jsonResp := true
	res := &pklLLM.ResourceChat{
		Model:           "llama2",
		Prompt:          &prompt,
		Role:            &role,
		JSONResponse:    &jsonResp,
		TimeoutDuration: &pkl.Duration{Value: 5, Unit: pkl.Second},
	}
	m := map[string]*pklLLM.ResourceChat{"id1": res}

	pklStr := resolverpkg.GeneratePklContent(m, ctx, logger, "test-request-id")

	// Basic sanity checks
	if !strings.Contains(pklStr, "Resources {") || !strings.Contains(pklStr, "\"id1\"") {
		t.Errorf("generated PKL missing expected identifiers: %s", pklStr)
	}
	if !strings.Contains(pklStr, "Model = \"llama2\"") {
		t.Errorf("model field not serialized correctly: %s", pklStr)
	}
	if !strings.Contains(pklStr, "Prompt = \"Hello\"") {
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
			result := resolverpkg.SummarizeMessageHistory(tt.history)
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
			result := resolverpkg.BuildSystemPrompt(tt.jsonResponse, tt.jsonResponseKeys, tt.tools)
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
			expectedRole: resolverpkg.RoleHuman,
			expectedType: llms.ChatMessageTypeHuman,
		},
		{
			name:         "empty role",
			rolePtr:      stringPtr(""),
			expectedRole: resolverpkg.RoleHuman,
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
			role, msgType := resolverpkg.GetRoleAndType(tt.rolePtr)
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
			result := resolverpkg.ProcessScenarioMessages(tt.scenario, logger)
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
			result := resolverpkg.MapRoleToLLMMessageType(tt.role)
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

func setupTestExecResolver(t *testing.T) *resolverpkg.DependencyResolver {
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

		err := dr.DecodeExecBlock(execBlock)
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

		err := dr.DecodeExecBlock(execBlock)
		assert.NoError(t, err)
		assert.Equal(t, "test_value", (*execBlock.Env)["TEST_KEY"])
	})

	t.Run("InvalidBase64Command", func(t *testing.T) {
		execBlock := &exec.ResourceExec{
			Command: "invalid base64",
		}

		err := dr.DecodeExecBlock(execBlock)
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

func TestDecodeErrorMessage_Handler(t *testing.T) {
	logger := logging.GetLogger()
	plain := "hello"
	enc := utils.EncodeValue(plain)
	if got := resolverpkg.DecodeErrorMessage(enc, logger); got != plain {
		t.Errorf("expected decoded value, got %s", got)
	}
	// non-base64 string passes through
	if got := resolverpkg.DecodeErrorMessage("not-encoded", logger); got != "not-encoded" {
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
	out := resolverpkg.FormatValue(m)
	if !strings.Contains(out, "[\"k\"]") || !strings.Contains(out, "v") {
		t.Errorf("formatValue map missing expected content: %s", out)
	}

	// Nil pointer should render textual <nil>
	var ptr *sampleStruct
	if got := resolverpkg.FormatValue(ptr); !strings.Contains(got, "<nil>") {
		t.Errorf("expected output to contain <nil> for nil pointer, got %s", got)
	}

	// Struct pointer
	s := &sampleStruct{FieldA: "foo", FieldB: 42}
	out2 := resolverpkg.FormatValue(s)
	if !strings.Contains(out2, "FieldA") || !strings.Contains(out2, "foo") || !strings.Contains(out2, "42") {
		t.Errorf("formatValue struct output unexpected: %s", out2)
	}
}

func TestDecodeErrorMessage_Extra(t *testing.T) {
	orig := "hello world"
	enc := base64.StdEncoding.EncodeToString([]byte(orig))

	// base64 encoded
	if got := resolverpkg.DecodeErrorMessage(enc, logging.NewTestLogger()); got != orig {
		t.Errorf("expected decoded message %q, got %q", orig, got)
	}

	// plain string remains unchanged
	if got := resolverpkg.DecodeErrorMessage(orig, logging.NewTestLogger()); got != orig {
		t.Errorf("plain string should remain unchanged: got %q", got)
	}

	// empty string returns empty
	if got := resolverpkg.DecodeErrorMessage("", logging.NewTestLogger()); got != "" {
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

	resolver := &resolverpkg.DependencyResolver{
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
		var nilResolver *resolverpkg.DependencyResolver
		err := nilResolver.CreateResponsePklFile(utils.NewAPIServerResponse(true, nil, 0, "", "test-request-nil"))
		assert.ErrorContains(t, err, "dependency resolver or database is nil")
	})

	t.Run("NilDatabase", func(t *testing.T) {
		dr := &resolverpkg.DependencyResolver{
			Logger: logging.NewTestLogger(),
			Fs:     afero.NewMemMapFs(),
			DBs:    nil,
		}
		err := dr.CreateResponsePklFile(utils.NewAPIServerResponse(true, nil, 0, "", "test-request-valid"))
		assert.ErrorContains(t, err, "dependency resolver or database is nil")
	})
}

func TestEnsureResponsePklFileNotExists(t *testing.T) {
	dr := &resolverpkg.DependencyResolver{
		Fs:     afero.NewMemMapFs(),
		Logger: logging.NewTestLogger(),
	}

	t.Run("FileDoesNotExist", func(t *testing.T) {
		err := dr.EnsureResponsePklFileNotExists()
		assert.NoError(t, err)
	})

	t.Run("FileExists", func(t *testing.T) {
		// Create a test file
		err := afero.WriteFile(dr.Fs, dr.ResponsePklFile, []byte("test"), 0o644)
		require.NoError(t, err)

		err = dr.EnsureResponsePklFileNotExists()
		assert.NoError(t, err)

		exists, err := afero.Exists(dr.Fs, dr.ResponsePklFile)
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestBuildResponseSections(t *testing.T) {
	dr := &resolverpkg.DependencyResolver{
		Fs:     afero.NewMemMapFs(),
		Logger: logging.NewTestLogger(),
	}

	t.Run("FullResponse", func(t *testing.T) {
		sections, err := dr.BuildResponseSections()
		require.NoError(t, err)
		joined := strings.Join(sections, "\n")
		assert.Contains(t, joined, "Success = true")
	})

	t.Run("ResponseWithError", func(t *testing.T) {
		sections, err := dr.BuildResponseSections()
		require.NoError(t, err)
		joined := strings.Join(sections, "\n")
		assert.Contains(t, joined, "Success = false")
	})
}

func TestFormatResponseData(t *testing.T) {
	t.Run("NilResponse", func(t *testing.T) {
		result := resolverpkg.FormatResponseData(nil)
		assert.Empty(t, result)
	})

	t.Run("EmptyData", func(t *testing.T) {
		response := &apiserverresponse.APIServerResponseBlock{
			Data: []any{},
		}
		result := resolverpkg.FormatResponseData(response)
		assert.Empty(t, result)
	})

	t.Run("WithData", func(t *testing.T) {
		response := &apiserverresponse.APIServerResponseBlock{
			Data: []any{"test"},
		}
		result := resolverpkg.FormatResponseData(response)
		assert.Contains(t, result, "Response")
		assert.Contains(t, result, "Data")
	})
}

func TestFormatResponseMeta(t *testing.T) {
	t.Run("NilMeta", func(t *testing.T) {
		result := resolverpkg.FormatResponseMeta("test-id", nil)
		assert.Contains(t, result, "RequestID = \"test-id\"")
	})

	t.Run("EmptyMeta", func(t *testing.T) {
		meta := &apiserverresponse.APIServerResponseMetaBlock{
			Headers:    &map[string]string{},
			Properties: &map[string]string{},
		}
		result := resolverpkg.FormatResponseMeta("test-id", meta)
		assert.Contains(t, result, "RequestID = \"test-id\"")
	})

	t.Run("WithHeadersAndProperties", func(t *testing.T) {
		headers := map[string]string{"Content-Type": "application/json"}
		properties := map[string]string{"key": "value"}
		meta := &apiserverresponse.APIServerResponseMetaBlock{
			Headers:    &headers,
			Properties: &properties,
		}
		result := resolverpkg.FormatResponseMeta("test-id", meta)
		assert.Contains(t, result, "RequestID = \"test-id\"")
		assert.Contains(t, result, "Content-Type")
		assert.Contains(t, result, "key")
	})
}

func TestFormatErrors(t *testing.T) {
	logger := logging.NewTestLogger()

	t.Run("NilErrors", func(t *testing.T) {
		result := resolverpkg.FormatErrors(nil, logger)
		assert.Empty(t, result)
	})

	t.Run("EmptyErrors", func(t *testing.T) {
		errors := &[]*apiserverresponse.APIServerErrorsBlock{}
		result := resolverpkg.FormatErrors(errors, logger)
		assert.Empty(t, result)
	})

	t.Run("WithErrors", func(t *testing.T) {
		errors := &[]*apiserverresponse.APIServerErrorsBlock{
			{
				Code:    404,
				Message: "Resource not found",
			},
		}
		result := resolverpkg.FormatErrors(errors, logger)
		assert.Contains(t, result, "Errors")
		assert.Contains(t, result, "Code = 404")
		assert.Contains(t, result, "Resource not found")
	})
}

func TestDecodeErrorMessage(t *testing.T) {
	logger := logging.NewTestLogger()

	t.Run("EmptyMessage", func(t *testing.T) {
		result := resolverpkg.DecodeErrorMessage("", logger)
		assert.Empty(t, result)
	})

	t.Run("PlainMessage", func(t *testing.T) {
		message := "test message"
		result := resolverpkg.DecodeErrorMessage(message, logger)
		assert.Equal(t, message, result)
	})

	t.Run("Base64Message", func(t *testing.T) {
		message := "dGVzdCBtZXNzYWdl"
		result := resolverpkg.DecodeErrorMessage(message, logger)
		assert.Equal(t, "test message", result)
	})
}

func TestHandleAPIErrorResponse(t *testing.T) {
	t.Skip("Skipping HandleAPIErrorResponse tests due to external PKL dependency")
	dr := &resolverpkg.DependencyResolver{
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
	formatted := resolverpkg.FormatMap(simpleMap)
	require.Contains(t, formatted, "new Mapping {")
	require.Contains(t, formatted, "value")

	// Value wrappers
	require.Equal(t, "null", resolverpkg.FormatValue(nil))

	// Map[string]interface{}
	m := map[string]interface{}{"key": "val"}
	formattedMap := resolverpkg.FormatValue(m)
	require.Contains(t, formattedMap, "\"key\"")
	require.Contains(t, formattedMap, "val")

	// Struct pointer should deref
	type sample struct{ A string }
	s := &sample{A: "x"}
	formattedStruct := resolverpkg.FormatValue(s)
	require.Contains(t, formattedStruct, "A")
	require.Contains(t, formattedStruct, "x")

	// structToMap should reflect fields
	stMap := resolverpkg.StructToMap(sample{A: "y"})
	require.Equal(t, "y", stMap["A"])
}

func TestDecodeErrorMessageExtra(t *testing.T) {
	logger := logging.NewTestLogger()
	src := "hello"
	encoded := base64.StdEncoding.EncodeToString([]byte(src))
	// Should decode base64
	out := resolverpkg.DecodeErrorMessage(encoded, logger)
	require.Equal(t, src, out)

	// Non-base64 should return original
	require.Equal(t, src, resolverpkg.DecodeErrorMessage(src, logger))
}

// Simple struct for structToMap / formatValue tests
type demo struct {
	FieldA string
	FieldB int
}

func TestFormatValueVariousTypes(t *testing.T) {
	// nil becomes "null"
	assert.Contains(t, resolverpkg.FormatValue(nil), "null")

	// map[string]interface{}
	m := map[string]interface{}{"k1": "v1"}
	out := resolverpkg.FormatValue(m)
	assert.Contains(t, out, "[\"k1\"]")
	assert.Contains(t, out, "v1")

	// pointer to struct
	d := &demo{FieldA: "abc", FieldB: 123}
	out2 := resolverpkg.FormatValue(d)
	assert.Contains(t, out2, "FieldA")
	assert.Contains(t, out2, "abc")
}

func TestValidatePklFileExtension(t *testing.T) {
	fs := afero.NewMemMapFs()
	dr := &resolverpkg.DependencyResolver{Fs: fs, ResponsePklFile: "/file.pkl", ResponseTargetFile: "/out.json"}
	assert.NoError(t, dr.ValidatePklFileExtension())

	dr.ResponsePklFile = "/file.txt"
	assert.Error(t, dr.ValidatePklFileExtension())
}

func TestEnsureResponseTargetFileNotExists(t *testing.T) {
	fs := afero.NewMemMapFs()
	path := "/out.json"
	_ = afero.WriteFile(fs, path, []byte("x"), 0o644)

	dr := &resolverpkg.DependencyResolver{Fs: fs, ResponseTargetFile: path}
	assert.NoError(t, dr.EnsureResponseTargetFileNotExists())
	exists, _ := afero.Exists(fs, path)
	assert.False(t, exists)
}
