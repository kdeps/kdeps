// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package http

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	stdhttp "net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/parser/yaml"
	"github.com/kdeps/kdeps/v2/pkg/validator"
)

type nopMultipartFile struct {
	*strings.Reader
}

func (nopMultipartFile) Close() error { return nil }

func TestClientIPFromAddr_NoPort(t *testing.T) {
	assert.Equal(t, "192.168.1.1", clientIPFromAddr("192.168.1.1"))
}

func TestClientIPFromAddr_WithPort(t *testing.T) {
	assert.Equal(t, "192.168.1.1", clientIPFromAddr("192.168.1.1:8080"))
}

func TestClientIPFromAddr_IPv6(t *testing.T) {
	assert.Equal(t, "2001:db8::1", clientIPFromAddr("[2001:db8::1]:443"))
}

func TestIPLimiterStore_CleanupLoop(t *testing.T) {
	orig := limiterCleanupInterval
	limiterCleanupInterval = 5 * time.Millisecond
	t.Cleanup(func() { limiterCleanupInterval = orig })

	store := &ipLimiterStore{
		limiters: make(map[string]*ipLimiter),
		rps:      10,
		burst:    5,
	}
	store.limiters["stale"] = &ipLimiter{
		lastSeen: time.Now().Add(-2 * limiterIdleExpiry),
	}

	done := make(chan struct{})
	go func() {
		store.cleanup()
		close(done)
	}()

	time.Sleep(20 * time.Millisecond)
	_, exists := store.limiters["stale"]
	assert.False(t, exists)
}

func TestResolvePackageEntryPath_AbsError(t *testing.T) {
	orig := filepathAbs
	t.Cleanup(func() { filepathAbs = orig })
	filepathAbs = func(string) (string, error) {
		return "", errors.New("abs failed")
	}

	_, err := resolvePackageEntryPath("/tmp/dest", "file.txt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve target path")
}

func TestExtractPackageEntry_PrefixGuard(t *testing.T) {
	var total int64
	err := extractPackageEntry(
		&tar.Header{Name: "file.txt"},
		"/tmp/base",
		"/other/file.txt",
		tar.NewReader(bytes.NewReader(nil)),
		&total,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid path in package")
}

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

func TestExtractKdepsPackage_InvalidGzip(t *testing.T) {
	tmpDir := t.TempDir()
	err := extractKdepsPackage([]byte("not-a-gzip-archive"), tmpDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a valid gzip archive")
}

func TestExtractKdepsPackage_MaxEntryCount(t *testing.T) {
	orig := maxPackageEntryCountLimit
	maxPackageEntryCountLimit = 2
	t.Cleanup(func() { maxPackageEntryCountLimit = orig })

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)
	for i := range 3 {
		name := fmt.Sprintf("file%d.txt", i)
		content := "x"
		require.NoError(t, tw.WriteHeader(&tar.Header{Name: name, Mode: 0600, Size: int64(len(content))}))
		_, err := tw.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gzw.Close())

	tmpDir := t.TempDir()
	err := extractKdepsPackage(buf.Bytes(), tmpDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "maximum entry count")
}

func TestWriteExtractedFile_PrefixGuard(t *testing.T) {
	var total int64
	err := writeExtractedFile("/tmp/base", "/other/file.txt", bytes.NewReader([]byte("x")), &total)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid target path")
}

func TestWriteExtractedFile_FileSizeLimit(t *testing.T) {
	orig := maxPackageFileSizeLimit
	maxPackageFileSizeLimit = 4
	t.Cleanup(func() { maxPackageFileSizeLimit = orig })

	tmpDir := t.TempDir()
	baseAbs, err := filepath.Abs(tmpDir)
	require.NoError(t, err)
	target := filepath.Join(tmpDir, "large.bin")

	var total int64
	err = writeExtractedFile(baseAbs, target, bytes.NewReader(bytes.Repeat([]byte("x"), 8)), &total)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum allowed size")
}

func TestWriteExtractedFile_TotalSizeLimit(t *testing.T) {
	origFile := maxPackageFileSizeLimit
	origTotal := maxPackageTotalUncompressedLimit
	maxPackageFileSizeLimit = 16
	maxPackageTotalUncompressedLimit = 6
	t.Cleanup(func() {
		maxPackageFileSizeLimit = origFile
		maxPackageTotalUncompressedLimit = origTotal
	})

	tmpDir := t.TempDir()
	baseAbs, err := filepath.Abs(tmpDir)
	require.NoError(t, err)
	target := filepath.Join(tmpDir, "part.bin")

	var total int64
	err = writeExtractedFile(baseAbs, target, bytes.NewReader([]byte("1234567")), &total)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "maximum total uncompressed size")
}

func TestWriteExtractedFile_CloseError(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "file.txt")

	orig := closeExtractedFile
	t.Cleanup(func() { closeExtractedFile = orig })
	closeExtractedFile = func(_ *os.File) error {
		return errors.New("close failed")
	}

	baseAbs, err := filepath.Abs(tmpDir)
	require.NoError(t, err)
	var totalBytes int64
	err = writeExtractedFile(baseAbs, target, bytes.NewReader([]byte("payload")), &totalBytes)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "close failed")
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

func TestPathRegisteredForMethod_UnknownMethod(t *testing.T) {
	router := NewRouter()
	router.GET("/api/test", func(stdhttp.ResponseWriter, *stdhttp.Request) {})

	assert.False(t, router.pathRegisteredForMethod(stdhttp.MethodPost, "/api/test"))
}

func TestRespondWithSuccess_EncodeError(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	req = req.WithContext(context.WithValue(context.Background(), RequestIDKey, "req-1"))

	RespondWithSuccess(rec, req, make(chan int), nil)
	assert.Equal(t, stdhttp.StatusOK, rec.Code)
}

func TestIsSecureRequest_TLS(t *testing.T) {
	req := httptest.NewRequest(stdhttp.MethodGet, "https://example.com/", nil)
	req.TLS = &tls.ConnectionState{}
	assert.True(t, isSecureRequest(req))
}

func TestRecoverPanic_HeadersAlreadyWritten(t *testing.T) {
	rec := httptest.NewRecorder()
	wrapper := &ResponseWriterWrapper{ResponseWriter: rec}
	wrapper.WriteHeader(stdhttp.StatusOK)

	req := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	func() {
		defer RecoverPanic(wrapper, req, false)
		panic("after headers")
	}()

	assert.Equal(t, stdhttp.StatusOK, rec.Code)
}

func TestTemporaryFileStore_CleanupLoop(t *testing.T) {
	orig := cleanupLoopInterval
	cleanupLoopInterval = 5 * time.Millisecond
	t.Cleanup(func() { cleanupLoopInterval = orig })

	tmpDir := t.TempDir()
	store, err := NewTemporaryFileStore(tmpDir)
	require.NoError(t, err)

	file, err := store.Store("old.txt", []byte("data"), "text/plain")
	require.NoError(t, err)
	store.files[file.ID].UploadedAt = time.Now().Add(-2 * time.Hour)

	time.Sleep(20 * time.Millisecond)
	_ = store.Close()
}

func TestServer_SetupHotReload_AbsPathWarning(t *testing.T) {
	orig := filepathAbs
	t.Cleanup(func() { filepathAbs = orig })
	filepathAbs = func(path string) (string, error) {
		if strings.HasSuffix(path, "workflow.yaml") {
			return "", errors.New("abs failed")
		}
		return path, nil
	}

	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "test"},
		Settings: domain.WorkflowSettings{
			APIServer: &domain.APIServerConfig{Routes: []domain.Route{}},
		},
	}
	server, err := NewServer(workflow, nil, slog.Default())
	require.NoError(t, err)
	server.SetWorkflowPath("workflow.yaml")
	server.SetWatcher(&callbackFileWatcher{})

	err = server.SetupHotReload()
	require.NoError(t, err)
}

func TestNewWorkflowParser_SchemaValidatorError(t *testing.T) {
	orig := schemaValidatorFactory
	t.Cleanup(func() { schemaValidatorFactory = orig })
	schemaValidatorFactory = func() (*validator.SchemaValidator, error) {
		return nil, errors.New("schema validator failed")
	}

	_, err := newWorkflowParser()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create schema validator")
}

func TestServer_SetupHotReload_ParserFactoryError(t *testing.T) {
	orig := workflowParserFactory
	t.Cleanup(func() { workflowParserFactory = orig })
	workflowParserFactory = func() (*yaml.Parser, error) {
		return nil, errors.New("parser factory failed")
	}

	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "test"},
	}
	server, err := NewServer(workflow, nil, slog.Default())
	require.NoError(t, err)
	server.SetWatcher(&callbackFileWatcher{})

	err = server.SetupHotReload()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parser factory failed")
}

func TestServer_ReloadWorkflow_EnsureReloadReadyError(t *testing.T) {
	orig := workflowParserFactory
	t.Cleanup(func() { workflowParserFactory = orig })
	workflowParserFactory = func() (*yaml.Parser, error) {
		return nil, errors.New("parser init failed")
	}

	server, err := NewServer(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "test"}}, nil, slog.Default())
	require.NoError(t, err)
	server.parser = nil

	err = server.reloadWorkflow()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parser init failed")
}

func TestServer_EnsureReloadReady_AbsError(t *testing.T) {
	orig := filepathAbs
	t.Cleanup(func() { filepathAbs = orig })
	filepathAbs = func(string) (string, error) {
		return "", errors.New("abs failed")
	}

	server, err := NewServer(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "test"}}, nil, slog.Default())
	require.NoError(t, err)
	server.workflowPath = ""

	err = server.ensureReloadReady()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve workflow path")
}

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

func TestWebServer_HandleAppRequest_ParseURLError(t *testing.T) {
	orig := parseProxyURL
	t.Cleanup(func() { parseProxyURL = orig })
	parseProxyURL = func(string) (*url.URL, error) {
		return nil, errors.New("parse failed")
	}

	workflow := &domain.Workflow{Settings: domain.WorkflowSettings{WebServer: &domain.WebServerConfig{}}}
	webServer, err := NewWebServer(workflow, slog.Default())
	require.NoError(t, err)

	route := &domain.WebRoute{AppPort: 8080, Path: "/"}
	req := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	webServer.HandleAppRequest(rec, req, route)
	assert.Equal(t, stdhttp.StatusInternalServerError, rec.Code)
}

func TestWebServer_HandleWebSocketProxy_NonSwitchingResponse(t *testing.T) {
	echoSrv := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		upgrader := websocket.Upgrader{CheckOrigin: func(*stdhttp.Request) bool { return true }}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
	}))
	defer echoSrv.Close()

	wsURL := "ws://" + echoSrv.Listener.Addr().String()
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	orig := dialTargetWebSocketHook
	t.Cleanup(func() { dialTargetWebSocketHook = orig })
	dialTargetWebSocketHook = func(_ url.URL, _ stdhttp.Header) (*websocket.Conn, *stdhttp.Response, error) {
		return conn, &stdhttp.Response{StatusCode: stdhttp.StatusBadRequest, Body: stdhttp.NoBody}, nil
	}

	workflow := &domain.Workflow{Settings: domain.WorkflowSettings{WebServer: &domain.WebServerConfig{}}}
	webServer, err := NewWebServer(workflow, slog.Default())
	require.NoError(t, err)

	req := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("Sec-WebSocket-Version", "13")
	rec := httptest.NewRecorder()

	targetURL, err := url.Parse(wsURL)
	require.NoError(t, err)
	webServer.HandleWebSocketProxy(rec, req, targetURL, &domain.WebRoute{Path: "/"})
	assert.Equal(t, stdhttp.StatusBadGateway, rec.Code)
}

func TestWebServer_HandleWebSocketProxy_HandshakeFailure(t *testing.T) {
	targetSrv := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		w.WriteHeader(stdhttp.StatusBadRequest)
	}))
	defer targetSrv.Close()

	targetURL, err := url.Parse("ws://" + targetSrv.Listener.Addr().String())
	require.NoError(t, err)

	workflow := &domain.Workflow{Settings: domain.WorkflowSettings{WebServer: &domain.WebServerConfig{}}}
	webServer, err := NewWebServer(workflow, slog.Default())
	require.NoError(t, err)

	req := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("Sec-WebSocket-Version", "13")
	rec := httptest.NewRecorder()

	webServer.HandleWebSocketProxy(rec, req, targetURL, &domain.WebRoute{Path: "/"})
	assert.Equal(t, stdhttp.StatusBadGateway, rec.Code)
}

func TestRelayWebSocketMessages_UnexpectedClose(t *testing.T) {
	serverA := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		upgrader := websocket.Upgrader{CheckOrigin: func(*stdhttp.Request) bool { return true }}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			if _, _, readErr := conn.ReadMessage(); readErr != nil {
				return
			}
		}
	}))
	defer serverA.Close()

	dialer := websocket.Dialer{}
	wsURL := "ws://" + serverA.Listener.Addr().String()
	srcConn, _, err := dialer.Dial(wsURL, nil)
	require.NoError(t, err)

	dstConn, _, err := dialer.Dial(wsURL, nil)
	require.NoError(t, err)

	errCh := make(chan error, 1)
	go relayWebSocketMessages(srcConn, dstConn, "src", "dst", slog.Default(), errCh)

	require.NoError(t, srcConn.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "boom"),
	))

	select {
	case relayErr := <-errCh:
		require.Error(t, relayErr)
	case <-time.After(2 * time.Second):
		t.Fatal("expected unexpected close error")
	}
}

func TestRelayWebSocketMessages_WriteError(t *testing.T) {
	serverA := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		upgrader := websocket.Upgrader{CheckOrigin: func(*stdhttp.Request) bool { return true }}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			mt, msg, readErr := conn.ReadMessage()
			if readErr != nil {
				return
			}
			_ = conn.WriteMessage(mt, msg)
		}
	}))
	defer serverA.Close()

	dialer := websocket.Dialer{}
	wsURL := "ws://" + serverA.Listener.Addr().String()
	srcConn, _, err := dialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer srcConn.Close()

	dstConn, _, err := dialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer dstConn.Close()

	orig := writeWebSocketMessageHook
	t.Cleanup(func() { writeWebSocketMessageHook = orig })
	writeWebSocketMessageHook = func(*websocket.Conn, int, []byte) error {
		return errors.New("write failed")
	}

	errCh := make(chan error, 1)
	go relayWebSocketMessages(srcConn, dstConn, "src", "dst", slog.Default(), errCh)

	require.NoError(t, srcConn.WriteMessage(websocket.TextMessage, []byte("ping")))
	select {
	case relayErr := <-errCh:
		require.Error(t, relayErr)
		assert.Contains(t, relayErr.Error(), "write failed")
	case <-time.After(2 * time.Second):
		t.Fatal("expected write error")
	}
}
