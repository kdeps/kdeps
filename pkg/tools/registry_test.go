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

func TestToLLMTools_WithExecute(t *testing.T) {
	r := NewRegistry()
	r.Register(&Tool{
		Name:        "exec_tool",
		Description: "A tool with an Execute function",
		Parameters: map[string]domain.ToolParam{
			"arg": {Type: "string", Description: "An argument"},
		},
		Execute: func(_ map[string]interface{}) (string, error) {
			return "executed", nil
		},
	})

	llmTools := r.ToLLMTools()
	if len(llmTools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(llmTools))
	}
	dt := llmTools[0]
	if dt.Execute == nil {
		t.Fatal("expected Execute to be non-nil when source tool has Execute")
	}
	result, err := dt.Execute(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "executed" {
		t.Errorf("expected 'executed', got %q", result)
	}
}

func TestToLLMTools_Mixed(t *testing.T) {
	r := NewRegistry()
	r.Register(&Tool{Name: "no_exec", Description: "no execute"})
	r.Register(&Tool{
		Name:        "with_exec",
		Description: "has execute",
		Execute: func(_ map[string]interface{}) (string, error) {
			return "from_exec", nil
		},
	})

	llmTools := r.ToLLMTools()
	if len(llmTools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(llmTools))
	}

	for _, dt := range llmTools {
		if dt.Name == "no_exec" {
			if dt.Execute != nil {
				t.Error("expected nil Execute for tool without Execute")
			}
		}
		if dt.Name == "with_exec" {
			if dt.Execute == nil {
				t.Fatal("expected non-nil Execute for tool with Execute")
			}
			result, err := dt.Execute(nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != "from_exec" {
				t.Errorf("expected 'from_exec', got %q", result)
			}
		}
	}
}
