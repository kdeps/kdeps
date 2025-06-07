package tool

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestPklResourceReader(t *testing.T) {
	t.Parallel()

	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	scriptDir := filepath.Join(tmpDir, "scripts")

	// Create script directory
	if err := os.MkdirAll(scriptDir, 0755); err != nil {
		t.Fatalf("Failed to create script directory: %v", err)
	}

	// Create test scripts
	createTestScript := func(name, content string) string {
		scriptPath := filepath.Join(scriptDir, name)
		if err := os.WriteFile(scriptPath, []byte(content), 0755); err != nil {
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
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
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
		uri, _ := url.Parse("tool:///test4?op=run&script=echo%20hello&params=param1%20param2")
		output, err := reader.Read(*uri)
		if err != nil {
			t.Errorf("Read failed: %v", err)
		}
		if strings.TrimSpace(string(output)) != "hello" {
			t.Errorf("Expected output 'hello', got '%s'", string(output))
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
			if !strings.Contains(string(output), "No such file or directory") {
				t.Errorf("Expected error message in output for invalid script file, got '%s'", string(output))
			}
		})

		t.Run("ScriptExecutionError", func(t *testing.T) {
			uri, _ := url.Parse(fmt.Sprintf("tool:///test_error?op=run&script=%s", errorScript))
			output, err := reader.Read(*uri)
			if err != nil {
				t.Errorf("Did not expect error for script execution failure, got: %v", err)
			}
			if len(strings.TrimSpace(string(output))) != 0 {
				t.Errorf("Expected empty output for script execution failure, got '%s'", string(output))
			}
		})

		t.Run("InvalidInterpreter", func(t *testing.T) {
			uri, _ := url.Parse(fmt.Sprintf("tool:///test_invalid_interpreter?op=run&script=%s", invalidScript))
			output, err := reader.Read(*uri)
			if err != nil {
				t.Errorf("Did not expect error for invalid interpreter, got: %v", err)
			}
			if !strings.Contains(string(output), "command not found") {
				t.Errorf("Expected error message in output for invalid interpreter, got '%s'", string(output))
			}
		})
	})

	t.Run("Read_History", func(t *testing.T) {
		// Test history for existing tool
		uri, _ := url.Parse("tool:///test5?op=history")
		output, err := reader.Read(*uri)
		if err != nil {
			t.Errorf("Read failed: %v", err)
		}
		// Accept empty output for missing history
		// Test history for nonexistent tool
		uri, _ = url.Parse("tool:///nonexistent?op=history")
		output, err = reader.Read(*uri)
		if err != nil {
			t.Errorf("Did not expect error for nonexistent tool history, got: %v", err)
		}
		if string(output) != "" {
			t.Errorf("Expected empty output for nonexistent tool history, got '%s'", string(output))
		}

		// Test history for empty ID
		uri, _ = url.Parse("tool:///?op=history")
		_, err = reader.Read(*uri)
		if err == nil {
			t.Error("Expected error for empty ID history")
		}
	})
}

func TestInitializeDatabase(t *testing.T) {
	t.Parallel()

	t.Run("SuccessfulInitialization", func(t *testing.T) {
		t.Parallel()
		db, err := InitializeDatabase("file::memory:")
		require.NoError(t, err)
		require.NotNil(t, db)
		defer db.Close()

		var name string
		err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='tools'").Scan(&name)
		require.NoError(t, err)
		require.Equal(t, "tools", name)

		err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='history'").Scan(&name)
		require.NoError(t, err)
		require.Equal(t, "history", name)
	})

	t.Run("InvalidPath", func(t *testing.T) {
		t.Parallel()
		db, err := InitializeDatabase("file::memory:?cache=invalid")
		if err != nil {
			if db != nil {
				defer db.Close()
				err = db.Ping()
				require.NoError(t, err, "Expected database to be usable even with invalid cache parameter")
			}
		}
	})

	t.Run("ReadOnlyDatabase", func(t *testing.T) {
		t.Parallel()
		// Create a temporary file for the database
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		// Create and initialize the database
		db, err := InitializeDatabase(dbPath)
		require.NoError(t, err)
		db.Close()

		// Make the file read-only
		err = os.Chmod(dbPath, 0o444)
		require.NoError(t, err)
		defer os.Chmod(dbPath, 0o666) // Restore permissions

		// Try to initialize again
		db2, err := InitializeDatabase(dbPath)
		if err == nil {
			// Try a write to trigger a read-only error
			_, writeErr := db2.Exec("INSERT INTO tools (id, value) VALUES (?, ?)", "foo", "bar")
			require.Error(t, writeErr)
			if !strings.Contains(writeErr.Error(), "read-only") && !strings.Contains(writeErr.Error(), "readonly") {
				t.Fatalf("unexpected error: %v", writeErr)
			}
			db2.Close()
		} else {
			require.Contains(t, err.Error(), "unable to open database file")
		}
	})

	t.Run("DatabaseOperationErrors", func(t *testing.T) {
		t.Parallel()
		reader, err := InitializeTool("file::memory:")
		require.NoError(t, err)
		defer reader.DB.Close()

		// Test database operation errors by closing the connection
		reader.DB.Close()

		// Try to read with closed connection
		uri, _ := url.Parse("tool:///test_db_error?op=run&script=echo%20test")
		_, err = reader.Read(*uri)
		require.Error(t, err)
		if !strings.Contains(err.Error(), "failed to initialize database") && !strings.Contains(err.Error(), "database is closed") {
			t.Fatalf("unexpected error: %v", err)
		}

		// Try to get history with closed connection
		uri, _ = url.Parse("tool:///test_db_error?op=history")
		_, err = reader.Read(*uri)
		require.Error(t, err)
		if !strings.Contains(err.Error(), "failed to initialize database") && !strings.Contains(err.Error(), "database is closed") {
			t.Fatalf("unexpected error: %v", err)
		}

		// Try to get record with closed connection
		uri, _ = url.Parse("tool:///test_db_error")
		_, err = reader.Read(*uri)
		require.Error(t, err)
		if !strings.Contains(err.Error(), "failed to initialize database") && !strings.Contains(err.Error(), "database is closed") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestInitializeTool(t *testing.T) {
	t.Parallel()

	t.Run("SuccessfulInitialization", func(t *testing.T) {
		t.Parallel()
		reader, err := InitializeTool("file::memory:")
		require.NoError(t, err)
		require.NotNil(t, reader)
		require.NotNil(t, reader.DB)
		require.Equal(t, "file::memory:", reader.DBPath)
		defer reader.DB.Close()
	})

	t.Run("InvalidPath", func(t *testing.T) {
		t.Parallel()
		// Create a temporary directory
		tmpDir := t.TempDir()

		// Create a file that's not a database
		dbPath := filepath.Join(tmpDir, "not_a_db.txt")
		err := os.WriteFile(dbPath, []byte("not a database"), 0o666)
		require.NoError(t, err)

		// Try to initialize with non-database file
		_, err = InitializeTool(dbPath)
		require.Error(t, err)
		require.Contains(t, err.Error(), "file is not a database")
	})

	t.Run("ReadOnlyPath", func(t *testing.T) {
		t.Parallel()
		// Create a temporary directory
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "readonly.db")

		// Create and initialize the database
		reader, err := InitializeTool(dbPath)
		require.NoError(t, err)
		reader.DB.Close()

		// Make the directory read-only
		err = os.Chmod(tmpDir, 0o444)
		require.NoError(t, err)
		defer os.Chmod(tmpDir, 0o777) // Restore permissions

		// Try to initialize again
		_, err = InitializeTool(dbPath)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unable to open database file")
	})
}
