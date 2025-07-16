package session_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/pkg/session"
)

func TestInitializeDatabase(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Test successful database initialization
	db, err := session.InitializeDatabase(dbPath)
	require.NoError(t, err)
	assert.NotNil(t, db)

	// Verify the database file was created
	_, err = os.Stat(dbPath)
	require.NoError(t, err)

	// Test that we can execute a simple query
	_, err = db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY)")
	require.NoError(t, err)

	// Clean up
	db.Close()
}

func TestInitializeDatabaseWithInvalidPath(t *testing.T) {
	// Test with an invalid path
	invalidPath := "/nonexistent/path/test.db"

	db, err := session.InitializeDatabase(invalidPath)
	require.Error(t, err)
	assert.Nil(t, db)
}

func TestInitializeSession(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Test successful session initialization
	sess, err := session.InitializeSession(dbPath)
	require.NoError(t, err)
	assert.NotNil(t, sess)
	assert.NotNil(t, sess.DB)

	// Verify the database file was created
	_, err = os.Stat(dbPath)
	require.NoError(t, err)

	// Clean up
	sess.DB.Close()
}

func TestInitializeSessionWithInvalidPath(t *testing.T) {
	// Test with an invalid path
	invalidPath := "/nonexistent/path/test.db"

	sess, err := session.InitializeSession(invalidPath)
	require.Error(t, err)
	assert.Nil(t, sess)
}

func TestSessionMethods(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Initialize session
	sess, err := session.InitializeSession(dbPath)
	require.NoError(t, err)
	defer sess.DB.Close()

	// Test that we can execute a simple query
	_, err = sess.DB.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY)")
	require.NoError(t, err)
}

func TestSessionConcurrentAccess(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Test concurrent access to the same database
	done := make(chan bool, 2)

	go func() {
		sess, err := session.InitializeSession(dbPath)
		assert.NoError(t, err)
		defer sess.DB.Close()
		done <- true
	}()

	go func() {
		sess, err := session.InitializeSession(dbPath)
		assert.NoError(t, err)
		defer sess.DB.Close()
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done
}

func TestPklResourceReader(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Initialize session
	sess, err := session.InitializeSession(dbPath)
	require.NoError(t, err)
	defer sess.DB.Close()

	// Test that we can access the session (which is a PklResourceReader)
	assert.NotNil(t, sess)
}
