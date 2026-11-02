// Package codegraph hands the code graph's business rules to Kartographer. The
// rules are the "rules" item of the "codegraph" asset set.
package codegraph

import (
	"fmt"
	"os"

	kgraph "github.com/kdeps/kartographer/codegraph"
	"gopkg.in/yaml.v3"

	"github.com/kdeps/kdeps/v2/pkg/assets"
)

// The code graph's rules are the "rules" item of the "codegraph" asset set.
const (
	rulesSet  = "codegraph"
	rulesItem = "rules"
)

// Configure reads the rules from the codegraph asset set and gives them to
// Kartographer. It must run before the code graph is built.
//
//kdeps:flow codegraph: the code graph is built or queried > [its rules are read from the codegraph set at start] > ranking weights, stopwords and limits apply to each query
func Configure() error {
	data, err := assets.Open(rulesSet).ReadFile(rulesSet + "/" + rulesItem + ".yaml")
	if err != nil {
		return err
	}
	return kgraph.Configure(data)
}

//nolint:gochecknoinits // registers and loads the codegraph asset set
func init() {
	assets.Register(rulesSet, assets.Spec{
		Validate: func(data []byte, _ string) error {
			var v map[string]any
			return yaml.Unmarshal(data, &v)
		},
		Required: []string{rulesItem},
	})
	if err := Configure(); err != nil {
		fmt.Fprintf(os.Stderr, "codegraph: rules: %v\n", err)
	}
}
