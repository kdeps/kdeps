package resolver

import (
	"bufio"
	"fmt"
	"io"
	"kdeps/pkg/data"
	"kdeps/pkg/schema"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	pklData "github.com/kdeps/schema/gen/data"
	pklExec "github.com/kdeps/schema/gen/exec"
	pklHttp "github.com/kdeps/schema/gen/http"
	pklLLM "github.com/kdeps/schema/gen/llm"
	pklPython "github.com/kdeps/schema/gen/python"
	"github.com/kdeps/schema/gen/utils"
	"github.com/spf13/afero"
)

func (dr *DependencyResolver) PrependDynamicImports(pklFile string) error {
	content, err := afero.ReadFile(dr.Fs, pklFile)
	if err != nil {
		return err
	}

	// Define a regular expression to match "{{value}}"
	re := regexp.MustCompile(`\@\((.*)\)`)

	importCheck := map[string]string{
		dr.RequestPklFile: "request",
		filepath.Join(dr.ActionDir, "/llm/"+dr.RequestId+"__llm_output.pkl"):       "llm",
		filepath.Join(dr.ActionDir, "/client/"+dr.RequestId+"__client_output.pkl"): "client",
		filepath.Join(dr.ActionDir, "/exec/"+dr.RequestId+"__exec_output.pkl"):     "exec",
		filepath.Join(dr.ActionDir, "/python/"+dr.RequestId+"__python_output.pkl"): "python",
		filepath.Join(dr.ActionDir, "/data/"+dr.RequestId+"__data_output.pkl"):     "dataFiles",
	}

	// Define core imports and declarations
	coreImports := []string{
		`import "pkl:json"`,
		`import "pkl:test"`,
		`import "pkl:math"`,
		`import "pkl:platform"`,
		`import "pkl:semver"`,
		`import "pkl:shell"`,
		`import "pkl:xml"`,
		`import "pkl:yaml"`,
	}
	coreDeclarations := []string{
		`
local function jsonParser(data: String) =
  if (test.catchOrNull(() -> (new json.Parser { useMapping = true }).parse(data)) == null)
    (new json.Parser { useMapping = false }).parse(data)
  else
    null
`,
		`
local function jsonParserMapping(data: String) =
  if (test.catchOrNull(() -> (new json.Parser { useMapping = true }).parse(data)) == null)
    (new json.Parser { useMapping = true }).parse(data)
  else
    null
`,
		`
local function jsonRenderDocument(data: String) =
  if (test.catchOrNull(() -> (new JsonRenderer {}).renderDocument(data)) == null)
    (new JsonRenderer {}).renderDocument(data)
  else
    ""
`,
		`
local function jsonRenderValue(data: String) =
  if (test.catchOrNull(() -> (new JsonRenderer {}).renderValue(data)) == null)
    (new JsonRenderer {}).renderValue(data)
  else
    ""
`,
		`
local function yamlRenderDocument(data: String) =
  if (test.catchOrNull(() -> (new YamlRenderer {}).renderDocument(data)) == null)
    (new YamlRenderer {}).renderDocument(data)
  else
    ""
`,
		`
local function yamlRenderValue(data: String) =
  if (test.catchOrNull(() -> (new YamlRenderer {}).renderValue(data)) == null)
    (new YamlRenderer {}).renderValue(data)
  else
    ""
`,
		`
local function xmlRenderDocument(data: String) =
  if (test.catchOrNull(() -> (new PListRenderer {}).renderDocument(data)) == null)
    (new PListRenderer {}).renderDocument(data)
  else
    ""
`,
		`
local function xmlRenderValue(data: String) =
  if (test.catchOrNull(() -> (new PListRenderer {}).renderValue(data)) == null)
    (new PListRenderer {}).renderValue(data)
  else
    ""
`,
	}

	var importFiles, localVariables string

	// Add core imports and declarations
	importFiles += strings.Join(coreImports, "\n") + "\n"
	localVariables += strings.Join(coreDeclarations, "\n") + "\n"

	for file, variable := range importCheck {
		if exists, _ := afero.Exists(dr.Fs, file); exists {
			// Check if the import line already exists
			importLine := fmt.Sprintf(`import "%s" as %s_output`, file, variable)
			if !strings.Contains(string(content), importLine) {
				importFiles += importLine + "\n"
			}
			if variable != "" {
				importName := strings.TrimSuffix(filepath.Base(variable), ".pkl")
				localVarLine := fmt.Sprintf("local %s = %s_output\n", variable, importName)
				// Check if the local variable line already exists
				if !strings.Contains(string(content), localVarLine) {
					localVariables += localVarLine
				}
			}
		}
	}

	// Only proceed if there are new imports or local variables to add
	if importFiles != "" || localVariables != "" {
		importFiles += "\n" + localVariables + "\n"

		// Convert the content to a string and find the "amends" line
		contentStr := string(content)
		amendsIndex := strings.Index(contentStr, "amends")

		// If "amends" line is found, insert the dynamic imports after it
		if amendsIndex != -1 {
			amendsLineEnd := strings.Index(contentStr[amendsIndex:], "\n") + amendsIndex + 1
			newContent := contentStr[:amendsLineEnd] + importFiles + contentStr[amendsLineEnd:]
			newContent = re.ReplaceAllString(newContent, `\($1)`)
			err = afero.WriteFile(dr.Fs, pklFile, []byte(newContent), 0644)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (dr *DependencyResolver) PrepareImportFiles() error {
	files := map[string]string{
		"llm":    filepath.Join(dr.ActionDir, "/llm/"+dr.RequestId+"__llm_output.pkl"),
		"client": filepath.Join(dr.ActionDir, "/client/"+dr.RequestId+"__client_output.pkl"),
		"exec":   filepath.Join(dr.ActionDir, "/exec/"+dr.RequestId+"__exec_output.pkl"),
		"python": filepath.Join(dr.ActionDir, "/python/"+dr.RequestId+"__python_output.pkl"),
		"data":   filepath.Join(dr.ActionDir, "/data/"+dr.RequestId+"__data_output.pkl"),
	}

	for key, file := range files {
		dir := filepath.Dir(file)
		if err := dr.Fs.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", key, err)
		}

		// Check if the file exists, if not, create it
		exists, err := afero.Exists(dr.Fs, file)
		if err != nil {
			return fmt.Errorf("failed to check if %s file exists: %w", key, err)
		}

		if !exists {
			// Create the file if it doesn't exist
			f, err := dr.Fs.Create(file)
			if err != nil {
				return fmt.Errorf("failed to create %s file: %w", key, err)
			}
			defer f.Close()

			// Use packageUrl in the header writing
			packageUrl := fmt.Sprintf("package://schema.kdeps.com/core@%s#/", schema.SchemaVersion)
			writer := bufio.NewWriter(f)

			var schemaFile, blockType string
			switch key {
			case "exec":
				schemaFile = "Exec.pkl"
				blockType = "resources"
			case "python":
				schemaFile = "Python.pkl"
				blockType = "resources"
			case "client":
				schemaFile = "Http.pkl"
				blockType = "resources"
			case "llm":
				schemaFile = "LLM.pkl"
				blockType = "resources"
			case "data":
				schemaFile = "Data.pkl"
				blockType = "files" // Special case for "data"
			}

			// Write header using packageUrl and schemaFile
			if _, err := writer.WriteString(fmt.Sprintf("extends \"%s%s\"\n\n", packageUrl, schemaFile)); err != nil {
				return fmt.Errorf("failed to write header for %s: %w", key, err)
			}

			// Write the block (resources or files)
			if _, err := writer.WriteString(fmt.Sprintf("%s {\n}\n", blockType)); err != nil {
				return fmt.Errorf("failed to write block for %s: %w", key, err)
			}

			// Flush the writer
			if err := writer.Flush(); err != nil {
				return fmt.Errorf("failed to flush output for %s: %w", key, err)
			}
		}
	}

	return nil
}

func (dr *DependencyResolver) PrepareWorkflowDir() error {
	src := dr.ProjectDir
	dest := dr.AgentDir
	fs := dr.Fs

	// Check if the destination exists and remove it if it does
	exists, err := afero.Exists(fs, dest)
	if err != nil {
		return fmt.Errorf("failed to check if destination exists: %w", err)
	}
	if exists {
		if err := fs.RemoveAll(dest); err != nil {
			return fmt.Errorf("failed to remove existing destination: %w", err)
		}
	}

	// Walk through the source directory
	err = afero.Walk(fs, src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Determine the relative path and destination path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dest, relPath)

		if info.IsDir() {
			// Create directories in the destination
			if err := fs.MkdirAll(targetPath, info.Mode()); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		} else {
			// Copy file contents to the destination
			in, err := fs.Open(path)
			if err != nil {
				return err
			}
			defer in.Close()

			out, err := fs.Create(targetPath)
			if err != nil {
				return err
			}
			defer out.Close()

			// Copy file contents
			if _, err := io.Copy(out, in); err != nil {
				return err
			}

			// Set file permissions to match the source file
			if err := fs.Chmod(targetPath, info.Mode()); err != nil {
				return err
			}
		}
		return nil
	})

	return err
}

func (dr *DependencyResolver) AddPlaceholderImports(filePath string) error {
	// Open the file using afero file system (dr.Fs)
	file, err := dr.Fs.Open(filePath)
	if err != nil {
		return fmt.Errorf("could not open file: %v", err)
	}
	defer file.Close()

	// Use a regular expression to find the id in the file
	re := regexp.MustCompile(`id\s*=\s*"([^"]+)"`)
	var actionId string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Check if the line contains the id
		matches := re.FindStringSubmatch(line)
		if len(matches) > 1 {
			actionId = matches[1]
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %v", err)
	}

	if actionId == "" {
		return fmt.Errorf("action id not found in the file")
	}

	// Create placeholder entries using the parsed actionId
	type DataImpl struct {
		*utils.UtilsImpl

		// Files in the data folder mapped with the agent name and version
		Files *map[string]map[string]string `pkl:"files"`
	}

	dataFileList, err := data.PopulateDataFileRegistry(dr.Fs, dr.DataDir)
	dataFiles := &pklData.DataImpl{
		Files: dataFileList,
	}
	llmChat := &pklLLM.ResourceChat{}
	execCmd := &pklExec.ResourceExec{}
	pythonCmd := &pklPython.ResourcePython{}
	httpClient := &pklHttp.ResourceHTTPClient{
		Method: "GET",
	}

	if err := dr.AppendDataEntry(actionId, dataFiles); err != nil {
		return err
	}

	if err := dr.AppendChatEntry(actionId, llmChat); err != nil {
		return err
	}

	if err := dr.AppendExecEntry(actionId, execCmd); err != nil {
		return err
	}

	if err := dr.AppendHttpEntry(actionId, httpClient); err != nil {
		return err
	}

	if err := dr.AppendPythonEntry(actionId, pythonCmd); err != nil {
		return err
	}

	return nil
}
