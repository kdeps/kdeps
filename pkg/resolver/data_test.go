package resolver

import (
	"encoding/base64"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
	apiserverresponse "github.com/kdeps/schema/gen/api_server_response"
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
	tests := []struct {
		name          string
		setup         func(dr *DependencyResolver) *data.DataImpl
		expectError   bool
		expectedError string
	}{
		{
			name: "Context is nil",
			setup: func(dr *DependencyResolver) *data.DataImpl {
				dr.Context = nil
				return nil
			},
			expectError:   true,
			expectedError: "context is nil",
		},
		{
			name: "PKL file load failure",
			setup: func(dr *DependencyResolver) *data.DataImpl {
				if err := afero.WriteFile(dr.Fs, filepath.Join(dr.ActionDir, "data", dr.RequestID+"__data_output.pkl"), []byte("invalid content"), 0o644); err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				// Return valid data to trigger PKL loading, which should fail due to invalid content
				files := map[string]map[string]string{
					"agent1": {
						"file1": "content1",
					},
				}
				return &data.DataImpl{
					Files: &files,
				}
			},
			expectError:   false,
			expectedError: "",
		},
		{
			name: "New data is nil",
			setup: func(dr *DependencyResolver) *data.DataImpl {
				return nil
			},
			expectError:   true,
			expectedError: "",
		},
		{
			name: "Valid data merge",
			setup: func(dr *DependencyResolver) *data.DataImpl {
				files := map[string]map[string]string{
					"agent1": {
						"file1": "content1",
					},
				}
				return &data.DataImpl{
					Files: &files,
				}
			},
			expectError:   false,
			expectedError: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tmp := t.TempDir()
			actionDir := filepath.Join(tmp, "action")
			fs := afero.NewOsFs()
			_ = fs.MkdirAll(filepath.Join(actionDir, "data"), 0o755)

			dr := &DependencyResolver{
				Fs:        fs,
				Context:   &MockContext{},
				ActionDir: actionDir,
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

func TestFormatDataValue(t *testing.T) {
	// Simple string value should embed JSONRenderDocument lines
	out := formatDataValue("hello")
	if !strings.Contains(out, "JSONRenderDocument") {
		t.Errorf("expected JSONRenderDocument in output, got %s", out)
	}

	// Map value path should still produce block
	m := map[string]interface{}{"k": "v"}
	out2 := formatDataValue(m)
	if !strings.Contains(out2, "k") {
		t.Errorf("map key lost in formatting: %s", out2)
	}
}

func TestFormatErrorsMultiple(t *testing.T) {
	logger := logging.NewTestLogger()
	msg := base64.StdEncoding.EncodeToString([]byte("decoded msg"))
	errorsSlice := &[]*apiserverresponse.APIServerErrorsBlock{
		{Code: 400, Message: "bad"},
		{Code: 500, Message: msg},
	}
	out := formatErrors(errorsSlice, logger)
	if !strings.Contains(out, "Code = 400") || !strings.Contains(out, "Code = 500") {
		t.Errorf("codes missing: %s", out)
	}
	if !strings.Contains(out, "decoded msg") {
		t.Errorf("base64 not decoded: %s", out)
	}
}

// TestFormatValueVariantsBasic exercises several branches of the reflection-based
// formatValue helper to bump coverage and guard against panics when handling
// diverse inputs.
func TestFormatValueVariantsBasic(t *testing.T) {
	type custom struct{ X string }

	variants := []interface{}{
		nil,
		map[string]interface{}{"k": "v"},
		custom{X: "val"},
	}

	for _, v := range variants {
		out := formatValue(v)
		if out == "" {
			t.Errorf("formatValue produced empty output for %+v", v)
		}
	}
}
