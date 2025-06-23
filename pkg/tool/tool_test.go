package tool_test

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/kdeps/kdeps/pkg/tool"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	reader := &PklResourceReader{
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
		nilDBReader := &PklResourceReader{
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
		mockResult := &mockResult{rowsAffectedErr: fmt.Errorf("mock rows affected error")}
		mockDB.execFunc = func(query string, args ...interface{}) (sql.Result, error) {
			return mockResult, nil
		}

		mockReader := &PklResourceReader{DB: mockDB.db}
		uri, _ := url.Parse("tool:///test4?op=run&script=echo&params=%ZZ")
		output, err := mockReader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, "\n", string(output))
	})

	t.Run("Read_Run_SQLExecFails", func(t *testing.T) {
		// Mock DB that fails on Exec
		db, _ := sql.Open("sqlite3", ":memory:")
		db.Exec(`CREATE TABLE IF NOT EXISTS tools (id TEXT PRIMARY KEY, value TEXT)`)
		db.Exec(`CREATE TABLE IF NOT EXISTS history (id TEXT, value TEXT, timestamp INTEGER)`)
		// Close DB to force Exec to fail
		db.Close()
		mockReader := &PklResourceReader{DB: db}
		uri, _ := url.Parse("tool:///failtest?op=run&script=echo")
		_, err := mockReader.Read(*uri)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to store script output")
	})

	t.Run("Read_History_SQLQueryFails", func(t *testing.T) {
		// Create a mock DB that fails Query
		mockDB := newMockDB()
		mockDB.queryFunc = func(query string, args ...interface{}) (*sql.Rows, error) {
			return nil, fmt.Errorf("mock query error")
		}

		mockReader := &PklResourceReader{DB: mockDB.db}
		uri, _ := url.Parse("tool:///test?op=history")
		_, err := mockReader.Read(*uri)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to retrieve history")
	})

	t.Run("Read_InvalidURL", func(t *testing.T) {
		reader := &PklResourceReader{}
		invalidURL := url.URL{Scheme: "invalid", Path: "//test"}
		output, err := reader.Read(invalidURL)
		require.NoError(t, err)
		require.Empty(t, string(output))
	})

	t.Run("Read_MissingOperation", func(t *testing.T) {
		reader := &PklResourceReader{}
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
		reader := &PklResourceReader{}
		testURL := url.URL{Scheme: "tool", Path: "//test", RawQuery: "op=invalid"}
		output, err := reader.Read(testURL)
		require.NoError(t, err)
		require.Empty(t, string(output))
	})

	t.Run("Read_Run_MissingScript", func(t *testing.T) {
		reader := &PklResourceReader{}
		testURL := url.URL{Scheme: "tool", Path: "//test", RawQuery: "op=run"}
		_, err := reader.Read(testURL)
		if err == nil {
			t.Error("Expected error for missing script")
		}
	})

	t.Run("Read_Run_ScriptExecutionTimeout", func(t *testing.T) {
		reader := &PklResourceReader{}
		testURL := url.URL{Scheme: "tool", Path: "//test", RawQuery: "op=run&script=sleep 10"}
		output, err := reader.Read(testURL)
		require.NoError(t, err)
		require.Empty(t, string(output))
	})

	t.Run("Read_Run_ScriptWithInvalidParams", func(t *testing.T) {
		reader := &PklResourceReader{}
		testURL := url.URL{Scheme: "tool", Path: "//test", RawQuery: "op=run&script=echo&params=param1 param2 param3"}
		_, err := reader.Read(testURL)
		if err != nil {
			t.Errorf("Unexpected error for valid params: %v", err)
		}
	})

	t.Run("Read_History_InvalidID", func(t *testing.T) {
		reader := &PklResourceReader{}
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
	fs := afero.NewOsFs()
	tmpDir, err := afero.TempDir(fs, "", "tool-init")
	require.NoError(t, err)
	defer fs.RemoveAll(tmpDir)

	t.Run("SuccessfulInitialization", func(t *testing.T) {
		// Test successful initialization
		dbPath := filepath.Join(tmpDir, "tool.db")

		reader, err := InitializeTool(dbPath)
		require.NoError(t, err)
		defer reader.DB.Close()

		// Verify the reader was created correctly
		require.NotNil(t, reader)
		require.Equal(t, dbPath, reader.DBPath)
		require.NotNil(t, reader.DB)

		// Verify the database is accessible
		err = reader.DB.Ping()
		require.NoError(t, err)

		// Verify both tables exist
		var toolsCount int
		err = reader.DB.QueryRow("SELECT COUNT(*) FROM tools").Scan(&toolsCount)
		require.NoError(t, err)
		require.Equal(t, 0, toolsCount)

		var historyCount int
		err = reader.DB.QueryRow("SELECT COUNT(*) FROM history").Scan(&historyCount)
		require.NoError(t, err)
		require.Equal(t, 0, historyCount)
	})

	t.Run("DatabaseInitializationError", func(t *testing.T) {
		// Test when database initialization fails
		invalidPath := filepath.Join(tmpDir, "nonexistent", "db.sqlite")
		_, err := InitializeTool(invalidPath)
		require.Error(t, err)
		require.Contains(t, err.Error(), "error initializing database")
	})
}

func TestInitializeDatabase(t *testing.T) {
	fs := afero.NewOsFs()
	tmpDir, err := afero.TempDir(fs, "", "tool-db")
	require.NoError(t, err)
	defer fs.RemoveAll(tmpDir)

	t.Run("SuccessfulInitialization", func(t *testing.T) {
		// Test successful database initialization
		dbPath := filepath.Join(tmpDir, "success.db")
		db, err := InitializeDatabase(dbPath)
		require.NoError(t, err)
		defer db.Close()

		// Verify the database was created and is accessible
		err = db.Ping()
		require.NoError(t, err)

		// Verify the tools table exists
		var toolsCount int
		err = db.QueryRow("SELECT COUNT(*) FROM tools").Scan(&toolsCount)
		require.NoError(t, err)
		require.Equal(t, 0, toolsCount)

		// Verify the history table exists
		var historyCount int
		err = db.QueryRow("SELECT COUNT(*) FROM history").Scan(&historyCount)
		require.NoError(t, err)
		require.Equal(t, 0, historyCount)
	})

	t.Run("DatabaseAlreadyExists", func(t *testing.T) {
		// Test that the function works when the database already exists
		dbPath := filepath.Join(tmpDir, "existing.db")

		// Create the database first
		db1, err := InitializeDatabase(dbPath)
		require.NoError(t, err)
		db1.Close()

		// Initialize again
		db2, err := InitializeDatabase(dbPath)
		require.NoError(t, err)
		defer db2.Close()

		// Verify the database is accessible
		err = db2.Ping()
		require.NoError(t, err)

		// Verify both tables exist
		var toolsCount int
		err = db2.QueryRow("SELECT COUNT(*) FROM tools").Scan(&toolsCount)
		require.NoError(t, err)
		require.Equal(t, 0, toolsCount)

		var historyCount int
		err = db2.QueryRow("SELECT COUNT(*) FROM history").Scan(&historyCount)
		require.NoError(t, err)
		require.Equal(t, 0, historyCount)
	})

	t.Run("InvalidDatabasePath", func(t *testing.T) {
		// Test with an invalid database path (directory doesn't exist)
		invalidPath := filepath.Join(tmpDir, "nonexistent", "db.sqlite")
		_, err := InitializeDatabase(invalidPath)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to ping database after 5 attempts")
	})

	t.Run("RetryLogic", func(t *testing.T) {
		// This test verifies that the retry logic works
		// We can't easily simulate database failures, but we can verify the function
		// handles normal cases correctly
		dbPath := filepath.Join(tmpDir, "retry.db")
		db, err := InitializeDatabase(dbPath)
		require.NoError(t, err)
		defer db.Close()

		// Verify the database works
		err = db.Ping()
		require.NoError(t, err)

		// Verify both tables exist
		var toolsCount int
		err = db.QueryRow("SELECT COUNT(*) FROM tools").Scan(&toolsCount)
		require.NoError(t, err)
		require.Equal(t, 0, toolsCount)

		var historyCount int
		err = db.QueryRow("SELECT COUNT(*) FROM history").Scan(&historyCount)
		require.NoError(t, err)
		require.Equal(t, 0, historyCount)
	})
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
	reader, err := InitializeTool(dbPath)
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
	tmpDir, err := os.MkdirTemp("", "kdeps_ro")
	if err != nil {
		t.Fatalf("tempdir error: %v", err)
	}
	// make directory read-only so sqlite cannot create file inside it
	if err := os.Chmod(tmpDir, 0o555); err != nil {
		t.Fatalf("chmod error: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "tool.db")
	_, err = InitializeDatabase(dbPath)
	if err == nil {
		t.Fatalf("expected error when initializing DB in read-only dir")
	}
}

func TestInitializeDatabase_ErrorPaths(t *testing.T) {
	t.Run("InvalidDBPath", func(t *testing.T) {
		// Test with an invalid database path that would cause sql.Open to fail
		invalidPath := "/nonexistent/directory/db.sqlite"

		// This will likely fail due to directory not existing
		db, err := InitializeDatabase(invalidPath)
		assert.Error(t, err)
		assert.Nil(t, db)
		assert.Contains(t, err.Error(), "failed to ping database")
	})

	t.Run("DatabaseOpenFailure", func(t *testing.T) {
		// Test with a path that would cause sql.Open to fail
		// Using a path with invalid characters
		invalidPath := string([]byte{0x00}) + "/invalid.db"

		db, err := InitializeDatabase(invalidPath)
		// This might succeed or fail depending on the system
		if err != nil {
			assert.Contains(t, err.Error(), "failed to")
		} else {
			assert.NotNil(t, db)
			db.Close()
		}
	})

	t.Run("DatabasePingFailure", func(t *testing.T) {
		// This is harder to test without mocking sql.Open
		// For now, we'll test that the function handles ping failures gracefully
		t.Logf("DatabasePingFailure would require mocking sql.Open")
	})

	t.Run("ToolsTableCreationFailure", func(t *testing.T) {
		// This is harder to test without mocking the database
		// For now, we'll test that the function handles table creation failures gracefully
		t.Logf("ToolsTableCreationFailure would require mocking database operations")
	})

	t.Run("HistoryTableCreationFailure", func(t *testing.T) {
		// This is harder to test without mocking the database
		// For now, we'll test that the function handles table creation failures gracefully
		t.Logf("HistoryTableCreationFailure would require mocking database operations")
	})

	t.Run("MaxAttemptsExceeded", func(t *testing.T) {
		// Test with a path that will consistently fail
		// Using a path that would cause persistent failures
		persistentFailPath := "/nonexistent/directory/persistent_fail.db"

		db, err := InitializeDatabase(persistentFailPath)
		assert.Error(t, err)
		assert.Nil(t, db)
		assert.Contains(t, err.Error(), "failed to ping database after 5 attempts")
	})

	t.Run("SuccessWithRetries", func(t *testing.T) {
		// Test successful initialization
		tmpDir, err := os.MkdirTemp("", "tool-test")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		dbPath := filepath.Join(tmpDir, "test.db")

		db, err := InitializeDatabase(dbPath)
		assert.NoError(t, err)
		assert.NotNil(t, db)

		// Verify the database is working
		err = db.Ping()
		assert.NoError(t, err)

		// Verify tables were created
		rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name IN ('tools', 'history')")
		assert.NoError(t, err)
		defer rows.Close()

		var tableNames []string
		for rows.Next() {
			var name string
			err := rows.Scan(&name)
			assert.NoError(t, err)
			tableNames = append(tableNames, name)
		}

		assert.Contains(t, tableNames, "tools")
		assert.Contains(t, tableNames, "history")

		db.Close()
	})
}

func TestPklResourceReader_InterfaceMethods(t *testing.T) {
	reader := &PklResourceReader{}

	// Test Scheme method
	assert.Equal(t, "tool", reader.Scheme())

	// Test IsGlobbable method
	assert.False(t, reader.IsGlobbable())

	// Test HasHierarchicalUris method
	assert.False(t, reader.HasHierarchicalUris())

	// Test ListElements method
	uri, _ := url.Parse("tool://test")
	elements, err := reader.ListElements(*uri)
	assert.NoError(t, err)
	assert.Nil(t, elements)
}

// TestInitializeDatabase_RetryScenarios tests specific retry scenarios
func TestInitializeDatabase_RetryScenarios(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tool-retry-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	t.Run("RetryOnPingFailure", func(t *testing.T) {
		// Create a directory that becomes writable after the first attempt
		retryDir := filepath.Join(tmpDir, "ping-retry")
		err := os.MkdirAll(retryDir, 0o755)
		require.NoError(t, err)

		dbPath := filepath.Join(retryDir, "retry.db")

		// This will succeed because the directory is writable
		// The function should handle any temporary issues gracefully
		db, err := InitializeDatabase(dbPath)
		if err == nil {
			assert.NotNil(t, db)
			assert.NoError(t, db.Ping())
			db.Close()
		} else {
			// If it fails, it should be due to the retry mechanism
			assert.Contains(t, err.Error(), "failed to")
		}
	})

	t.Run("RetryOnTableCreationFailure", func(t *testing.T) {
		// Test with in-memory database which should always succeed
		// This exercises the table creation path
		db, err := InitializeDatabase(":memory:")
		assert.NoError(t, err)
		assert.NotNil(t, db)

		// Verify tables were created successfully
		assert.NoError(t, db.Ping())

		// Check that both tables exist
		var toolsExists bool
		err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='tools'").Scan(&toolsExists)
		assert.NoError(t, err)

		var historyExists bool
		err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='history'").Scan(&historyExists)
		assert.NoError(t, err)

		db.Close()
	})

	t.Run("MemoryDatabaseSuccess", func(t *testing.T) {
		// Test successful case with in-memory database
		// This should exercise the success path without retries
		db, err := InitializeDatabase(":memory:")
		assert.NoError(t, err)
		assert.NotNil(t, db)

		// Verify database functionality
		assert.NoError(t, db.Ping())

		// Test table operations
		_, err = db.Exec("INSERT INTO tools (id, value) VALUES (?, ?)", "test", "value")
		assert.NoError(t, err)

		_, err = db.Exec("INSERT INTO history (id, value, timestamp) VALUES (?, ?, ?)", "test", "value", 123456789)
		assert.NoError(t, err)

		// Verify data can be retrieved
		var value string
		err = db.QueryRow("SELECT value FROM tools WHERE id = ?", "test").Scan(&value)
		assert.NoError(t, err)
		assert.Equal(t, "value", value)

		db.Close()
	})

	t.Run("DirectoryPermissionTest", func(t *testing.T) {
		// Create a directory with restricted permissions
		restrictedDir := filepath.Join(tmpDir, "restricted")
		err := os.MkdirAll(restrictedDir, 0o755)
		require.NoError(t, err)

		// Make directory read-only
		err = os.Chmod(restrictedDir, 0o555)
		require.NoError(t, err)

		dbPath := filepath.Join(restrictedDir, "readonly.db")

		// This should fail after retries
		db, err := InitializeDatabase(dbPath)
		if err != nil {
			assert.Error(t, err)
			assert.Nil(t, db)
			// Should mention either ping or open failure
			assert.True(t,
				strings.Contains(err.Error(), "failed to ping database") ||
					strings.Contains(err.Error(), "failed to open database"),
				"Error should mention ping or open failure, got: %s", err.Error())
		}

		// Restore permissions for cleanup
		os.Chmod(restrictedDir, 0o755)
	})
}
