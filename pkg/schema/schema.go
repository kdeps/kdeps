package schema

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/kdeps/kdeps/pkg/utils"
	"github.com/kdeps/kdeps/pkg/version"
)

// Export global variables for testing.
var (
	VersionCache sync.Map
	UseLatest    = false
	// Add exitFunc for testability.
	ExitFunc = os.Exit
)

// VersionDeps holds dependencies for VersionWithDeps, enabling test injection.
type VersionDeps struct {
	UseLatest    bool
	Fetcher      func(context.Context, string, string) (string, error)
	ExitFunc     func(int)
	VersionCache *sync.Map
}

// VersionWithDeps fetches and returns the schema version using injected dependencies.
func VersionWithDeps(ctx context.Context, deps VersionDeps) string {
	if deps.UseLatest {
		if cached, ok := deps.VersionCache.Load("version"); ok {
			if cachedStr, okStr := cached.(string); okStr {
				return cachedStr
			}
			return ""
		}
		schemaVersion, err := deps.Fetcher(ctx, "kdeps/schema", "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Unable to fetch the latest schema version for 'kdeps/schema': %v\n", err)
			deps.ExitFunc(1)
		}
		deps.VersionCache.Store("version", schemaVersion)
		return schemaVersion
	}
	return version.DefaultSchemaVersion
}

// Version fetches and returns the schema version using global dependencies.
func Version(ctx context.Context) string {
	return VersionWithDeps(ctx, VersionDeps{
		UseLatest:    UseLatest,
		Fetcher:      utils.GitHubReleaseFetcher,
		ExitFunc:     ExitFunc,
		VersionCache: &VersionCache,
	})
}
