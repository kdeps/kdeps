package version

// Application version information
var (
	Version = "dev"
	Commit  = ""
)

// Component version constants
const (
	// Default schema version used when not fetching latest
	DefaultSchemaVersion = "0.4.2"

	// Default Anaconda version for Docker images
	DefaultAnacondaVersion = "20.4.20-1"

	// Default PKL version for Docker images
	DefaultPklVersion = "0.28.2"

	// Default Ollama image tag version for base Docker images
	DefaultOllamaImageTag = "0.9.2"

	// Default kdeps installation version tag
	DefaultKdepsInstallVersion = "latest"

	// Minimum supported schema version - versions below this are not supported
	MinimumSchemaVersion = "0.4.2"
)
