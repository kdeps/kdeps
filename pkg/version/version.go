package version

// Application version information
var (
	Version = "dev"
	Commit  = ""
)

// Component version constants
const (
	// Default schema version used when not fetching latest
	DefaultSchemaVersion = "0.2.44"

	// Default Anaconda version for Docker images
	DefaultAnacondaVersion = "2024.10-1"

	// Default PKL version for Docker images
	DefaultPklVersion = "0.29.0"

	// Default Ollama image tag version for base Docker images
	DefaultOllamaImageTag = "0.9.6"

	// Default kdeps installation version tag
	DefaultKdepsInstallVersion = "latest"

	// Minimum supported schema version - versions below this are not supported
	MinimumSchemaVersion = "0.2.44"
)
