package docker

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/stretchr/testify/require"
)

func TestGenerateURLs_NoLatest(t *testing.T) {
	ctx := context.Background()
	originalLatest := schema.UseLatest
	schema.UseLatest = false
	defer func() { schema.UseLatest = originalLatest }()

	items, err := GenerateURLs(ctx)
	require.NoError(t, err)
	// Expect 2 items for supported architectures (pkl + anaconda) relevant to current arch
	require.Len(t, items, 2)

	// Basic validation each item populated
	for _, it := range items {
		require.NotEmpty(t, it.URL)
		require.NotEmpty(t, it.LocalName)
	}
}
