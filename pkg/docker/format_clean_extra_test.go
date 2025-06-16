package docker

import (
	"encoding/json"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/spf13/afero"
)

func TestFormatResponseJSON_NestedData(t *testing.T) {
	// Build a response where data[0] is a JSON string
	payload := APIResponse{
		Success:  true,
		Response: ResponseData{Data: []string{`{"foo":123}`}},
		Meta:     ResponseMeta{RequestID: "id"},
	}
	raw, _ := json.Marshal(payload)
	pretty := formatResponseJSON(raw)

	// The nested JSON should have been parsed → data[0] becomes an object not string
	var out map[string]interface{}
	if err := json.Unmarshal(pretty, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	resp, ok := out["response"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing response field")
	}
	dataArr, ok := resp["data"].([]interface{})
	if !ok || len(dataArr) != 1 {
		t.Fatalf("unexpected data field: %v", resp["data"])
	}
	first, ok := dataArr[0].(map[string]interface{})
	if !ok {
		t.Fatalf("data[0] still a string after formatting")
	}
	if val, ok := first["foo"].(float64); !ok || val != 123 {
		t.Fatalf("nested JSON not preserved: %v", first)
	}
}

func TestCleanOldFilesUnique(t *testing.T) {
	fs := afero.NewOsFs()
	tmpDir, _ := afero.TempDir(fs, "", "clean")
	target := tmpDir + "/resp.json"
	_ = afero.WriteFile(fs, target, []byte("data"), 0o644)

	dr := &resolver.DependencyResolver{Fs: fs, Logger: logging.NewTestLogger(), ResponseTargetFile: target}
	if err := cleanOldFiles(dr); err != nil {
		t.Fatalf("cleanOldFiles error: %v", err)
	}
	if exists, _ := afero.Exists(fs, target); exists {
		t.Fatalf("file still exists after cleanOldFiles")
	}
}
