package embedding

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func TestExecute_DBOpenError(t *testing.T) {
	orig := sqlOpen
	t.Cleanup(func() { sqlOpen = orig })
	sqlOpen = func(_, _ string) (*sql.DB, error) {
		return nil, errors.New("open failed")
	}

	e := NewExecutor()
	config := &domain.EmbeddingConfig{
		Operation:  "index",
		Text:       "test",
		DBPath:     ":memory:",
		Collection: "test",
	}
	_, err := e.Execute(nil, config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open database")
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
