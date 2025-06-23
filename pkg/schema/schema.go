package schema

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/kdeps/kdeps/pkg/utils"
	versionpkg "github.com/kdeps/kdeps/pkg/version"
)

var (
	VersionCache     sync.Map
	UseLatest        bool   = false
	SpecifiedVersion string = versionpkg.SchemaVersion // Default specified version
	// Add ExitFunc for testability
	ExitFunc = os.Exit
)

// SchemaVersion(ctx) fetches and returns the schema version based on the cmd.Latest flag.
func SchemaVersion(ctx context.Context) string {
	if UseLatest { // Reference the global Latest flag
		// Try to get from cache first
		if cached, ok := VersionCache.Load("version"); ok {
			return cached.(string)
		}

		// If not in cache, fetch it
		version, err := utils.GitHubReleaseFetcher(ctx, "kdeps/schema", "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Unable to fetch the latest schema version for 'kdeps/schema': %v\n", err)
			ExitFunc(1)
		}

		// Store in cache
		VersionCache.Store("version", version)
		return version
	}

	// Use the specified version if not using the latest
	return SpecifiedVersion
}
