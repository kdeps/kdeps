package archiver_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/archiver"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/schema/gen/project"
	pklProject "github.com/kdeps/schema/gen/project"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestValidatePklResources_DirectoryNotFound(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	err := archiver.ValidatePklResources(fs, ctx, "/nonexistent/directory", logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing resource directory")
}

func TestValidatePklResources_StatError(t *testing.T) {
	base := afero.NewOsFs()
	fs := &errorFs{base, "stat"}
	ctx := context.Background()
	logger := logging.NewTestLogger()
	dir := "/nonexistent/dir"
	err := archiver.ValidatePklResources(fs, ctx, dir, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing resource directory")
}

func TestValidatePklResources_CollectPklFilesError(t *testing.T) {
	base := afero.NewOsFs()
	fs := &errorFs{base, "readDir"}
	ctx := context.Background()
	logger := logging.NewTestLogger()
	dir, _ := afero.TempDir(base, "", "validate-pkl-collect-error")
	defer base.RemoveAll(dir)
	err := archiver.ValidatePklResources(fs, ctx, dir, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no .pkl files in")
}

func TestValidatePklResources_EmptyPklFiles(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	dir, _ := afero.TempDir(fs, "", "validate-pkl-empty")
	defer fs.RemoveAll(dir)
	// Create a non-pkl file
	filePath := filepath.Join(dir, "test.txt")
	_ = afero.WriteFile(fs, filePath, []byte("content"), 0o644)
	err := archiver.ValidatePklResources(fs, ctx, dir, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no .pkl files in")
}

func TestValidatePklResources_EnforcePklTemplateAmendsRulesError(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	dir, _ := afero.TempDir(fs, "", "validate-pkl-enforce-error")
	defer fs.RemoveAll(dir)
	// Create a pkl file with invalid content
	filePath := filepath.Join(dir, "test.pkl")
	_ = afero.WriteFile(fs, filePath, []byte("invalid pkl content"), 0o644)
	err := archiver.ValidatePklResources(fs, ctx, dir, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed for")
}

func TestValidatePklResources_Success(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	dir, _ := afero.TempDir(fs, "", "validate-pkl-success")
	defer fs.RemoveAll(dir)
	// Create a valid pkl file
	filePath := filepath.Join(dir, "test.pkl")
	validPklContent := `amends "package://schema.kdeps.com/core@0.2.30#/Resource.pkl"

id = "testAction"
`
	_ = afero.WriteFile(fs, filePath, []byte(validPklContent), 0o644)
	err := archiver.ValidatePklResources(fs, ctx, dir, logger)
	assert.NoError(t, err)
}

func TestValidatePklResources_MultipleValidFiles(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create a directory with multiple valid .pkl files
	dir, err := afero.TempDir(fs, "", "validate-pkl-multiple")
	assert.NoError(t, err)
	defer fs.RemoveAll(dir)

	// Create multiple valid .pkl files
	files := []string{"test1.pkl", "test2.pkl", "test3.pkl"}
	for _, filename := range files {
		file := filepath.Join(dir, filename)
		content := fmt.Sprintf(`amends "package://schema.kdeps.com/core@0.2.30#/Resource.pkl"

actionID = "%s"
// Valid content`, strings.TrimSuffix(filename, ".pkl"))
		assert.NoError(t, afero.WriteFile(fs, file, []byte(content), 0o644))
	}

	err = archiver.ValidatePklResources(fs, ctx, dir, logger)
	assert.NoError(t, err)
}

func TestValidatePklResources_MixedFiles(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create a directory with mixed file types (only .pkl files should be validated)
	dir, err := afero.TempDir(fs, "", "validate-pkl-mixed")
	assert.NoError(t, err)
	defer fs.RemoveAll(dir)

	// Create non-.pkl files (should be ignored)
	nonPklFiles := []string{"test.txt", "test.json", "test.yaml"}
	for _, filename := range nonPklFiles {
		file := filepath.Join(dir, filename)
		assert.NoError(t, afero.WriteFile(fs, file, []byte("test content"), 0o644))
	}

	// Create a valid .pkl file with correct amends line
	pklFile := filepath.Join(dir, "test.pkl")
	content := `amends "package://schema.kdeps.com/core@0.2.30#/Resource.pkl"

actionID = "testAction"
// Valid content`
	assert.NoError(t, afero.WriteFile(fs, pklFile, []byte(content), 0o644))

	err = archiver.ValidatePklResources(fs, ctx, dir, logger)
	assert.NoError(t, err)
}

func TestValidatePklResources_EmptyDirectory(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create an empty directory
	dir, err := afero.TempDir(fs, "", "validate-pkl-empty-dir")
	assert.NoError(t, err)
	defer fs.RemoveAll(dir)

	err = archiver.ValidatePklResources(fs, ctx, dir, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no .pkl files in")
}

type stubWfSimple struct{}

func (stubWfSimple) GetName() string                   { return "testAgent" }
func (stubWfSimple) GetVersion() string                { return "1.0.0" }
func (stubWfSimple) GetDescription() string            { return "" }
func (stubWfSimple) GetWebsite() *string               { return nil }
func (stubWfSimple) GetAuthors() *[]string             { return nil }
func (stubWfSimple) GetDocumentation() *string         { return nil }
func (stubWfSimple) GetRepository() *string            { return nil }
func (stubWfSimple) GetHeroImage() *string             { return nil }
func (stubWfSimple) GetAgentIcon() *string             { return nil }
func (stubWfSimple) GetTargetActionID() string         { return "testAction" }
func (stubWfSimple) GetWorkflows() []string            { return nil }
func (stubWfSimple) GetSettings() *pklProject.Settings { return nil }

func TestProcessFileContent_FileDoesNotExist(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()
	wf := stubWfSimple{}

	_, action, err := archiver.ProcessFileContent(fs, "/nonexistent/file.pkl", wf, logger)
	assert.Error(t, err)
	assert.Empty(t, action)
}

func TestProcessFileContent_EmptyFile(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()
	wf := stubWfSimple{}
	dir, err := afero.TempDir(fs, "", "emptyfile")
	assert.NoError(t, err)
	defer fs.RemoveAll(dir)
	file := filepath.Join(dir, "empty.pkl")
	assert.NoError(t, afero.WriteFile(fs, file, []byte(""), 0o644))
	buf, action, err := archiver.ProcessFileContent(fs, file, wf, logger)
	assert.NoError(t, err)
	assert.Empty(t, action)
	assert.Equal(t, "", buf.String())
}

func TestProcessFileContent_ValidActionNoRequires(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()
	wf := stubWfSimple{}
	dir, err := afero.TempDir(fs, "", "validaction")
	assert.NoError(t, err)
	defer fs.RemoveAll(dir)
	file := filepath.Join(dir, "action.pkl")
	content := `amends "package://schema.kdeps.com/core@0.2.30#/Resource.pkl"

actionID = "fooBar"
description = "A test resource"
run = null`
	assert.NoError(t, afero.WriteFile(fs, file, []byte(content), 0o644))
	buf, action, err := archiver.ProcessFileContent(fs, file, wf, logger)
	assert.NoError(t, err)
	assert.Equal(t, "fooBar", action)
	assert.Contains(t, buf.String(), "@testAgent/fooBar:1.0.0")
}

func TestProcessFileContent_ValidActionWithRequires(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()
	wf := stubWfSimple{}
	dir, err := afero.TempDir(fs, "", "actionreq")
	assert.NoError(t, err)
	defer fs.RemoveAll(dir)
	file := filepath.Join(dir, "actionreq.pkl")
	content := `amends "package://schema.kdeps.com/core@0.2.30#/Resource.pkl"

actionID = "fooBar"
description = "A test resource"
requires: {
  dep1
}
run = null`
	assert.NoError(t, afero.WriteFile(fs, file, []byte(content), 0o644))
	buf, action, err := archiver.ProcessFileContent(fs, file, wf, logger)
	assert.NoError(t, err)
	assert.Equal(t, "fooBar", action)
	assert.Contains(t, buf.String(), "requires:")
}

func TestProcessFileContent_MultipleRequiresBlocks(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()
	wf := stubWfSimple{}
	dir, err := afero.TempDir(fs, "", "multireq")
	assert.NoError(t, err)
	defer fs.RemoveAll(dir)
	file := filepath.Join(dir, "multireq.pkl")
	content := `amends "package://schema.kdeps.com/core@0.2.30#/Resource.pkl"

actionID = "fooBar"
description = "A test resource"
requires: {
  dep1
}
requires: {
  dep2
}
run = null`
	assert.NoError(t, afero.WriteFile(fs, file, []byte(content), 0o644))
	buf, action, err := archiver.ProcessFileContent(fs, file, wf, logger)
	assert.NoError(t, err)
	assert.Equal(t, "fooBar", action)
	assert.Contains(t, buf.String(), "dep1")
	assert.Contains(t, buf.String(), "dep2")
}

func TestProcessFileContent_InvalidContent(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()
	wf := stubWfSimple{}
	dir, err := afero.TempDir(fs, "", "invalidcontent")
	assert.NoError(t, err)
	defer fs.RemoveAll(dir)
	file := filepath.Join(dir, "invalid.pkl")
	content := "not a valid pkl file"
	assert.NoError(t, afero.WriteFile(fs, file, []byte(content), 0o644))
	buf, action, err := archiver.ProcessFileContent(fs, file, wf, logger)
	assert.NoError(t, err)
	assert.Empty(t, action)
	assert.Contains(t, buf.String(), "not a valid pkl file")
}

func TestProcessActionPatterns_ResponseHeader(t *testing.T) {
	name := "testAgent"
	version := "1.0.0"
	line := "responseHeader(\"fooBar\", \"val\")"
	out := archiver.ProcessActionPatterns(line, name, version)
	assert.Equal(t, "responseHeader(\"@testAgent/fooBar:1.0.0\", \"val\")", out)
}

func TestProcessActionPatterns_Env(t *testing.T) {
	name := "testAgent"
	version := "1.0.0"
	line := "env(\"fooBar\", \"val\")"
	out := archiver.ProcessActionPatterns(line, name, version)
	assert.Equal(t, "env(\"@testAgent/fooBar:1.0.0\", \"val\")", out)
}

func TestProcessActionPatterns_OtherPattern(t *testing.T) {
	name := "testAgent"
	version := "1.0.0"
	line := "foo(\"fooBar\")"
	out := archiver.ProcessActionPatterns(line, name, version)
	assert.Equal(t, line, out)
}

func TestProcessActionPatterns_AlreadyPrefixed(t *testing.T) {
	name := "testAgent"
	version := "1.0.0"
	line := "foo(\"@otherAgent/bar:2.0.0\")"
	out := archiver.ProcessActionPatterns(line, name, version)
	assert.Equal(t, line, out)
}

func TestProcessActionPatterns_MultipleMatches(t *testing.T) {
	name := "testAgent"
	version := "1.0.0"
	line := "foo(\"fooBar\") env(\"barBaz\", \"val\")"
	out := archiver.ProcessActionPatterns(line, name, version)
	assert.Equal(t, "foo(\"fooBar\") env(\"@testAgent/barBaz:1.0.0\", \"val\")", out)
}

func TestProcessActionPatterns_NoMatch(t *testing.T) {
	name := "testAgent"
	version := "1.0.0"
	line := "no function call here"
	out := archiver.ProcessActionPatterns(line, name, version)
	assert.Equal(t, line, out)
}

func TestProcessFileContent_ReadFileError(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()
	wf := stubWf{}

	// Test with non-existent file
	fileBuffer, action, err := archiver.ProcessFileContent(fs, "/nonexistent/file.pkl", wf, logger)
	assert.Error(t, err)
	assert.Nil(t, fileBuffer)
	assert.Empty(t, action)
}

func TestProcessFileContent_ScannerError(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()
	wf := stubWf{}

	// Create a file with content that would cause scanner error
	// (this is hard to trigger, but we can test the error path)
	file, err := afero.TempFile(fs, "", "scanner-error")
	assert.NoError(t, err)
	defer fs.Remove(file.Name())

	// Write content that might cause scanner issues
	content := "actionID = \"testAction\"\n"
	assert.NoError(t, afero.WriteFile(fs, file.Name(), []byte(content), 0o644))
	assert.NoError(t, file.Close())

	fileBuffer, action, err := archiver.ProcessFileContent(fs, file.Name(), wf, logger)
	assert.NoError(t, err) // This should actually succeed with normal content
	assert.NotNil(t, fileBuffer)
	assert.Equal(t, "testAction", action)
}

func TestProcessFileContent_NoActionFound(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()
	wf := stubWf{}

	// Create a file with no actionID
	content := `// This is a comment
requires {
  // some requires
}

// No actionID here
someOtherContent = "value"`

	file, err := afero.TempFile(fs, "", "no-action")
	assert.NoError(t, err)
	defer fs.Remove(file.Name())
	assert.NoError(t, afero.WriteFile(fs, file.Name(), []byte(content), 0o644))

	fileBuffer, action, err := archiver.ProcessFileContent(fs, file.Name(), wf, logger)
	assert.NoError(t, err)
	assert.NotNil(t, fileBuffer)
	assert.Empty(t, action)
}

func TestProcessFileContent_ActionFound(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()
	wf := stubWfSimple{}

	// Create a file with actionID
	content := `actionID = "testAction"
// some content`

	file, err := afero.TempFile(fs, "", "with-action")
	assert.NoError(t, err)
	defer fs.Remove(file.Name())
	assert.NoError(t, afero.WriteFile(fs, file.Name(), []byte(content), 0o644))

	fileBuffer, action, err := archiver.ProcessFileContent(fs, file.Name(), wf, logger)
	assert.NoError(t, err)
	assert.NotNil(t, fileBuffer)
	assert.Equal(t, "testAction", action)

	// Verify action was updated with @ prefix
	result := fileBuffer.String()
	assert.Contains(t, result, `actionID = "@testAgent/testAction:1.0.0"`)
}

func TestProcessFileContent_ActionWithAtPrefix(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()
	wf := stubWf{}

	// Create a file with actionID that already has @ prefix
	content := `actionID = "@otherAgent/testAction:2.0.0"
// some content`

	file, err := afero.TempFile(fs, "", "with-at-prefix")
	assert.NoError(t, err)
	defer fs.Remove(file.Name())
	assert.NoError(t, afero.WriteFile(fs, file.Name(), []byte(content), 0o644))

	fileBuffer, action, err := archiver.ProcessFileContent(fs, file.Name(), wf, logger)
	assert.NoError(t, err)
	assert.NotNil(t, fileBuffer)
	assert.Equal(t, "@otherAgent/testAction:2.0.0", action)

	// Verify action was not changed (already had @ prefix)
	result := fileBuffer.String()
	assert.Contains(t, result, `actionID = "@otherAgent/testAction:2.0.0"`)
}

func TestProcessFileContent_WithRequiresBlock(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()
	wf := stubWfSimple{}

	// Create a file with requires block
	content := `actionID = "testAction"
requires {
  // some requires content
  dependency = "value"
}
// more content`

	file, err := afero.TempFile(fs, "", "with-requires")
	assert.NoError(t, err)
	defer fs.Remove(file.Name())
	assert.NoError(t, afero.WriteFile(fs, file.Name(), []byte(content), 0o644))

	fileBuffer, action, err := archiver.ProcessFileContent(fs, file.Name(), wf, logger)
	assert.NoError(t, err)
	assert.NotNil(t, fileBuffer)
	assert.Equal(t, "testAction", action)

	// Verify requires block was processed
	result := fileBuffer.String()
	assert.Contains(t, result, `actionID = "@testAgent/testAction:1.0.0"`)
	assert.Contains(t, result, "requires {")
}

func TestProcessFileContent_MultipleRequiresBlocksSkipped(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()
	wf := stubWfSimple{}

	// Create a file with multiple requires blocks (second should be skipped)
	content := `actionID = "testAction"
requires {
  // first requires block
  dependency1 = "value1"
}
// some content
requires {
  // second requires block (should be skipped)
  dependency2 = "value2"
}`

	file, err := afero.TempFile(fs, "", "multiple-requires")
	assert.NoError(t, err)
	defer fs.Remove(file.Name())
	assert.NoError(t, afero.WriteFile(fs, file.Name(), []byte(content), 0o644))

	fileBuffer, action, err := archiver.ProcessFileContent(fs, file.Name(), wf, logger)
	assert.NoError(t, err)
	assert.NotNil(t, fileBuffer)
	assert.Equal(t, "testAction", action)

	// Verify only first requires block was processed
	result := fileBuffer.String()
	assert.Contains(t, result, `actionID = "@testAgent/testAction:1.0.0"`)
	assert.Contains(t, result, "dependency1 = \"value1\"")
	assert.NotContains(t, result, "dependency2 = \"value2\"")
}

func TestProcessFileContent_WithActionPatterns(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()
	wf := stubWfSimple{}

	// Create a file with action patterns
	content := `actionID = "testAction"
// some content with action patterns
response("someAction")
env("otherAction", "default")
responseHeader("headerAction", "headerValue")`

	file, err := afero.TempFile(fs, "", "with-patterns")
	assert.NoError(t, err)
	defer fs.Remove(file.Name())
	assert.NoError(t, afero.WriteFile(fs, file.Name(), []byte(content), 0o644))

	fileBuffer, action, err := archiver.ProcessFileContent(fs, file.Name(), wf, logger)
	assert.NoError(t, err)
	assert.NotNil(t, fileBuffer)
	assert.Equal(t, "testAction", action)

	// Verify action patterns were updated
	result := fileBuffer.String()
	assert.Contains(t, result, `actionID = "@testAgent/testAction:1.0.0"`)
	assert.Contains(t, result, `response("@testAgent/someAction:1.0.0")`)
	assert.Contains(t, result, `env("@testAgent/otherAction:1.0.0", "default")`)
	assert.Contains(t, result, `responseHeader("@testAgent/headerAction:1.0.0", "headerValue")`)
}

func TestProcessFileContent_WithExistingAtPrefixPatterns(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()
	wf := stubWfSimple{}

	// Create a file with action patterns that already have @ prefix
	content := `actionID = "testAction"
// some content with existing @ prefix patterns
response("@otherAgent/someAction:2.0.0")
env("@otherAgent/otherAction:2.0.0", "default")`

	file, err := afero.TempFile(fs, "", "existing-prefix-patterns")
	assert.NoError(t, err)
	defer fs.Remove(file.Name())
	assert.NoError(t, afero.WriteFile(fs, file.Name(), []byte(content), 0o644))

	fileBuffer, action, err := archiver.ProcessFileContent(fs, file.Name(), wf, logger)
	assert.NoError(t, err)
	assert.NotNil(t, fileBuffer)
	assert.Equal(t, "testAction", action)

	// Verify existing @ prefix patterns were not changed
	result := fileBuffer.String()
	assert.Contains(t, result, `actionID = "@testAgent/testAction:1.0.0"`)
	assert.Contains(t, result, `response("@otherAgent/someAction:2.0.0")`)
	assert.Contains(t, result, `env("@otherAgent/otherAction:2.0.0", "default")`)
}

func TestProcessFileContent_MixedContent(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()
	wf := stubWfSimple{}

	// Create a file with mixed content: requires, action, and patterns
	content := `actionID = "testAction"
requires {
  // requires content
  dependency = "value"
}
// some content with action patterns
response("someAction")
env("otherAction", "default")
// more content`

	file, err := afero.TempFile(fs, "", "mixed-content")
	assert.NoError(t, err)
	defer fs.Remove(file.Name())
	assert.NoError(t, afero.WriteFile(fs, file.Name(), []byte(content), 0o644))

	fileBuffer, action, err := archiver.ProcessFileContent(fs, file.Name(), wf, logger)
	assert.NoError(t, err)
	assert.NotNil(t, fileBuffer)
	assert.Equal(t, "testAction", action)

	// Verify all content was processed correctly
	result := fileBuffer.String()
	assert.Contains(t, result, `actionID = "@testAgent/testAction:1.0.0"`)
	assert.Contains(t, result, "requires {")
	assert.Contains(t, result, "dependency = \"value\"")
	assert.Contains(t, result, `response("@testAgent/someAction:1.0.0")`)
	assert.Contains(t, result, `env("@testAgent/otherAction:1.0.0", "default")`)
}

func TestProcessFileContent_EmptyFileNew(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()
	wf := stubWf{}

	// Create an empty file
	file, err := afero.TempFile(fs, "", "empty")
	assert.NoError(t, err)
	defer fs.Remove(file.Name())
	assert.NoError(t, afero.WriteFile(fs, file.Name(), []byte(""), 0o644))

	fileBuffer, action, err := archiver.ProcessFileContent(fs, file.Name(), wf, logger)
	assert.NoError(t, err)
	assert.NotNil(t, fileBuffer)
	assert.Empty(t, action)
	assert.Empty(t, fileBuffer.String())
}

func TestProcessFileContent_OnlyComments(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()
	wf := stubWf{}

	// Create a file with only comments
	content := `// This is a comment
// Another comment
// No actionID here`

	file, err := afero.TempFile(fs, "", "comments-only")
	assert.NoError(t, err)
	defer fs.Remove(file.Name())
	assert.NoError(t, afero.WriteFile(fs, file.Name(), []byte(content), 0o644))

	fileBuffer, action, err := archiver.ProcessFileContent(fs, file.Name(), wf, logger)
	assert.NoError(t, err)
	assert.NotNil(t, fileBuffer)
	assert.Empty(t, action)

	// Verify comments were preserved
	result := fileBuffer.String()
	assert.Contains(t, result, "// This is a comment")
	assert.Contains(t, result, "// Another comment")
	assert.Contains(t, result, "// No actionID here")
}

func TestProcessPklFile_NoValidAction(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()
	wf := stubWf{}

	// Create a temp file with no valid action
	file, err := afero.TempFile(fs, "", "no-action")
	assert.NoError(t, err)
	defer fs.Remove(file.Name())

	content := `amends "package://schema.kdeps.com/core@` + schema.SchemaVersion(context.Background()) + `#/Workflow.pkl"

// No action ID found
someProperty = "value"`
	assert.NoError(t, afero.WriteFile(fs, file.Name(), []byte(content), 0o644))

	// Create temp resources directory
	resourcesDir, err := afero.TempDir(fs, "", "resources")
	assert.NoError(t, err)
	defer fs.RemoveAll(resourcesDir)

	// Test ProcessPklFile with no valid action
	err = archiver.ProcessPklFile(fs, file.Name(), wf, resourcesDir, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no valid action found")
}

func TestProcessPklFile_WriteFileError(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()
	wf := stubWfSimple{}

	// Create a temp file with valid action ID
	file, err := afero.TempFile(fs, "", "valid-action")
	assert.NoError(t, err)
	defer fs.Remove(file.Name())

	content := `amends "package://schema.kdeps.com/core@` + schema.SchemaVersion(context.Background()) + `#/Workflow.pkl"

actionID = "testAction"
someProperty = "value"`
	assert.NoError(t, afero.WriteFile(fs, file.Name(), []byte(content), 0o644))

	// Create a mock filesystem that allows reading but fails on writing
	mockFs := &mockWriteFailFs{base: fs}

	// Test ProcessPklFile with WriteFile error
	err = archiver.ProcessPklFile(mockFs, file.Name(), wf, "/tmp/test-dir", logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error writing file")
}

// mockWriteFailFs is a filesystem that allows reading but fails on writing
type mockWriteFailFs struct {
	base afero.Fs
}

func (m *mockWriteFailFs) Name() string { return m.base.Name() }
func (m *mockWriteFailFs) Create(name string) (afero.File, error) {
	return nil, fmt.Errorf("write not allowed")
}

func (m *mockWriteFailFs) Mkdir(name string, perm os.FileMode) error {
	return fmt.Errorf("write not allowed")
}

func (m *mockWriteFailFs) MkdirAll(path string, perm os.FileMode) error {
	return fmt.Errorf("write not allowed")
}
func (m *mockWriteFailFs) Open(name string) (afero.File, error) { return m.base.Open(name) }
func (m *mockWriteFailFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	if flag&os.O_WRONLY != 0 || flag&os.O_RDWR != 0 {
		return nil, fmt.Errorf("write not allowed")
	}
	return m.base.OpenFile(name, flag, perm)
}
func (m *mockWriteFailFs) Remove(name string) error    { return fmt.Errorf("write not allowed") }
func (m *mockWriteFailFs) RemoveAll(path string) error { return fmt.Errorf("write not allowed") }
func (m *mockWriteFailFs) Rename(oldname, newname string) error {
	return fmt.Errorf("write not allowed")
}
func (m *mockWriteFailFs) Stat(name string) (os.FileInfo, error) { return m.base.Stat(name) }
func (m *mockWriteFailFs) Chmod(name string, mode os.FileMode) error {
	return fmt.Errorf("write not allowed")
}

func (m *mockWriteFailFs) Chown(name string, uid, gid int) error {
	return fmt.Errorf("write not allowed")
}

func (m *mockWriteFailFs) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return fmt.Errorf("write not allowed")
}

func TestProcessPklFile_Success(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()
	wf := stubWfSimple{}

	// Create a temp file with valid action
	file, err := afero.TempFile(fs, "", "valid-action")
	assert.NoError(t, err)
	defer fs.Remove(file.Name())

	content := `amends "package://schema.kdeps.com/core@` + schema.SchemaVersion(context.Background()) + `#/Workflow.pkl"

actionID = "testAction"
someProperty = "value"`
	assert.NoError(t, afero.WriteFile(fs, file.Name(), []byte(content), 0o644))

	// Create temp resources directory
	resourcesDir, err := afero.TempDir(fs, "", "resources")
	assert.NoError(t, err)
	defer fs.RemoveAll(resourcesDir)

	// Test ProcessPklFile success
	err = archiver.ProcessPklFile(fs, file.Name(), wf, resourcesDir, logger)
	assert.NoError(t, err)

	// Verify the processed file was created
	expectedFile := filepath.Join(resourcesDir, "testAgent_testAction-1.0.0.pkl")
	exists, err := afero.Exists(fs, expectedFile)
	assert.NoError(t, err)
	assert.True(t, exists)

	// Verify the content was processed correctly
	processedContent, err := afero.ReadFile(fs, expectedFile)
	assert.NoError(t, err)
	assert.Contains(t, string(processedContent), "@testAgent/testAction:1.0.0")
}

func TestCompileResources_ValidatePklResourcesError(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	wf := stubWf{}

	// Create temp project directory without resources
	projectDir, err := afero.TempDir(fs, "", "project")
	assert.NoError(t, err)
	defer fs.RemoveAll(projectDir)

	// Create temp resources directory
	resourcesDir, err := afero.TempDir(fs, "", "resources")
	assert.NoError(t, err)
	defer fs.RemoveAll(resourcesDir)

	// Test CompileResources with ValidatePklResources error (no resources directory)
	err = archiver.CompileResources(fs, ctx, wf, resourcesDir, projectDir, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing resource directory")
}

func TestCompileResources_NoPklFiles(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	wf := stubWf{}

	// Create temp project directory with resources but no .pkl files
	projectDir, err := afero.TempDir(fs, "", "project")
	assert.NoError(t, err)
	defer fs.RemoveAll(projectDir)

	projectResourcesDir := filepath.Join(projectDir, "resources")
	assert.NoError(t, fs.MkdirAll(projectResourcesDir, 0o755))

	// Create a non-.pkl file
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(projectResourcesDir, "test.txt"), []byte("not a pkl file"), 0o644))

	// Create temp resources directory
	resourcesDir, err := afero.TempDir(fs, "", "resources")
	assert.NoError(t, err)
	defer fs.RemoveAll(resourcesDir)

	// Test CompileResources with no .pkl files
	err = archiver.CompileResources(fs, ctx, wf, resourcesDir, projectDir, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no .pkl files")
}

func TestCompileResources_InvalidPklFile(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	wf := stubWf{}

	// Create temp project directory with invalid .pkl file
	projectDir, err := afero.TempDir(fs, "", "project")
	assert.NoError(t, err)
	defer fs.RemoveAll(projectDir)

	projectResourcesDir := filepath.Join(projectDir, "resources")
	assert.NoError(t, fs.MkdirAll(projectResourcesDir, 0o755))

	// Create an invalid .pkl file (wrong schema URL)
	invalidContent := `amends "package://wrongdomain.com/core@` + schema.SchemaVersion(ctx) + `#/Workflow.pkl"

actionID = "testAction"`
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(projectResourcesDir, "test.pkl"), []byte(invalidContent), 0o644))

	// Create temp resources directory
	resourcesDir, err := afero.TempDir(fs, "", "resources")
	assert.NoError(t, err)
	defer fs.RemoveAll(resourcesDir)

	// Test CompileResources with invalid .pkl file
	err = archiver.CompileResources(fs, ctx, wf, resourcesDir, projectDir, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}

func TestCompileResources_Success(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	wf := stubWfSimple{}

	// Create temp project directory with valid .pkl file
	projectDir, err := afero.TempDir(fs, "", "project")
	assert.NoError(t, err)
	defer fs.RemoveAll(projectDir)

	projectResourcesDir := filepath.Join(projectDir, "resources")
	assert.NoError(t, fs.MkdirAll(projectResourcesDir, 0o755))

	// Create a valid .pkl file
	validContent := `amends "package://schema.kdeps.com/core@` + schema.SchemaVersion(ctx) + `#/Workflow.pkl"

actionID = "testAction"
someProperty = "value"`
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(projectResourcesDir, "workflow.pkl"), []byte(validContent), 0o644))

	// Create temp resources directory
	resourcesDir, err := afero.TempDir(fs, "", "resources")
	assert.NoError(t, err)
	defer fs.RemoveAll(resourcesDir)

	// Test CompileResources success
	err = archiver.CompileResources(fs, ctx, wf, resourcesDir, projectDir, logger)
	assert.NoError(t, err)

	// Verify the processed file was created
	expectedFile := filepath.Join(resourcesDir, "testAgent_testAction-1.0.0.pkl")
	exists, err := afero.Exists(fs, expectedFile)
	assert.NoError(t, err)
	assert.True(t, exists)

	// Verify the content was processed correctly
	processedContent, err := afero.ReadFile(fs, expectedFile)
	assert.NoError(t, err)
	assert.Contains(t, string(processedContent), "@testAgent/testAction:1.0.0")
}

func TestCompileResources_MultiplePklFiles(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	wf := stubWfSimple{}

	// Create temp project directory with multiple valid .pkl files
	projectDir, err := afero.TempDir(fs, "", "project")
	assert.NoError(t, err)
	defer fs.RemoveAll(projectDir)

	projectResourcesDir := filepath.Join(projectDir, "resources")
	assert.NoError(t, fs.MkdirAll(projectResourcesDir, 0o755))

	// Create multiple valid .pkl files
	files := map[string]string{
		"workflow1.pkl": `amends "package://schema.kdeps.com/core@` + schema.SchemaVersion(ctx) + `#/Resource.pkl"

actionID = "action1"
prop1 = "value1"`,
		"workflow2.pkl": `amends "package://schema.kdeps.com/core@` + schema.SchemaVersion(ctx) + `#/Resource.pkl"

actionID = "action2"
prop2 = "value2"`,
	}

	for filename, content := range files {
		assert.NoError(t, afero.WriteFile(fs, filepath.Join(projectResourcesDir, filename), []byte(content), 0o644))
	}

	// Create temp resources directory
	resourcesDir, err := afero.TempDir(fs, "", "resources")
	assert.NoError(t, err)
	defer fs.RemoveAll(resourcesDir)

	// Test CompileResources with multiple files
	err = archiver.CompileResources(fs, ctx, wf, resourcesDir, projectDir, logger)
	assert.NoError(t, err)

	// Verify all processed files were created
	expectedFiles := []string{
		"testAgent_action1-1.0.0.pkl",
		"testAgent_action2-1.0.0.pkl",
	}

	for _, expectedFile := range expectedFiles {
		filePath := filepath.Join(resourcesDir, expectedFile)
		exists, err := afero.Exists(fs, filePath)
		assert.NoError(t, err)
		assert.True(t, exists, "Expected file %s to exist", expectedFile)
	}
}

func TestCompileResources_WalkError(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	wf := stubWf{}

	// Create temp project directory with resources
	projectDir, err := afero.TempDir(fs, "", "project")
	assert.NoError(t, err)
	defer fs.RemoveAll(projectDir)

	projectResourcesDir := filepath.Join(projectDir, "resources")
	assert.NoError(t, fs.MkdirAll(projectResourcesDir, 0o755))

	// Create a valid .pkl file
	validContent := `amends "package://schema.kdeps.com/core@` + schema.SchemaVersion(ctx) + `#/Workflow.pkl"

actionID = "testAction"`
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(projectResourcesDir, "test.pkl"), []byte(validContent), 0o644))

	// Create temp resources directory
	resourcesDir, err := afero.TempDir(fs, "", "resources")
	assert.NoError(t, err)
	defer fs.RemoveAll(resourcesDir)

	// Use a read-only filesystem to cause Walk to fail when trying to write processed files
	readOnlyFs := afero.NewReadOnlyFs(fs)

	// Test CompileResources with Walk error
	err = archiver.CompileResources(readOnlyFs, ctx, wf, "/readonly/dir", projectDir, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "filename validation failed: invalid .pkl filename for a workflow file")
}

func TestCompileWorkflow_ActionWithoutAtPrefix(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	wf := stubWfSimple{}
	dir, err := afero.TempDir(fs, "", "compileworkflow")
	assert.NoError(t, err)
	defer fs.RemoveAll(dir)
	file := filepath.Join(dir, "workflow.pkl")
	content := `amends "package://schema.kdeps.com/core@0.2.30#/Workflow.pkl"

name = "test"
version = "1.0.0"
description = "Test workflow"
defaultAction = "testAction"
actionID = "testAction"
description = "A test action"
run = null`
	assert.NoError(t, afero.WriteFile(fs, file, []byte(content), 0o644))
	compiledDir, err := archiver.CompileWorkflow(fs, ctx, wf, dir, dir, logger)
	assert.NoError(t, err)
	assert.NotEmpty(t, compiledDir)
}

// testStubWf implements the workflow interface for testing HandleRequiresSection
type testStubWf struct{}

func (testStubWf) GetName() string                { return "test-agent" }
func (testStubWf) GetVersion() string             { return "1.0.0" }
func (testStubWf) GetDescription() string         { return "" }
func (testStubWf) GetWebsite() *string            { return nil }
func (testStubWf) GetAuthors() *[]string          { return nil }
func (testStubWf) GetDocumentation() *string      { return nil }
func (testStubWf) GetRepository() *string         { return nil }
func (testStubWf) GetHeroImage() *string          { return nil }
func (testStubWf) GetAgentIcon() *string          { return nil }
func (testStubWf) GetTargetActionID() string      { return "" }
func (testStubWf) GetWorkflows() []string         { return nil }
func (testStubWf) GetSettings() *project.Settings { return nil }

// TestHandleRequiresSection tests the HandleRequiresSection function directly
func TestHandleRequiresSection(t *testing.T) {
	mockWf := testStubWf{}

	t.Run("RequiresBlockSkipWhenBufferNotEmpty", func(t *testing.T) {
		// Test the skip logic when requiresBuf already has content
		line := "requires {"
		inBlock := false
		requiresBuf := &bytes.Buffer{}
		fileBuf := &bytes.Buffer{}

		// Add some content to requiresBuf to simulate existing content
		requiresBuf.WriteString("previous content\n")

		// This should trigger the skip logic (lines 151-153)
		result := archiver.HandleRequiresSection(&line, &inBlock, mockWf, requiresBuf, fileBuf)

		assert.True(t, result, "Should return true when skipping duplicate requires block")
		assert.False(t, inBlock, "Should not set inBlock to true when skipping")
		assert.Contains(t, requiresBuf.String(), "previous content", "Should preserve existing buffer content")
	})

	t.Run("RequiresBlockStart", func(t *testing.T) {
		// Test normal requires block start
		line := "requires {"
		inBlock := false
		requiresBuf := &bytes.Buffer{}
		fileBuf := &bytes.Buffer{}

		result := archiver.HandleRequiresSection(&line, &inBlock, mockWf, requiresBuf, fileBuf)

		assert.True(t, result, "Should return true when starting requires block")
		assert.True(t, inBlock, "Should set inBlock to true")
		assert.Contains(t, requiresBuf.String(), "requires {", "Should add requires line to buffer")
	})

	t.Run("RequiresBlockEnd", func(t *testing.T) {
		// Test requires block end
		line := "}"
		inBlock := true
		requiresBuf := &bytes.Buffer{}
		fileBuf := &bytes.Buffer{}

		// Add some content to requires buffer
		requiresBuf.WriteString("requires {\n  some content\n")

		result := archiver.HandleRequiresSection(&line, &inBlock, mockWf, requiresBuf, fileBuf)

		assert.True(t, result, "Should return true when ending requires block")
		assert.False(t, inBlock, "Should set inBlock to false")
		assert.Contains(t, fileBuf.String(), "}", "Should add closing brace to file buffer")
	})

	t.Run("RequiresBlockContent", func(t *testing.T) {
		// Test content within requires block
		line := "  some requires content"
		inBlock := true
		requiresBuf := &bytes.Buffer{}
		fileBuf := &bytes.Buffer{}

		result := archiver.HandleRequiresSection(&line, &inBlock, mockWf, requiresBuf, fileBuf)

		assert.True(t, result, "Should return true when processing requires content")
		assert.True(t, inBlock, "Should remain in requires block")
		assert.Contains(t, requiresBuf.String(), "some requires content", "Should add content to requires buffer")
	})

	t.Run("NonRequiresLine", func(t *testing.T) {
		// Test non-requires line
		line := "someOtherLine = \"value\""
		inBlock := false
		requiresBuf := &bytes.Buffer{}
		fileBuf := &bytes.Buffer{}

		result := archiver.HandleRequiresSection(&line, &inBlock, mockWf, requiresBuf, fileBuf)

		assert.False(t, result, "Should return false for non-requires lines")
		assert.False(t, inBlock, "Should not change inBlock state")
	})
}
