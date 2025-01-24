package docker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/kdeps/pkg/utils"
)

// URLInfo represents information about a URL template and repository details.
type URLInfo struct {
	BaseURL       string   // Base URL template with placeholders for version and architecture
	Repo          string   // Repository name, e.g., "kdeps/kdeps"
	IsAnaconda    bool     // Special handling for Anaconda URLs
	Version       string   // Specific version or a placeholder "{latest}" for the latest version
	Architectures []string // Architectures to include in the URLs
}

// GetCurrentArchitecture returns the architecture of the current machine.
func GetCurrentArchitecture(ctx context.Context, repo string) string {
	arch := runtime.GOARCH
	switch arch {
	case "amd64":
		if repo == "apple/pkl" {
			return "amd64" // PKL uses "amd64"
		}
		return "x86_64" // Anaconda uses "x86_64"
	case "arm64":
		return "aarch64" // Both use "aarch64"
	default:
		return arch
	}
}

// CompareVersions compares two versions, returning true if v1 > v2.
func CompareVersions(ctx context.Context, v1, v2 string) bool {
	parts1 := parseVersion(v1)
	parts2 := parseVersion(v2)

	for i := 0; i < len(parts1) || i < len(parts2); i++ {
		var p1, p2 int
		if i < len(parts1) {
			p1 = parts1[i]
		}
		if i < len(parts2) {
			p2 = parts2[i]
		}
		if p1 != p2 {
			return p1 > p2
		}
	}
	return false
}

// parseVersion parses a version string like "2024.10-1" into a slice of integers.
func parseVersion(version string) []int {
	parts := strings.FieldsFunc(version, func(r rune) bool {
		return r == '.' || r == '-'
	})
	result := make([]int, len(parts))
	for i, part := range parts {
		num, _ := strconv.Atoi(part)
		result[i] = num
	}
	return result
}

// GetLatestAnacondaVersions fetches the latest Anaconda versions for both architectures.
func GetLatestAnacondaVersions(ctx context.Context) (map[string]string, map[string]string, error) {
	archiveURL := "https://repo.anaconda.com/archive/"
	resp, err := http.Get(archiveURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch Anaconda archive: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Regex for extracting version and architecture from filenames.
	re := regexp.MustCompile(`Anaconda3-(\d+\.\d+-\d+)-Linux-(x86_64|aarch64)\.sh`)
	matches := re.FindAllStringSubmatch(string(body), -1)
	if len(matches) == 0 {
		return nil, nil, errors.New("no Anaconda versions found")
	}

	// Map to hold the latest version for each architecture.
	latestBuilds := map[string]string{
		"x86_64":  "",
		"aarch64": "",
	}
	latestVersions := map[string]string{
		"x86_64":  "",
		"aarch64": "",
	}

	for _, match := range matches {
		version := match[1]
		arch := match[2]

		// Compare and update the latest version for each architecture.
		if latestVersions[arch] == "" || CompareVersions(ctx, version, latestVersions[arch]) {
			latestVersions[arch] = version
			latestBuilds[arch] = match[0] // Full filename for the latest version and architecture.
		}
	}

	return latestVersions, latestBuilds, nil
}

func GenerateURLs(ctx context.Context) ([]string, error) {
	urlInfos := []URLInfo{
		{
			BaseURL:       "https://github.com/apple/pkl/releases/download/{version}/pkl-linux-{arch}",
			Repo:          "apple/pkl",
			Version:       "0.27.2",
			Architectures: []string{"amd64", "aarch64"},
		},
		{
			BaseURL:       "https://repo.anaconda.com/archive/Anaconda3-{version}-Linux-{arch}.sh",
			IsAnaconda:    true,
			Version:       "2024.10-1",
			Architectures: []string{"x86_64", "aarch64"},
		},
	}

	var urls []string

	for _, info := range urlInfos {
		currentArch := GetCurrentArchitecture(ctx, info.Repo) // Pass repo to get correct arch

		if info.IsAnaconda {
			// Handle Anaconda URLs
			version := info.Version
			if schema.UseLatest {
				latestVersions, _, err := GetLatestAnacondaVersions(ctx)
				if err != nil {
					return nil, fmt.Errorf("failed to get Anaconda versions: %w", err)
				}
				version = latestVersions[currentArch]
				if version == "" {
					return nil, fmt.Errorf("no latest version found for architecture: %s", currentArch)
				}
			}
			url := strings.ReplaceAll(info.BaseURL, "{version}", version)
			url = strings.ReplaceAll(url, "{arch}", currentArch)
			urls = append(urls, url)
		} else {
			// Handle other URLs (e.g., PKL)
			version := info.Version
			if schema.UseLatest {
				latestVersion, err := utils.GetLatestGitHubRelease(ctx, info.Repo, "")
				if err != nil {
					return nil, fmt.Errorf("failed to get latest version for %s: %w", info.Repo, err)
				}
				version = latestVersion
				if version == "" {
					return nil, fmt.Errorf("no latest version found for repo: %s", info.Repo)
				}
			}

			for _, arch := range info.Architectures {
				if arch == currentArch {
					url := strings.ReplaceAll(info.BaseURL, "{version}", version)
					url = strings.ReplaceAll(url, "{arch}", arch)
					urls = append(urls, url)
				}
			}
		}
	}

	return urls, nil
}
