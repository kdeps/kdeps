package resolver

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/tool"
	pklHTTP "github.com/kdeps/schema/gen/http"
	pklLLM "github.com/kdeps/schema/gen/llm"
	"github.com/spf13/afero"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
)

// Injectable functions for testing
var (
	// HTTP client functions
	HttpGet    = http.Get
	HttpPost   = http.Post
	NewRequest = http.NewRequest

	// Filesystem operations
	AferoReadFile  = afero.ReadFile
	AferoWriteFile = afero.WriteFile
	AferoExists    = afero.Exists
	AferoReadDir   = afero.ReadDir
	AferoWalk      = afero.Walk

	// Time operations
	TimeNow   = time.Now
	TimeSleep = time.Sleep

	// LLM operations
	OllamaNew          = ollama.New
	LLMGenerateContent = func(llm llms.Model, ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
		return llm.GenerateContent(ctx, messages, options...)
	}

	// Gin operations
	GinCreateTestContext = gin.CreateTestContext

	// Tool operations
	ProcessToolCallsFunc func(toolCalls []llms.ToolCall, toolreader *tool.PklResourceReader, chatBlock *pklLLM.ResourceChat, logger *logging.Logger, messageHistory *[]llms.MessageContent, originalPrompt string, toolOutputs map[string]string) error

	// Resource operations
	GenerateChatResponseFunc func(ctx context.Context, fs afero.Fs, llm *ollama.LLM, chatBlock *pklLLM.ResourceChat, toolreader *tool.PklResourceReader, logger *logging.Logger) (string, error)
)

// Mock implementations for testing
type MockExecBlock struct {
	Command         string
	Args            []string
	TimeoutDuration *int64
}

func (m *MockExecBlock) GetCommand() string {
	return m.Command
}

func (m *MockExecBlock) GetArgs() *[]string {
	if m.Args == nil {
		return nil
	}
	return &m.Args
}

func (m *MockExecBlock) GetTimeoutDuration() interface{} {
	return m.TimeoutDuration
}

type MockHTTPBlock struct {
	URL     string
	Method  string
	Headers map[string]string
	Body    *string
}

func (m *MockHTTPBlock) GetURL() string {
	return m.URL
}

func (m *MockHTTPBlock) GetMethod() string {
	return m.Method
}

func (m *MockHTTPBlock) GetHeaders() *map[string]string {
	if m.Headers == nil {
		return nil
	}
	return &m.Headers
}

func (m *MockHTTPBlock) GetBody() *string {
	return m.Body
}

type MockPythonBlock struct {
	Script          string
	Args            []string
	TimeoutDuration *int64
}

func (m *MockPythonBlock) GetScript() string {
	return m.Script
}

func (m *MockPythonBlock) GetArgs() *[]string {
	if m.Args == nil {
		return nil
	}
	return &m.Args
}

func (m *MockPythonBlock) GetTimeoutDuration() interface{} {
	return m.TimeoutDuration
}

// SetupTestableEnvironment sets up the environment for testing
func SetupTestableEnvironment() {
	// Override with test implementations if needed
	HttpGet = func(url string) (*http.Response, error) {
		// Return a mock response for testing
		return &http.Response{
			StatusCode: 200,
			Body:       http.NoBody,
		}, nil
	}

	OllamaNew = func(options ...ollama.Option) (*ollama.LLM, error) {
		// Return a mock LLM for testing
		return &ollama.LLM{}, nil
	}
}

// ResetEnvironment resets all injectable functions to their defaults
func ResetEnvironment() {
	HttpGet = http.Get
	HttpPost = http.Post
	NewRequest = http.NewRequest
	AferoReadFile = afero.ReadFile
	AferoWriteFile = afero.WriteFile
	AferoExists = afero.Exists
	AferoReadDir = afero.ReadDir
	AferoWalk = afero.Walk
	TimeNow = time.Now
	TimeSleep = time.Sleep
	OllamaNew = ollama.New
	GinCreateTestContext = gin.CreateTestContext
	ProcessToolCallsFunc = nil
	GenerateChatResponseFunc = nil
}

// CreateMockExecBlock creates a mock exec block for testing
func CreateMockExecBlock(command string, args []string) interface{} {
	// This would need to return an actual exec resource implementation
	// For now, return a mock struct
	return &MockExecBlock{
		Command: command,
		Args:    args,
	}
}

// CreateMockHTTPBlock creates a mock HTTP block for testing
func CreateMockHTTPBlock(url, method string, headers map[string]string, body *string) *pklHTTP.ResourceHTTPClient {
	// Return a basic mock HTTP client
	return &pklHTTP.ResourceHTTPClient{
		Url:     url,
		Method:  method,
		Headers: &headers,
		Data:    nil,
	}
}

// CreateMockPythonBlock creates a mock Python block for testing
func CreateMockPythonBlock(script string, args []string) interface{} {
	// This would need to return an actual python resource implementation
	// For now, return a mock struct
	return &MockPythonBlock{
		Script: script,
		Args:   args,
	}
}
