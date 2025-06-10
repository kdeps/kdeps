package tool

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

	t.Run("Read_NilReceiver", func(t *testing.T) {
		var nilReader *PklResourceReader
		uri, _ := url.Parse("tool:///test1")
		_, err := nilReader.Read(*uri)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to initialize PklResourceReader")
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
		// Add some history entries
		now := time.Now().Unix()
		historyData := []struct {
			id        string
			value     string
			timestamp int64
		}{
			{"test5", "output1", now - 2},
			{"test5", "output2", now - 1},
			{"test5", "output3", now},
		}

		for _, data := range historyData {
			if _, err := db.Exec(
				"INSERT INTO history (id, value, timestamp) VALUES (?, ?, ?)",
				data.id, data.value, data.timestamp,
			); err != nil {
				t.Fatalf("Failed to insert history data: %v", err)
			}
		}

		// Test history for existing tool
		uri, _ := url.Parse("tool:///test5?op=history")
		output, err := reader.Read(*uri)
		if err != nil {
			t.Errorf("Read failed: %v", err)
		}
		outputStr := string(output)
		require.Contains(t, outputStr, "output1")
		require.Contains(t, outputStr, "output2")
		require.Contains(t, outputStr, "output3")

		// Test history for nonexistent tool
		uri, _ = url.Parse("tool:///nonexistent?op=history")
		output, err = reader.Read(*uri)
		if err != nil {
			t.Errorf("Read failed: %v", err)
		}
		if string(output) != "" {
			t.Errorf("Expected empty output for nonexistent tool history, got '%s'", string(output))
		}

		// Test empty ID
		uri, _ = url.Parse("tool:///?op=history")
		_, err = reader.Read(*uri)
		if err == nil {
			t.Error("Expected error for empty ID")
		}

		// Test database error during history query
		// Close the database connection to simulate an error
		db.Close()
		uri, _ = url.Parse("tool:///test5?op=history")
		_, err = reader.Read(*uri)
		if err == nil {
			t.Error("Expected error for closed database connection")
		}
	})
}

// MockDBInitializer implements DBInitializer for testing
type MockDBInitializer struct {
	OpenFunc  func(dsn string) (*sql.DB, error)
	PingFunc  func(db *sql.DB) error
	ExecFunc  func(db *sql.DB, query string) error
	OpenCount int
	PingCount int
	ExecCount int
}

func (m *MockDBInitializer) Open(dsn string) (*sql.DB, error) {
	m.OpenCount++
	if m.OpenFunc != nil {
		return m.OpenFunc(dsn)
	}
	return nil, fmt.Errorf("Open not implemented")
}

func (m *MockDBInitializer) Ping(db *sql.DB) error {
	m.PingCount++
	if m.PingFunc != nil {
		return m.PingFunc(db)
	}
	return fmt.Errorf("Ping not implemented")
}

func (m *MockDBInitializer) Exec(db *sql.DB, query string) error {
	m.ExecCount++
	if m.ExecFunc != nil {
		return m.ExecFunc(db, query)
	}
	return fmt.Errorf("Exec not implemented")
}

func TestInitializeDatabase(t *testing.T) {
	t.Run("Successful Initialization", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")
		db, err := InitializeDatabase(dbPath, &DefaultDBInitializer{})
		require.NoError(t, err)
		require.NotNil(t, db)
		defer db.Close()

		// Verify tables exist
		rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table'")
		require.NoError(t, err)
		defer rows.Close()

		var tables []string
		for rows.Next() {
			var name string
			require.NoError(t, rows.Scan(&name))
			tables = append(tables, name)
		}
		require.NoError(t, rows.Err())
		require.Contains(t, tables, "tools")
		require.Contains(t, tables, "history")
	})

	t.Run("Failing to Open Database", func(t *testing.T) {
		mock := &MockDBInitializer{
			OpenFunc: func(dsn string) (*sql.DB, error) {
				// Return a valid DB but always error
				db, _ := sql.Open("sqlite3", ":memory:")
				return db, fmt.Errorf("mock open error")
			},
		}
		db, err := InitializeDatabase("test.db", mock)
		require.Error(t, err)
		require.Nil(t, db)
		require.Contains(t, err.Error(), "failed to open database after 5 attempts")
		require.Equal(t, 5, mock.OpenCount)
	})

	t.Run("Failing to Ping Database", func(t *testing.T) {
		mock := &MockDBInitializer{
			OpenFunc: func(dsn string) (*sql.DB, error) {
				// Return a valid but unusable DB
				db, _ := sql.Open("sqlite3", ":memory:")
				return db, nil
			},
			PingFunc: func(db *sql.DB) error {
				return fmt.Errorf("mock ping error")
			},
		}
		db, err := InitializeDatabase("test.db", mock)
		require.Error(t, err)
		require.Nil(t, db)
		require.Contains(t, err.Error(), "failed to ping database after 5 attempts")
		require.Equal(t, 5, mock.OpenCount)
		require.Equal(t, 5, mock.PingCount)
	})

	t.Run("Failing to Create Tools Table", func(t *testing.T) {
		mock := &MockDBInitializer{
			OpenFunc: func(dsn string) (*sql.DB, error) {
				db, _ := sql.Open("sqlite3", ":memory:")
				return db, nil
			},
			PingFunc: func(db *sql.DB) error {
				return nil
			},
			ExecFunc: func(db *sql.DB, query string) error {
				if strings.Contains(query, "CREATE TABLE IF NOT EXISTS tools") {
					return fmt.Errorf("mock tools table error")
				}
				return nil
			},
		}
		db, err := InitializeDatabase("test.db", mock)
		require.Error(t, err)
		require.Nil(t, db)
		require.Contains(t, err.Error(), "failed to create tools table after 5 attempts")
		require.Equal(t, 5, mock.OpenCount)
		require.Equal(t, 5, mock.PingCount)
		require.Equal(t, 5, mock.ExecCount)
	})

	t.Run("Failing to Create History Table", func(t *testing.T) {
		mock := &MockDBInitializer{
			OpenFunc: func(dsn string) (*sql.DB, error) {
				db, _ := sql.Open("sqlite3", ":memory:")
				return db, nil
			},
			PingFunc: func(db *sql.DB) error {
				return nil
			},
			ExecFunc: func(db *sql.DB, query string) error {
				if strings.Contains(query, "CREATE TABLE IF NOT EXISTS history") {
					return fmt.Errorf("mock history table error")
				}
				return nil
			},
		}
		db, err := InitializeDatabase("test.db", mock)
		require.Error(t, err)
		require.Nil(t, db)
		require.Contains(t, err.Error(), "failed to create history table after 5 attempts")
		require.Equal(t, 5, mock.OpenCount)
		require.Equal(t, 5, mock.PingCount)
		require.Equal(t, 10, mock.ExecCount)
	})

	t.Run("Nil Initializer Uses Default", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")
		db, err := InitializeDatabase(dbPath, nil)
		require.NoError(t, err)
		require.NotNil(t, db)
		defer db.Close()

		// Verify tables exist
		rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table'")
		require.NoError(t, err)
		defer rows.Close()

		var tables []string
		for rows.Next() {
			var name string
			require.NoError(t, rows.Scan(&name))
			tables = append(tables, name)
		}
		require.NoError(t, rows.Err())
		require.Contains(t, tables, "tools")
		require.Contains(t, tables, "history")
	})
}

func TestInitializeTool(t *testing.T) {
	t.Parallel()

	// Create a temporary directory for the test database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Test successful initialization
	reader, err := InitializeTool(dbPath)
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
	_, err = InitializeTool("/nonexistent/path/test.db")
	if err == nil {
		t.Error("Expected error for invalid path")
	}
}
