package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
)

// GetLatestOllamaVersion fetches the latest semantic version tag from Docker Hub for ollama/ollama
func GetLatestOllamaVersion(ctx context.Context) (string, error) {
	url := "https://hub.docker.com/v2/repositories/ollama/ollama/tags?page_size=100"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var result struct {
		Results []struct {
			Name string `json:"name"`
		} `json:"results"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse JSON response: %w", err)
	}

	// Filter to only semantic version tags (e.g., "0.13.5", not "latest" or "0.13.5-rc1")
	semverPattern := regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
	var versions []string
	for _, tag := range result.Results {
		if semverPattern.MatchString(tag.Name) {
			versions = append(versions, tag.Name)
		}
	}

	if len(versions) == 0 {
		return "", fmt.Errorf("no semantic version tags found")
	}

	// Sort versions and return the latest (first one should already be latest from Docker Hub)
	sort.Slice(versions, func(i, j int) bool {
		return compareSemanticVersions(versions[i], versions[j]) > 0
	})

	return versions[0], nil
}

// compareSemanticVersions compares two semantic version strings
// Returns: 1 if v1 > v2, -1 if v1 < v2, 0 if equal
func compareSemanticVersions(v1, v2 string) int {
	v1Parts := parseSemVer(v1)
	v2Parts := parseSemVer(v2)

	for i := 0; i < 3; i++ {
		if v1Parts[i] > v2Parts[i] {
			return 1
		}
		if v1Parts[i] < v2Parts[i] {
			return -1
		}
	}
	return 0
}

// parseSemVer parses a semantic version string into [major, minor, patch]
func parseSemVer(v string) [3]int {
	var parts [3]int
	fmt.Sscanf(v, "%d.%d.%d", &parts[0], &parts[1], &parts[2])
	return parts
}
