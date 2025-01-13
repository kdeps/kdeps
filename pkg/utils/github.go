package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

// GetLatestGitHubRelease fetches the latest release version from a GitHub repository.
func GetLatestGitHubRelease(repo string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	// Add Authorization header with the token
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return "", fmt.Errorf("GITHUB_TOKEN environment variable is not set")
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Extract the version from the response JSON
	var result struct {
		TagName string `json:"tag_name"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	return strings.TrimPrefix(result.TagName, "v"), nil // Strip "v" prefix if present
}
