package tool_test

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/pkg/tool"
)

func TestPklResourceReader(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	scriptDir := filepath.Join(tmpDir, "scripts")

	// Create script directory
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		t.Fatalf("Failed to create script directory: %v", err)
	}

	// Create test scripts
	createTestScript := func(name, content string) string {
		scriptPath := filepath.Join(scriptDir, name)
		if err := os.WriteFile(scriptPath, []byte(content), 0o755); err != nil {
			t.Fatalf("Failed to create test script %s: %v", name, err)
		}
		return scriptPath
	}

	// Create test scripts
	pythonScript := createTestScript("test.py", "print('Hello from Python')")
	jsScript := createTestScript("test.js", "console.log('Hello from JavaScript')")
	rubyScript := createTestScript("test.rb", "puts 'Hello from Ruby'")
	shellScript := createTestScript("test.sh", "echo 'Hello from Shell'")
	errorScript := createTestScript("test_error.sh", "exit 1")
	invalidScript := createTestScript("test.invalid", "invalid content")

	// Initialize database with test data
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create tables
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS tools (
			id TEXT PRIMARY KEY,
			value TEXT
		);
		CREATE TABLE IF NOT EXISTS history (
			id TEXT,
			timestamp INTEGER NOT NULL,
			value TEXT
		);
	`); err != nil {
		t.Fatalf("Failed to create tables: %v", err)
	}

	// Insert test data
	testData := []struct {
		id    string
		value string
	}{
		{"test1", "output1"},
		{"test2", "output2"},
		{"test3", "output3"},
		{"test4", "output4"},
		{"test5", "output5"},
		{"test6", "output6"},
	}

	for _, data := range testData {
		if _, err := db.Exec("INSERT INTO tools (id, value) VALUES (?, ?)", data.id, data.value); err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}
	}

	// Create reader with the test database
	reader := &tool.PklResourceReader{
		DB: db,
	}

	t.Run("Scheme", func(t *testing.T) {
		if reader.Scheme() != "tool" {
			t.Errorf("Expected scheme 'tool', got '%s'", reader.Scheme())
		}
	})

	t.Run("IsGlobbable", func(t *testing.T) {
		if reader.IsGlobbable() {
			t.Error("Expected IsGlobbable to return false")
		}
	})

	t.Run("HasHierarchicalUris", func(t *testing.T) {
		if reader.HasHierarchicalUris() {
			t.Error("Expected HasHierarchicalUris to return false")
		}
	})

	t.Run("ListElements", func(t *testing.T) {
		uri, _ := url.Parse("tool:///")
		elements, err := reader.ListElements(*uri)
		if err != nil {
			t.Errorf("ListElements failed: %v", err)
		}
		if len(elements) != 0 {
			t.Errorf("Expected 0 elements, got %d", len(elements))
		}
	})

	t.Run("Read_NilDB", func(t *testing.T) {
		nilDBReader := &tool.PklResourceReader{
			DBPath: dbPath,
		}
		uri, _ := url.Parse("tool:///test1")
		output, err := nilDBReader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, "output1", string(output))
	})

	t.Run("Read_GetItem", func(t *testing.T) {
		// Test successful read
		uri, _ := url.Parse("tool:///test1")
		output, err := reader.Read(*uri)
		if err != nil {
			t.Errorf("Read failed: %v", err)
		}
		if string(output) != "output1" {
			t.Errorf("Expected output 'output1', got '%s'", string(output))
		}

		// Test nonexistent item
		uri, _ = url.Parse("tool:///nonexistent")
		output, err = reader.Read(*uri)
		if err != nil {
			t.Errorf("Did not expect error for nonexistent item, got: %v", err)
		}
		if string(output) != "" {
			t.Errorf("Expected empty output for nonexistent item, got '%s'", string(output))
		}

		// Test empty ID
		uri, _ = url.Parse("tool:///")
		_, err = reader.Read(*uri)
		if err == nil {
			t.Error("Expected error for empty ID")
		}
	})

	t.Run("Read_Run_InlineScript", func(t *testing.T) {
		// Test with URL-encoded parameters
		uri, _ := url.Parse("tool:///test4?op=run&script=echo%20hello&params=param1%20param2")
		output, err := reader.Read(*uri)
		if err != nil {
			t.Errorf("Read failed: %v", err)
		}
		if strings.TrimSpace(string(output)) != "hello" {
			t.Errorf("Expected output 'hello', got '%s'", string(output))
		}

		// Test with empty parameters
		uri, _ = url.Parse("tool:///test4?op=run&script=echo%20hello")
		output, err = reader.Read(*uri)
		if err != nil {
			t.Errorf("Read failed: %v", err)
		}
		if strings.TrimSpace(string(output)) != "hello" {
			t.Errorf("Expected output 'hello', got '%s'", string(output))
		}

		// Test with invalid URL encoding (should not error, just pass empty string)
		uri, _ = url.Parse("tool:///test4?op=run&script=echo%20hello&params=%")
		output, err = reader.Read(*uri)
		if err != nil {
			t.Errorf("Read failed for invalid URL encoding: %v", err)
		}
		if strings.TrimSpace(string(output)) != "hello" {
			t.Errorf("Expected output 'hello' for invalid URL encoding, got '%s'", string(output))
		}

		// Test with empty params after trimming
		uri, _ = url.Parse("tool:///test4?op=run&script=echo%20hello&params=%20%20%20")
		output, err = reader.Read(*uri)
		if err != nil {
			t.Errorf("Read failed for empty params after trimming: %v", err)
		}
		if strings.TrimSpace(string(output)) != "hello" {
			t.Errorf("Expected output 'hello' for empty params after trimming, got '%s'", string(output))
		}
	})

	t.Run("Read_Run_FileScript", func(t *testing.T) {
		t.Run("Python_script", func(t *testing.T) {
			uri, _ := url.Parse(fmt.Sprintf("tool:///test_py?op=run&script=%s&params=param1%%20param2", pythonScript))
			output, err := reader.Read(*uri)
			if err != nil {
				t.Errorf("Read failed: %v", err)
			}
			if !strings.Contains(string(output), "Hello from Python") {
				t.Errorf("Expected output to contain 'Hello from Python', got '%s'", string(output))
			}
		})

		t.Run("JavaScript_script", func(t *testing.T) {
			uri, _ := url.Parse(fmt.Sprintf("tool:///test_js?op=run&script=%s&params=param1%%20param2", jsScript))
			output, err := reader.Read(*uri)
			if err != nil {
				t.Errorf("Read failed: %v", err)
			}
			if !strings.Contains(string(output), "Hello from JavaScript") {
				t.Errorf("Expected output to contain 'Hello from JavaScript', got '%s'", string(output))
			}
		})

		t.Run("Ruby_script", func(t *testing.T) {
			uri, _ := url.Parse(fmt.Sprintf("tool:///test_rb?op=run&script=%s&params=param1%%20param2", rubyScript))
			output, err := reader.Read(*uri)
			if err != nil {
				t.Errorf("Read failed: %v", err)
			}
			if !strings.Contains(string(output), "Hello from Ruby") {
				t.Errorf("Expected output to contain 'Hello from Ruby', got '%s'", string(output))
			}
		})

		t.Run("Shell_script", func(t *testing.T) {
			uri, _ := url.Parse(fmt.Sprintf("tool:///test_sh?op=run&script=%s&params=param1%%20param2", shellScript))
			output, err := reader.Read(*uri)
			if err != nil {
				t.Errorf("Read failed: %v", err)
			}
			if !strings.Contains(string(output), "Hello from Shell") {
				t.Errorf("Expected output to contain 'Hello from Shell', got '%s'", string(output))
			}
		})

		t.Run("InvalidScriptFile", func(t *testing.T) {
			uri, _ := url.Parse("tool:///test_invalid?op=run&script=/nonexistent/script.sh")
			output, err := reader.Read(*uri)
			if err != nil {
				t.Errorf("Did not expect error for invalid script file, got: %v", err)
			}
			if !strings.Contains(strings.ToLower(string(output)), "no such file or directory") && !strings.Contains(strings.ToLower(string(output)), "not found") {
				t.Errorf("Expected error message in output for invalid script file, got '%s'", string(output))
			}
		})

		t.Run("ScriptExecutionError", func(t *testing.T) {
			uri, _ := url.Parse(fmt.Sprintf("tool:///test_error?op=run&script=%s", errorScript))
			output, err := reader.Read(*uri)
			if err != nil {
				t.Errorf("Did not expect error for script execution failure, got: %v", err)
			}
			if strings.TrimSpace(string(output)) != "" {
				t.Errorf("Expected empty output for script execution failure, got '%s'", string(output))
			}
		})

		t.Run("InvalidInterpreter", func(t *testing.T) {
			uri, _ := url.Parse(fmt.Sprintf("tool:///test_invalid_interpreter?op=run&script=%s", invalidScript))
			output, err := reader.Read(*uri)
			if err != nil {
				t.Errorf("Did not expect error for invalid interpreter, got: %v", err)
			}
			if !strings.Contains(string(output), "not found") {
				t.Errorf("Expected error message in output for invalid interpreter, got '%s'", string(output))
			}
		})
	})

	t.Run("Read_Run_InterpreterNotFound", func(t *testing.T) {
		// Test with a non-existent interpreter
		uri, _ := url.Parse("tool:///test4?op=run&script=test.fake")
		output, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Contains(t, strings.ToLower(string(output)), "not found")
	})

	t.Run("Read_History", func(t *testing.T) {
		// First run a script to create some history
		uri, _ := url.Parse("tool:///history_test?op=run&script=echo%20test%20history")
		_, err := reader.Read(*uri)
		require.NoError(t, err)

		// Now test history retrieval
		uri, _ = url.Parse("tool:///history_test?op=history")
		output, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Contains(t, string(output), "test history")

		// Test history for non-existent ID
		uri, _ = url.Parse("tool:///nonexistent_history?op=history")
		output, err = reader.Read(*uri)
		require.NoError(t, err)
		require.Empty(t, string(output))
	})

	t.Run("Read_Run_InvalidParamsEncoding", func(t *testing.T) {
		// Create a mock DB that fails RowsAffected
		mockDB := newMockDB()
		mockDB.db.Exec(`CREATE TABLE IF NOT EXISTS tools (id TEXT PRIMARY KEY, value TEXT)`)
		mockDB.db.Exec(`CREATE TABLE IF NOT EXISTS history (id TEXT, value TEXT, timestamp INTEGER)`)

		// Create a mock result that fails RowsAffected
		mockResult := &mockResult{rowsAffectedErr: errors.New("mock rows affected error")}
		mockDB.execFunc = func(_ string, _ ...interface{}) (sql.Result, error) {
			return mockResult, nil
		}

		mockReader := &tool.PklResourceReader{DB: mockDB.db}
		uri, _ := url.Parse("tool:///test4?op=run&script=echo&params=%ZZ")
		output, err := mockReader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, "\n", string(output))
	})

	t.Run("Read_Run_SQLExecFails", func(t *testing.T) {
		// Create a mock DB that fails on Exec
		mockDB := newMockDB()
		mockDB.execFunc = func(_ string, _ ...interface{}) (sql.Result, error) {
			return nil, errors.New("mock exec error")
		}

		mockReader := &tool.PklResourceReader{DB: mockDB.db}
		uri, _ := url.Parse("tool:///failtest?op=run&script=echo")
		_, err := mockReader.Read(*uri)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to store script output")
	})

	t.Run("Read_History_SQLQueryFails", func(t *testing.T) {
		// Create a mock DB that fails Query
		mockDB := newMockDB()
		mockDB.queryFunc = func(_ string, _ ...interface{}) (*sql.Rows, error) {
			return nil, errors.New("mock query error")
		}

		mockReader := &tool.PklResourceReader{DB: mockDB.db}
		uri, _ := url.Parse("tool:///test?op=history")
		_, err := mockReader.Read(*uri)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to retrieve history")
	})

	t.Run("Read_InvalidURL", func(t *testing.T) {
		reader := &tool.PklResourceReader{}
		invalidURL := url.URL{Scheme: "invalid", Path: "//test"}
		output, err := reader.Read(invalidURL)
		require.NoError(t, err)
		require.Empty(t, string(output))
	})

	t.Run("Read_MissingOperation", func(t *testing.T) {
		reader := &tool.PklResourceReader{}
		uri := url.URL{
			Scheme:   "tool",
			Path:     "/test",
			RawQuery: "script=echo",
		}
		result, err := reader.Read(uri)
		if err != nil {
			t.Errorf("Expected no error for missing operation, got: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("Expected empty result for missing operation, got: %v", result)
		}
	})

	t.Run("Read_InvalidOperation", func(t *testing.T) {
		reader := &tool.PklResourceReader{}
		testURL := url.URL{Scheme: "tool", Path: "//test", RawQuery: "op=invalid"}
		output, err := reader.Read(testURL)
		require.NoError(t, err)
		require.Empty(t, string(output))
	})

	t.Run("Read_Run_MissingScript", func(t *testing.T) {
		reader := &tool.PklResourceReader{}
		testURL := url.URL{Scheme: "tool", Path: "//test", RawQuery: "op=run"}
		_, err := reader.Read(testURL)
		if err == nil {
			t.Error("Expected error for missing script")
		}
	})

	t.Run("Read_Run_ScriptExecutionTimeout", func(t *testing.T) {
		reader := &tool.PklResourceReader{}
		testURL := url.URL{Scheme: "tool", Path: "//test", RawQuery: "op=run&script=sleep 10"}
		output, err := reader.Read(testURL)
		require.NoError(t, err)
		require.Empty(t, string(output))
	})

	t.Run("Read_Run_ScriptWithInvalidParams", func(t *testing.T) {
		reader := &tool.PklResourceReader{}
		testURL := url.URL{Scheme: "tool", Path: "//test", RawQuery: "op=run&script=echo&params=param1 param2 param3"}
		_, err := reader.Read(testURL)
		if err != nil {
			t.Errorf("Unexpected error for valid params: %v", err)
		}
	})

	t.Run("Read_History_InvalidID", func(t *testing.T) {
		reader := &tool.PklResourceReader{}
		uri := url.URL{
			Scheme:   "tool",
			Path:     "/",
			RawQuery: "op=history",
		}
		result, err := reader.Read(uri)
		require.Error(t, err)
		require.Empty(t, string(result))
	})
}

// Mock interfaces for testing
type mockResult struct {
	rowsAffectedErr error
}

func (m *mockResult) LastInsertId() (int64, error) { return 0, nil }
func (m *mockResult) RowsAffected() (int64, error) { return 0, m.rowsAffectedErr }

type mockDB struct {
	db        *sql.DB
	execFunc  func(query string, args ...interface{}) (sql.Result, error)
	queryFunc func(query string, args ...interface{}) (*sql.Rows, error)
}

func newMockDB() *mockDB {
	db, _ := sql.Open("sqlite3", ":memory:")
	return &mockDB{db: db}
}

func (m *mockDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	if m.execFunc != nil {
		return m.execFunc(query, args...)
	}
	return m.db.Exec(query, args...)
}

func (m *mockDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	if m.queryFunc != nil {
		return m.queryFunc(query, args...)
	}
	return m.db.Query(query, args...)
}

func (m *mockDB) QueryRow(query string, args ...interface{}) *sql.Row {
	return m.db.QueryRow(query, args...)
}

func (m *mockDB) Close() error { return m.db.Close() }
func (m *mockDB) Ping() error  { return m.db.Ping() }

// mockRows implements the Rows interface for testing
type mockRows struct {
	nextFunc  func() bool
	scanFunc  func(dest ...interface{}) error
	errFunc   func() error
	closeFunc func() error
}

func (m *mockRows) Next() bool {
	return m.nextFunc()
}

func (m *mockRows) Scan(dest ...interface{}) error {
	return m.scanFunc(dest...)
}

func (m *mockRows) Err() error {
	return m.errFunc()
}

func (m *mockRows) Close() error {
	return m.closeFunc()
}

func TestInitializeTool(t *testing.T) {
	// Create a temporary directory for the test database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Test successful initialization
	reader, err := tool.InitializeTool(dbPath)
	if err != nil {
		t.Errorf("InitializeTool failed: %v", err)
	}
	if reader == nil {
		t.Error("InitializeTool returned nil reader")
	}
	if reader.DB == nil {
		t.Error("InitializeTool returned reader with nil DB")
	}
	if reader.DBPath != dbPath {
		t.Errorf("Expected DBPath %s, got %s", dbPath, reader.DBPath)
	}

	// Test initialization with invalid path
	_, err = tool.InitializeTool("/nonexistent/path/test.db")
	if err == nil {
		t.Error("Expected error for invalid path")
	}
}

func TestInitializeDatabaseAndHistory(t *testing.T) {
	// Create temp dir and DB path using afero.NewOsFs
	fs := afero.NewOsFs()
	tmpDir, err := afero.TempDir(fs, "", "tooldb")
	if err != nil {
		t.Fatalf("TempDir error: %v", err)
	}
	dbPath := filepath.Join(tmpDir, "kdeps_tool.db")

	// Initialize the reader (this implicitly calls InitializeDatabase).
	reader, err := tool.InitializeTool(dbPath)
	if err != nil {
		t.Fatalf("InitializeTool error: %v", err)
	}
	defer reader.DB.Close()

	// Manually insert a couple of history rows.
	now := time.Now().Unix()
	_, err = reader.DB.Exec("INSERT INTO history (id, value, timestamp) VALUES (?, ?, ?)", "someid", "hello-1", now)
	if err != nil {
		t.Fatalf("insert history err: %v", err)
	}
	_, err = reader.DB.Exec("INSERT INTO history (id, value, timestamp) VALUES (?, ?, ?)", "someid", "hello-2", now+1)
	if err != nil {
		t.Fatalf("insert history err: %v", err)
	}

	// Request history via the reader.Read API.
	uri := url.URL{Scheme: "tool", Path: "/someid", RawQuery: "op=history"}
	out, err := reader.Read(uri)
	if err != nil {
		t.Fatalf("Read history error: %v", err)
	}

	got := string(out)
	if !strings.Contains(got, "hello-1") || !strings.Contains(got, "hello-2") {
		t.Fatalf("unexpected history output: %s", got)
	}
}

// TestInitializeDatabaseFailure exercises the retry + failure branch by pointing
// the DB path into a directory that is not writable, ensuring all attempts fail.
func TestInitializeDatabaseFailure(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.MkdirAll(tmpDir, 0o755)
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Make the directory read-only to cause database initialization to fail
	err = os.Chmod(tmpDir, 0o444)
	require.NoError(t, err)

	dbPath := filepath.Join(tmpDir, "tool.db")
	_, err = tool.InitializeDatabase(dbPath)
	if err == nil {
		t.Fatalf("expected error when initializing DB in read-only dir")
	}
}
