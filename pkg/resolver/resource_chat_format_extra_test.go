package resolver

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/pkg/logging"
	pklLLM "github.com/kdeps/schema/gen/llm"
	"github.com/spf13/afero"
)

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
	fs := afero.NewMemMapFs()
	dr := &DependencyResolver{
		Fs:        fs,
		FilesDir:  "/files",
		RequestID: "req123",
		Logger:    logging.NewTestLogger(),
	}
	_ = fs.MkdirAll(dr.FilesDir, 0o755)

	resp := "this is the content"
	encoded := base64.StdEncoding.EncodeToString([]byte(resp))

	// Base64 encoded input
	path, err := dr.WriteResponseToFile("resID", &encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := afero.ReadFile(fs, path)
	if string(data) != resp {
		t.Errorf("decoded content mismatch: got %q, want %q", string(data), resp)
	}

	// Plain text input
	path2, err := dr.WriteResponseToFile("resID2", &resp)
	if err != nil {
		t.Fatalf("unexpected error (plain): %v", err)
	}
	data2, _ := afero.ReadFile(fs, path2)
	if string(data2) != resp {
		t.Errorf("plain content mismatch: got %q, want %q", string(data2), resp)
	}
}
