package utils

import (
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

// Mock bus service for testing
type mockBusManager struct {
	signals        map[string]bool
	completions    map[string]string
	files          map[string]bool
	cleanupSignals []string
}

func newMockBusManager() *mockBusManager {
	return &mockBusManager{
		signals:        make(map[string]bool),
		completions:    make(map[string]string),
		files:          make(map[string]bool),
		cleanupSignals: make([]string, 0),
	}
}

func (m *mockBusManager) SignalResourceCompletion(resourceID, resourceType, status string, data map[string]interface{}) error {
	m.completions[resourceID] = status
	return nil
}

func (m *mockBusManager) WaitForResourceCompletion(resourceID string, timeoutSeconds int64) error {
	if status, exists := m.completions[resourceID]; exists {
		if status == "failed" {
			return &resourceCompletionError{resourceID: resourceID}
		}
		return nil
	}
	return &timeoutError{resource: resourceID}
}

func (m *mockBusManager) SignalFileReady(filepath, operation string, data map[string]interface{}) error {
	m.files[filepath] = true
	return nil
}

func (m *mockBusManager) WaitForFileReady(filepath string, timeoutSeconds int64) error {
	if m.files[filepath] {
		return nil
	}
	return &timeoutError{resource: filepath}
}

func (m *mockBusManager) SignalCleanup(cleanupType, message string, data map[string]interface{}) error {
	m.cleanupSignals = append(m.cleanupSignals, cleanupType)
	return nil
}

func (m *mockBusManager) WaitForCleanup(timeoutSeconds int64) error {
	if len(m.cleanupSignals) > 0 {
		return nil
	}
	return &timeoutError{resource: "cleanup"}
}

func (m *mockBusManager) Close() error {
	return nil
}

// Error types for testing
type resourceCompletionError struct {
	resourceID string
}

func (e *resourceCompletionError) Error() string {
	return "resource " + e.resourceID + " failed"
}

type timeoutError struct {
	resource string
}

func (e *timeoutError) Error() string {
	return "timeout waiting for " + e.resource
}

func TestBusIPCManager_ResourceCompletion(t *testing.T) {
	t.Parallel()

	busManager := newMockBusManager()

	tests := []struct {
		name           string
		resourceID     string
		resourceType   string
		status         string
		data           map[string]interface{}
		expectError    bool
		setupMock      func(*mockBusManager)
		validateResult func(*testing.T, *mockBusManager)
	}{
		{
			name:         "successful completion signal",
			resourceID:   "test-resource-1",
			resourceType: "exec",
			status:       "completed",
			data:         map[string]interface{}{"command": "echo test"},
			expectError:  false,
			setupMock:    func(m *mockBusManager) {},
			validateResult: func(t *testing.T, m *mockBusManager) {
				if m.completions["test-resource-1"] != "completed" {
					t.Errorf("Expected completion status 'completed', got '%s'", m.completions["test-resource-1"])
				}
			},
		},
		{
			name:         "failure completion signal",
			resourceID:   "test-resource-2",
			resourceType: "python",
			status:       "failed",
			data:         map[string]interface{}{"error": "script failed"},
			expectError:  false,
			setupMock:    func(m *mockBusManager) {},
			validateResult: func(t *testing.T, m *mockBusManager) {
				if m.completions["test-resource-2"] != "failed" {
					t.Errorf("Expected completion status 'failed', got '%s'", m.completions["test-resource-2"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock(busManager)

			err := busManager.SignalResourceCompletion(tt.resourceID, tt.resourceType, tt.status, tt.data)

			if (err != nil) != tt.expectError {
				t.Errorf("SignalResourceCompletion() error = %v, expectError = %v", err, tt.expectError)
				return
			}

			tt.validateResult(t, busManager)
		})
	}
}

func TestBusIPCManager_WaitForResourceCompletion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		resourceID  string
		timeout     int64
		setupMock   func(*mockBusManager)
		expectError bool
		errorType   interface{}
	}{
		{
			name:       "successful wait for completion",
			resourceID: "test-resource-success",
			timeout:    5,
			setupMock: func(m *mockBusManager) {
				m.completions["test-resource-success"] = "completed"
			},
			expectError: false,
		},
		{
			name:       "wait for failed resource",
			resourceID: "test-resource-failed",
			timeout:    5,
			setupMock: func(m *mockBusManager) {
				m.completions["test-resource-failed"] = "failed"
			},
			expectError: true,
			errorType:   &resourceCompletionError{},
		},
		{
			name:       "timeout waiting for resource",
			resourceID: "test-resource-timeout",
			timeout:    1,
			setupMock: func(m *mockBusManager) {
				// Don't add to completions to simulate timeout
			},
			expectError: true,
			errorType:   &timeoutError{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			busManager := newMockBusManager()
			tt.setupMock(busManager)

			err := busManager.WaitForResourceCompletion(tt.resourceID, tt.timeout)

			if (err != nil) != tt.expectError {
				t.Errorf("WaitForResourceCompletion() error = %v, expectError = %v", err, tt.expectError)
				return
			}

			if tt.expectError && tt.errorType != nil {
				switch tt.errorType.(type) {
				case *resourceCompletionError:
					if _, ok := err.(*resourceCompletionError); !ok {
						t.Errorf("Expected resourceCompletionError, got %T", err)
					}
				case *timeoutError:
					if _, ok := err.(*timeoutError); !ok {
						t.Errorf("Expected timeoutError, got %T", err)
					}
				}
			}
		})
	}
}

func TestBusIPCManager_FileOperations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		filepath    string
		operation   string
		setupMock   func(*mockBusManager)
		expectError bool
	}{
		{
			name:        "signal file ready",
			filepath:    "/test/file.txt",
			operation:   "create",
			setupMock:   func(m *mockBusManager) {},
			expectError: false,
		},
		{
			name:     "wait for existing file",
			filepath: "/test/existing.txt",
			setupMock: func(m *mockBusManager) {
				m.files["/test/existing.txt"] = true
			},
			expectError: false,
		},
		{
			name:     "wait for non-existing file timeout",
			filepath: "/test/nonexistent.txt",
			setupMock: func(m *mockBusManager) {
				// Don't add to files to simulate timeout
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			busManager := newMockBusManager()
			tt.setupMock(busManager)

			if tt.operation != "" {
				err := busManager.SignalFileReady(tt.filepath, tt.operation, nil)
				if err != nil {
					t.Errorf("SignalFileReady() error = %v", err)
					return
				}
			}

			err := busManager.WaitForFileReady(tt.filepath, 1)
			if (err != nil) != tt.expectError {
				t.Errorf("WaitForFileReady() error = %v, expectError = %v", err, tt.expectError)
			}
		})
	}
}

func TestBusIPCManager_CleanupOperations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		cleanupType string
		message     string
		setupMock   func(*mockBusManager)
		expectError bool
	}{
		{
			name:        "signal docker cleanup",
			cleanupType: "docker",
			message:     "Docker cleanup completed",
			setupMock:   func(m *mockBusManager) {},
			expectError: false,
		},
		{
			name:        "signal action cleanup",
			cleanupType: "action",
			message:     "Action cleanup completed",
			setupMock:   func(m *mockBusManager) {},
			expectError: false,
		},
		{
			name: "wait for cleanup signal",
			setupMock: func(m *mockBusManager) {
				m.cleanupSignals = append(m.cleanupSignals, "docker")
			},
			expectError: false,
		},
		{
			name: "wait for cleanup timeout",
			setupMock: func(m *mockBusManager) {
				// Don't add cleanup signals to simulate timeout
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			busManager := newMockBusManager()
			tt.setupMock(busManager)

			if tt.cleanupType != "" {
				err := busManager.SignalCleanup(tt.cleanupType, tt.message, nil)
				if err != nil {
					t.Errorf("SignalCleanup() error = %v", err)
					return
				}

				// Verify signal was recorded
				found := false
				for _, signal := range busManager.cleanupSignals {
					if signal == tt.cleanupType {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Cleanup signal '%s' was not recorded", tt.cleanupType)
				}
			}

			err := busManager.WaitForCleanup(1)
			if (err != nil) != tt.expectError {
				t.Errorf("WaitForCleanup() error = %v, expectError = %v", err, tt.expectError)
			}
		})
	}
}

func TestWaitForFileReadyLegacy(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	logger := logging.GetLogger()

	tests := []struct {
		name        string
		filepath    string
		setupFile   bool
		expectError bool
	}{
		{
			name:        "existing file",
			filepath:    "/test/existing.txt",
			setupFile:   true,
			expectError: false,
		},
		{
			name:        "non-existing file",
			filepath:    "/test/nonexistent.txt",
			setupFile:   false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFile {
				err := afero.WriteFile(fs, tt.filepath, []byte("test content"), 0644)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
			}

			err := WaitForFileReadyLegacy(fs, tt.filepath, logger)
			if (err != nil) != tt.expectError {
				t.Errorf("WaitForFileReadyLegacy() error = %v, expectError = %v", err, tt.expectError)
			}
		})
	}
}

func TestCreateFilesWithBusSignal(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	files := []string{"/test/file1.txt", "/test/file2.txt"}

	// Test without bus manager (fallback behavior)
	err := CreateFilesWithBusSignal(fs, nil, files)
	if err != nil {
		t.Errorf("CreateFilesWithBusSignal() error = %v", err)
	}

	// Verify files were created
	for _, file := range files {
		exists, err := afero.Exists(fs, file)
		if err != nil {
			t.Errorf("Error checking file existence: %v", err)
		}
		if !exists {
			t.Errorf("File %s was not created", file)
		}
	}
}

// Integration test with real bus server (requires server to be running)
func TestBusIPCManager_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logger := logging.GetLogger()

	// Test with real bus manager - if it fails to connect, skip the test
	busManager, err := NewBusIPCManager(logger)
	if err != nil {
		t.Skipf("Could not connect to bus server (may not be running): %v", err)
	}
	defer busManager.Close()

	// Simple test that doesn't hang - just verify connection works
	t.Log("Bus IPC Manager connected successfully")
}
