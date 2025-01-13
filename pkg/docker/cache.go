package docker

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"kdeps/pkg/utils"
)

// URLInfo represents information about a URL template and repository details.
type URLInfo struct {
	BaseURL    string // Base URL template with placeholders for version and architecture
	Repo       string // Repository name, e.g., "kdeps/kdeps"
	IsAnaconda bool   // Special handling for Anaconda URLs
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
func GetLatestAnacondaVersions() (string, map[string]string, error) {
	archiveURL := "https://repo.anaconda.com/archive/"
	resp, err := http.Get(archiveURL)
	if err != nil {
		return "", nil, fmt.Errorf("failed to fetch Anaconda archive: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Regex for extracting version and architecture from filenames.
	re := regexp.MustCompile(`Anaconda3-(\d+\.\d+-\d+)-Linux-(x86_64|aarch64)\.sh`)
	matches := re.FindAllStringSubmatch(string(body), -1)
	if len(matches) == 0 {
		return "", nil, fmt.Errorf("no Anaconda versions found")
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

	return latestVersions["x86_64"], latestBuilds, nil
}

func GenerateURLs() ([]string, error) {
	urlInfos := []URLInfo{
		{BaseURL: "https://github.com/apple/pkl/releases/download/{version}/pkl-linux-{arch}", Repo: "apple/pkl"},
		{BaseURL: "https://repo.anaconda.com/archive/{build}", IsAnaconda: true},
	}

	architectures := []string{"amd64", "aarch64"}
	var urls []string

	for _, info := range urlInfos {
		if info.IsAnaconda {
			_, builds, err := GetLatestAnacondaVersions()
			if err != nil {
				return nil, fmt.Errorf("failed to get Anaconda versions: %w", err)
			}

			for _, build := range builds {
				url := strings.ReplaceAll(info.BaseURL, "{build}", build)
				urls = append(urls, url)
			}
		} else {
			latestVersion, err := utils.GetLatestGitHubRelease(info.Repo)
			if err != nil {
				return nil, fmt.Errorf("failed to get latest version for %s: %w", info.Repo, err)
			}

			for _, arch := range architectures {
				url := strings.ReplaceAll(info.BaseURL, "{version}", latestVersion)
				url = strings.ReplaceAll(url, "{arch}", arch)
				urls = append(urls, url)
			}
		}
	}

	return urls, nil
}
