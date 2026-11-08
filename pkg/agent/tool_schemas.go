package agent

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/kdeps/kdeps/v2/pkg/assets"
	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/subagent"
	kdepstools "github.com/kdeps/kdeps/v2/pkg/tools"
)

// toolNameLoadTools is the tool that loads deferred tool definitions.
const toolNameLoadTools = "load_tools"

// toolSchemaPolicyName is the policy asset that names the core tools.
const (
	toolSchemaPolicyName = "tool-schemas"
	toolSchemaPolicyFile = toolSchemaPolicyName + ".yaml"
)

// toolCatalogDescChars is how much of a deferred tool's description its
// load_tools catalog line keeps. A display width, not policy.
const toolCatalogDescChars = 70

// toolSchemaPolicy is assets/policy/tool-schemas.yaml: the tools whose full
// definition goes on every request.
type toolSchemaPolicy struct {
	Core []string `yaml:"core"`
}

// parseToolSchemaPolicy reads the policy file. An empty core list is an
// error: it would hide every definition, including load_tools itself.
func parseToolSchemaPolicy(data []byte) (toolSchemaPolicy, error) {
	var p toolSchemaPolicy
	if err := yaml.Unmarshal(data, &p); err != nil {
		return toolSchemaPolicy{}, err
	}
	if len(p.Core) == 0 {
		return toolSchemaPolicy{}, errors.New("tool-schemas policy: core lists no tools")
	}
	return p, nil
}

// validatePolicyAsset checks one file of the policy set with the parser its
// name selects.
func validatePolicyAsset(data []byte, name string) error {
	base := filepath.Base(name)
	if strings.TrimSuffix(base, filepath.Ext(base)) == toolSchemaPolicyName {
		_, err := parseToolSchemaPolicy(data)
		return err
	}
	_, err := parsePermissionPolicy(data)
	return err
}

// toolSchemaCache holds the core set until the assets are reloaded. nil core
// with loaded set means the file is missing: send every definition.
//
//nolint:gochecknoglobals // one parsed policy per process, reset by ReloadAssets
var toolSchemaCache struct {
	sync.Mutex
	loaded bool
	core   map[string]bool
}

// initToolSchemaPolicy drops the cached core set. ReloadAssets calls it.
func initToolSchemaPolicy() {
	toolSchemaCache.Lock()
	defer toolSchemaCache.Unlock()
	toolSchemaCache.loaded = false
	toolSchemaCache.core = nil
}

// coreToolSet returns the core tool names, or nil when every definition is
// sent (no policy file, or one that does not parse). The user's
// ~/.kdeps/policy/tool-schemas.yaml overrides the built-in file.
func coreToolSet() map[string]bool {
	toolSchemaCache.Lock()
	defer toolSchemaCache.Unlock()
	if toolSchemaCache.loaded {
		return toolSchemaCache.core
	}
	toolSchemaCache.loaded = true
	data, err := readToolSchemaPolicy()
	if err != nil {
		return nil
	}
	p, err := parseToolSchemaPolicy(data)
	if err != nil {
		warnAsset(err)
		return nil
	}
	core := make(map[string]bool, len(p.Core))
	for _, n := range p.Core {
		core[n] = true
	}
	toolSchemaCache.core = core
	return core
}

// readToolSchemaPolicy reads the user's override, else the built-in file.
func readToolSchemaPolicy() ([]byte, error) {
	if home, err := os.UserHomeDir(); err == nil {
		if data, readErr := os.ReadFile(filepath.Join(home, ".kdeps", "policy", toolSchemaPolicyFile)); readErr == nil {
			return data, nil
		}
	}
	return fs.ReadFile(assets.Open("policy"), "policy/"+toolSchemaPolicyFile)
}

// requestTools is the tool list for the next request. With a core set, it
// holds the core tools, the tools kdeps does not ship (project, MCP) and the
// tools the model loaded this session; load_tools' description names the rest. Without one,
// or when nothing would be held back, it is every registered tool.
func (l *Loop) requestTools() []domain.Tool {
	if l.registry == nil {
		return nil
	}
	core := coreToolSet()
	if core == nil || l.registry.Get(toolNameLoadTools) == nil {
		return l.registry.ToLLMTools()
	}
	sent := func(t *kdepstools.Tool) bool { return !t.OnDemand || core[t.Name] || l.loadedTools[t.Name] }
	var deferred []*kdepstools.Tool
	for _, t := range l.registry.List() {
		if !sent(t) {
			deferred = append(deferred, t)
		}
	}
	if len(deferred) == 0 {
		return l.registry.ToLLMTools()
	}
	out := l.registry.ToLLMToolsWhere(sent)
	for i := range out {
		if out[i].Name == toolNameLoadTools {
			out[i].Description += "\n\nTools you can load:\n" + toolCatalog(deferred)
		}
	}
	return out
}

// toolCatalog is one line per deferred tool: its name and the start of its
// description.
func toolCatalog(deferred []*kdepstools.Tool) string {
	lines := make([]string, 0, len(deferred))
	for _, t := range deferred {
		desc := strings.Join(strings.Fields(t.Description), " ")
		if first, _, ok := strings.Cut(desc, ". "); ok {
			desc = first
		}
		lines = append(lines, "- "+t.Name+": "+truncateEllipsis(desc, toolCatalogDescChars))
	}
	return strings.Join(lines, "\n")
}

// registerToolLoader registers load_tools, which adds deferred definitions to
// every later request this session.
func (l *Loop) registerToolLoader() {
	if l.registry == nil {
		return
	}
	l.registry.Register(defined(&kdepstools.Tool{
		Name: toolNameLoadTools,
		Execute: func(args map[string]any) (string, error) {
			return l.loadTools(subagent.ToolNames(args["names"]))
		},
	}))
}

// loadTools marks names as loaded. Unknown names are reported, not fatal.
func (l *Loop) loadTools(names []string) (string, error) {
	if len(names) == 0 {
		return "", errors.New("load_tools: names is required")
	}
	if l.loadedTools == nil {
		l.loadedTools = map[string]bool{}
	}
	var loaded, unknown []string
	for _, n := range names {
		if l.registry.Get(n) == nil {
			unknown = append(unknown, n)
			continue
		}
		l.loadedTools[n] = true
		loaded = append(loaded, n)
	}
	var sb strings.Builder
	if len(loaded) > 0 {
		fmt.Fprintf(
			&sb,
			"Loaded: %s. Their definitions are on your next request; call them now.",
			strings.Join(loaded, ", "),
		)
	}
	if len(unknown) > 0 {
		if sb.Len() > 0 {
			sb.WriteString(" ")
		}
		fmt.Fprintf(&sb, "Not tools: %s.", strings.Join(unknown, ", "))
	}
	if len(loaded) == 0 {
		return "", errors.New("load_tools: " + sb.String())
	}
	return sb.String(), nil
}
