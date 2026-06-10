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
	"strings"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

type nopMultipartFile struct {
	*strings.Reader
}

func (nopMultipartFile) Close() error { return nil }

// errFileStore implements domain.FileStore and returns an error on Delete
// while delegating all other operations to a real TemporaryFileStore.
type errFileStore struct {
	domain.FileStore
}

func (s *errFileStore) Delete(_ string) error {
	return errors.New("simulated delete error")
}

// whiteboxMockExecutor is a minimal WorkflowExecutor for whitebox tests.
type whiteboxMockExecutor struct{}

func (e *whiteboxMockExecutor) Execute(_ *domain.Workflow, _ interface{}) (interface{}, error) {
	return map[string]interface{}{"result": "ok"}, nil
}

// callbackFileWatcher is a mock watcher that captures registered callbacks
// for later invocation.
type callbackFileWatcher struct {
	watchedPaths []string
	callbacks    []func()
}

func (w *callbackFileWatcher) Watch(path string, callback func()) error {
	w.watchedPaths = append(w.watchedPaths, path)
	w.callbacks = append(w.callbacks, callback)
	return nil
}

func (w *callbackFileWatcher) Close() error {
	return nil
}
