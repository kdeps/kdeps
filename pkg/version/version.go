package version

// Application version and build information
var (
	Version = "dev"
	Commit  = ""
)

// External tool versions
const (
	// PKL language version
	PklVersion = "0.28.2"

	// Anaconda distribution version
	AnacondaVersion = "2024.10-1"

	// Schema version for kdeps core
	SchemaVersion = "0.2.40"
)

// Docker image tags
const (
	// Default Ollama image tag
	DefaultOllamaImageTag = "0.9.2"

	// Latest tag for dynamic versioning
	LatestTag = "latest"
)

// Version placeholders for "latest" mode
const (
	LatestVersionPlaceholder = "latest"
)
