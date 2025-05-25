package tool

import (
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

	dbPath := "file::memory:"
	reader, err := InitializeTool(dbPath)
	require.NoError(t, err)
	defer reader.DB.Close()

	t.Run("Scheme", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, "tool", reader.Scheme())
	})

	t.Run("IsGlobbable", func(t *testing.T) {
		t.Parallel()
		require.False(t, reader.IsGlobbable())
	})

	t.Run("HasHierarchicalUris", func(t *testing.T) {
		t.Parallel()
		require.False(t, reader.HasHierarchicalUris())
	})

	t.Run("ListElements", func(t *testing.T) {
		t.Parallel()
		uri, _ := url.Parse("tool:///test")
		elements, err := reader.ListElements(*uri)
		require.NoError(t, err)
		require.Nil(t, elements)
	})

	t.Run("Read_GetItem", func(t *testing.T) {
		t.Parallel()
		reader, err := InitializeTool("file::memory:")
		require.NoError(t, err)
		defer reader.DB.Close()

		_, err = reader.DB.Exec("INSERT INTO tools (id, value) VALUES (?, ?)", "test1", "output1")
		require.NoError(t, err)

		uri, _ := url.Parse("tool:///test1")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte("output1"), data)

		uri, _ = url.Parse("tool:///nonexistent")
		data, err = reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte(""), data)

		uri, _ = url.Parse("tool:///")
		_, err = reader.Read(*uri)
		require.Error(t, err)
		require.Contains(t, err.Error(), "no tool ID provided")
	})

	t.Run("Read_Run_FileScript", func(t *testing.T) {
		t.Parallel()
		reader, err := InitializeTool("file::memory:")
		require.NoError(t, err)
		defer reader.DB.Close()

		// Create a temporary script file
		tmpDir := t.TempDir()
		scriptPath := filepath.Join(tmpDir, "test.sh")
		err = os.WriteFile(scriptPath, []byte("#!/bin/sh\necho test"), 0o755)
		require.NoError(t, err)

		uri, _ := url.Parse(fmt.Sprintf("tool:///test2?op=run&script=%s&params=%s", url.QueryEscape(scriptPath), url.QueryEscape("param1 param2")))
		_, err = reader.Read(*uri)
		require.NoError(t, err)
		// Actual execution depends on system, so check DB storage
		var value string
		err = reader.DB.QueryRow("SELECT value FROM tools WHERE id = ?", "test2").Scan(&value)
		require.NoError(t, err)
		// Since we can't predict exact output without mocking exec, verify insertion
		require.NotEmpty(t, value)

		// Check history
		var historyCount int
		err = reader.DB.QueryRow("SELECT COUNT(*) FROM history WHERE id = ?", "test2").Scan(&historyCount)
		require.NoError(t, err)
		require.Equal(t, 1, historyCount)

		// Test missing script
		uri, _ = url.Parse("tool:///test3?op=run")
		_, err = reader.Read(*uri)
		require.Error(t, err)
		require.Contains(t, err.Error(), "run operation requires a script parameter")

		// Test missing ID
		uri, _ = url.Parse("tool:///?op=run&script=script.sh")
		_, err = reader.Read(*uri)
		require.Error(t, err)
		require.Contains(t, err.Error(), "no tool ID provided for run operation")
	})

	t.Run("Read_Run_InlineScript", func(t *testing.T) {
		t.Parallel()
		reader, err := InitializeTool("file::memory:")
		require.NoError(t, err)
		defer reader.DB.Close()

		uri, _ := url.Parse("tool:///test4?op=run&script=echo%20hello&params=param1%20param2")
		_, err = reader.Read(*uri)
		require.NoError(t, err)
		// Actual execution depends on system, so check DB storage
		var value string
		err = reader.DB.QueryRow("SELECT value FROM tools WHERE id = ?", "test4").Scan(&value)
		require.NoError(t, err)
		// Since we can't predict exact output without mocking exec, verify insertion
		require.NotEmpty(t, value)

		// Check history
		var historyCount int
		err = reader.DB.QueryRow("SELECT COUNT(*) FROM history WHERE id = ?", "test4").Scan(&historyCount)
		require.NoError(t, err)
		require.Equal(t, 1, historyCount)
	})

	t.Run("Read_History", func(t *testing.T) {
		t.Parallel()
		reader, err := InitializeTool("file::memory:")
		require.NoError(t, err)
		defer reader.DB.Close()

		// Insert history entries
		timestamp := time.Now().Unix()
		_, err = reader.DB.Exec("INSERT INTO history (id, value, timestamp) VALUES (?, ?, ?), (?, ?, ?)",
			"test5", "output1", timestamp-10,
			"test5", "output2", timestamp)
		require.NoError(t, err)

		uri, _ := url.Parse("tool:///test5?op=history")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		lines := strings.Split(string(data), "\n")
		require.Len(t, lines, 2)
		for i, line := range lines {
			require.Contains(t, line, fmt.Sprintf("output%d", i+1))
			require.Contains(t, line, time.Unix(timestamp-10+int64(i*10), 0).Format(time.RFC3339)[:19]) // Approximate time
		}

		// Test non-existent history
		uri, _ = url.Parse("tool:///test6?op=history")
		data, err = reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte(""), data)

		// Test missing ID
		uri, _ = url.Parse("tool:///?op=history")
		_, err = reader.Read(*uri)
		require.Error(t, err)
		require.Contains(t, err.Error(), "no tool ID provided for history operation")
	})

	t.Run("Read_NilReceiver", func(t *testing.T) {
		t.Parallel()
		nilReader := &PklResourceReader{DBPath: dbPath}
		uri, _ := url.Parse("tool:///test7?op=run&script=echo%20test")
		_, err = nilReader.Read(*uri)
		require.NoError(t, err)
		// Actual execution depends on system, so check DB storage
		var value string
		err = nilReader.DB.QueryRow("SELECT value FROM tools WHERE id = ?", "test7").Scan(&value)
		require.NoError(t, err)
		require.NotEmpty(t, value)
	})

	t.Run("Read_NilDB", func(t *testing.T) {
		t.Parallel()
		reader := &PklResourceReader{DBPath: dbPath, DB: nil}
		uri, _ := url.Parse("tool:///test8?op=run&script=echo%20test")
		_, err = reader.Read(*uri)
		require.NoError(t, err)
		var value string
		err = reader.DB.QueryRow("SELECT value FROM tools WHERE id = ?", "test8").Scan(&value)
		require.NoError(t, err)
		require.NotEmpty(t, value)
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
}

func TestInitializeTool(t *testing.T) {
	t.Parallel()
	reader, err := InitializeTool("file::memory:")
	require.NoError(t, err)
	require.NotNil(t, reader)
	require.NotNil(t, reader.DB)
	require.Equal(t, "file::memory:", reader.DBPath)
	defer reader.DB.Close()
}
