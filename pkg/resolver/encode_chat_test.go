package resolver_test

import (
	"context"
	"encoding/base64"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/kdeps/kdeps/pkg/utils"
	pklHTTP "github.com/kdeps/schema/gen/http"
	pklLLM "github.com/kdeps/schema/gen/llm"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestEncodeChat_AllFields(t *testing.T) {
	logger := logging.NewTestLogger()
	model := "llama2"
	prompt := "Tell me a joke"
	role := "system"
	scRole := "user"
	scPrompt := "You are helpful"
	scenario := []*pklLLM.MultiChat{{
		Role:   &scRole,
		Prompt: &scPrompt,
	}}
	// Tool definition with one parameter
	req := true
	paramType := "string"
	paramDesc := "echo value"
	params := map[string]*pklLLM.ToolProperties{
		"value": {Required: &req, Type: &paramType, Description: &paramDesc},
	}
	toolName := "echo"
	toolScript := "echo foo"
	toolDesc := "simple echo"
	tools := []*pklLLM.Tool{{
		Name:        &toolName,
		Script:      &toolScript,
		Description: &toolDesc,
		Parameters:  &params,
	}}
	// Use temporary directory for test files
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "file.txt")
	files := []string{filePath}
	chat := &pklLLM.ResourceChat{
		Model:    model,
		Prompt:   &prompt,
		Role:     &role,
		Scenario: &scenario,
		Tools:    &tools,
		Files:    &files,
	}
	encoded := resolver.EncodeChat(chat, logger)
	// Check that the encoded string contains the expected encoded values
	if !strings.Contains(encoded, utils.EncodeValue(model)) {
		t.Errorf("model not encoded correctly: %s", encoded)
	}
	if !strings.Contains(encoded, utils.EncodeValue(prompt)) {
		t.Errorf("prompt not encoded correctly: %s", encoded)
	}
	if !strings.Contains(encoded, utils.EncodeValue(role)) {
		t.Errorf("role not encoded correctly: %s", encoded)
	}
	if !strings.Contains(encoded, utils.EncodeValue(scRole)) {
		t.Errorf("scenario role not encoded: %s", encoded)
	}
	if !strings.Contains(encoded, utils.EncodeValue(scPrompt)) {
		t.Errorf("scenario prompt not encoded: %s", encoded)
	}
	if !strings.Contains(encoded, utils.EncodeValue(files[0])) {
		t.Errorf("file not encoded: %s", encoded)
	}
	if !strings.Contains(encoded, utils.EncodeValue(toolName)) {
		t.Errorf("tool name not encoded: %s", encoded)
	}
	if !strings.Contains(encoded, utils.EncodeValue(toolScript)) {
		t.Errorf("tool script not encoded: %s", encoded)
	}
	if !strings.Contains(encoded, utils.EncodeValue(paramType)) {
		t.Errorf("param type not encoded: %s", encoded)
	}
	if !strings.Contains(encoded, utils.EncodeValue(paramDesc)) {
		t.Errorf("param desc not encoded: %s", encoded)
	}
	// Check for default timeout and timestamp
	if !strings.Contains(encoded, "TimeoutDuration") {
		t.Error("timeout should be auto-populated")
	}
	if !strings.Contains(encoded, "Timestamp") {
		t.Error("timestamp should be auto-populated")
	}
}

func TestEncodeJSONResponseKeys_Nil(t *testing.T) {
	if resolver.EncodeJSONResponseKeys(nil) != nil {
		t.Errorf("expected nil when keys nil")
	}

	keys := []string{"k1"}
	enc := resolver.EncodeJSONResponseKeys(&keys)
	if (*enc)[0] != utils.EncodeValue("k1") {
		t.Errorf("key not encoded: %s", (*enc)[0])
	}
}

func TestEncodeExecHelpers(t *testing.T) {
	dr := &resolver.DependencyResolver{}

	t.Run("ExecEnv_Nil", func(t *testing.T) {
		require.Nil(t, dr.EncodeExecEnv(nil))
	})

	t.Run("ExecEnv_Encode", func(t *testing.T) {
		env := map[string]string{"KEY": "value"}
		enc := dr.EncodeExecEnv(&env)
		require.NotNil(t, enc)
		require.Equal(t, "dmFsdWU=", (*enc)["KEY"])
	})

	t.Run("ExecOutputs", func(t *testing.T) {
		stderr := "err"
		stdout := "out"
		es, eo := dr.EncodeExecOutputs(&stderr, &stdout)
		require.Equal(t, "ZXJy", *es)
		require.Equal(t, "b3V0", *eo)
	})

	t.Run("ExecOutputs_Nil", func(t *testing.T) {
		es, eo := dr.EncodeExecOutputs(nil, nil)
		require.Nil(t, es)
		require.Nil(t, eo)
	})

	t.Run("EncodeStderr", func(t *testing.T) {
		txt := "oops"
		s := dr.EncodeExecStderr(&txt)
		require.Contains(t, s, txt)
		require.Contains(t, s, "Stderr = #\"\"\"")
	})

	t.Run("EncodeStderr_Nil", func(t *testing.T) {
		require.Equal(t, "    Stderr = \"\"\n", dr.EncodeExecStderr(nil))
	})

	t.Run("EncodeStdout", func(t *testing.T) {
		txt := "yay"
		s := dr.EncodeExecStdout(&txt)
		require.Contains(t, s, txt)
		require.Contains(t, s, "Stdout = #\"\"\"")
	})

	t.Run("EncodeStdout_Nil", func(t *testing.T) {
		require.Equal(t, "    Stdout = \"\"\n", dr.EncodeExecStdout(nil))
	})
}

func newMemResolver() *resolver.DependencyResolver {
	fs := afero.NewMemMapFs()
	fs.MkdirAll("/files", 0o755)
	return &resolver.DependencyResolver{
		Fs:        fs,
		FilesDir:  "/files",
		ActionDir: "/action",
		RequestID: "req1",
		Context:   context.Background(),
		Logger:    logging.NewTestLogger(),
	}
}

func TestEncodeResponseHeadersAndBody(t *testing.T) {
	dr := newMemResolver()

	body := "hello"
	hdrs := map[string]string{"X-Test": "val"}
	resp := &pklHTTP.ResponseBlock{
		Headers: &hdrs,
		Body:    &body,
	}

	// Test headers
	headersStr := resolver.EncodeResponseHeaders(resp)
	if !strings.Contains(headersStr, "X-Test") {
		t.Fatalf("expected header name in output, got %s", headersStr)
	}

	// Test body encoding & file writing
	bodyStr := resolver.EncodeResponseBody(resp, dr, "res1")
	encoded := base64.StdEncoding.EncodeToString([]byte(body))
	if !strings.Contains(bodyStr, encoded) {
		t.Fatalf("expected encoded body in output, got %s", bodyStr)
	}
	// The file should be created with decoded content
	files, _ := afero.ReadDir(dr.Fs, dr.FilesDir)
	if len(files) == 0 {
		t.Fatalf("expected file to be written in %s", dr.FilesDir)
	}
	content, _ := afero.ReadFile(dr.Fs, dr.FilesDir+"/"+files[0].Name())
	if string(content) != body {
		t.Fatalf("expected file content %q, got %q", body, string(content))
	}
}
