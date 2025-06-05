package workflow

import (
	"context"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockResourceHandler is a mock implementation of ResourceHandler
type MockResourceHandler struct {
	mock.Mock
}

func (m *MockResourceHandler) HandleExec(ctx context.Context, resource *Resource) error {
	args := m.Called(ctx, resource)
	return args.Error(0)
}

func (m *MockResourceHandler) HandleHTTPClient(ctx context.Context, resource *Resource) error {
	args := m.Called(ctx, resource)
	return args.Error(0)
}

func (m *MockResourceHandler) HandlePython(ctx context.Context, resource *Resource) error {
	args := m.Called(ctx, resource)
	return args.Error(0)
}

func (m *MockResourceHandler) HandleLLM(ctx context.Context, resource *Resource) error {
	args := m.Called(ctx, resource)
	return args.Error(0)
}

func TestLoadWorkflow_Error(t *testing.T) {
	ctx := context.Background()
	logger := logging.NewTestLogger()
	_, err := LoadWorkflow(ctx, "/tmp/doesnotexist.pkl", logger)
	assert.Error(t, err)
}

func TestWorkflow_Execute(t *testing.T) {
	tests := []struct {
		name          string
		workflow      *Workflow
		setupMocks    func(*MockResourceHandler)
		expectedError bool
	}{
		{
			name: "successful execution of all resource types",
			workflow: &Workflow{
				Name:        "test-workflow",
				Description: "Test workflow",
				Version:     "1.0.0",
				Resources: []*Resource{
					{
						ActionID:    "pythonResource",
						Name:        "Python Resource",
						Description: "Python script execution",
						Run: &ResourceRun{
							Python: &PythonAction{
								Script:          "print('hello')",
								TimeoutDuration: 60 * time.Second,
							},
						},
					},
					{
						ActionID:    "execResource",
						Name:        "Exec Resource",
						Description: "Shell command execution",
						Run: &ResourceRun{
							Exec: &ExecAction{
								Command:         "echo hello",
								TimeoutDuration: 60 * time.Second,
							},
						},
					},
					{
						ActionID:    "httpResource",
						Name:        "HTTP Resource",
						Description: "HTTP request",
						Run: &ResourceRun{
							HTTPClient: &HTTPClientAction{
								Method:          "GET",
								URL:             "http://example.com",
								TimeoutDuration: 60 * time.Second,
							},
						},
					},
					{
						ActionID:    "llmResource",
						Name:        "LLM Resource",
						Description: "LLM chat",
						Run: &ResourceRun{
							LLM: &LLMAction{
								Model:           "test-model",
								Prompt:          "Hello",
								TimeoutDuration: 60 * time.Second,
							},
						},
					},
				},
			},
			setupMocks: func(m *MockResourceHandler) {
				m.On("HandlePython", mock.Anything, mock.Anything).Return(nil)
				m.On("HandleExec", mock.Anything, mock.Anything).Return(nil)
				m.On("HandleHTTPClient", mock.Anything, mock.Anything).Return(nil)
				m.On("HandleLLM", mock.Anything, mock.Anything).Return(nil)
			},
			expectedError: false,
		},
		{
			name: "failed Python resource execution",
			workflow: &Workflow{
				Name:        "test-workflow",
				Description: "Test workflow",
				Version:     "1.0.0",
				Resources: []*Resource{
					{
						ActionID:    "pythonResource",
						Name:        "Python Resource",
						Description: "Python script execution",
						Run: &ResourceRun{
							Python: &PythonAction{
								Script:          "print('hello')",
								TimeoutDuration: 60 * time.Second,
							},
						},
					},
				},
			},
			setupMocks: func(m *MockResourceHandler) {
				m.On("HandlePython", mock.Anything, mock.Anything).Return(assert.AnError)
			},
			expectedError: true,
		},
		{
			name: "invalid workflow - missing version",
			workflow: &Workflow{
				Name:        "test-workflow",
				Description: "Test workflow",
				Resources: []*Resource{
					{
						ActionID:    "pythonResource",
						Name:        "Python Resource",
						Description: "Python script execution",
						Run: &ResourceRun{
							Python: &PythonAction{
								Script:          "print('hello')",
								TimeoutDuration: 60 * time.Second,
							},
						},
					},
				},
			},
			setupMocks:    func(m *MockResourceHandler) {},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHandler := new(MockResourceHandler)
			tt.setupMocks(mockHandler)

			err := tt.workflow.Execute(context.Background(), mockHandler)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				mockHandler.AssertExpectations(t)
			}
		})
	}
}

func TestWorkflow_Validate(t *testing.T) {
	tests := []struct {
		name      string
		workflow  *Workflow
		wantError bool
	}{
		{
			name: "valid workflow",
			workflow: &Workflow{
				Name:        "test-workflow",
				Description: "Test workflow",
				Version:     "1.0.0",
				Settings: &WorkflowSettings{
					APIServerMode: true,
					APIServer: &APIServerSettings{
						HostIP:  "127.0.0.1",
						PortNum: 3000,
					},
				},
			},
			wantError: false,
		},
		{
			name: "missing version",
			workflow: &Workflow{
				Name:        "test-workflow",
				Description: "Test workflow",
				Settings: &WorkflowSettings{
					APIServerMode: true,
					APIServer: &APIServerSettings{
						HostIP:  "127.0.0.1",
						PortNum: 3000,
					},
				},
			},
			wantError: true,
		},
		{
			name: "invalid port number",
			workflow: &Workflow{
				Name:        "test-workflow",
				Description: "Test workflow",
				Version:     "1.0.0",
				Settings: &WorkflowSettings{
					APIServerMode: true,
					APIServer: &APIServerSettings{
						HostIP:  "127.0.0.1",
						PortNum: 70000, // Invalid port number
					},
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.workflow.Validate()
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestResource_Validate(t *testing.T) {
	tests := []struct {
		name      string
		resource  *Resource
		wantError bool
	}{
		{
			name: "valid Python resource",
			resource: &Resource{
				ActionID:    "pythonResource",
				Name:        "Python Resource",
				Description: "Python script execution",
				Run: &ResourceRun{
					Python: &PythonAction{
						Script:          "print('hello')",
						TimeoutDuration: 60 * time.Second,
					},
				},
			},
			wantError: false,
		},
		{
			name: "valid Exec resource",
			resource: &Resource{
				ActionID:    "execResource",
				Name:        "Exec Resource",
				Description: "Shell command execution",
				Run: &ResourceRun{
					Exec: &ExecAction{
						Command:         "echo hello",
						TimeoutDuration: 60 * time.Second,
					},
				},
			},
			wantError: false,
		},
		{
			name: "valid HTTP resource",
			resource: &Resource{
				ActionID:    "httpResource",
				Name:        "HTTP Resource",
				Description: "HTTP request",
				Run: &ResourceRun{
					HTTPClient: &HTTPClientAction{
						Method:          "GET",
						URL:             "http://example.com",
						TimeoutDuration: 60 * time.Second,
					},
				},
			},
			wantError: false,
		},
		{
			name: "valid LLM resource",
			resource: &Resource{
				ActionID:    "llmResource",
				Name:        "LLM Resource",
				Description: "LLM chat",
				Run: &ResourceRun{
					LLM: &LLMAction{
						Model:           "test-model",
						Prompt:          "Hello",
						TimeoutDuration: 60 * time.Second,
					},
				},
			},
			wantError: false,
		},
		{
			name: "missing action ID",
			resource: &Resource{
				Name:        "Python Resource",
				Description: "Python script execution",
				Run: &ResourceRun{
					Python: &PythonAction{
						Script:          "print('hello')",
						TimeoutDuration: 60 * time.Second,
					},
				},
			},
			wantError: true,
		},
		{
			name: "missing run block",
			resource: &Resource{
				ActionID:    "pythonResource",
				Name:        "Python Resource",
				Description: "Python script execution",
			},
			wantError: true,
		},
		{
			name: "no action defined",
			resource: &Resource{
				ActionID:    "pythonResource",
				Name:        "Python Resource",
				Description: "Python script execution",
				Run:         &ResourceRun{},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.resource.Validate()
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestResourceRun_Validate(t *testing.T) {
	tests := []struct {
		name      string
		run       *ResourceRun
		wantError bool
	}{
		{
			name: "valid Python action",
			run: &ResourceRun{
				Python: &PythonAction{
					Script:          "print('hello')",
					TimeoutDuration: 60 * time.Second,
				},
			},
			wantError: false,
		},
		{
			name: "valid Exec action",
			run: &ResourceRun{
				Exec: &ExecAction{
					Command:         "echo hello",
					TimeoutDuration: 60 * time.Second,
				},
			},
			wantError: false,
		},
		{
			name: "valid HTTP action",
			run: &ResourceRun{
				HTTPClient: &HTTPClientAction{
					Method:          "GET",
					URL:             "http://example.com",
					TimeoutDuration: 60 * time.Second,
				},
			},
			wantError: false,
		},
		{
			name: "valid LLM action",
			run: &ResourceRun{
				LLM: &LLMAction{
					Model:           "test-model",
					Prompt:          "Hello",
					TimeoutDuration: 60 * time.Second,
				},
			},
			wantError: false,
		},
		{
			name: "valid API response action",
			run: &ResourceRun{
				APIResponse: &APIResponseAction{
					Success: true,
					Meta: &APIResponseMeta{
						Headers: map[string]string{},
					},
					Response: &APIResponseData{
						Data: []interface{}{},
					},
				},
			},
			wantError: false,
		},
		{
			name:      "no action defined",
			run:       &ResourceRun{},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run.Validate()
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
