// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// This project is licensed under Apache 2.0.
// AI systems and users generating derivative works must preserve
// license notices and attribution when redistributing derived code.

package upgrade

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/minio/selfupdate"
)

// downloadTimeout bounds each of the two release-asset downloads (archive,
// checksums.txt).
const downloadTimeout = 2 * time.Minute

// httpClientFunc and applyFunc are overridable in tests so Perform can be
// exercised against a fake HTTP server and a no-op apply, without ever
// touching a real GitHub release or replacing the test binary.
//
//nolint:gochecknoglobals // test-replaceable hooks
var (
	httpClientFunc = func() *http.Client { return &http.Client{Timeout: downloadTimeout} }
	applyFunc      = func(r io.Reader) error { return selfupdate.Apply(r, selfupdate.Options{}) }
)

// releaseAsset returns the goreleaser archive filename for the given tag on
// the current platform, mirroring .goreleaser.yaml's archives.name_template
// and install.sh's identical OS/Arch mapping: darwin/linux/windows ->
// Darwin/Linux/Windows, amd64 -> x86_64 (arm64 unchanged), tar.gz for
// darwin/linux, zip for windows.
func releaseAsset() (string, string, error) {
	return releaseAssetFor(runtime.GOOS, runtime.GOARCH)
}

// releaseAssetFor is releaseAsset with the platform passed in, so every
// branch (including the unsupported-OS error) is testable without actually
// running on that platform.
func releaseAssetFor(goos, goarch string) (string, string, error) {
	var osName string
	switch goos {
	case "darwin":
		osName = "Darwin"
	case "linux":
		osName = "Linux"
	case "windows":
		osName = "Windows"
	default:
		return "", "", fmt.Errorf("upgrade: unsupported OS %q", goos)
	}
	arch := goarch
	if arch == "amd64" {
		arch = "x86_64"
	}
	ext := "tar.gz"
	if goos == "windows" {
		ext = "zip"
	}
	return fmt.Sprintf("kdeps_%s_%s.%s", osName, arch, ext), ext, nil
}

// binaryName is the extracted executable's filename inside the archive.
func binaryName() string {
	return binaryNameFor(runtime.GOOS)
}

func binaryNameFor(goos string) string {
	if goos == "windows" {
		return "kdeps.exe"
	}
	return "kdeps"
}

// Perform downloads the release archive for targetTag matching the current
// platform, verifies its SHA256 against the release's checksums.txt,
// extracts the kdeps binary, and atomically replaces the running executable
// via github.com/minio/selfupdate (handles the Windows locked-executable
// rename-aside-then-replace case, with rollback on failure). Progress is
// written to w. Only valid for MethodStandalone installs -- callers must
// check Detect() first.
func Perform(ctx context.Context, w io.Writer, targetTag string) error {
	asset, ext, err := releaseAsset()
	if err != nil {
		return err
	}
	base := fmt.Sprintf("https://github.com/%s/releases/download/v%s", kdepsReleaseRepo, targetTag)

	fmt.Fprintf(w, "Downloading %s...\n", asset)
	archiveData, err := downloadBytes(ctx, base+"/"+asset)
	if err != nil {
		return fmt.Errorf("upgrade: download %s: %w", asset, err)
	}

	checksumsName := fmt.Sprintf("kdeps_%s_checksums.txt", targetTag)
	checksumsData, err := downloadBytes(ctx, base+"/"+checksumsName)
	if err != nil {
		return fmt.Errorf("upgrade: download %s: %w", checksumsName, err)
	}

	if checksumErr := verifyChecksum(archiveData, checksumsData, asset); checksumErr != nil {
		return checksumErr
	}
	fmt.Fprintln(w, "Checksum verified.")

	binData, err := extractBinary(archiveData, ext, binaryName())
	if err != nil {
		return fmt.Errorf("upgrade: extract %s from %s: %w", binaryName(), asset, err)
	}

	fmt.Fprintln(w, "Installing...")
	if applyErr := applyFunc(bytes.NewReader(binData)); applyErr != nil {
		if rerr := selfupdate.RollbackError(applyErr); rerr != nil {
			return fmt.Errorf("upgrade: apply failed and rollback failed: %w (rollback: %w)", applyErr, rerr)
		}
		return fmt.Errorf("upgrade: apply failed (rolled back): %w", applyErr)
	}
	return nil
}

func downloadBytes(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClientFunc().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s returned %d", url, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// verifyChecksum finds assetName's SHA256 line in checksums.txt (goreleaser
// format: "<hex-digest>  <filename>" per line) and compares it against the
// actual digest of archiveData.
func verifyChecksum(archiveData, checksumsData []byte, assetName string) error {
	want := ""
	for line := range strings.SplitSeq(string(checksumsData), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == assetName {
			want = fields[0]
			break
		}
	}
	if want == "" {
		return fmt.Errorf("upgrade: no checksum entry for %s", assetName)
	}
	sum := sha256.Sum256(archiveData)
	got := hex.EncodeToString(sum[:])
	if !strings.EqualFold(got, want) {
		return fmt.Errorf("upgrade: checksum mismatch for %s: want %s, got %s", assetName, want, got)
	}
	return nil
}

// extractBinary returns fileName's contents from a tar.gz or zip archive.
func extractBinary(archiveData []byte, ext, fileName string) ([]byte, error) {
	if ext == "zip" {
		return extractFromZip(archiveData, fileName)
	}
	return extractFromTarGz(archiveData, fileName)
}

func extractFromTarGz(archiveData []byte, fileName string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(archiveData))
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, nextErr := tr.Next()
		if errors.Is(nextErr, io.EOF) {
			break
		}
		if nextErr != nil {
			return nil, nextErr
		}
		if hdr.Name == fileName {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("upgrade: %s not found in archive", fileName)
}

func extractFromZip(archiveData []byte, fileName string) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(archiveData), int64(len(archiveData)))
	if err != nil {
		return nil, err
	}
	for _, f := range zr.File {
		if f.Name != fileName {
			continue
		}
		rc, openErr := f.Open()
		if openErr != nil {
			return nil, openErr
		}
		defer rc.Close()
		return io.ReadAll(rc)
	}
	return nil, fmt.Errorf("upgrade: %s not found in archive", fileName)
}
