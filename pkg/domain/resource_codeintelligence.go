package domain

// CodeIntelligenceOperation is a code-intelligence operation kind.
type CodeIntelligenceOperation string

const (
	CodeIntOpSymbolSearch    CodeIntelligenceOperation = "symbolSearch"
	CodeIntOpDefinition      CodeIntelligenceOperation = "definition"
	CodeIntOpReferences      CodeIntelligenceOperation = "references"
	CodeIntOpDocumentSymbols CodeIntelligenceOperation = "documentSymbols"
	CodeIntOpHover           CodeIntelligenceOperation = "hover"
	CodeIntOpDiagnostics     CodeIntelligenceOperation = "diagnostics"
)

// CodeIntelligenceConfig holds configuration for a code-intelligence resource.
type CodeIntelligenceConfig struct {
	Operation CodeIntelligenceOperation `yaml:"operation"`            // required
	Path      string                     `yaml:"path,omitempty"`      // file or directory to search
	Query     string                     `yaml:"query,omitempty"`     // symbol name or search pattern
	Symbol    string                     `yaml:"symbol,omitempty"`    // specific symbol for definition/references
	Pattern   string                     `yaml:"pattern,omitempty"`   // file glob filter (e.g. "*.go")
	Language  string                     `yaml:"language,omitempty"`  // rg --type value
	Context   int                        `yaml:"context,omitempty"`   // rg -C context lines
	Limit     int                        `yaml:"limit,omitempty"`     // max results (0 = unlimited)
	Include   []string                   `yaml:"include,omitempty"`   // rg --include patterns
	Exclude   []string                   `yaml:"exclude,omitempty"`   // rg --exclude patterns
	Recursive bool                       `yaml:"recursive,omitempty"` // search subdirectories
}
