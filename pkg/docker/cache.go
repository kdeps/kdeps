package docker

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"kdeps/pkg/schema"
	"kdeps/pkg/utils"
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
func GetCurrentArchitecture() string {
	arch := runtime.GOARCH
	switch arch {
	case "amd64":
		return "x86_64" // For Anaconda, maps "amd64" to "x86_64"
	case "arm64":
		return "aarch64" // For Anaconda and pkl, uses "aarch64"
	default:
		return arch
	}
}

// CompareVersions compares two versions, returning true if v1 > v2.
func CompareVersions(v1, v2 string) bool {
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
func GetLatestAnacondaVersions() (map[string]string, map[string]string, error) {
	archiveURL := "https://repo.anaconda.com/archive/"
	resp, err := http.Get(archiveURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch Anaconda archive: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Regex for extracting version and architecture from filenames.
	re := regexp.MustCompile(`Anaconda3-(\d+\.\d+-\d+)-Linux-(x86_64|aarch64)\.sh`)
	matches := re.FindAllStringSubmatch(string(body), -1)
	if len(matches) == 0 {
		return nil, nil, fmt.Errorf("no Anaconda versions found")
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
		if latestVersions[arch] == "" || CompareVersions(version, latestVersions[arch]) {
			latestVersions[arch] = version
			latestBuilds[arch] = match[0] // Full filename for the latest version and architecture.
		}
	}

	return latestVersions, latestBuilds, nil
}

func GenerateURLs() ([]string, error) {
	urlInfos := []URLInfo{
		{
			BaseURL:       "https://github.com/apple/pkl/releases/download/{version}/pkl-linux-{arch}",
			Repo:          "apple/pkl",
			Version:       "0.27.1",
			Architectures: []string{"amd64", "aarch64"},
		},
		{
			BaseURL:       "https://repo.anaconda.com/archive/Anaconda3-{version}-Linux-{arch}.sh",
			IsAnaconda:    true,
			Version:       "2024.10-1", // Default version when not using the latest
			Architectures: []string{"x86_64", "aarch64"},
		},
	}

	var urls []string
	currentArch := GetCurrentArchitecture()

	for _, info := range urlInfos {
		if info.IsAnaconda {
			// Anaconda URLs
			version := info.Version
			if schema.UseLatest {
				latestVersions, _, err := GetLatestAnacondaVersions()
				if err != nil {
					return nil, fmt.Errorf("failed to get Anaconda versions: %w", err)
				}
				version = latestVersions[currentArch]
			}
			url := strings.ReplaceAll(info.BaseURL, "{version}", version)
			url = strings.ReplaceAll(url, "{arch}", currentArch)
			urls = append(urls, url)
		} else {
			// Other repositories
			version := info.Version
			if schema.UseLatest {
				latestVersion, err := utils.GetLatestGitHubRelease(info.Repo, "")
				if err != nil {
					return nil, fmt.Errorf("failed to get latest version for %s: %w", info.Repo, err)
				}
				version = latestVersion
			}

			// Generate URLs for the current architecture
			for _, arch := range info.Architectures {
				if arch == currentArch || (arch == "arm64" && currentArch == "aarch64") {
					url := strings.ReplaceAll(info.BaseURL, "{version}", version)
					url = strings.ReplaceAll(url, "{arch}", arch)
					urls = append(urls, url)
				}
			}
		}
	}

	return urls, nil
}
