package resolver_test

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/kdeps/schema/gen/data"
	"github.com/spf13/afero"
)

// MockContext implements context.Context.
type MockContext struct{}

func (mc *MockContext) Deadline() (time.Time, bool) {
	return time.Now(), false
}

func (mc *MockContext) Done() <-chan struct{} {
	return nil
}

func (mc *MockContext) Err() error {
	return nil
}

func (mc *MockContext) Value(key interface{}) interface{} {
	return nil
}

func TestAppendDataEntry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		setup         func(dr *resolver.DependencyResolver) *data.DataImpl
		expectError   bool
		expectedError string
	}{
		{
			name: "Context is nil",
			setup: func(dr *resolver.DependencyResolver) *data.DataImpl {
				//nolint:fatcontext
				dr.Context = ctx
				return nil
			},
			expectError:   true,
			expectedError: "context is nil",
		},
		{
			name: "PKL file load failure",
			setup: func(dr *resolver.DependencyResolver) *data.DataImpl {
				if err := afero.WriteFile(dr.Fs, filepath.Join(dr.ActionDir, "data", dr.RequestID+"__data_output.pkl"), []byte("invalid content"), 0o644); err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				return nil
			},
			expectError:   true,
			expectedError: "failed to load PKL file",
		},
		{
			name: "New data is nil",
			setup: func(dr *resolver.DependencyResolver) *data.DataImpl {
				return nil
			},
			expectError:   true,
			expectedError: "",
		},
		{
			name: "Valid data merge",
			setup: func(dr *resolver.DependencyResolver) *data.DataImpl {
				files := map[string]map[string]string{
					"agent1": {
						"file1": "content1",
					},
				}
				return &data.DataImpl{
					Files: &files,
				}
			},
			expectError:   true,
			expectedError: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			dr := &resolver.DependencyResolver{
				Fs:        afero.NewMemMapFs(),
				Context:   &MockContext{},
				ActionDir: "action",
				RequestID: "testRequestID",
				Logger:    logging.GetLogger(),
			}

			newData := test.setup(dr)
			err := dr.AppendDataEntry("testResourceID", newData)

			if test.expectError {
				if err == nil || !strings.Contains(err.Error(), test.expectedError) {
					t.Fatalf("expected error: %s, got: %v", test.expectedError, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				// Verify the written file exists
				pklPath := filepath.Join(dr.ActionDir, "data", dr.RequestID+"__data_output.pkl")
				_, err := afero.ReadFile(dr.Fs, pklPath)
				if err != nil {
					t.Fatalf("file not written: %v", err)
				}
			}
		})
	}
}
