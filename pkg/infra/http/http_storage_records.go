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

package http

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"time"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func storedUploadPath(baseDir, id, filename string) string {
	return filepath.Join(baseDir, id+"_"+filename)
}

func newUploadedFileRecord(
	id, filename, contentType, filePath string,
	size int64,
) *domain.UploadedFile {
	return &domain.UploadedFile{
		ID:          id,
		Filename:    filename,
		ContentType: contentType,
		Size:        size,
		Path:        filePath,
		UploadedAt:  time.Now(),
		Metadata:    newUploadMetadataMap(),
	}
}

func lookupStoredFile(
	files map[string]*domain.UploadedFile,
	id string,
) (*domain.UploadedFile, error) {
	file, exists := files[id]
	if !exists {
		return nil, fileNotFoundError(id)
	}
	return file, nil
}

func generateUploadID(content []byte) string {
	hash := sha256.Sum256(append(content, []byte(time.Now().String())...))
	return hex.EncodeToString(hash[:])[:16]
}

func expiredFileIDs(files map[string]*domain.UploadedFile, cutoff time.Time) []string {
	var ids []string
	for id, file := range files {
		if file.UploadedAt.Before(cutoff) {
			ids = append(ids, id)
		}
	}
	return ids
}

func removeStoredFileEntry(files map[string]*domain.UploadedFile, id string) {
	file := files[id]
	_ = removeUploadedFile(file)
	delete(files, id)
}

func removeUploadedFile(file *domain.UploadedFile) error {
	if err := os.Remove(file.Path); err != nil && !isNotExistErr(err) {
		return storageDeleteFileFailed(err)
	}
	return nil
}
