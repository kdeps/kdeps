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

// Version fetches and returns the schema version based on the cmd.Latest flag.
func Version(ctx context.Context) string {
	if UseLatest { // Reference the global Latest flag from cmd package
		// Try to get from cache first
		if cached, ok := VersionCache.Load("version"); ok {
			if cachedStr, okStr := cached.(string); okStr {
				return cachedStr
			}
			return ""
		}

		// If not in cache, fetch it
		schemaVersion, err := utils.GitHubReleaseFetcher(ctx, "kdeps/schema", "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Unable to fetch the latest schema version for 'kdeps/schema': %v\n", err)
			ExitFunc(1)
		}

		// Store in cache
		VersionCache.Store("version", schemaVersion)
		return schemaVersion
	}

	// Use the centralized default schema version
	return version.DefaultSchemaVersion
}
