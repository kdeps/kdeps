//go:build !js

package llm

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

// ---- injectChainOfThought ----

func TestInjectChainOfThought_AppendToExistingSystem(t *testing.T) {
	msgs := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, "Be helpful."),
		llms.TextParts(llms.ChatMessageTypeHuman, "hello"),
	}
	out := injectChainOfThought(msgs)
	require.Len(t, out, 2)
	assert.Equal(t, llms.ChatMessageTypeSystem, out[0].Role)
	text := out[0].Parts[0].(llms.TextContent).Text
	assert.True(t, strings.Contains(text, "Be helpful."), "should keep original system text")
	assert.True(t, strings.Contains(text, chainOfThoughtInstruction), "should append CoT")
}

func TestInjectChainOfThought_PrependWhenNoSystem(t *testing.T) {
	msgs := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, "hello"),
	}
	out := injectChainOfThought(msgs)
	require.Len(t, out, 2)
	assert.Equal(t, llms.ChatMessageTypeSystem, out[0].Role)
	assert.Equal(t, llms.ChatMessageTypeHuman, out[1].Role)
}

func TestInjectChainOfThought_EmptyMessages(t *testing.T) {
	out := injectChainOfThought(nil)
	require.Len(t, out, 1)
	assert.Equal(t, llms.ChatMessageTypeSystem, out[0].Role)
}

func TestInjectChainOfThought_NonTextParts(t *testing.T) {
	msgs := []llms.MessageContent{
		{
			Role: llms.ChatMessageTypeSystem,
			Parts: []llms.ContentPart{
				llms.ImageURLContent{URL: "https://example.com/img.png"},
				llms.TextContent{Text: "describe this"},
			},
		},
	}
	out := injectChainOfThought(msgs)
	require.Len(t, out, 1)
	// Image URL part should be preserved.
	_, isImg := out[0].Parts[0].(llms.ImageURLContent)
	assert.True(t, isImg, "non-text parts should be preserved as-is")
}

// ---- parseToolCallParts ----

func TestParseToolCallParts_SingleCall(t *testing.T) {
	raw := []map[string]any{
		{
			"id":   "call_abc",
			"type": "function",
			"function": map[string]any{
				"name":      "getWeather",
				"arguments": `{"city":"London"}`,
			},
		},
	}
	parts := parseToolCallParts(raw)
	require.Len(t, parts, 1)
	tc, ok := parts[0].(llms.ToolCall)
	require.True(t, ok)
	assert.Equal(t, "call_abc", tc.ID)
	assert.Equal(t, "getWeather", tc.FunctionCall.Name)
	assert.Equal(t, `{"city":"London"}`, tc.FunctionCall.Arguments)
}

func TestParseToolCallParts_MultipleCalls(t *testing.T) {
	raw := []map[string]any{
		{"id": "c1", "type": "function", "function": map[string]any{"name": "fn1", "arguments": "{}"}},
		{"id": "c2", "type": "function", "function": map[string]any{"name": "fn2", "arguments": "{}"}},
	}
	parts := parseToolCallParts(raw)
	assert.Len(t, parts, 2)
}

func TestParseToolCallParts_UnmarshalError(t *testing.T) {
	// A channel cannot be marshalled to JSON, so json.Marshal will return an error.
	parts := parseToolCallParts(make(chan int))
	assert.Nil(t, parts)
}

func TestParseToolCallParts_EmptySlice(t *testing.T) {
	parts := parseToolCallParts([]map[string]any{})
	assert.Empty(t, parts)
}

// ---- fileContentPart ----

func TestFileContentPart_HTTPUrl_NoScheme(t *testing.T) {
	part, ok := fileContentPart("http://example.com/image.jpg")
	assert.True(t, ok)
	_, isURL := part.(llms.ImageURLContent)
	assert.True(t, isURL)
}

func TestFileContentPart_NonexistentFile(t *testing.T) {
	_, ok := fileContentPart("/does/not/exist.jpg")
	assert.False(t, ok)
}

func TestFileContentPart_LocalFile(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "test-*.png")
	require.NoError(t, err)
	_, _ = f.Write([]byte{0x89, 0x50, 0x4e, 0x47})
	require.NoError(t, f.Close())

	part, ok := fileContentPart(f.Name())
	assert.True(t, ok)
	bp, isBinary := part.(llms.BinaryContent)
	assert.True(t, isBinary)
	assert.Equal(t, "image/png", bp.MIMEType)
}

func TestFileContentPart_LocalFile_UnknownExt(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "data-*.bin")
	require.NoError(t, err)
	_, _ = f.Write([]byte{0x00, 0x01})
	require.NoError(t, f.Close())

	part, ok := fileContentPart(f.Name())
	assert.True(t, ok)
	bp, isBinary := part.(llms.BinaryContent)
	assert.True(t, isBinary)
	assert.Equal(t, "application/octet-stream", bp.MIMEType)
}

// ---- mapLLMError ----

func TestMapLLMError_Nil(t *testing.T) {
	assert.NoError(t, mapLLMError("anthropic", nil))
	assert.NoError(t, mapLLMError("google", nil))
	assert.NoError(t, mapLLMError("openai", nil))
}

func TestMapLLMError_Anthropic(t *testing.T) {
	err := mapLLMError(backendAnthropic, errors.New("unauthorized: api key not valid"))
	assert.Error(t, err)
}

func TestMapLLMError_Google(t *testing.T) {
	err := mapLLMError(backendGoogle, errors.New("quota exceeded"))
	assert.Error(t, err)
}

func TestMapLLMError_Default(t *testing.T) {
	err := mapLLMError("openai", errors.New("connection refused"))
	assert.Error(t, err)
}

func TestMapLLMError_UnknownBackend(t *testing.T) {
	err := mapLLMError("bedrock", errors.New("some error"))
	assert.Error(t, err)
}

// ---- buildStreamingReasoningOpts ----

func TestBuildStreamingReasoningOpts_Nil_WhenNoThinking(t *testing.T) {
	cfg := &domain.ChatConfig{}
	opts := buildStreamingReasoningOpts(cfg, &bytes.Buffer{})
	assert.Nil(t, opts)
}

func TestBuildStreamingReasoningOpts_Nil_WhenThinkingModeNone(t *testing.T) {
	cfg := &domain.ChatConfig{
		Thinking: &domain.ThinkingConfig{
			Mode:           domain.ThinkingModeNone,
			StreamThinking: true,
		},
	}
	opts := buildStreamingReasoningOpts(cfg, &bytes.Buffer{})
	assert.Nil(t, opts)
}

func TestBuildStreamingReasoningOpts_Nil_WhenStreamThinkingFalse(t *testing.T) {
	cfg := &domain.ChatConfig{
		Thinking: &domain.ThinkingConfig{
			Mode:           domain.ThinkingModeMedium,
			StreamThinking: false,
		},
	}
	opts := buildStreamingReasoningOpts(cfg, &bytes.Buffer{})
	assert.Nil(t, opts)
}

func TestBuildStreamingReasoningOpts_ReturnsOpt_WhenEnabled(t *testing.T) {
	cfg := &domain.ChatConfig{
		Thinking: &domain.ThinkingConfig{
			Mode:           domain.ThinkingModeMedium,
			StreamThinking: true,
		},
	}
	var buf bytes.Buffer
	opts := buildStreamingReasoningOpts(cfg, &buf)
	assert.NotEmpty(t, opts)
}

func TestBuildStreamingReasoningOpts_UsesThinkingWriter(t *testing.T) {
	var thinkBuf bytes.Buffer
	cfg := &domain.ChatConfig{
		Thinking: &domain.ThinkingConfig{
			Mode:           domain.ThinkingModeMedium,
			StreamThinking: true,
			ThinkingWriter: &thinkBuf,
		},
	}
	var mainBuf bytes.Buffer
	opts := buildStreamingReasoningOpts(cfg, &mainBuf)
	require.NotEmpty(t, opts)

	// Invoke the streaming reasoning function by applying the option.
	var callOpts llms.CallOptions
	opts[0](&callOpts)
	require.NotNil(t, callOpts.StreamingReasoningFunc)
	err := callOpts.StreamingReasoningFunc(context.Background(), []byte("think"), nil)
	require.NoError(t, err)
	assert.Equal(t, "think", thinkBuf.String())
	assert.Empty(t, mainBuf.String())
}

func TestBuildStreamingReasoningOpts_EmptyChunkNoOp(t *testing.T) {
	cfg := &domain.ChatConfig{
		Thinking: &domain.ThinkingConfig{
			Mode:           domain.ThinkingModeMedium,
			StreamThinking: true,
		},
	}
	var buf bytes.Buffer
	opts := buildStreamingReasoningOpts(cfg, &buf)
	require.NotEmpty(t, opts)

	var callOpts llms.CallOptions
	opts[0](&callOpts)
	err := callOpts.StreamingReasoningFunc(context.Background(), []byte{}, nil)
	require.NoError(t, err)
	assert.Empty(t, buf.String())
}
