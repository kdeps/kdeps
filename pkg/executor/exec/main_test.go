package exec_test

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Use in-memory database for all tests in this package to avoid locking issues
	os.Setenv("KDEPS_MEMORY_DB_PATH", ":memory:")

	// Run tests
	code := m.Run()

	// Cleanup (though process exit will clean up env vars for the process, good practice)
	os.Unsetenv("KDEPS_MEMORY_DB_PATH")

	os.Exit(code)
}
