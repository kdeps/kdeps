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
	"os"
	"path/filepath"
	"strings"
)

type mockRunnerForCoverage struct{}

func (m *mockRunnerForCoverage) Build(_ context.Context, _, _, _, _, _ string) error {
	return nil
}

func (m *mockRunnerForCoverage) CacheImport(_ context.Context, _ string) error {
	return nil
}

type mockRawBIOSRunner struct{}

func (m *mockRawBIOSRunner) Build(
	_ context.Context,
	configPath, format, _, outputDir, _ string,
) error {
	if format != "kernel+initrd" {
		return nil
	}

	base := filepath.Base(configPath)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	files := map[string]string{
		base + "-kernel":     "fake-kernel",
		base + "-initrd.img": "fake-initrd",
		base + "-cmdline":    "console=ttyS0",
	}
	for name, content := range files {
		if writeErr := os.WriteFile(filepath.Join(outputDir, name), []byte(content), 0644); writeErr != nil {
			return writeErr
		}
	}
	return nil
}

func (m *mockRawBIOSRunner) CacheImport(_ context.Context, _ string) error {
	return nil
}
