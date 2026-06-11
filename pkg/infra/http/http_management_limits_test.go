// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package http

import (
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractKdepsPackage_AbsDestError(t *testing.T) {
	orig := filepathAbs
	t.Cleanup(func() { filepathAbs = orig })
	filepathAbs = func(path string) (string, error) {
		if path == "/bad/dest" {
			return "", errors.New("abs failed")
		}
		return path, nil
	}

	err := extractKdepsPackage([]byte{}, "/bad/dest")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve destination directory")
}

func TestGetManagementWorkflowPath_AppDirectoryWithWorkflowFile(t *testing.T) {
	origStat := osStat
	origFind := findWorkflowFileHook
	t.Cleanup(func() {
		osStat = origStat
		findWorkflowFileHook = origFind
	})
	osStat = func(name string) (os.FileInfo, error) {
		if name == "/app" {
			return nil, nil
		}
		return nil, os.ErrNotExist
	}
	findWorkflowFileHook = func(string) string {
		return "/app/workflow.yaml.j2"
	}

	server, err := NewServer(nil, nil, slog.Default())
	require.NoError(t, err)

	assert.Equal(t, "/app/workflow.yaml.j2", server.getManagementWorkflowPath())
}

func TestGetManagementWorkflowPath_AppDirectory(t *testing.T) {
	orig := osStat
	t.Cleanup(func() { osStat = orig })
	osStat = func(name string) (os.FileInfo, error) {
		if name == "/app" {
			return nil, nil
		}
		return nil, os.ErrNotExist
	}

	server, err := NewServer(nil, nil, slog.Default())
	require.NoError(t, err)

	assert.Equal(t, "/app/workflow.yaml", server.getManagementWorkflowPath())
}
