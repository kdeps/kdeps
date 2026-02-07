// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package iso

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

const (
	linuxkitVersion = "v1.8.2"
	linuxkitBaseURL = "https://github.com/linuxkit/linuxkit/releases/download"
)

// LinuxKitRunner executes linuxkit CLI commands.
type LinuxKitRunner interface {
	Build(ctx context.Context, configPath, format, arch, outputDir string) error
}

// DefaultLinuxKitRunner runs the actual linuxkit binary.
type DefaultLinuxKitRunner struct {
	BinaryPath string
}

// Build runs "linuxkit build" with the given config, format, architecture, and output directory.
func (r *DefaultLinuxKitRunner) Build(ctx context.Context, configPath, format, arch, outputDir string) error {
	args := []string{"build", "--format", format, "--arch", arch, "--dir", outputDir, configPath}

	cmd := exec.CommandContext(ctx, r.BinaryPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Fprintf(os.Stdout, "Running: linuxkit %v\n", args)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("linuxkit build failed: %w", err)
	}

	return nil
}

// EnsureLinuxKit locates or downloads the linuxkit binary.
// Search order: PATH → cache dir → download from GitHub releases.
func EnsureLinuxKit(ctx context.Context) (string, error) {
	// 1. Check PATH
	if path, err := exec.LookPath("linuxkit"); err == nil {
		return path, nil
	}

	// 2. Check cache
	cacheDir, err := linuxkitCacheDir()
	if err != nil {
		return "", err
	}

	cachedBinary := filepath.Join(cacheDir, fmt.Sprintf("linuxkit-%s", linuxkitVersion))
	if info, statErr := os.Stat(cachedBinary); statErr == nil && !info.IsDir() {
		return cachedBinary, nil
	}

	// 3. Download
	fmt.Fprintf(os.Stdout, "Downloading linuxkit %s...\n", linuxkitVersion)

	downloadURL := LinuxKitDownloadURL()

	if mkdirErr := os.MkdirAll(cacheDir, 0755); mkdirErr != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", mkdirErr)
	}

	if dlErr := downloadFile(ctx, downloadURL, cachedBinary); dlErr != nil {
		return "", fmt.Errorf(
			"failed to download linuxkit: %w\nInstall manually: go install github.com/linuxkit/linuxkit/src/cmd/linuxkit@latest",
			dlErr,
		)
	}

	if chmodErr := os.Chmod(cachedBinary, 0755); chmodErr != nil {
		return "", fmt.Errorf("failed to make linuxkit executable: %w", chmodErr)
	}

	fmt.Fprintf(os.Stdout, "LinuxKit %s installed to %s\n", linuxkitVersion, cachedBinary)

	return cachedBinary, nil
}

func linuxkitCacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	return filepath.Join(home, ".cache", "kdeps", "linuxkit"), nil
}

// LinuxKitDownloadURL returns the download URL for the linuxkit binary.
func LinuxKitDownloadURL() string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	return fmt.Sprintf("%s/%s/linuxkit-%s-%s", linuxkitBaseURL, linuxkitVersion, goos, goarch)
}

func downloadFile(ctx context.Context, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d from %s", resp.StatusCode, url)
	}

	tmpFile := dest + ".tmp"

	out, err := os.Create(tmpFile)
	if err != nil {
		return err
	}

	_, err = io.Copy(out, resp.Body)
	out.Close()

	if err != nil {
		os.Remove(tmpFile)
		return err
	}

	return os.Rename(tmpFile, dest)
}
