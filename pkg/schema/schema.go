package schema

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/kdeps/kdeps/pkg/utils"
	"github.com/kdeps/kdeps/pkg/version"
)

var (
	versionCache sync.Map
	UseLatest    bool = false
	// Add exitFunc for testability
	exitFunc = os.Exit
)

// SchemaVersion(ctx) fetches and returns the schema version based on the cmd.Latest flag.
func SchemaVersion(ctx context.Context) string {
	if UseLatest { // Reference the global Latest flag from cmd package
		// Try to get from cache first
		if cached, ok := versionCache.Load("latest"); ok {
			if version, ok := cached.(string); ok {
				return version
			}
		}

		// If not in cache, fetch it
		schemaVersion, err := utils.GitHubReleaseFetcher(ctx, "kdeps/schema", "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Unable to fetch the latest schema version for 'kdeps/schema': %v\n", err)
			exitFunc(1)
		}

		// Store in cache
		versionCache.Store("latest", schemaVersion)
		return schemaVersion
	}

	// Use the centralized default schema version
	return version.DefaultSchemaVersion
}
