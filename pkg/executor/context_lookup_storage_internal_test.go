package executor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func newTestCtx(t *testing.T) *ExecutionContext {
	t.Helper()
	ctx, err := NewExecutionContext(&domain.Workflow{})
	require.NoError(t, err)
	return ctx
}

func TestGetOutput_HitAndMiss(t *testing.T) {
	t.Parallel()
	ctx := newTestCtx(t)
	ctx.Outputs = map[string]interface{}{"key": "val"}

	got, err := ctx.getOutput("key")
	require.NoError(t, err)
	assert.Equal(t, "val", got)

	_, err = ctx.getOutput("missing")
	assert.Error(t, err)
}

func TestGetMemory_HitAndMiss(t *testing.T) {
	t.Parallel()
	ctx := newTestCtx(t)
	require.NoError(t, ctx.Memory.Set("k", "v"))

	got, err := ctx.getMemory("k")
	require.NoError(t, err)
	assert.Equal(t, "v", got)

	_, err = ctx.getMemory("missing")
	assert.Error(t, err)
}

func TestGetSession_HitAndMiss(t *testing.T) {
	t.Parallel()
	wf := &domain.Workflow{Settings: domain.WorkflowSettings{Session: &domain.SessionConfig{}}}
	ctx, err := NewExecutionContext(wf)
	require.NoError(t, err)
	require.NoError(t, ctx.Session.Set("sk", "sv"))

	got, err := ctx.getSession("sk")
	require.NoError(t, err)
	assert.Equal(t, "sv", got)

	_, err = ctx.getSession("missing")
	assert.Error(t, err)
}

func TestGetByType_UnknownType(t *testing.T) {
	t.Parallel()
	ctx := newTestCtx(t)
	_, err := ctx.getByType("name", "totally_unknown_type")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown storage type")
}

func TestGetByType_Output(t *testing.T) {
	t.Parallel()
	ctx := newTestCtx(t)
	ctx.Outputs = map[string]interface{}{"result": "42"}

	got, err := ctx.getByType("result", "output")
	require.NoError(t, err)
	assert.Equal(t, "42", got)
}

func TestLookupStorageItem_Empty(t *testing.T) {
	t.Parallel()
	ctx := newTestCtx(t)
	ctx.Items = map[string]interface{}{itemKeyCurrent: "row1"}

	got, err := lookupStorageItem(ctx, "")
	require.NoError(t, err)
	assert.Equal(t, "row1", got)
}

func TestStorageTypeHandlers_ContainsAll(t *testing.T) {
	t.Parallel()
	handlers := storageTypeHandlers()
	for _, key := range []string{"item", "loop", "memory", "session", "output", "param", "header", "file", "info", "data", "body", "filepath", "filetype"} {
		assert.Contains(t, handlers, key, "handler for %q should exist", key)
	}
}
