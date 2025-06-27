package item

import (
	"database/sql"
	"net/url"

	"github.com/spf13/afero"
)

// Injectable functions for testability
var (
	// Database operations
	SqlOpenFunc = func(driverName, dataSourceName string) (*sql.DB, error) {
		return sql.Open(driverName, dataSourceName)
	}

	// File system operations for testing
	AferoNewOsFsFunc = func() afero.Fs {
		return afero.NewOsFs()
	}

	// URL parsing functions
	UrlParseFunc = func(rawurl string) (*url.URL, error) {
		return url.Parse(rawurl)
	}

	// Database initialization wrapper
	InitializeDatabaseFunc = InitializeDatabase

	// Item initialization wrapper
	InitializeItemFunc = InitializeItem
)

// SetupTestableDatabase sets up a testable database configuration
func SetupTestableDatabase() {
	// Override database operations for testing
	SqlOpenFunc = func(driverName, dataSourceName string) (*sql.DB, error) {
		// Always use memory database for testing
		if dataSourceName != ":memory:" {
			dataSourceName = ":memory:"
		}
		return sql.Open(driverName, dataSourceName)
	}
}

// ResetDatabase resets database functions to defaults
func ResetDatabase() {
	SqlOpenFunc = sql.Open
}
