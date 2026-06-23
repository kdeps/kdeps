//go:build !js

package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	stdhttp "net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	hfAPIBase     = "https://huggingface.co/api"
	hfResolveBase = "https://huggingface.co"
	hfAPIModels   = hfAPIBase + "/models"
	hfSearchLimit = 20
	hfHTTPTimeout = 30 * time.Second
	hfTokenEnvVar = "HF_TOKEN"
)

// HFModelResult is a single entry from the HuggingFace model search response.
// Siblings is populated when the search is made with full=true.
type HFModelResult struct {
	ID        string        `json:"id"`
	Downloads int           `json:"downloads"`
	Likes     int           `json:"likes"`
	Tags      []string      `json:"tags"`
	Siblings  []HFFileEntry `json:"siblings"`
}

// HFFileEntry is a single file inside a HuggingFace repo.
type HFFileEntry struct {
	Filename string `json:"rfilename"`
	Size     int64  `json:"size"`
}

// HFRepoInfo is the model-info API response.
type HFRepoInfo struct {
	ID       string        `json:"id"`
	Siblings []HFFileEntry `json:"siblings"`
}

// hfRequest makes an authenticated GET request to the HuggingFace API.
// Uses HF_TOKEN when present.
func hfRequest(ctx context.Context, rawURL string) (*stdhttp.Response, error) {
	req, err := stdhttp.NewRequestWithContext(ctx, stdhttp.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	if tok := os.Getenv(hfTokenEnvVar); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	req.Header.Set("Accept", "application/json")
	client := &stdhttp.Client{Timeout: hfHTTPTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("hf api: %w", err)
	}
	if resp.StatusCode != stdhttp.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("hf api: HTTP %d for %s", resp.StatusCode, rawURL)
	}
	return resp, nil
}

// HFSearchGGUF queries HuggingFace for GGUF repos matching query.
func HFSearchGGUF(ctx context.Context, query string, limit int) ([]HFModelResult, error) {
	return HFSearchGGUFWithBase(ctx, hfAPIModels, query, limit)
}

// HFSearchGGUFWithBase is the testable form of HFSearchGGUF with an injectable base URL.
func HFSearchGGUFWithBase(ctx context.Context, apiModelsURL, query string, limit int) ([]HFModelResult, error) {
	if limit <= 0 {
		limit = hfSearchLimit
	}
	params := url.Values{
		"search":    {query},
		"filter":    {"gguf"},
		"sort":      {"downloads"},
		"direction": {"-1"},
		"limit":     {strconv.Itoa(limit)},
		"full":      {"true"}, // include siblings (file list) in each result
	}
	resp, err := hfRequest(ctx, apiModelsURL+"?"+params.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var results []HFModelResult
	if decErr := json.NewDecoder(resp.Body).Decode(&results); decErr != nil {
		return nil, fmt.Errorf("hf search: decode: %w", decErr)
	}
	// If nothing matched the gguf tag, retry without the tag filter so authors
	// whose repos contain GGUF files but lack the tag are still found.
	if len(results) == 0 {
		results = hfSearchWithoutFilter(ctx, apiModelsURL, params)
	}
	return results, nil
}

// hfSearchWithoutFilter retries a search without filter=gguf and returns only
// repos that have at least one .gguf sibling.
func hfSearchWithoutFilter(ctx context.Context, apiModelsURL string, params url.Values) []HFModelResult {
	params.Del("filter")
	resp, err := hfRequest(ctx, apiModelsURL+"?"+params.Encode())
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	var all []HFModelResult
	if decErr := json.NewDecoder(resp.Body).Decode(&all); decErr != nil {
		return nil
	}
	var out []HFModelResult
	for _, m := range all {
		if len(HFGGUFFiles(m.Siblings)) > 0 {
			out = append(out, m)
		}
	}
	return out
}

// HFRepoFiles returns info about a HuggingFace repo including its file list.
func HFRepoFiles(ctx context.Context, repoID string) (HFRepoInfo, error) {
	return HFRepoFilesWithBase(ctx, hfAPIModels, repoID)
}

// HFRepoFilesWithBase is the testable form of HFRepoFiles with an injectable base URL.
func HFRepoFilesWithBase(ctx context.Context, apiModelsURL, repoID string) (HFRepoInfo, error) {
	resp, err := hfRequest(ctx, apiModelsURL+"/"+repoID)
	if err != nil {
		return HFRepoInfo{}, err
	}
	defer resp.Body.Close()
	var info HFRepoInfo
	if decErr := json.NewDecoder(resp.Body).Decode(&info); decErr != nil {
		return HFRepoInfo{}, fmt.Errorf("hf info: decode: %w", decErr)
	}
	return info, nil
}

// HFGGUFFiles returns only the .gguf entries from a file list.
func HFGGUFFiles(files []HFFileEntry) []HFFileEntry {
	var out []HFFileEntry
	for _, f := range files {
		if strings.HasSuffix(strings.ToLower(f.Filename), ".gguf") {
			out = append(out, f)
		}
	}
	return out
}

// HFDownloadURL returns the direct download URL for a file in a repo.
func HFDownloadURL(repoID, filename string) string {
	return hfResolveBase + "/" + repoID + "/resolve/main/" + filename
}

// HFDownloadGGUF downloads a GGUF file and registers it in the local registry.
// Returns the local file path and the alias registered for it.
func HFDownloadGGUF(ctx context.Context, repoID, filename string, logger *slog.Logger) (string, string, error) {
	if logger == nil {
		logger = slog.Default()
	}
	dir, err := DefaultModelsDir()
	if err != nil {
		return "", "", err
	}
	downloadURL := HFDownloadURL(repoID, filename)
	dest := filepath.Join(dir, filepath.Base(filename))

	if _, statErr := AppFS.Stat(dest); statErr != nil {
		if dlErr := hfDownloadFile(ctx, downloadURL, filename, dest, dir, logger); dlErr != nil {
			return "", "", dlErr
		}
	}

	alias := strings.TrimSuffix(filename, ".gguf")
	alias = strings.TrimSuffix(alias, ".GGUF")
	entry := GGUFEntry{
		Alias:    alias,
		URL:      downloadURL,
		Filename: filename,
		Repo:     repoID,
	}
	if regErr := HFRegisterGGUFEntry(entry); regErr != nil {
		logger.WarnContext(ctx, "hf: could not register in local registry", "err", regErr)
	}
	return dest, alias, nil
}

// hfDownloadFile downloads a file. aria2c is used when available for
// multi-connection downloads; falls back to plain HTTP (with HF_TOKEN header
// when set).
func hfDownloadFile(
	ctx context.Context,
	downloadURL, filename, dest, dir string,
	logger *slog.Logger,
) error {
	tok := os.Getenv(hfTokenEnvVar)
	if aria2cPath, lookErr := exec.LookPath("aria2c"); lookErr == nil {
		return hfDownloadAria2c(ctx, aria2cPath, downloadURL, dest, tok, logger)
	}
	if tok != "" {
		return hfDownloadWithToken(ctx, downloadURL, dest, tok)
	}
	_, err := downloadModelFile(downloadURL, filename, dir, logger, AppFS)
	return err
}

// hfDownloadAria2c downloads via aria2c with up to 8 parallel connections.
func hfDownloadAria2c(
	ctx context.Context,
	aria2cPath, downloadURL, dest, token string,
	logger *slog.Logger,
) error {
	args := []string{
		"--max-connection-per-server=8",
		"--split=8",
		"--min-split-size=10M",
		"--continue=true",
		"--file-allocation=none",
		"--console-log-level=warn",
		"--dir=" + filepath.Dir(dest),
		"--out=" + filepath.Base(dest),
	}
	if token != "" {
		args = append(args, "--header=Authorization: Bearer "+token)
	}
	args = append(args, downloadURL)
	logger.InfoContext(ctx, "hf: downloading via aria2c", "url", downloadURL, "dest", dest)
	cmd := exec.CommandContext(ctx, aria2cPath, args...)
	cmd.Stdout = os.Stderr // progress to stderr so REPL stdout stays clean
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("aria2c: %w", err)
	}
	return nil
}

// hfDownloadWithToken downloads via HTTP with an Authorization header (for gated models).
func hfDownloadWithToken(ctx context.Context, rawURL, dest, token string) error {
	req, err := stdhttp.NewRequestWithContext(ctx, stdhttp.MethodGet, rawURL, nil)
	if err != nil {
		return fmt.Errorf("hf download: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	client := &stdhttp.Client{} // no timeout for large files
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("hf download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != stdhttp.StatusOK {
		return fmt.Errorf("hf download: HTTP %d", resp.StatusCode)
	}
	body := newProgressReader(resp.Body, resp.ContentLength, filepath.Base(dest))
	return writeDownloadToFile(dest, body)
}

// HFRegisterGGUFEntry adds or replaces an entry in the local GGUF registry.
// The registry is at ~/.kdeps/gguf_versions.yaml. On success, the in-memory
// registry is invalidated so the next lookup picks up the new entry.
func HFRegisterGGUFEntry(entry GGUFEntry) error {
	localPath := localGGUFRegistryPath()
	// loadOrSeedLocalFile creates the file on first call but returns false.
	// Re-read the file directly so we always have content to work with.
	raw, _ := loadOrSeedLocalFile(localPath, defaultGGUFVersionsYAML)
	if raw == nil {
		// File was just seeded; read it back.
		var readErr error
		raw, readErr = os.ReadFile(localPath)
		if readErr != nil {
			return fmt.Errorf("hf register: could not read registry at %s: %w", localPath, readErr)
		}
	}
	reg := parseGGUFYAML(raw)
	if reg == nil {
		reg = &ggufVersions{Version: 1}
	}
	replaced := false
	for i, e := range reg.GGUFs {
		if e.Alias == entry.Alias {
			reg.GGUFs[i] = entry
			replaced = true
			break
		}
	}
	if !replaced {
		reg.GGUFs = append(reg.GGUFs, entry)
	}
	data, err := yaml.Marshal(reg)
	if err != nil {
		return fmt.Errorf("hf register: marshal: %w", err)
	}
	if writeErr := os.WriteFile(localPath, data, 0o600); writeErr != nil {
		return fmt.Errorf("hf register: write: %w", writeErr)
	}
	ReloadGGUFRegistry()
	return nil
}
