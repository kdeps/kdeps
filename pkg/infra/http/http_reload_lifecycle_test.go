// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package http

import (
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func TestLogReloadedWorkflow_NilWorkflow(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	require.NotPanics(t, func() {
		logReloadedWorkflow(&Server{logger: logger, Workflow: nil})
	})
}

func TestLogReloadedWorkflow_WithDetail(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	require.NotPanics(t, func() {
		logReloadedWorkflow(&Server{
			logger: logger,
			Workflow: &domain.Workflow{
				Metadata: domain.WorkflowMetadata{Name: "wf", Version: "1.0"},
			},
		})
	})
}
