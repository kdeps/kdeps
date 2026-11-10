package codegraph

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	kgraph "github.com/kdeps/kartographer/codegraph"
	"github.com/kdeps/kartographer/mcp"
	"gopkg.in/yaml.v3"

	"github.com/kdeps/kdeps/v3/pkg/assets"
	"github.com/kdeps/kdeps/v3/pkg/domain"
	kdepstools "github.com/kdeps/kdeps/v3/pkg/tools"
)

// The exposed tools are the "tools" item of the "codegraph" asset set.
const toolsItem = "tools"

// ToolPrefix is put before each Kartographer tool name the agent is offered.
const ToolPrefix = "code_"

// exposedNames reads the names of the Kartographer tools the agent is offered.
func exposedNames() ([]string, error) {
	data, err := assets.Open(rulesSet).ReadFile(rulesSet + "/" + toolsItem + ".yaml")
	if err != nil {
		return nil, err
	}
	var v struct {
		Tools []string `yaml:"tools"`
	}
	if err = yaml.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return v.Tools, nil
}

// Register offers the exposed code-graph tools to the agent. The index of dir
// is built the first time one of them runs, so registering costs nothing.
func Register(reg *kdepstools.Registry, dir string) error {
	names, err := exposedNames()
	if err != nil {
		return err
	}
	// The schemas do not depend on the index, so an empty one gives them.
	schemas := map[string]mcp.Tool{}
	for _, t := range kgraph.Tools(new(kgraph.Index)) {
		schemas[t.Name] = t
	}

	var once sync.Once
	var loadErr error
	byName := map[string]mcp.Tool{}
	load := func() error {
		once.Do(func() {
			ix, berr := kgraph.BuildCached(dir, mustCacheDir())
			if berr != nil {
				loadErr = berr
				return
			}
			for _, t := range kgraph.Tools(ix) {
				byName[t.Name] = t
			}
		})
		return loadErr
	}

	for _, name := range names {
		src, ok := schemas[name]
		if !ok {
			return fmt.Errorf("codegraph: %q is not a Kartographer tool", name)
		}
		reg.Register(&kdepstools.Tool{
			Name:        ToolPrefix + name,
			Description: src.Description,
			Parameters:  toolParams(src.InputSchema),
			Category:    "code",
			Execute: func(args map[string]any) (string, error) {
				if lerr := load(); lerr != nil {
					return "", lerr
				}
				raw, merr := json.Marshal(kdepstools.ModelArgs(args))
				if merr != nil {
					return "", merr
				}
				return byName[name].Handle(raw)
			},
		})
	}
	return nil
}

// mustCacheDir is the saved-index directory, or the temp directory if the user
// cache is unavailable.
func mustCacheDir() string {
	dir, err := kgraph.CacheDir()
	if err != nil {
		return os.TempDir()
	}
	return dir
}

// toolParams turns a Kartographer JSON Schema into kdeps parameters.
func toolParams(schema map[string]any) map[string]domain.ToolParam {
	out := map[string]domain.ToolParam{}
	props, _ := schema["properties"].(map[string]any)
	required := map[string]bool{}
	if req, ok := schema["required"].([]string); ok {
		for _, r := range req {
			required[r] = true
		}
	}
	for name, raw := range props {
		p, _ := raw.(map[string]any)
		param := domain.ToolParam{Required: required[name]}
		param.Type, _ = p["type"].(string)
		param.Description, _ = p["description"].(string)
		param.Default = p["default"]
		out[name] = param
	}
	return out
}
