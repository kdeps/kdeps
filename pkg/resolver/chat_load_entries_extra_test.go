package resolver

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	pklLLM "github.com/kdeps/schema/gen/llm"
	pklRes "github.com/kdeps/schema/gen/resource"
	"github.com/tmc/langchaingo/llms/ollama"
)

func TestGenerateChatResponseBasic(t *testing.T) {
	t.Parallel()

	// Create stub HTTP client to satisfy Ollama client without network
	httpClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			// Return NDJSON single line with completed message
			body := `{"message":{"content":"stub-response"},"done":true}` + "\n"
			resp := &http.Response{
				StatusCode: 200,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(body)),
			}
			resp.Header.Set("Content-Type", "application/x-ndjson")
			return resp, nil
		}),
	}

	llm, errNew := ollama.New(
		ollama.WithHTTPClient(httpClient),
		ollama.WithServerURL("http://stub"),
	)
	assert.NoError(t, errNew)

	fs := afero.NewMemMapFs()
	logger := logging.GetLogger()
	ctx := context.Background()

	prompt := "Hello"
	role := "user"
	chatBlock := &pklLLM.ResourceChat{
		Model:  "test-model",
		Prompt: &prompt,
		Role:   &role,
	}

	resp, err := generateChatResponse(ctx, fs, llm, chatBlock, nil, logger)
	assert.NoError(t, err)
	assert.Equal(t, "stub-response", resp)
}

func TestLoadResourceEntriesInjected(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	logger := logging.GetLogger()

	// Setup workflow resources directory and dummy .pkl file
	workflowDir := "/workflow"
	resourcesDir := workflowDir + "/resources"
	_ = fs.MkdirAll(resourcesDir, 0o755)
	dummyFile := resourcesDir + "/dummy.pkl"
	_ = afero.WriteFile(fs, dummyFile, []byte("dummy"), 0o644)

	dr := &DependencyResolver{
		Fs:                   fs,
		Logger:               logger,
		WorkflowDir:          workflowDir,
		ResourceDependencies: make(map[string][]string),
		Resources:            []ResourceNodeEntry{},
		LoadResourceFn: func(_ context.Context, _ string, _ ResourceType) (interface{}, error) {
			return &pklRes.Resource{ActionID: "action1"}, nil
		},
		PrependDynamicImportsFn: func(string) error { return nil },
		AddPlaceholderImportsFn: func(string) error { return nil },
	}

	err := dr.LoadResourceEntries()
	assert.NoError(t, err)
	assert.Len(t, dr.Resources, 1)
	assert.Contains(t, dr.ResourceDependencies, "action1")
}

// roundTripFunc allows defining inline RoundTripper functions.
type roundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip implements http.RoundTripper.
func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
