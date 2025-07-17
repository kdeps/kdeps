package version

// VersionInfo contains application version information.
type VersionInfo struct {
	Version   string
	Commit    string
	LocalMode string
}

var versionInfo = VersionInfo{
	Version:   "dev",
	Commit:    "",
	LocalMode: "0",
}

// GetVersionInfo returns the current version and commit info.
func GetVersionInfo() VersionInfo {
	return versionInfo
}

// SetVersionInfo allows setting version info (for main or tests).
// Deprecated: Prefer dependency injection in new code.
func SetVersionInfo(v, c string) {
	versionInfo.Version = v
	versionInfo.Commit = c
}

// SetLocalMode allows setting local mode info (for main or tests).
func SetLocalMode(mode string) {
	versionInfo.LocalMode = mode
}

// Component version constants
const (
	// Default schema version used when not fetching latest
	DefaultSchemaVersion = "0.4.5"

	// Default Anaconda version for Docker images
	DefaultAnacondaVersion = "20.4.20-1"

	// Default PKL version for Docker images
	DefaultPklVersion = "0.28.2"

	// Default Ollama image tag version for base Docker images
	DefaultOllamaImageTag = "0.9.2"

	// Default kdeps installation version tag
	DefaultKdepsInstallVersion = "latest"

	// Minimum supported schema version - versions below this are not supported
	MinimumSchemaVersion = "0.4.5"
)
