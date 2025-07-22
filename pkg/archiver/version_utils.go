package archiver

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/messages"
)

// CompareVersions compares version numbers and returns the latest.
func CompareVersions(versions []string, logger *logging.Logger) string {
	logger.Debug(messages.MsgComparingVersions, "versions", versions)
	sort.Slice(versions, func(i, j int) bool {
		// Split the version strings into parts
		v1 := strings.Split(versions[i], ".")
		v2 := strings.Split(versions[j], ".")

		// Compare each part of the version (major, minor, patch)
		for k := range v1 {
			if v1[k] != v2[k] {
				result := v1[k] > v2[k]
				logger.Debug(messages.MsgVersionComparisonResult, "v1", v1, "v2", v2, "result", result)
				return result
			}
		}
		return false
	})

	// Return the first version (which will be the latest after sorting)
	latestVersion := versions[0]
	logger.Debug(messages.MsgLatestVersionDetermined, "version", latestVersion)
	return latestVersion
}

var GetLatestVersion = func(directory string, logger *logging.Logger) (string, error) {
	var versions []string

	// Walk through the directory to collect version names
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logger.Error("error walking the path", "path", path, "error", err)
			return err
		}

		// Collect directory names that match the version pattern
		if info.IsDir() && strings.Count(info.Name(), ".") == 2 {
			versions = append(versions, info.Name())
			logger.Debug(messages.MsgFoundVersionDirectory, "directory", info.Name())
		}
		return nil
	})
	if err != nil {
		logger.Error("error while walking the directory", "directory", directory, "error", err)
		return "", err
	}

	// Check if versions were found
	if len(versions) == 0 {
		err = errors.New("no versions found")
		logger.Warn("no versions found", "directory", directory)
		return "", err
	}

	// Find the latest version
	latestVersion := CompareVersions(versions, logger)
	return latestVersion, nil
}
