// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package http

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func TestUploadHandler_NilMultipartFileMap(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewTemporaryFileStore(tmpDir)
	require.NoError(t, err)
	handler := NewUploadHandler(store, 1024)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	require.NoError(t, writer.WriteField("note", "value only"))
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(stdhttp.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	require.NoError(t, req.ParseMultipartForm(MaxMemory))
	req.MultipartForm.File = nil

	files, err := handler.HandleUpload(req)
	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestUploadHandler_NilMultipartForm(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewTemporaryFileStore(tmpDir)
	require.NoError(t, err)
	handler := NewUploadHandler(store, 1024)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	require.NoError(t, writer.WriteField("note", "value only"))
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(stdhttp.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	files, err := handler.HandleUpload(req)
	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestUploadHandler_EmptyMultipartForm(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewTemporaryFileStore(tmpDir)
	require.NoError(t, err)
	handler := NewUploadHandler(store, 1024)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(stdhttp.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	files, err := handler.HandleUpload(req)
	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestUploadHandler_CollectAllUploadFiles_SkipEmptyField(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewTemporaryFileStore(tmpDir)
	require.NoError(t, err)
	handler := NewUploadHandler(store, 1024)

	files, err := handler.collectAllUploadFiles(map[string][]*multipart.FileHeader{
		"empty": {},
	})
	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestUploadHandler_ProcessFileHeader_HeaderSizeLimit(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewTemporaryFileStore(tmpDir)
	require.NoError(t, err)
	handler := NewUploadHandler(store, 10)

	_, err = handler.processFileHeader(&multipart.FileHeader{Filename: "big.txt", Size: 20}, "file")
	require.Error(t, err)
	var appErr *domain.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, domain.ErrCodeRequestTooLarge, appErr.Code)
}

func TestUploadHandler_ProcessFileHeader_ContentSizeAfterRead(t *testing.T) {
	origOpen := openMultipartFile
	origRead := readMultipartFile
	t.Cleanup(func() {
		openMultipartFile = origOpen
		readMultipartFile = origRead
	})
	openMultipartFile = func(*multipart.FileHeader) (multipart.File, error) {
		return nopMultipartFile{strings.NewReader("small")}, nil
	}
	readMultipartFile = func(io.Reader) ([]byte, error) {
		return bytes.Repeat([]byte("x"), 20), nil
	}

	tmpDir := t.TempDir()
	store, err := NewTemporaryFileStore(tmpDir)
	require.NoError(t, err)
	handler := NewUploadHandler(store, 10)

	_, err = handler.processFileHeader(&multipart.FileHeader{Filename: "big.txt", Size: 5}, "file")
	require.Error(t, err)
	var appErr *domain.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, domain.ErrCodeRequestTooLarge, appErr.Code)
}

func TestUploadHandler_ProcessFileHeader_OpenError(t *testing.T) {
	orig := openMultipartFile
	t.Cleanup(func() { openMultipartFile = orig })
	openMultipartFile = func(*multipart.FileHeader) (multipart.File, error) {
		return nil, errors.New("open failed")
	}

	tmpDir := t.TempDir()
	store, err := NewTemporaryFileStore(tmpDir)
	require.NoError(t, err)
	handler := NewUploadHandler(store, 1024)

	_, err = handler.processFileHeader(&multipart.FileHeader{Filename: "bad.txt", Size: 1}, "file")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open uploaded file")
}

func TestUploadHandler_ProcessFileHeader_ReadError(t *testing.T) {
	orig := readMultipartFile
	t.Cleanup(func() { readMultipartFile = orig })
	readMultipartFile = func(io.Reader) ([]byte, error) {
		return nil, errors.New("read failed")
	}

	tmpDir := t.TempDir()
	store, err := NewTemporaryFileStore(tmpDir)
	require.NoError(t, err)
	handler := NewUploadHandler(store, 1024)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "test.txt")
	require.NoError(t, err)
	_, err = part.Write([]byte("data"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(stdhttp.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	files, err := handler.HandleUpload(req)
	require.Error(t, err)
	assert.Nil(t, files)
	assert.Contains(t, err.Error(), "failed to read file content")
}

func TestUploadHandler_CollectAllUploadFiles_SkipEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewTemporaryFileStore(tmpDir)
	require.NoError(t, err)
	handler := NewUploadHandler(store, 1024)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("custom", "a.txt")
	require.NoError(t, err)
	_, err = part.Write([]byte("ok"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(stdhttp.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	files, err := handler.HandleUpload(req)
	require.NoError(t, err)
	require.Len(t, files, 1)
}
