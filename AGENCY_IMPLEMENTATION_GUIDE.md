# KDeps Repository Analysis - Agency.yml Implementation Guide

## Overview
The kdeps repository is a workflow orchestration system that currently supports single-agent workflows via `workflow.yaml`. To implement `agency.yml` support for orchestrating multiple agents, you'll need to understand the existing architecture.

---

## 1. Workflow Structure (`pkg/domain/workflow.go`)

### Main Workflow Struct
```go
type Workflow struct {
    APIVersion string           `yaml:"apiVersion"`        // "kdeps.io/v1"
    Kind       string           `yaml:"kind"`              // "Workflow"
    Metadata   WorkflowMetadata `yaml:"metadata"`
    Settings   WorkflowSettings `yaml:"settings"`
    Resources  []*Resource      `yaml:"resources,omitempty"`
}
```

### WorkflowMetadata
```go
type WorkflowMetadata struct {
    Name           string   `yaml:"name"`
    Description    string   `yaml:"description"`
    Version        string   `yaml:"version"`
    TargetActionID string   `yaml:"targetActionId"`      // Entry point action ID
    Workflows      []string `yaml:"workflows,omitempty"`  // NOTE: Currently unused field!
}
```

### WorkflowSettings
```go
type WorkflowSettings struct {
    APIServerMode  bool                     `yaml:"apiServerMode"`
    WebServerMode  bool                     `yaml:"webServerMode"`
    HostIP         string                   `yaml:"hostIp,omitempty"`
    PortNum        int                      `yaml:"portNum,omitempty"`
    APIServer      *APIServerConfig         `yaml:"apiServer,omitempty"`
    WebServer      *WebServerConfig         `yaml:"webServer,omitempty"`
    AgentSettings  AgentSettings            `yaml:"agentSettings"`
    SQLConnections map[string]SQLConnection `yaml:"sqlConnections,omitempty"`
    Session        *SessionConfig           `yaml:"session,omitempty"`
    WebApp         *WebAppConfig            `yaml:"webApp,omitempty"`
    Input          *InputConfig             `yaml:"input,omitempty"`
}
```

**Key Insight:** The `WorkflowMetadata.Workflows` field exists but is unused! This could be the foundation for agency.yml support.

---

## 2. Validator Schema (`pkg/validator/schemas/workflow.json`)

The workflow schema requires:
```json
{
  "required": ["apiVersion", "kind", "metadata", "settings"],
  "properties": {
    "apiVersion": { "enum": ["kdeps.io/v1"] },
    "kind": { "enum": ["Workflow"] },
    "metadata": {
      "required": ["name", "targetActionId"],
      "properties": {
        "name": { "type": "string" },
        "targetActionId": { "type": "string" },
        "workflows": { "type": "array", "items": { "type": "string" } }
      }
    }
  }
}
```

**Note:** The schema already supports a `workflows` array in metadata but doesn't enforce/validate it!

---

## 3. Parser Implementation (`pkg/parser/yaml/parser.go`)

### Key Functions:
```go
// Main workflow parser
func (p *Parser) ParseWorkflow(path string) (*domain.Workflow, error)

// Resource parser
func (p *Parser) ParseResource(path string) (*domain.Resource, error)

// Internal: loads resources from resources/ directory
func (p *Parser) loadResources(workflow *domain.Workflow, workflowPath string) error
```

### Parsing Flow:
1. Read YAML file
2. Apply Jinja2 preprocessing (only for `{%`, `{#` tags; `{{}}` expressions handled at runtime)
3. Validate against JSON schema
4. Parse into Go structs
5. Auto-load resources from `resources/` directory

**Important:** Resources are automatically discovered from the `resources/` subdirectory alongside the workflow file.

---

## 4. Execution Flow (`cmd/run.go`)

### Key Functions:

#### `FindWorkflowFile(dir string) string`
```go
// Returns the first found workflow file in this priority:
// 1. workflow.yaml
// 2. workflow.yaml.j2
// 3. workflow.yml
// 4. workflow.yml.j2
// 5. workflow.j2 (pure Jinja2)
```
Used in: build.go, package.go, prepackage.go, run.go, scaffold.go, validate.go

#### `ParseWorkflowFile(path string) (*domain.Workflow, error)`
```go
// Creates schema validator + expression parser
// Creates YAML parser
// Parses workflow (auto-loads resources from resources/ dir)
// Returns *domain.Workflow
```

#### `RunWorkflowWithFlags(cmd *cobra.Command, args []string, flags *RunFlags) error`
```go
// 1. Resolves workflow path (handles .kdeps packages or directories)
// 2. Calls ExecuteWorkflowStepsWithFlags
```

#### `ExecuteWorkflowStepsWithFlags(cmd *cobra.Command, workflowPath string, flags *RunFlags) error`
```go
// 5-step execution pipeline:
// [1/5] Parse workflow
// [2/5] Validate workflow
// [3/5] Setup environment (Python, packages)
// [4/5] Check LLM backend (Ollama)
// [5/5] Execute workflow or start HTTP server
```

---

## 5. Scaffold Command (`cmd/scaffold.go`)

Adds resources to existing agents:

```go
func RunScaffoldWithFlags(cmd *cobra.Command, args []string, flags *ScaffoldFlags) error
```

**Valid resources:**
- http-client
- llm
- sql
- python
- exec
- response

**Process:**
1. Find workflow file with `FindWorkflowFile(flags.Dir)`
2. Create `resources/` directory if needed
3. Generate resource files from templates
4. Skip if file exists (unless `--force`)

---

## 6. New Command (`cmd/new.go`)

Creates new AI agents from templates:

```go
func RunNewWithFlags(cmd *cobra.Command, args []string, flags *NewFlags) error
```

**Key functions:**
- `determineTemplateName()` - Select template (api-service, sql-agent)
- `collectTemplateData()` - Gather template variables
- `generateProject()` - Create project from template

**Templates available:**
1. api-service (default)
2. sql-agent

Each template generates:
- workflow.yaml.j2
- README.md.j2
- env.example.j2 (or other files)

---

## 7. Template System (`pkg/templates/templates/`)

### Template Files:
```
templates/
├── api-service/
│   ├── README.md.j2
│   ├── env.example.j2
│   └── workflow.yaml.j2
├── sql-agent/
│   ├── README.md.j2
│   └── workflow.yaml.j2
└── resources/
    ├── exec.yaml.j2
    ├── http-client.yaml.j2
    ├── llm.yaml.j2
    ├── python.yaml.j2
    ├── response.yaml.j2
    ├── sql.yaml.j2
    └── .gitkeep
```

All templates use **Jinja2** syntax (`{{ var }}`, `{% if %}`, etc.)

---

## 8. HTTP Management (`pkg/infra/http/management.go` lines 450-467)

```go
// clearResourcesDir removes all .yaml/.yml files from resources directory
func clearResourcesDir(dir string) {
    entries, err := os.ReadDir(dir)
    if err != nil {
        return // directory does not exist
    }
    
    for _, entry := range entries {
        if entry.IsDir() {
            continue
        }
        
        name := entry.Name()
        if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
            _ = os.Remove(filepath.Join(dir, name))
        }
    }
}

// getManagementWorkflowPath returns workflow path for management operations
func (s *Server) getManagementWorkflowPath() string {
    // Prefers configured path, falls back to /app/workflow.yaml (Docker)
    // Then falls back to workflow.yaml (local)
}
```

---

## 9. Agency.yml Implementation Recommendations

Based on the codebase analysis, here's how to implement agency.yml:

### Proposed agency.yml Structure:
```yaml
apiVersion: kdeps.io/v1
kind: Agency
metadata:
  name: my-agency
  description: Multi-agent orchestration
  version: 1.0.0
settings:
  coordinationMode: sequential|parallel|reactive
  sharedSettings:
    apiServerMode: true
    portNum: 16395
    agentSettings:
      timezone: UTC
agents:
  - name: agent-1
    path: agents/agent-1/workflow.yaml  # or agents/agent-1 (auto-discover)
    settings:
      # Override shared settings for this agent
  - name: agent-2
    path: agents/agent-2
coordination:
  # Define message passing, dependencies, etc.
```

### Implementation Steps:

1. **Create Agency struct** in `pkg/domain/workflow.go`:
   - Similar to Workflow struct
   - Include agents array with agent references
   - Support coordination configuration

2. **Extend schema** in `pkg/validator/schemas/`:
   - Create `agency.json` schema
   - Define agent reference structure
   - Validate coordination rules

3. **Update parser** in `pkg/parser/yaml/parser.go`:
   - Add `ParseAgency(path string)` method
   - Support `agents/**/workflow.yaml` pattern matching
   - Merge shared + agent-specific settings

4. **Update FindWorkflowFile** in `cmd/run.go`:
   - Add support for `agency.yaml` / `agency.yml`
   - Return path to agency file if found

5. **Create new runner** in `cmd/`:
   - `RunAgencyWithFlags()` - orchestrate multiple agents
   - `ExecuteAgencyStepsWithFlags()` - 5-step pipeline for agency
   - Support coordination modes (sequential, parallel, reactive)

6. **Update templates**:
   - Add agency template: `pkg/templates/templates/agency/`
   - Generate example agency.yml structure

7. **Resource discovery**:
   - Pattern: `agents/*/workflow.yaml`
   - Merge resources from multiple agents
   - Handle resource ID conflicts

### Key Considerations:

- **Backward compatibility:** Keep single-agent workflow.yaml fully functional
- **Auto-discovery:** Support both explicit paths and glob patterns like `agents/**/workflow.yaml`
- **Settings inheritance:** Agents inherit from shared settings but can override
- **Resource isolation:** Each agent has its own resources/ directory
- **Execution model:** Support sequential (one agent at a time), parallel (concurrent), reactive (event-driven)
- **Message passing:** Define how agents communicate (likely through shared actions)
- **Validation:** Extend workflow validator to check agency dependencies

---

## 10. Current "Workflows" Field Usage

**Unused field discovered:**
```go
type WorkflowMetadata struct {
    Workflows []string `yaml:"workflows,omitempty"`
}
```

This field is **declared but not used anywhere** in the codebase. This appears to be a placeholder for multi-workflow support that was partially implemented. Perfect foundation for agency.yml!

---

## Summary of Key Files for Agency.yml Implementation

| File | Purpose | Lines |
|------|---------|-------|
| `pkg/domain/workflow.go` | Add Agency struct | ~917 lines |
| `pkg/parser/yaml/parser.go` | Add ParseAgency method | 287 lines |
| `cmd/run.go` | Add RunAgencyWithFlags | 1569 lines |
| `pkg/validator/schemas/agency.json` | NEW - Agency schema | - |
| `cmd/agency.go` | NEW - Agency command | - |
| `pkg/templates/templates/agency/` | NEW - Agency templates | - |

---

## File Locations (Absolute Paths for Reference)

- `/home/runner/work/kdeps/kdeps/pkg/domain/workflow.go` - Domain structs
- `/home/runner/work/kdeps/kdeps/cmd/run.go` - Execution orchestration
- `/home/runner/work/kdeps/kdeps/pkg/parser/yaml/parser.go` - YAML parser
- `/home/runner/work/kdeps/kdeps/pkg/validator/schemas/workflow.json` - Schema
- `/home/runner/work/kdeps/kdeps/cmd/scaffold.go` - Resource scaffolding
- `/home/runner/work/kdeps/kdeps/cmd/new.go` - Project generation
- `/home/runner/work/kdeps/kdeps/pkg/templates/templates/` - Jinja2 templates
