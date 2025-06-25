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

	"github.com/kdeps/kdeps/pkg/download"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/kdeps/pkg/utils"
	"github.com/kdeps/kdeps/pkg/version"
)

type URLInfo struct {
	BaseURL           string
	Repo              string
	IsAnaconda        bool
	Version           string
	Architectures     []string
	LocalNameTemplate string
}

var ArchMappings = map[string]map[string]string{
	"apple/pkl": {"amd64": "amd64", "arm64": "aarch64"},
	"default":   {"amd64": "x86_64", "arm64": "aarch64"},
}

// GetCurrentArchitecture gets the current system architecture
func GetCurrentArchitecture(ctx context.Context, repo string) string {
	goArch := runtime.GOARCH
	mapping, ok := ArchMappings[repo]
	if !ok {
		mapping = ArchMappings["default"]
	}
	if arch, ok := mapping[goArch]; ok {
		return arch
	}
	return goArch
}

// CompareVersions compares two version strings
func CompareVersions(ctx context.Context, v1, v2 string) bool {
	p1, p2 := ParseVersion(v1), ParseVersion(v2)
	maxLen := max(len(p1), len(p2))

	for i := range maxLen {
		n1, n2 := 0, 0
		if i < len(p1) {
			n1 = p1[i]
		}
		if i < len(p2) {
			n2 = p2[i]
		}
		if n1 != n2 {
			return n1 > n2
		}
	}
	return false
}

// ParseVersion parses a version string into a slice of integers
func ParseVersion(v string) []int {
	parts := strings.FieldsFunc(v, func(r rune) bool { return r == '.' || r == '-' })
	res := make([]int, len(parts))
	for i, p := range parts {
		num, _ := strconv.Atoi(p)
		res[i] = num
	}
	return res
}

// GetLatestAnacondaVersions gets the latest Anaconda versions
func GetLatestAnacondaVersions(ctx context.Context) (map[string]string, error) {
	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://repo.anaconda.com/archive/", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Anaconda archive: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	re := regexp.MustCompile(`Anaconda3-(\d+\.\d+-\d+)-Linux-(x86_64|aarch64)\.sh`)
	matches := re.FindAllStringSubmatch(string(body), -1)
	if len(matches) == 0 {
		return nil, errors.New("no Anaconda versions found")
	}

	versions := map[string]string{"x86_64": "", "aarch64": ""}
	for _, m := range matches {
		v, arch := m[1], m[2]
		if versions[arch] == "" || CompareVersions(ctx, v, versions[arch]) {
			versions[arch] = v
		}
	}
	return versions, nil
}

// BuildURL builds a download URL
func BuildURL(baseURL, version, arch string) string {
	return strings.NewReplacer("{version}", version, "{arch}", arch).Replace(baseURL)
}

// GenerateURLs generates download URLs
func GenerateURLs(ctx context.Context) ([]download.DownloadItem, error) {
	return GenerateURLsWithOptions(ctx, true)
}

// GenerateURLsWithOptions generates download URLs with options
func GenerateURLsWithOptions(ctx context.Context, includeAnaconda bool) ([]download.DownloadItem, error) {
	urlInfos := []URLInfo{
		{
			BaseURL:           "https://github.com/apple/pkl/releases/download",
			Repo:              "apple/pkl",
			IsAnaconda:        false,
			Version:           version.PklVersion,
			Architectures:     []string{"amd64", "aarch64"},
			LocalNameTemplate: "pkl-linux-{version}-{arch}",
		},
	}

	if includeAnaconda {
		urlInfos = append(urlInfos, URLInfo{
			BaseURL:           "https://repo.anaconda.com/archive/Anaconda3-{version}-Linux-{arch}.sh",
			IsAnaconda:        true,
			Version:           version.AnacondaVersion,
			Architectures:     []string{"x86_64", "aarch64"},
			LocalNameTemplate: "anaconda-linux-{version}-{arch}.sh",
		})
	}

	var items []download.DownloadItem
	for _, info := range urlInfos {
		currentArch := GetCurrentArchitecture(ctx, info.Repo)
		version := info.Version

		if info.IsAnaconda && schema.UseLatest {
			versions, err := GetLatestAnacondaVersions(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get Anaconda versions: %w", err)
			}
			if version = versions[currentArch]; version == "" {
				return nil, fmt.Errorf("no Anaconda version for %s", currentArch)
			}
		} else if schema.UseLatest {
			latest, err := utils.GetLatestGitHubRelease(ctx, info.Repo, "")
			if err != nil {
				return nil, fmt.Errorf("failed to get latest GitHub release: %w", err)
			}
			version = latest
		}

		if utils.ContainsString(info.Architectures, currentArch) {
			url := BuildURL(info.BaseURL, version, currentArch)

			localVersion := version
			if schema.UseLatest {
				localVersion = "latest"
			}

			var localName string
			if info.LocalNameTemplate != "" {
				localName = strings.NewReplacer(
					"{version}", localVersion,
					"{arch}", currentArch,
				).Replace(info.LocalNameTemplate)
			}

			items = append(items, download.DownloadItem{
				URL:       url,       // full URL with actual version
				LocalName: localName, // friendly/stable name like "anaconda-latest-aarch64.sh"
			})
		}
	}

	return items, nil
}
