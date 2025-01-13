package schema

import (
	"fmt"
	"os"
	"sync"

	"kdeps/pkg/utils"
)

var (
	cachedVersion string
	once          sync.Once
)

// SchemaVersion fetches and returns the latest schema version, using a cached value if available.
func SchemaVersion() string {
	once.Do(func() {
		var err error
		cachedVersion, err = utils.GetLatestGitHubRelease("kdeps/schema")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Unable to fetch the latest schema version for 'kdeps/schema': %v\n", err)
			os.Exit(1)
		}
	})

	return cachedVersion
}
