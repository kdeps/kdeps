package schema

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/kdeps/kdeps/pkg/utils"
)

var (
	cachedVersion    string
	once             sync.Once
	specifiedVersion string = "0.2.12" // Default specified version
	UseLatest        bool   = false
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

	// Use the specified version if not using the latest
	return specifiedVersion
}
