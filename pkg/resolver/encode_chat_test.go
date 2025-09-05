package resolver

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/utils"
	pklHTTP "github.com/kdeps/schema/gen/http"
	pklLLM "github.com/kdeps/schema/gen/llm"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestEncodeChat_AllFields(t *testing.T) {
	logger := logging.GetLogger()

	// Build a fully-populated chat block using plain-text strings.
	role := RoleHuman
	prompt := "Say hi"
	model := "mistral:7b"

	// Scenario entry
	scRole := RoleSystem
	scPrompt := "contextual prompt"
	scenario := []pklLLM.MultiChat{{Role: &scRole, Prompt: &scPrompt}}

	// Tool definition with one parameter
	req := true
	paramType := "string"
	paramDesc := "echo value"
	params := map[string]pklLLM.ToolProperties{
		"value": {Required: &req, Type: &paramType, Description: &paramDesc},
	}
	toolName := "echo"
	toolScript := "echo foo"
	toolDesc := "simple echo"
	tools := []pklLLM.Tool{{
		Name:        &toolName,
		Script:      &toolScript,
		Description: &toolDesc,
		Parameters:  &params,
	}}

	files := []string{"/tmp/file.txt"}

	chat := &pklLLM.ResourceChat{
		Model:    model,
		Prompt:   &prompt,
		Role:     &role,
		Scenario: &scenario,
		Tools:    &tools,
		Files:    &files,
		// leave Timestamp/Timeout nil so encodeChat will populate defaults
	}

	encoded := encodeChat(chat, logger)

	// Basic top-level encodings
	if encoded.Model != utils.EncodeValue(model) {
		t.Errorf("model not encoded correctly: %s", encoded.Model)
	}
	if utils.SafeDerefString(encoded.Prompt) != utils.EncodeValue(prompt) {
		t.Errorf("prompt not encoded correctly: %s", utils.SafeDerefString(encoded.Prompt))
	}
	if utils.SafeDerefString(encoded.Role) != utils.EncodeValue(role) {
		t.Errorf("role not encoded correctly: %s", utils.SafeDerefString(encoded.Role))
	}

	// Scenario should be encoded
	if encoded.Scenario == nil || len(*encoded.Scenario) != 1 {
		t.Fatalf("scenario length mismatch")
	}
	sc := (*encoded.Scenario)[0]
	if utils.SafeDerefString(sc.Role) != utils.EncodeValue(scRole) {
		t.Errorf("scenario role not encoded: %s", utils.SafeDerefString(sc.Role))
	}
	if utils.SafeDerefString(sc.Prompt) != utils.EncodeValue(scPrompt) {
		t.Errorf("scenario prompt not encoded: %s", utils.SafeDerefString(sc.Prompt))
	}

	// Files encoded
	if encoded.Files == nil || (*encoded.Files)[0] != utils.EncodeValue(files[0]) {
		t.Errorf("file not encoded: %v", encoded.Files)
	}

	// Tools encoded
	if encoded.Tools == nil || len(*encoded.Tools) != 1 {
		t.Fatalf("encoded tools missing")
	}
	et := (*encoded.Tools)[0]
	if utils.SafeDerefString(et.Name) != utils.EncodeValue(toolName) {
		t.Errorf("tool name not encoded")
	}
	if utils.SafeDerefString(et.Script) != utils.EncodeValue(toolScript) {
		t.Errorf("tool script not encoded")
	}
	gotParam := (*et.Parameters)["value"]
	if utils.SafeDerefString(gotParam.Type) != utils.EncodeValue(paramType) {
		t.Errorf("param type not encoded: %s", utils.SafeDerefString(gotParam.Type))
	}
	if utils.SafeDerefString(gotParam.Description) != utils.EncodeValue(paramDesc) {
		t.Errorf("param desc not encoded: %s", utils.SafeDerefString(gotParam.Description))
	}

	// Defaults populated
	if encoded.Timestamp == nil {
		t.Error("timestamp should be auto-populated")
	}
	if encoded.TimeoutDuration == nil || encoded.TimeoutDuration.Value != 60 {
		t.Errorf("timeout default incorrect: %+v", encoded.TimeoutDuration)
	}
}

func TestEncodeJSONResponseKeys_Nil(t *testing.T) {
	if encodeJSONResponseKeys(nil) != nil {
		t.Errorf("expected nil when keys nil")
	}

	keys := []string{"k1"}
	enc := encodeJSONResponseKeys(&keys)
	if (*enc)[0] != utils.EncodeValue("k1") {
		t.Errorf("key not encoded: %s", (*enc)[0])
	}
}

func TestEncodeExecHelpers(t *testing.T) {
	dr := &DependencyResolver{}

	t.Run("ExecEnv_Nil", func(t *testing.T) {
		require.Nil(t, dr.encodeExecEnv(nil))
	})

	t.Run("ExecEnv_Encode", func(t *testing.T) {
		env := map[string]string{"KEY": "value"}
		enc := dr.encodeExecEnv(&env)
		require.NotNil(t, enc)
		require.Equal(t, "dmFsdWU=", (*enc)["KEY"])
	})

	t.Run("ExecOutputs", func(t *testing.T) {
		stderr := "err"
		stdout := "out"
		es, eo := dr.encodeExecOutputs(&stderr, &stdout)
		require.Equal(t, "ZXJy", *es)
		require.Equal(t, "b3V0", *eo)
	})

	t.Run("ExecOutputs_Nil", func(t *testing.T) {
		es, eo := dr.encodeExecOutputs(nil, nil)
		require.Nil(t, es)
		require.Nil(t, eo)
	})

	t.Run("EncodeStderr", func(t *testing.T) {
		txt := "oops"
		s := dr.encodeExecStderr(&txt)
		require.Contains(t, s, txt)
		require.Contains(t, s, "Stderr = #\"\"\"")
	})

	t.Run("EncodeStderr_Nil", func(t *testing.T) {
		require.Equal(t, "    Stderr = \"\"\n", dr.encodeExecStderr(nil))
	})

	t.Run("EncodeStdout", func(t *testing.T) {
		txt := "yay"
		s := dr.encodeExecStdout(&txt)
		require.Contains(t, s, txt)
		require.Contains(t, s, "Stdout = #\"\"\"")
	})

	t.Run("EncodeStdout_Nil", func(t *testing.T) {
		require.Equal(t, "    Stdout = \"\"\n", dr.encodeExecStdout(nil))
	})
}

func newMemResolver() *DependencyResolver {
	fs := afero.NewMemMapFs()
	fs.MkdirAll("/files", 0o755)
	return &DependencyResolver{
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
	headersStr := encodeResponseHeaders(resp)
	if !strings.Contains(headersStr, "X-Test") {
		t.Fatalf("expected header name in output, got %s", headersStr)
	}

	// Test body encoding & file writing
	bodyStr := encodeResponseBody(resp, dr, "res1")
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
