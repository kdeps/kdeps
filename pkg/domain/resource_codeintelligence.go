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

	// CodeIntOpIndexFolder indexes the current working directory into a
	// persistent graph database (references + topics), backing the
	// graphFile/graphTopic/graphAll operations. Path is ignored -- it always
	// indexes the CWD, never an arbitrary caller-supplied path.
	CodeIntOpIndexFolder CodeIntelligenceOperation = "indexFolder"
	// CodeIntOpGraphFile returns the reference graph plus every other indexed
	// file that shares a topic with Path.
	CodeIntOpGraphFile CodeIntelligenceOperation = "graphFile"
	// CodeIntOpGraphTopic returns every indexed file tagged with Topic, plus
	// the full reference graph.
	CodeIntOpGraphTopic CodeIntelligenceOperation = "graphTopic"
	// CodeIntOpGraphAll returns the full reference graph plus every root file
	// (files nothing else references) — graphs everything indexed.
	CodeIntOpGraphAll CodeIntelligenceOperation = "graphAll"
)

// CodeIntelligenceConfig holds configuration for a code-intelligence resource.
type CodeIntelligenceConfig struct {
	Operation  CodeIntelligenceOperation `yaml:"operation"`            // required
	Path       string                    `yaml:"path,omitempty"`       // file or directory to search
	Query      string                    `yaml:"query,omitempty"`      // symbol name or search pattern
	Symbol     string                    `yaml:"symbol,omitempty"`     // specific symbol for definition/references
	Pattern    string                    `yaml:"pattern,omitempty"`    // file glob filter (e.g. "*.go") — rg only
	Language   string                    `yaml:"language,omitempty"`   // rg --type value or LSP language ID override
	LanguageID string                    `yaml:"languageId,omitempty"` // explicit LSP language ID (go, python, rust, etc.)
	Context    int                       `yaml:"context,omitempty"`    // rg -C context lines
	Limit      int                       `yaml:"limit,omitempty"`      // max results (0 = unlimited) — rg only
	Include    []string                  `yaml:"include,omitempty"`    // rg --include patterns
	Exclude    []string                  `yaml:"exclude,omitempty"`    // rg --exclude patterns
	Recursive  bool                      `yaml:"recursive,omitempty"`  // search subdirectories

	// Graph fields — indexFolder/graphFile/graphTopic/graphAll only.
	Topic      string   `yaml:"topic,omitempty"`      // topic name for graphTopic
	Extensions []string `yaml:"extensions,omitempty"` // file extensions to index for indexFolder (defaults to .md/.markdown/.txt/.yaml/.yml)
	// GraphDBPath is the bbolt graph index db path. Defaults to "<path>/.kdeps/graph.db",
	// or "<CWD>/.kdeps/graph.db" for indexFolder (which ignores Path) and whenever Path is
	// also unset.
	GraphDBPath string `yaml:"graphDBPath,omitempty"`
}
