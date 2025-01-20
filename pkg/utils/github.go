package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

// GitHubReleaseFetcher is a function variable that can be mocked in tests.
// By default, it points to the real GetLatestGitHubRelease function.
var GitHubReleaseFetcher = GetLatestGitHubRelease

// GetLatestGitHubRelease fetches the latest release version from a GitHub repository.
func GetLatestGitHubRelease(ctx context.Context, repo string, baseURL string) (string, error) {
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}
	url := fmt.Sprintf("%s/repos/%s/releases/latest", baseURL, repo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	token := os.Getenv("GITHUB_TOKEN")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	} else {
		fmt.Fprintln(os.Stderr, "Warning: GITHUB_TOKEN is not set; using unauthenticated requests with limited rate.")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return "", fmt.Errorf("unauthorized: check your GITHUB_TOKEN")
	}
	if resp.StatusCode == http.StatusForbidden {
		return "", fmt.Errorf("rate limit exceeded: ensure your GITHUB_TOKEN has appropriate permissions")
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d, response: %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var result struct {
		TagName string `json:"tag_name"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return strings.TrimPrefix(result.TagName, "v"), nil
}
