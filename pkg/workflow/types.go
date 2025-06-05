package workflow

import (
	"context"
	"time"
)

// Workflow represents a complete workflow configuration
type Workflow struct {
	Name        string
	Description string
	Version     string
	Settings    *WorkflowSettings
	Resources   []*Resource
}

// WorkflowSettings contains all the settings for a workflow
type WorkflowSettings struct {
	APIServerMode bool
	APIServer     *APIServerSettings
}

// APIServerSettings contains the API server configuration
type APIServerSettings struct {
	HostIP  string
	PortNum int
	Routes  []*Route
}

// Route represents an API route configuration
type Route struct {
	Path    string
	Methods []string
}

// Resource represents a single resource in the workflow
type Resource struct {
	ActionID    string
	Name        string
	Description string
	Category    string
	Requires    []string
	Run         *ResourceRun
}

// ResourceRun contains the execution configuration for a resource
type ResourceRun struct {
	RestrictToHTTPMethods []string
	RestrictToRoutes      []string
	AllowedHeaders        []string
	AllowedParams         []string
	SkipCondition         string
	PreflightCheck        *PreflightCheck
	Python                *PythonAction
	Exec                  *ExecAction
	HTTPClient            *HTTPClientAction
	LLM                   *LLMAction
	APIResponse           *APIResponseAction
}

// PreflightCheck contains validation rules for a resource
type PreflightCheck struct {
	Validations []string
	Error       *ErrorResponse
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Code    int
	Message string
}

// PythonAction represents a Python script execution
type PythonAction struct {
	CondaEnvironment string
	Script           string
	Env              map[string]string
	TimeoutDuration  time.Duration
}

// ExecAction represents a shell command execution
type ExecAction struct {
	Command         string
	Env             map[string]string
	TimeoutDuration time.Duration
}

// HTTPClientAction represents an HTTP client request
type HTTPClientAction struct {
	Method          string
	URL             string
	Data            map[string]interface{}
	Headers         map[string]string
	TimeoutDuration time.Duration
}

// LLMAction represents an LLM chat session
type LLMAction struct {
	Model            string
	Role             string
	Prompt           string
	Scenario         []*LLMScenario
	JSONResponse     bool
	JSONResponseKeys []string
	Files            []string
	TimeoutDuration  time.Duration
}

// LLMScenario represents a single scenario in an LLM chat session
type LLMScenario struct {
	Role   string
	Prompt string
}

// APIResponseAction represents an API response
type APIResponseAction struct {
	Success  bool
	Meta     *APIResponseMeta
	Response *APIResponseData
	Errors   []*ErrorResponse
}

// APIResponseMeta contains metadata for an API response
type APIResponseMeta struct {
	Headers    map[string]string
	Properties map[string]string
}

// APIResponseData contains the actual response data
type APIResponseData struct {
	Data []interface{}
}

// ResourceHandler defines the interface for handling different types of resources
type ResourceHandler interface {
	HandleExec(ctx context.Context, resource *Resource) error
	HandleHTTPClient(ctx context.Context, resource *Resource) error
	HandlePython(ctx context.Context, resource *Resource) error
	HandleLLM(ctx context.Context, resource *Resource) error
}
