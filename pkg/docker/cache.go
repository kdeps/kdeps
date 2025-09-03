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

var archMappings = map[string]map[string]string{
	"apple/pkl": {"amd64": "amd64", "arm64": "aarch64"},
	"default":   {"amd64": "x86_64", "arm64": "aarch64"},
}

func GetCurrentArchitecture(_ context.Context, repo string) string {
	goArch := runtime.GOARCH
	mapping, ok := archMappings[repo]
	if !ok {
		mapping = archMappings["default"]
	}
	if arch, ok := mapping[goArch]; ok {
		return arch
	}
	return goArch
}

func CompareVersions(_ context.Context, v1, v2 string) bool {
	p1, p2 := parseVersion(v1), parseVersion(v2)
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

func parseVersion(v string) []int {
	parts := strings.FieldsFunc(v, func(r rune) bool { return r == '.' || r == '-' })
	res := make([]int, len(parts))
	for i, p := range parts {
		num, _ := strconv.Atoi(p)
		res[i] = num
	}
	return res
}

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

func buildURL(baseURL, version, arch string) string {
	return strings.NewReplacer("{version}", version, "{arch}", arch).Replace(baseURL)
}

func GenerateURLs(ctx context.Context, installAnaconda bool) ([]download.DownloadItem, error) {
	urlInfos := setupURLInfos(installAnaconda)
	var items []download.DownloadItem

	for _, info := range urlInfos {
		item, err := processURLInfo(ctx, info)
		if err != nil {
			return nil, err
		}
		if item != nil {
			items = append(items, *item)
		}
	}

	return items, nil
}

func setupURLInfos(installAnaconda bool) []URLInfo {
	urlInfos := []URLInfo{
		{
			BaseURL:           "https://github.com/apple/pkl/releases/download/{version}/pkl-linux-{arch}",
			Repo:              "apple/pkl",
			Version:           version.DefaultPklVersion,
			Architectures:     []string{"amd64", "aarch64"},
			LocalNameTemplate: "pkl-linux-{version}-{arch}",
		},
	}

	// Only include anaconda if it should be installed
	if installAnaconda {
		urlInfos = append(urlInfos, URLInfo{
			BaseURL:           "https://repo.anaconda.com/archive/Anaconda3-{version}-Linux-{arch}.sh",
			IsAnaconda:        true,
			Version:           version.DefaultAnacondaVersion,
			Architectures:     []string{"x86_64", "aarch64"},
			LocalNameTemplate: "anaconda-linux-{version}-{arch}.sh",
		})
	}

	return urlInfos
}

func processURLInfo(ctx context.Context, info URLInfo) (*download.DownloadItem, error) {
	currentArch := GetCurrentArchitecture(ctx, info.Repo)
	version, err := resolveVersion(ctx, info, currentArch)
	if err != nil {
		return nil, err
	}

	if !utils.ContainsString(info.Architectures, currentArch) {
		return nil, nil
	}

	url := buildURL(info.BaseURL, version, currentArch)
	localName := buildLocalName(info, version, currentArch)

	return &download.DownloadItem{
		URL:       url,
		LocalName: localName,
	}, nil
}

func resolveVersion(ctx context.Context, info URLInfo, currentArch string) (string, error) {
	version := info.Version

	if info.IsAnaconda && schema.UseLatest {
		versions, err := GetLatestAnacondaVersions(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to get Anaconda versions: %w", err)
		}
		if version = versions[currentArch]; version == "" {
			return "", fmt.Errorf("no Anaconda version for %s", currentArch)
		}
	} else if schema.UseLatest {
		latest, err := utils.GetLatestGitHubRelease(ctx, info.Repo, "")
		if err != nil {
			return "", fmt.Errorf("failed to get latest GitHub release: %w", err)
		}
		version = latest
	}

	return version, nil
}

func buildLocalName(info URLInfo, version, currentArch string) string {
	if info.LocalNameTemplate == "" {
		return ""
	}

	localVersion := version
	if schema.UseLatest {
		localVersion = "latest"
	}

	return strings.NewReplacer(
		"{version}", localVersion,
		"{arch}", currentArch,
	).Replace(info.LocalNameTemplate)
}
