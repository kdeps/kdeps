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

//nolint:mnd // cleanup intervals are intentional
package http

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

// TemporaryFileStore implements FileStore for temporary uploads.
type TemporaryFileStore struct {
	baseDir string
	files   map[string]*domain.UploadedFile
	mu      sync.RWMutex
	stopCh  chan struct{}
	stopped bool
}

// NewTemporaryFileStore creates a new temporary file store.
func NewTemporaryFileStore(baseDir string) (*TemporaryFileStore, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}

	store := &TemporaryFileStore{
		baseDir: baseDir,
		files:   make(map[string]*domain.UploadedFile),
		stopCh:  make(chan struct{}),
		stopped: false,
	}

	// Start background cleanup
	go store.cleanupLoop(30 * time.Minute)

	return store, nil
}

// Store saves an uploaded file.
func (s *TemporaryFileStore) Store(filename string, content []byte, contentType string) (*domain.UploadedFile, error) {
	// Generate unique ID using hash of content + timestamp
	hash := sha256.Sum256(append(content, []byte(time.Now().String())...))
	id := hex.EncodeToString(hash[:])[:16]

	// Sanitize filename
	safeFilename := filepath.Base(filename)

	// Create file path
	filePath := filepath.Join(s.baseDir, id+"_"+safeFilename)

	// Write file to disk
	if err := os.WriteFile(filePath, content, 0600); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	// Create metadata
	file := &domain.UploadedFile{
		ID:          id,
		Filename:    safeFilename,
		ContentType: contentType,
		Size:        int64(len(content)),
		Path:        filePath,
		UploadedAt:  time.Now(),
		Metadata:    make(map[string]string),
	}

	// Store in memory index
	s.mu.Lock()
	s.files[id] = file
	s.mu.Unlock()

	return file, nil
}

// Get retrieves file metadata by ID.
func (s *TemporaryFileStore) Get(id string) (*domain.UploadedFile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	file, exists := s.files[id]
	if !exists {
		return nil, fmt.Errorf("file not found: %s", id)
	}

	return file, nil
}

// GetPath returns the filesystem path for a file ID.
func (s *TemporaryFileStore) GetPath(id string) (string, error) {
	file, err := s.Get(id)
	if err != nil {
		return "", err
	}
	return file.Path, nil
}

// Delete removes a file.
func (s *TemporaryFileStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, exists := s.files[id]
	if !exists {
		return fmt.Errorf("file not found: %s", id)
	}

	// Remove from disk
	if err := os.Remove(file.Path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	// Remove from memory
	delete(s.files, id)

	return nil
}

// Cleanup removes files older than TTL.
func (s *TemporaryFileStore) Cleanup(ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-ttl)
	var toDelete []string

	for id, file := range s.files {
		if file.UploadedAt.Before(cutoff) {
			toDelete = append(toDelete, id)
		}
	}

	for _, id := range toDelete {
		file := s.files[id]
		_ = os.Remove(file.Path) // Ignore errors for cleanup
		delete(s.files, id)
	}

	return nil
}

// Close stops the file store and cleanup background tasks.
func (s *TemporaryFileStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		return nil
	}

	close(s.stopCh)
	s.stopped = true

	// Clean up all remaining files
	for id, file := range s.files {
		_ = os.Remove(file.Path) // Ignore errors
		delete(s.files, id)
	}

	return nil
}

// cleanupLoop runs periodic cleanup.
func (s *TemporaryFileStore) cleanupLoop(ttl time.Duration) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_ = s.Cleanup(ttl) // Ignore errors in background cleanup
		case <-s.stopCh:
			return
		}
	}
}
