package resolver

import (
	"encoding/base64"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/utils"
	pklHTTP "github.com/kdeps/schema/gen/http"
	"github.com/spf13/afero"
)

func TestDecodeHTTPBlock_Base64(t *testing.T) {
	url := "https://example.com"
	urlEnc := base64.StdEncoding.EncodeToString([]byte(url))
	headerVal := utils.EncodeValue("application/json")
	paramVal := utils.EncodeValue("q")
	dataVal := utils.EncodeValue("body")

	client := &pklHTTP.ResourceHTTPClient{
		Url:     urlEnc,
		Headers: &map[string]string{"Content-Type": headerVal},
		Params:  &map[string]string{"search": paramVal},
		Data:    &[]string{dataVal},
	}

	dr := &DependencyResolver{Logger: logging.GetLogger()}
	if err := dr.decodeHTTPBlock(client); err != nil {
		t.Fatalf("decodeHTTPBlock returned error: %v", err)
	}

	if client.Url != url {
		t.Errorf("URL not decoded: %s", client.Url)
	}
	if (*client.Headers)["Content-Type"] != "application/json" {
		t.Errorf("header not decoded: %v", client.Headers)
	}
	if (*client.Params)["search"] != "q" {
		t.Errorf("param not decoded: %v", client.Params)
	}
	if (*client.Data)[0] != "body" {
		t.Errorf("data not decoded: %v", client.Data)
	}
}

func TestEncodeResponseHelpers(t *testing.T) {
	tmp := t.TempDir()
	fs := afero.NewOsFs()
	dr := &DependencyResolver{
		Fs:        fs,
		FilesDir:  tmp,
		RequestID: "rid",
		Logger:    logging.GetLogger(),
	}
	body := "hello world"
	headers := map[string]string{"X-Test": "val"}
	resp := &pklHTTP.ResponseBlock{Body: &body, Headers: &headers}

	encodedHeaders := encodeResponseHeaders(resp)
	if !strings.Contains(encodedHeaders, "X-Test") || !strings.Contains(encodedHeaders, utils.EncodeValue("val")) {
		t.Errorf("encoded headers missing values: %s", encodedHeaders)
	}

	resourceID := "res1"
	encodedBody := encodeResponseBody(resp, dr, resourceID)
	if !strings.Contains(encodedBody, utils.EncodeValue(body)) {
		t.Errorf("encoded body missing: %s", encodedBody)
	}

	// ensure file was created
	expectedFile := filepath.Join(tmp, utils.GenerateResourceIDFilename(resourceID, dr.RequestID))
	if exists, _ := afero.Exists(fs, expectedFile); !exists {
		t.Errorf("expected file not written: %s", expectedFile)
	}

	// Nil cases
	emptyHeaders := encodeResponseHeaders(nil)
	if emptyHeaders != "    headers {[\"\"] = \"\"}\n" {
		t.Errorf("unexpected default headers: %s", emptyHeaders)
	}
	emptyBody := encodeResponseBody(nil, dr, resourceID)
	if emptyBody != "    body=\"\"\n" {
		t.Errorf("unexpected default body: %s", emptyBody)
	}
}

func TestIsMethodWithBody(t *testing.T) {
	if !isMethodWithBody("POST") || !isMethodWithBody("put") {
		t.Errorf("expected POST/PUT to allow body")
	}
	if isMethodWithBody("GET") || isMethodWithBody("HEAD") {
		t.Errorf("expected GET/HEAD to not allow body")
	}
}
