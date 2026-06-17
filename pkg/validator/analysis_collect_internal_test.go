package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func TestCollectOnErrorStrings_Nil(t *testing.T) {
	t.Parallel()
	assert.Nil(t, collectOnErrorStrings(nil))
}

func TestCollectOnErrorStrings_WithExprs(t *testing.T) {
	t.Parallel()
	cfg := &domain.OnErrorConfig{
		Expr: []domain.Expression{{Raw: "output('a')"}},
		When: []domain.Expression{{Raw: "output('b')"}},
	}
	got := collectOnErrorStrings(cfg)
	assert.Contains(t, got, "output('a')")
	assert.Contains(t, got, "output('b')")
}

func TestCollectValidationStrings_Nil(t *testing.T) {
	t.Parallel()
	assert.Nil(t, collectValidationStrings(nil))
}

func TestCollectValidationStrings_WithExprs(t *testing.T) {
	t.Parallel()
	cfg := &domain.ValidationsConfig{
		Skip:  []domain.Expression{{Raw: "output('x')"}},
		Check: []domain.Expression{{Raw: "output('y')"}},
	}
	got := collectValidationStrings(cfg)
	assert.Contains(t, got, "output('x')")
	assert.Contains(t, got, "output('y')")
}

func TestCollectChatStrings_Nil(t *testing.T) {
	t.Parallel()
	assert.Nil(t, collectChatStrings(nil))
}

func TestCollectChatStrings_WithData(t *testing.T) {
	t.Parallel()
	cfg := &domain.ChatConfig{
		Prompt:   "hello",
		Files:    []string{"file.txt"},
		Scenario: []domain.ScenarioItem{{Prompt: "scenario-prompt"}},
	}
	got := collectChatStrings(cfg)
	assert.Contains(t, got, "hello")
	assert.Contains(t, got, "file.txt")
	assert.Contains(t, got, "scenario-prompt")
}

func TestCollectExecTypeStrings_Python(t *testing.T) {
	t.Parallel()
	r := &domain.Resource{Python: &domain.PythonConfig{Script: "print('hi')"}}
	got := collectExecTypeStrings(r)
	assert.Contains(t, got, "print('hi')")
}

func TestCollectExecTypeStrings_Exec(t *testing.T) {
	t.Parallel()
	r := &domain.Resource{Exec: &domain.ExecConfig{Command: "echo hi"}}
	got := collectExecTypeStrings(r)
	assert.Contains(t, got, "echo hi")
}

func TestCollectExecTypeStrings_HTTP(t *testing.T) {
	t.Parallel()
	r := &domain.Resource{HTTPClient: &domain.HTTPClientConfig{URL: "https://example.com", Data: "body"}}
	got := collectExecTypeStrings(r)
	assert.Contains(t, got, "https://example.com")
	assert.Contains(t, got, "body")
}

func TestCollectExecTypeStrings_SearchWeb(t *testing.T) {
	t.Parallel()
	r := &domain.Resource{SearchWeb: &domain.SearchWebConfig{Query: "golang test"}}
	got := collectExecTypeStrings(r)
	assert.Contains(t, got, "golang test")
}

func TestCollectInlineStrings_WithAll(t *testing.T) {
	t.Parallel()
	ac := &domain.ActionConfig{
		Expr:       "output('z')",
		Chat:       &domain.ChatConfig{Prompt: "inline-prompt"},
		Python:     &domain.PythonConfig{Script: "x=1"},
		Exec:       &domain.ExecConfig{Command: "ls"},
		HTTPClient: &domain.HTTPClientConfig{URL: "http://x.com"},
	}
	got := collectInlineStrings(ac)
	assert.Contains(t, got, "output('z')")
	assert.Contains(t, got, "inline-prompt")
	assert.Contains(t, got, "x=1")
	assert.Contains(t, got, "ls")
	assert.Contains(t, got, "http://x.com")
}

func TestCollectInlineListStrings_Empty(t *testing.T) {
	t.Parallel()
	assert.Empty(t, collectInlineListStrings(nil))
}

func TestCollectResourceStrings_Empty(t *testing.T) {
	t.Parallel()
	r := &domain.Resource{}
	got := collectResourceStrings(r)
	assert.Empty(t, got)
}
