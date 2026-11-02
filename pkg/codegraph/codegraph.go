// Package codegraph hands kdeps's code-graph business rules to Kartographer.
// The rules are YAML in this package; Kartographer holds no policy of its own.
package codegraph

import (
	_ "embed"

	kgraph "github.com/kdeps/kartographer/codegraph"
)

//go:embed rules.yaml
var rulesYAML []byte

// Configure gives Kartographer's code graph its rules. It must run before the
// code graph is built.
func Configure() error {
	return kgraph.Configure(rulesYAML)
}
