package archiver

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kdeps/kdeps/pkg/logging"
)

// Function to compare version numbers.
func compareVersions(ctx context.Context, versions []string, logger *logging.Logger) string {
	logger.Debug("Comparing versions", "versions", versions)
	sort.Slice(versions, func(i, j int) bool {
		// Split the version strings into parts
		v1 := strings.Split(versions[i], ".")
		v2 := strings.Split(versions[j], ".")

		// Compare each part of the version (major, minor, patch)
		for k := range v1 {
			if v1[k] != v2[k] {
				result := v1[k] > v2[k]
				logger.Debug("Version comparison result", "v1", v1, "v2", v2, "result", result)
				return result
			}
		}
		return false
	})

	// Return the first version (which will be the latest after sorting)
	latestVersion := versions[0]
	logger.Debug("Latest version determined", "version", latestVersion)
	return latestVersion
}

func getLatestVersion(ctx context.Context, directory string, logger *logging.Logger) (string, error) {
	var versions []string

	// Walk through the directory to collect version names
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logger.Error("Error walking the path", "path", path, "error", err)
			return err
		}

		// Collect directory names that match the version pattern
		if info.IsDir() && strings.Count(info.Name(), ".") == 2 {
			versions = append(versions, info.Name())
			logger.Debug("Found version directory", "directory", info.Name())
		}
		return nil
	})
	if err != nil {
		logger.Error("Error while walking the directory", "directory", directory, "error", err)
		return "", err
	}

	// Check if versions were found
	if len(versions) == 0 {
		err = errors.New("no versions found")
		logger.Warn("No versions found", "directory", directory)
		return "", err
	}

	// Find the latest version
	latestVersion := compareVersions(ctx, versions, logger)
	return latestVersion, nil
}
