package tools

import (
	"testing"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry returned nil")
	}
	if len(r.List()) != 0 {
		t.Error("new registry should be empty")
	}
}

func TestRegisterAndList(t *testing.T) {
	r := NewRegistry()

	r.Register(&Tool{
		Name:        "test_tool",
		Description: "A test tool",
		Parameters: map[string]domain.ToolParam{
			"arg": {Type: "string", Description: "An argument"},
		},
	})

	list := r.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(list))
	}
	if list[0].Name != "test_tool" {
		t.Errorf("expected 'test_tool', got %q", list[0].Name)
	}
}

func TestGet(t *testing.T) {
	r := NewRegistry()
	r.Register(&Tool{Name: "find_me", Description: "test"})

	if tool := r.Get("find_me"); tool == nil {
		t.Error("Get returned nil for registered tool")
	}
	if tool := r.Get("not_found"); tool != nil {
		t.Error("Get should return nil for unknown tool")
	}
}

func TestRegister_Overwrite(t *testing.T) {
	r := NewRegistry()
	r.Register(&Tool{Name: "same", Description: "first"})
	r.Register(&Tool{Name: "same", Description: "second"})

	tool := r.Get("same")
	if tool.Description != "second" {
		t.Errorf("expected 'second', got %q", tool.Description)
	}
}

func TestToLLMTools(t *testing.T) {
	r := NewRegistry()
	r.Register(&Tool{
		Name:        "llm_tool",
		Description: "An LLM tool",
		Parameters: map[string]domain.ToolParam{
			"input": {Type: "string", Description: "Input text", Required: true},
		},
	})

	llmTools := r.ToLLMTools()
	if len(llmTools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(llmTools))
	}
	if llmTools[0].Name != "llm_tool" {
		t.Errorf("expected 'llm_tool', got %q", llmTools[0].Name)
	}
	if llmTools[0].Parameters["input"].Required != true {
		t.Error("expected required parameter")
	}
}

func TestList_Empty(t *testing.T) {
	r := NewRegistry()
	if len(r.List()) != 0 {
		t.Error("List should be empty")
	}
	r.Register(&Tool{Name: "a"})
	r.Register(&Tool{Name: "b"})
	if len(r.List()) != 2 {
		t.Errorf("expected 2, got %d", len(r.List()))
	}
}
