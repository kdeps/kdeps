package llm

import (
	"context"
	"fmt"
	"log/slog"
	stdhttp "net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/pathologize"
	"gopkg.in/yaml.v3"
)

const (
	hfResolveBase = "https://huggingface.co"
	hfTokenEnvVar = "HF_TOKEN"
)

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
	dest := filepath.Join(dir, pathologize.Clean(filepath.Base(filename)))

	if _, statErr := AppFS.Stat(dest); statErr != nil || isPartialDownload(AppFS, dest) {
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
	_, err := downloadModelFile(ctx, downloadURL, filename, dir, logger, AppFS)
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
		"--summary-interval=1", // per-second progress; pipe output defaults to 60s
		"--dir=" + filepath.Dir(dest),
		"--out=" + filepath.Base(dest),
	}
	if token != "" {
		args = append(args, "--header=Authorization: Bearer "+token)
	}
	args = append(args, downloadURL)
	logger.InfoContext(ctx, "hf: downloading via aria2c", "url", downloadURL, "dest", dest)
	cmd := exec.CommandContext(ctx, aria2cPath, args...)
	// Route all output to stderr — readline manages stdout in raw mode and
	// \r-based progress lines corrupt its display state.
	outFilter := &aria2cNoiseFilter{w: os.Stderr} // progress bars to stderr
	errFilter := &aria2cNoiseFilter{w: os.Stderr} // filtered errors to stderr
	cmd.Stdout = outFilter
	cmd.Stderr = errFilter
	err := cmd.Run()
	outFilter.Flush()
	errFilter.Flush()
	if err != nil {
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
	raw, _ := loadLocalFile(localPath)
	reg := parseGGUFYAML(raw)
	if reg == nil {
		reg = &ggufVersions{}
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
	if mkErr := os.MkdirAll(filepath.Dir(localPath), 0o750); mkErr != nil {
		return fmt.Errorf("hf register: %w", mkErr)
	}
	if writeErr := os.WriteFile(localPath, data, 0o600); writeErr != nil {
		return fmt.Errorf("hf register: write: %w", writeErr)
	}
	ReloadGGUFRegistry()
	return nil
}
