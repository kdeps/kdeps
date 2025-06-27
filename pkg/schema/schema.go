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
	cachedVersion string
	once          sync.Once
	UseLatest     bool = false
)

// SchemaVersion(ctx) fetches and returns the schema version based on the cmd.Latest flag.
func SchemaVersion(ctx context.Context) string {
	if UseLatest { // Reference the global Latest flag from cmd package
		once.Do(func() {
			var err error
			cachedVersion, err = utils.GitHubReleaseFetcher(ctx, "kdeps/schema", "")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: Unable to fetch the latest schema version for 'kdeps/schema': %v\n", err)
				os.Exit(1)
			}
		})
		return cachedVersion
	}

	// Use the centralized version from pkg/version instead of hardcoded string
	return version.SchemaVersion
}
