package resolver

import (
	"bufio"
	"fmt"
	"io"
	"kdeps/pkg/schema"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	pklExec "github.com/kdeps/schema/gen/exec"
	pklHttp "github.com/kdeps/schema/gen/http"
	pklLLM "github.com/kdeps/schema/gen/llm"
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
	}

	var importFiles, localVariables string
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

			var schemaFile string
			switch key {
			case "exec":
				schemaFile = "Exec.pkl"
			case "client":
				schemaFile = "Http.pkl"
			case "llm":
				schemaFile = "LLM.pkl"
			}

			// Write header using packageUrl and schemaFile
			if _, err := writer.WriteString(fmt.Sprintf("extends \"%s%s\"\n\n", packageUrl, schemaFile)); err != nil {
				return fmt.Errorf("failed to write header for %s: %w", key, err)
			}

			// Write the resource block
			if _, err := writer.WriteString("resources {\n}\n"); err != nil {
				return fmt.Errorf("failed to write resource block for %s: %w", key, err)
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
	llmChat := &pklLLM.ResourceChat{}
	execCmd := &pklExec.ResourceExec{}
	httpClient := &pklHttp.ResourceHTTPClient{
		Method: "GET",
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

	return nil
}
