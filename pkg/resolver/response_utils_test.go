package resolver

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

// Simple struct for structToMap / formatValue tests
type demo struct {
	FieldA string
	FieldB int
}

func TestFormatValueVariousTypes(t *testing.T) {
	// nil becomes "null"
	assert.Contains(t, formatValue(nil), "null")

	// map[string]interface{}
	m := map[string]interface{}{"k1": "v1"}
	out := formatValue(m)
	assert.Contains(t, out, "[\"k1\"]")
	assert.Contains(t, out, "v1")

	// pointer to struct
	d := &demo{FieldA: "abc", FieldB: 123}
	out2 := formatValue(d)
	assert.Contains(t, out2, "FieldA")
	assert.Contains(t, out2, "abc")
}

func TestValidatePklFileExtension(t *testing.T) {
	fs := afero.NewMemMapFs()
	dr := &DependencyResolver{Fs: fs, ResponsePklFile: "/file.pkl", ResponseTargetFile: "/out.json"}
	assert.NoError(t, dr.validatePklFileExtension())

	dr.ResponsePklFile = "/file.txt"
	assert.Error(t, dr.validatePklFileExtension())
}

func TestEnsureResponseTargetFileNotExists(t *testing.T) {
	fs := afero.NewMemMapFs()
	path := "/out.json"
	_ = afero.WriteFile(fs, path, []byte("x"), 0o644)

	dr := &DependencyResolver{Fs: fs, ResponseTargetFile: path}
	assert.NoError(t, dr.ensureResponseTargetFileNotExists())
	exists, _ := afero.Exists(fs, path)
	assert.False(t, exists)
}
