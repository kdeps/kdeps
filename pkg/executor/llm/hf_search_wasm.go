//go:build js

package llm

import (
	"context"
	"errors"
	"log/slog"
)

// ErrHFNotSupported is returned when HuggingFace operations are attempted in WASM.
var ErrHFNotSupported = errors.New("huggingface search is not supported in WASM builds")

type HFModelResult struct {
	ID        string
	Downloads int
	Likes     int
	Tags      []string
}

type HFFileEntry struct {
	Filename string
	Size     int64
}

type HFRepoInfo struct {
	ID       string
	Siblings []HFFileEntry
}

func HFSearchGGUF(_ context.Context, _ string, _ int) ([]HFModelResult, error) {
	return nil, ErrHFNotSupported
}

func HFSearchGGUFWithBase(_ context.Context, _, _ string, _ int) ([]HFModelResult, error) {
	return nil, ErrHFNotSupported
}

func HFRepoFiles(_ context.Context, _ string) (HFRepoInfo, error) {
	return HFRepoInfo{}, ErrHFNotSupported
}

func HFRepoFilesWithBase(_ context.Context, _, _ string) (HFRepoInfo, error) {
	return HFRepoInfo{}, ErrHFNotSupported
}

func HFGGUFFiles(files []HFFileEntry) []HFFileEntry { return nil }

func HFDownloadURL(repoID, filename string) string { return "" }

func HFDownloadGGUF(_ context.Context, _, _ string, _ *slog.Logger) (string, string, error) {
	return "", "", ErrHFNotSupported
}

func HFRegisterGGUFEntry(_ GGUFEntry) error { return ErrHFNotSupported }
