package resolver

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"kdeps/pkg/environment"
	"kdeps/pkg/evaluator"
	"kdeps/pkg/logging"
	"kdeps/pkg/resource"
	"kdeps/pkg/schema"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/alexellis/go-execute/v2"
	"github.com/charmbracelet/log"
	"github.com/kdeps/kartographer/graph"
	apiserverresponse "github.com/kdeps/schema/gen/api_server_response"
	pklExec "github.com/kdeps/schema/gen/exec"
	pklLLM "github.com/kdeps/schema/gen/llm"
	pklRes "github.com/kdeps/schema/gen/resource"
	pklWf "github.com/kdeps/schema/gen/workflow"
	"github.com/spf13/afero"
	"github.com/tmc/langchaingo/llms/ollama"
)

type DependencyResolver struct {
	Fs                   afero.Fs
	Logger               *log.Logger
	Resources            []ResourceNodeEntry
	ResourceDependencies map[string][]string
	DependencyGraph      []string
	VisitedPaths         map[string]bool
	Context              *context.Context
	Graph                *graph.DependencyGraph
	Workflow             *pklWf.Workflow
	RequestPklFile       string
	ResponsePklFile      string
	ResponseTargetFile   string
	ResponseFlag         string
	ResponseType         string
	ProjectDir           string
	AgentDir             string
	ActionDir            string
	ApiServerMode        bool
}

type ResourceNodeEntry struct {
	Id   string `pkl:"id"`
	File string `pkl:"file"`
}

func NewGraphResolver(fs afero.Fs, logger *log.Logger, ctx context.Context, env *environment.Environment, agentDir string) (*DependencyResolver, error) {
	var actionDir, requestPklFile, responsePklFile, projectDir, pklWfFile, pklWfParentFile string

	if env.DockerMode == "1" {
		agentDir = filepath.Join(agentDir, "/workflow/")
		pklWfFile = filepath.Join(agentDir, "workflow.pkl")
		pklWfParentFile = filepath.Join(agentDir, "../workflow.pkl")

		// Check if "workflow.pkl" exists using afero.Exists
		exists, err := afero.Exists(fs, pklWfFile)
		if err != nil {
			return nil, fmt.Errorf("error checking %s: %v", pklWfFile, err)
		}

		logging.Info(pklWfFile)
		if !exists {
			// If "workflow.pkl" doesn't exist, check for "../workflow.pkl"
			existsParent, errParent := afero.Exists(fs, pklWfParentFile)
			if errParent != nil {
				return nil, fmt.Errorf("error checking %s: %v", pklWfParentFile, errParent)
			}

			if !existsParent {
				return nil, fmt.Errorf("neither %s nor %s exist", pklWfFile, pklWfParentFile)
			}

			// "../workflow.pkl" exists, update pklWfFile to point to it
			pklWfFile = pklWfParentFile
			agentDir = filepath.Join(agentDir, "../")
			projectDir = filepath.Join(agentDir, "/project/")
			actionDir = filepath.Join(agentDir, "/actions")
		} else {
			projectDir = filepath.Join(agentDir, "../project/")
			actionDir = filepath.Join(agentDir, "../actions")
		}

		requestPklFile = filepath.Join(actionDir, "/api/request.pkl")
		responsePklFile = filepath.Join(actionDir, "/api/response.pkl")

	}

	dependencyResolver := &DependencyResolver{
		Fs:                   fs,
		ResourceDependencies: make(map[string][]string),
		Logger:               logger,
		VisitedPaths:         make(map[string]bool),
		Context:              &ctx,
		AgentDir:             agentDir,
		ActionDir:            actionDir,
		RequestPklFile:       requestPklFile,
		ResponsePklFile:      responsePklFile,
		ProjectDir:           projectDir,
	}

	workflowConfiguration, err := pklWf.LoadFromPath(ctx, pklWfFile)
	if err != nil {
		return nil, err
	}
	dependencyResolver.Workflow = workflowConfiguration
	if workflowConfiguration.Settings != nil {
		dependencyResolver.ApiServerMode = workflowConfiguration.Settings.ApiServerMode
	}

	dependencyResolver.Graph = graph.NewDependencyGraph(fs, logger, dependencyResolver.ResourceDependencies)
	if dependencyResolver.Graph == nil {
		return nil, fmt.Errorf("failed to initialize dependency graph")
	}

	return dependencyResolver, nil
}

func (dr *DependencyResolver) LoadResourceEntries() error {
	workflowDir := filepath.Join(dr.AgentDir, "resources")
	var pklFiles []string
	if err := afero.Walk(dr.Fs, workflowDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Errorf("Error walking through files: %s - %s", workflowDir, err)
			return err
		}

		// Check if the file has a .pkl extension
		if !info.IsDir() && filepath.Ext(path) == ".pkl" {
			if err := dr.PrependDynamicImports(path); err != nil {
				fmt.Errorf("Failed to prepend dynamic imports "+path, err)
			}

			if err := dr.AddPlaceholderImports(path); err != nil {
				fmt.Errorf("Unable to create placeholder imports for .pkl file "+path, err)
			}

			pklFiles = append(pklFiles, path)
		}
		return nil
	}); err != nil {
		return err
	}

	for _, file := range pklFiles {
		// Load the resource file
		pklRes, err := resource.LoadResource(*dr.Context, file)
		if err != nil {
			fmt.Errorf("Error loading .pkl file "+file, err)
		}

		dr.Resources = append(dr.Resources, ResourceNodeEntry{
			Id:   pklRes.Id,
			File: file,
		})

		if pklRes.Requires != nil {
			dr.ResourceDependencies[pklRes.Id] = *pklRes.Requires
		} else {
			dr.ResourceDependencies[pklRes.Id] = nil
		}
	}

	return nil
}

func (dr *DependencyResolver) PrependDynamicImports(pklFile string) error {
	content, err := afero.ReadFile(dr.Fs, pklFile)
	if err != nil {
		return err
	}

	// Define a regular expression to match "{{value}}"
	re := regexp.MustCompile(`\@\((.*)\)`)

	importCheck := map[string]string{
		dr.RequestPklFile: "",
		filepath.Join(dr.ActionDir, "/llm/llm_output.pkl"):       "llm",
		filepath.Join(dr.ActionDir, "/client/client_output.pkl"): "client",
		filepath.Join(dr.ActionDir, "/exec/exec_output.pkl"):     "exec",
	}

	var importFiles, localVariables string
	for file, variable := range importCheck {
		if exists, _ := afero.Exists(dr.Fs, file); exists {
			// Check if the import line already exists
			importLine := fmt.Sprintf(`import "%s"`, file)
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

func WaitForFile(fs afero.Fs, filepath string) error {
	logging.Info("Waiting for file: ", filepath)

	// Create a ticker that checks for the file periodically
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check if the file exists
			exists, err := afero.Exists(fs, filepath)
			if err != nil {
				return fmt.Errorf("error checking file %s: %w", filepath, err)
			}
			if exists {
				logging.Info("File found: ", filepath)
				return nil
			}
		}
	}

	return nil
}

func (dr *DependencyResolver) PrepareImportFiles() error {
	files := map[string]string{
		"llm":    filepath.Join(dr.ActionDir, "/llm/llm_output.pkl"),
		"client": filepath.Join(dr.ActionDir, "/client/client_output.pkl"),
		"exec":   filepath.Join(dr.ActionDir, "/exec/exec_output.pkl"),
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
			if _, err := writer.WriteString(fmt.Sprintf("amends \"%s%s\"\n\n", packageUrl, schemaFile)); err != nil {
				return fmt.Errorf("failed to write header for %s: %w", key, err)
			}

			// Write the resource block
			if _, err := writer.WriteString("resource {\n}\n"); err != nil {
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

func NewAPIServerResponse(success bool, data []any, errorCode int, errorMessage string) apiserverresponse.APIServerResponse {
	responseBlock := &apiserverresponse.APIServerResponseBlock{Data: data}
	var errorsBlock *apiserverresponse.APIServerErrorsBlock

	// If there is an error, create the errors block
	if errorCode != 0 || errorMessage != "" {
		errorsBlock = &apiserverresponse.APIServerErrorsBlock{
			Code:    errorCode,
			Message: errorMessage,
		}
	}

	return apiserverresponse.APIServerResponse{
		Success:  success,
		Response: responseBlock,
		Errors:   errorsBlock,
	}
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

	if err := dr.AppendChatEntry(actionId, llmChat); err != nil {
		return err
	}

	if err := dr.AppendExecEntry(actionId, execCmd); err != nil {
		return err
	}

	return nil
}

func (dr *DependencyResolver) HandleRunAction() error {
	defer func() {
		if r := recover(); r != nil {
			logging.Error("Recovered from panic:", r)
			dr.HandleAPIErrorResponse(500, "Server panic occurred")
		}
	}()

	visited := make(map[string]bool)
	actionId := dr.Workflow.Action

	logging.Info("Processing resources...")
	if err := dr.LoadResourceEntries(); err != nil {
		return dr.HandleAPIErrorResponse(500, err.Error())
	}

	stack := dr.Graph.BuildDependencyStack(actionId, visited)
	for _, resNode := range stack {
		for _, res := range dr.Resources {
			if res.Id == resNode {
				rsc, err := pklRes.LoadFromPath(*dr.Context, res.File)
				if err != nil {
					return dr.HandleAPIErrorResponse(500, err.Error())
				}

				runBlock := rsc.Run
				if runBlock != nil {
					// Check Skip Condition
					if runBlock.SkipCondition != nil {
						if ShouldSkip(runBlock.SkipCondition) {
							logging.Info("Skip condition met, skipping:", res.Id)
							continue
						}
					}

					// Handle Preflight Check
					if runBlock.PreflightCheck != nil && runBlock.PreflightCheck.Validations != nil {
						if !AllConditionsMet(runBlock.PreflightCheck.Validations) {
							logging.Error("Preflight check not met, failing:", res.Id)
							if runBlock.PreflightCheck.Error != nil {
								return dr.HandleAPIErrorResponse(
									runBlock.PreflightCheck.Error.Code,
									fmt.Sprintf("%s: %s", runBlock.PreflightCheck.Error.Message, res.Id))
							}
							logging.Error("Preflight check not met, failing:", res.Id)
							return dr.HandleAPIErrorResponse(500, "Preflight check failed for resource: "+res.Id)
						}
					}

					if runBlock.Exec != nil && runBlock.Exec.Command != "" {
						timestamp, err := dr.GetCurrentTimestamp(res.Id, "exec")
						if err != nil {
							logging.Error("Exec error:", res.Id)
							return dr.HandleAPIErrorResponse(500, fmt.Sprintf("Exec failed for resource: %s - %s", res.Id, err))
						}

						if err := dr.HandleExec(res.Id, runBlock.Exec); err != nil {
							logging.Error("Exec error:", res.Id)
							return dr.HandleAPIErrorResponse(500, fmt.Sprintf("Exec failed for resource: %s - %s", res.Id, err))
						}

						if err := dr.WaitForTimestampChange(res.Id, timestamp, 60*time.Second, "exec"); err != nil {
							logging.Error("Exec error:", res.Id)
							return dr.HandleAPIErrorResponse(500, fmt.Sprintf("Exec timeout awaiting for output: %s - %s", res.Id, err))
						}

					}

					if runBlock.Chat != nil && runBlock.Chat.Model != "" && runBlock.Chat.Prompt != "" {
						timestamp, err := dr.GetCurrentTimestamp(res.Id, "llm")
						if err != nil {
							logging.Error("LLM chat error:", res.Id)
							return dr.HandleAPIErrorResponse(500, fmt.Sprintf("LLM chat failed for resource: %s - %s", res.Id, err))
						}

						if err := dr.HandleLLMChat(res.Id, runBlock.Chat); err != nil {
							logging.Error("LLM chat error:", res.Id)
							return dr.HandleAPIErrorResponse(500, fmt.Sprintf("LLM chat failed for resource: %s - %s", res.Id, err))
						}

						if err := dr.WaitForTimestampChange(res.Id, timestamp, 60*time.Second, "llm"); err != nil {
							logging.Error("LLM chat error:", res.Id)
							return dr.HandleAPIErrorResponse(500, fmt.Sprintf("LLM chat timeout awaiting for response: %s - %s", res.Id, err))
						}
					}

					// Handle Postflight Check
					if runBlock.PostflightCheck != nil && runBlock.PostflightCheck.Validations != nil {
						if !AllConditionsMet(runBlock.PostflightCheck.Validations) {
							if runBlock.PostflightCheck.Error != nil {
								return dr.HandleAPIErrorResponse(
									runBlock.PostflightCheck.Error.Code,
									fmt.Sprintf("%s: %s", runBlock.PostflightCheck.Error.Message, res.Id))
							}

							logging.Error("Postflight check not met, failing:", res.Id)
							return dr.HandleAPIErrorResponse(500, "Postflight check failed for resource: "+res.Id)
						}
					}

					// API Response
					if dr.ApiServerMode && runBlock.ApiResponse != nil {
						if err := dr.CreateResponsePklFile(runBlock.ApiResponse); err != nil {
							return dr.HandleAPIErrorResponse(500, err.Error())
						}
					}
				}
			}
		}
	}

	logging.Info("All resources finished processing")
	return nil
}

// Helper function to Handle API error responses
func (dr *DependencyResolver) HandleAPIErrorResponse(code int, message string) error {
	if dr.ApiServerMode {
		errorResponse := NewAPIServerResponse(false, nil, code, message)
		if err := dr.CreateResponsePklFile(&errorResponse); err != nil {
			logging.Error("Failed to create error response file:", err)
			return err
		}
	}
	return nil
}

func (dr *DependencyResolver) AppendExecEntry(resourceId string, newExec *pklExec.ResourceExec) error {
	// Define the path to the PKL file
	pklPath := filepath.Join(dr.ActionDir, "exec/exec_output.pkl")

	// Get the current timestamp
	newTimestamp := uint32(time.Now().UnixNano())

	// Load existing PKL data
	pklRes, err := pklExec.LoadFromPath(*dr.Context, pklPath)
	if err != nil {
		return fmt.Errorf("failed to load PKL file: %w", err)
	}

	// Ensure pklRes.Resource is of type *map[string]*llm.ResourceChat
	existingResources := *pklRes.Resource // Dereference the pointer to get the map

	// Create or update the ResourceChat entry
	existingResources[resourceId] = &pklExec.ResourceExec{
		Env:       newExec.Env, // Add Env field
		Command:   newExec.Command,
		Stderr:    newExec.Stderr,
		Stdout:    newExec.Stdout,
		Timestamp: &newTimestamp,
	}

	// Build the new content for the PKL file in the specified format
	var pklContent strings.Builder
	pklContent.WriteString("amends \"package://schema.kdeps.com/core@0.0.50#/Exec.pkl\"\n\n")
	pklContent.WriteString("resource {\n")

	for id, resource := range existingResources {
		pklContent.WriteString(fmt.Sprintf("  [\"%s\"] {\n", id))
		pklContent.WriteString(fmt.Sprintf("    command = \"\"\"\n%s\n\"\"\"\n", resource.Command))
		pklContent.WriteString(fmt.Sprintf("    timestamp = %d\n", *resource.Timestamp))

		// Write environment variables (if Env is not nil)
		if resource.Env != nil {
			pklContent.WriteString("    env {\n")
			for key, value := range *resource.Env {
				pklContent.WriteString(fmt.Sprintf("      [\"%s\"] = \"%s\"\n", key, value))
			}
			pklContent.WriteString("    }\n")
		} else {
			pklContent.WriteString("    env {}\n") // Handle nil case for Env
		}

		// Dereference to pass Stderr and Stdout correctly
		if resource.Stderr != nil {
			pklContent.WriteString(fmt.Sprintf("    stderr = \"\"\"\n%s\n\"\"\"\n", *resource.Stderr))
		} else {
			pklContent.WriteString("    stderr = \"\"\n") // Handle nil case
		}
		if resource.Stdout != nil {
			pklContent.WriteString(fmt.Sprintf("    stdout = \"\"\"\n%s\n\"\"\"\n", *resource.Stdout))
		} else {
			pklContent.WriteString("    stdout = \"\"\n") // Handle nil case
		}

		pklContent.WriteString("  }\n")
	}

	pklContent.WriteString("}\n")

	// Write the new PKL content to the file using afero
	err = afero.WriteFile(dr.Fs, pklPath, []byte(pklContent.String()), 0644)
	if err != nil {
		return fmt.Errorf("failed to write to PKL file: %w", err)
	}

	return nil
}

func (dr *DependencyResolver) AppendChatEntry(resourceId string, newChat *pklLLM.ResourceChat) error {
	// Define the path to the PKL file
	pklPath := filepath.Join(dr.ActionDir, "llm/llm_output.pkl")

	// Get the current timestamp
	newTimestamp := uint32(time.Now().UnixNano())

	// Load existing PKL data
	pklRes, err := pklLLM.LoadFromPath(*dr.Context, pklPath)
	if err != nil {
		return fmt.Errorf("failed to load PKL file: %w", err)
	}

	// Ensure pklRes.Resource is of type *map[string]*llm.ResourceChat
	existingResources := *pklRes.Resource // Dereference the pointer to get the map

	// Create or update the ResourceChat entry
	existingResources[resourceId] = &pklLLM.ResourceChat{
		Model:     newChat.Model,
		Prompt:    newChat.Prompt,
		Response:  newChat.Response,
		Timestamp: &newTimestamp,
	}

	// Build the new content for the PKL file in the specified format
	var pklContent strings.Builder
	pklContent.WriteString("amends \"package://schema.kdeps.com/core@0.0.50#/LLM.pkl\"\n\n")
	pklContent.WriteString("resource {\n")

	for id, resource := range existingResources {
		pklContent.WriteString(fmt.Sprintf("  [\"%s\"] {\n", id))
		pklContent.WriteString(fmt.Sprintf("    model = \"%s\"\n", resource.Model))
		pklContent.WriteString(fmt.Sprintf("    prompt = \"\"\"\n%s\n\"\"\"\n", resource.Prompt))
		pklContent.WriteString(fmt.Sprintf("    timestamp = %d\n", *resource.Timestamp))
		// Dereference response to pass it correctly
		if resource.Response != nil {
			pklContent.WriteString(fmt.Sprintf("    response = \"\"\"\n%s\n\"\"\"\n", *resource.Response))
		} else {
			pklContent.WriteString("    response = \"\"\n") // Handle nil case
		}

		pklContent.WriteString("  }\n")
	}

	pklContent.WriteString("}\n")

	// Write the new PKL content to the file using afero
	err = afero.WriteFile(dr.Fs, pklPath, []byte(pklContent.String()), 0644)
	if err != nil {
		return fmt.Errorf("failed to write to PKL file: %w", err)
	}

	return nil
}

func (dr *DependencyResolver) GetCurrentTimestamp(resourceId string, resourceType string) (uint32, error) {
	// Define file paths based on resource types
	files := map[string]string{
		"llm":    filepath.Join(dr.ActionDir, "llm/llm_output.pkl"),
		"client": filepath.Join(dr.ActionDir, "client/client_output.pkl"),
		"exec":   filepath.Join(dr.ActionDir, "exec/exec_output.pkl"),
	}

	// Check if the resource type is valid and get the corresponding path
	pklPath, exists := files[resourceType]
	if !exists {
		return 0, fmt.Errorf("invalid resourceType %s provided", resourceType)
	}

	// Load the appropriate PKL file based on the resourceType
	switch resourceType {
	case "exec":
		pklRes, err := pklExec.LoadFromPath(*dr.Context, pklPath)
		if err != nil {
			return 0, fmt.Errorf("failed to load exec PKL file: %w", err)
		}
		// Dereference the resource map for exec and handle ResourceExec
		existingResources := *pklRes.Resource
		if resource, exists := existingResources[resourceId]; exists {
			if resource.Timestamp == nil {
				return 0, fmt.Errorf("timestamp for resource ID %s is nil", resourceId)
			}
			return *resource.Timestamp, nil
		}
	case "llm":
		pklRes, err := pklLLM.LoadFromPath(*dr.Context, pklPath)
		if err != nil {
			return 0, fmt.Errorf("failed to load llm PKL file: %w", err)
		}
		// Dereference the resource map for llm and handle ResourceChat
		existingResources := *pklRes.Resource
		if resource, exists := existingResources[resourceId]; exists {
			if resource.Timestamp == nil {
				return 0, fmt.Errorf("timestamp for resource ID %s is nil", resourceId)
			}
			return *resource.Timestamp, nil
		}
	default:
		return 0, fmt.Errorf("unsupported resourceType %s provided", resourceType)
	}

	return 0, fmt.Errorf("resource ID %s does not exist in the file", resourceId)
}

// WaitForTimestampChange waits until the timestamp for the specified resource ID changes from the provided previous timestamp.
func (dr *DependencyResolver) WaitForTimestampChange(resourceId string, previousTimestamp uint32, timeout time.Duration,
	resourceType string) error {

	// Map containing the paths for different resource types
	files := map[string]string{
		"llm":    filepath.Join(dr.ActionDir, "llm/llm_output.pkl"),
		"client": filepath.Join(dr.ActionDir, "client/client_output.pkl"),
		"exec":   filepath.Join(dr.ActionDir, "exec/exec_output.pkl"),
	}

	// Retrieve the correct path based on resourceType
	pklPath, exists := files[resourceType]
	if !exists {
		return fmt.Errorf("invalid resourceType %s provided", resourceType)
	}

	// Start the waiting loop
	startTime := time.Now()
	for {
		// Check if the timeout has been exceeded
		if time.Since(startTime) > timeout {
			return fmt.Errorf("timeout exceeded while waiting for timestamp change for resource ID %s", resourceId)
		}

		// Reload the current state of the PKL file and handle based on the resourceType
		switch resourceType {
		case "exec":
			// Load exec type PKL file
			updatedRes, err := pklExec.LoadFromPath(*dr.Context, pklPath)
			if err != nil {
				return fmt.Errorf("failed to reload exec PKL file: %w", err)
			}

			// Get the resource map and check for timestamp changes
			updatedResources := *updatedRes.Resource // Dereference to get the map
			if updatedResource, exists := updatedResources[resourceId]; exists {
				// Compare the current timestamp with the previous timestamp
				if updatedResource.Timestamp != nil && *updatedResource.Timestamp != previousTimestamp {
					// Timestamp has changed
					return nil
				}
			} else {
				return fmt.Errorf("resource ID %s does not exist in the exec file", resourceId)
			}

		case "llm":
			// Load llm type PKL file
			updatedRes, err := pklLLM.LoadFromPath(*dr.Context, pklPath)
			if err != nil {
				return fmt.Errorf("failed to reload llm PKL file: %w", err)
			}

			// Get the resource map and check for timestamp changes
			updatedResources := *updatedRes.Resource // Dereference to get the map
			if updatedResource, exists := updatedResources[resourceId]; exists {
				// Compare the current timestamp with the previous timestamp
				if updatedResource.Timestamp != nil && *updatedResource.Timestamp != previousTimestamp {
					// Timestamp has changed
					return nil
				}
			} else {
				return fmt.Errorf("resource ID %s does not exist in the llm file", resourceId)
			}

		default:
			return fmt.Errorf("unsupported resourceType %s provided", resourceType)
		}

		// Sleep for a short duration before checking again
		time.Sleep(100 * time.Millisecond)
	}
}

func (dr *DependencyResolver) HandleExec(actionId string, execBlock *pklExec.ResourceExec) error {
	go func() error {
		err := dr.processExecBlock(actionId, execBlock)
		if err != nil {
			return err
		}

		return nil
	}()

	return nil
}

func (dr *DependencyResolver) processExecBlock(actionId string, execBlock *pklExec.ResourceExec) error {
	var env []string
	if execBlock.Env != nil {
		for key, value := range *execBlock.Env {
			env = append(env, fmt.Sprintf("%s=\"%s\"", key, value))
		}
	}

	cmd := execute.ExecTask{
		Command:     execBlock.Command,
		Shell:       true,
		Env:         env,
		StreamStdio: false,
	}

	// Execute the command
	result, err := cmd.Execute(context.Background())
	if err != nil {
		return err
	}

	execBlock.Stdout = &result.Stdout
	execBlock.Stderr = &result.Stderr

	if err := dr.AppendExecEntry(actionId, execBlock); err != nil {
		return err
	}

	return nil
}

func (dr *DependencyResolver) HandleLLMChat(actionId string, chatBlock *pklLLM.ResourceChat) error {
	go func() error {
		err := dr.processLLMChat(actionId, chatBlock)
		if err != nil {
			return err
		}

		return nil
	}()

	return nil
}

func (dr *DependencyResolver) processLLMChat(actionId string, chatBlock *pklLLM.ResourceChat) error {
	llm, err := ollama.New(ollama.WithModel(chatBlock.Model))
	if err != nil {
		return err
	}
	completion, err := llm.Call(*dr.Context, chatBlock.Prompt)
	if err != nil {
		return err
	}

	llmResponse := pklLLM.ResourceChat{
		Model:    chatBlock.Model,
		Prompt:   chatBlock.Prompt,
		Response: &completion,
	}

	if err := dr.AppendChatEntry(actionId, &llmResponse); err != nil {
		return err
	}

	return nil
}

func ShouldSkip(conditions *[]bool) bool {
	for _, condition := range *conditions {
		if condition {
			return true // Skip if any condition is true
		}
	}
	return false
}

// Function to check if all conditions in a pre/postflight check are met
func AllConditionsMet(conditions *[]bool) bool {
	for _, condition := range *conditions {
		if !condition {
			return false // Return false if any condition is not met
		}
	}
	return true // All conditions met
}

func (dr *DependencyResolver) CreateResponsePklFile(apiResponseBlock *apiserverresponse.APIServerResponse) error {
	success := apiResponseBlock.Success
	var responseData []string
	var errorsStr string

	// Check the ResponseFlag (assuming this is a precondition)
	if err := dr.GetResponseFlag(); err != nil {
		return err
	}

	// Check if the response file already exists, and remove it if so
	if _, err := dr.Fs.Stat(dr.ResponsePklFile); err == nil {
		if err := dr.Fs.RemoveAll(dr.ResponsePklFile); err != nil {
			logging.Error("Unable to delete old response file", "response-pkl-file", dr.ResponsePklFile)
			return err
		}
	}

	// Format the success as "success = true/false"
	successStr := fmt.Sprintf("success = %v", success)

	// Process the response block
	if apiResponseBlock.Response != nil && apiResponseBlock.Response.Data != nil {
		// Convert the data slice to a string representation
		responseData = make([]string, len(apiResponseBlock.Response.Data))
		for i, v := range apiResponseBlock.Response.Data {
			responseData[i] = fmt.Sprintf(`
"""
%v
"""
`, v) // Convert each item to a string
		}
	}

	// Format the response block as "response { data { ... } }"
	var responseStr string
	if len(responseData) > 0 {
		responseStr = fmt.Sprintf(`
response {
  data {
%s
  }
}`, strings.Join(responseData, "\n    ")) // Properly format the data block with indentation
	}

	// Process the errors block
	if apiResponseBlock.Errors != nil {
		errorsStr = fmt.Sprintf(`
errors {
  code = %d
  message = %q
}`, apiResponseBlock.Errors.Code, apiResponseBlock.Errors.Message)
	}

	// Combine everything into sections as []string
	sections := []string{successStr, responseStr, errorsStr}

	// Create and process the PKL file
	if err := evaluator.CreateAndProcessPklFile(dr.Fs, sections, dr.ResponsePklFile, "APIServerResponse.pkl",
		nil, evaluator.EvalPkl); err != nil {
		return err
	}

	return nil
}

func (dr *DependencyResolver) GetResponseFlag() error {
	responseFiles := []struct {
		Flag              string
		Ext               string
		PklResponseFormat string
	}{
		{"response-jsonnet", ".json", "jsonnet"},
		{"response-txtpb", ".txtpb", "textproto"},
		{"response-yaml", ".yaml", "yaml"},
		{"response-plist", ".plist", "plist"},
		{"response-xml", ".xml", "xml"},
		{"response-pcf", ".pcf", "pcf"},
		{"response-json", ".json", "json"},
	}

	// Loop through each response flag file and check its existence
	for _, file := range responseFiles {
		dr.ResponseFlag = filepath.Join(dr.ActionDir, "/api/"+file.Flag)

		// Check if the response flag file exists
		exists, err := afero.Exists(dr.Fs, dr.ResponseFlag)
		if err != nil {
			return fmt.Errorf("error checking file existence: %w", err)
		}

		if exists {
			// If the file exists, return the file extension and content type
			fmt.Printf("Response flag file found: %s\n", dr.ResponseFlag)
			dr.ResponseType = file.PklResponseFormat
			dr.ResponseTargetFile = filepath.Join(dr.ActionDir, fmt.Sprintf("/api/response%s", file.Ext))
			return nil
		}
	}

	// If no response flag file is found, return an error
	return fmt.Errorf("no valid response flag file found in %s", dr.ActionDir)
}

func (dr *DependencyResolver) EvalPklFormattedResponseFile() (string, error) {
	// Validate that the file has a .pkl extension
	if filepath.Ext(dr.ResponsePklFile) != ".pkl" {
		errMsg := fmt.Sprintf("file '%s' must have a .pkl extension", dr.ResponsePklFile)
		logging.Error(errMsg)
		return "", fmt.Errorf(errMsg)
	}

	if _, err := dr.Fs.Stat(dr.ResponseTargetFile); err == nil {
		if err := dr.Fs.RemoveAll(dr.ResponseTargetFile); err != nil {
			logging.Error("Unable to delete old response file", "response-file", dr.ResponseTargetFile)
			return "", err
		}
	}

	// Ensure that the 'pkl' binary is available
	if err := evaluator.EnsurePklBinaryExists(); err != nil {
		return "", err
	}

	cmd := execute.ExecTask{
		Command:     "pkl",
		Args:        []string{"eval", "--format", dr.ResponseType, "--output-path", dr.ResponseTargetFile, dr.ResponsePklFile},
		StreamStdio: false,
	}

	// Execute the command
	result, err := cmd.Execute(context.Background())
	if err != nil {
		errMsg := "command execution failed"
		logging.Error(errMsg, "error", err)
		return "", fmt.Errorf("%s: %w", errMsg, err)
	}

	// Check for non-zero exit code
	if result.ExitCode != 0 {
		errMsg := fmt.Sprintf("command failed with exit code %d: %s", result.ExitCode, result.Stderr)
		logging.Error(errMsg)
		return "", fmt.Errorf(errMsg)
	}

	return result.Stdout, nil
}
