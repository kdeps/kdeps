package domain

// GitOperation is a version-control operation kind for the git resource.
type GitOperation string

const (
	GitOpStatus   GitOperation = "status"
	GitOpDiff     GitOperation = "diff"
	GitOpLog      GitOperation = "log"
	GitOpShow     GitOperation = "show"
	GitOpBranch   GitOperation = "branch"
	GitOpRemote   GitOperation = "remote"
	GitOpAdd      GitOperation = "add"
	GitOpCommit   GitOperation = "commit"
	GitOpCheckout GitOperation = "checkout"
	GitOpInit     GitOperation = "init"
	GitOpClone    GitOperation = "clone"
	GitOpPush     GitOperation = "push"
	GitOpPull     GitOperation = "pull"
)

// GitResourceConfig holds configuration for a git resource.
type GitResourceConfig struct {
	Operation  GitOperation `yaml:"operation"`            // required
	WorkingDir string       `yaml:"workingDir,omitempty"` // working directory (default: cwd)
	Paths      []string     `yaml:"paths,omitempty"`      // file paths for add/checkout
	Message    string       `yaml:"message,omitempty"`    // commit message
	Branch     string       `yaml:"branch,omitempty"`     // branch name for checkout/branch
	URL        string       `yaml:"url,omitempty"`        // remote URL for clone
	Remote     string       `yaml:"remote,omitempty"`     // remote name (default: origin)
	Args       []string     `yaml:"args,omitempty"`       // additional git arguments
	MaxCount   int          `yaml:"maxCount,omitempty"`   // log limit (default: 10)
	DryRun     bool         `yaml:"dryRun,omitempty"`     // dry-run mode
	Format     string       `yaml:"format,omitempty"`     // custom git format string
}
