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
	"fmt"
	"io"
	"mime/multipart"
	stdhttp "net/http"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func emptyUploadFiles() []*domain.UploadedFile {
	return []*domain.UploadedFile{}
}

func uploadPreferredFieldNames() []string {
	return []string{uploadFieldArray, uploadFieldFiles, uploadFieldFile}
}

func isStandardUploadField(fieldName string) bool {
	return fieldName == uploadFieldFile ||
		fieldName == uploadFieldArray ||
		fieldName == uploadFieldFiles
}

func isEmptyMultipartForm(form *multipart.Form) bool {
	return form == nil || form.File == nil
}

func isEmptyFileList(files []*multipart.FileHeader) bool {
	return len(files) == 0
}

func singleFileSlice(files []*multipart.FileHeader) []*multipart.FileHeader {
	return files[:1]
}

func uploadFieldSuffix(fieldName string) string {
	if isStandardUploadField(fieldName) {
		return ""
	}
	return fmt.Sprintf(" from field %s", fieldName)
}

func readBoundedUploadContent(src io.Reader, maxSize int64) ([]byte, int64, error) {
	content, err := readMultipartFile(io.LimitReader(src, maxSize+1))
	if err != nil {
		return nil, 0, err
	}
	return content, int64(len(content)), nil
}

func isExplicitUploadContentType(contentType string) bool {
	return contentType != "" && contentType != octetStreamContentType
}

func resolveUploadContentType(content []byte, headerContentType string) string {
	contentType := stdhttp.DetectContentType(content)
	if isExplicitUploadContentType(headerContentType) {
		return headerContentType
	}
	return contentType
}

func isSingleStandardUpload(fieldName string, fileCount int) bool {
	return fieldName == uploadFieldFile && fileCount == 1
}
