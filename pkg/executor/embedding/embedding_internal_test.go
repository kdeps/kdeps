package embedding

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func TestExecute_DBOpenError(t *testing.T) {
	e := NewExecutor()
	// Use an invalid path that can't be opened
	config := &domain.EmbeddingConfig{
		Operation:  "index",
		Text:       "test",
		DBPath:     "/root/.nonexistent/noperms/test.db",
		Collection: "test",
	}
	_, err := e.Execute(nil, config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to ensure schema")
}

func TestExecute_UnknownOperation(t *testing.T) {
	e := NewExecutor()
	config := &domain.EmbeddingConfig{
		Operation:  "bogus",
		Text:       "test",
		DBPath:     ":memory:",
		Collection: "test",
	}
	_, err := e.Execute(nil, config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown operation")
}
