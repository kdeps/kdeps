package resolver

import (
	"context"
	"fmt"
	"io"
	"kdeps/pkg/environment"
	"kdeps/pkg/evaluator"
	"kdeps/pkg/logging"
	"kdeps/pkg/resource"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/alexellis/go-execute/v2"
	"github.com/charmbracelet/log"
	"github.com/kdeps/kartographer/graph"
	apiserverresponse "github.com/kdeps/schema/gen/api_server_response"
	pklRes "github.com/kdeps/schema/gen/resource"
	pklWf "github.com/kdeps/schema/gen/workflow"
	"github.com/spf13/afero"
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
	var actionDir, requestPklFile, responsePklFile, projectDir string

	if env.DockerMode == "1" {
		agentDir = filepath.Join(agentDir, "/workflow/")
		projectDir = filepath.Join(agentDir, "../project/")
		actionDir = filepath.Join(agentDir, "../actions")
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

	pklWfFile := filepath.Join(agentDir, "workflow.pkl")
	if err := WaitForFile(fs, pklWfFile); err != nil {
		return nil, err
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

	fmt.Printf(`
		AgentDir:             %s - agentDir,
		ActionDir:            %s - actionDir,
		RequestPklFile:       %s - requestPklFile,
		ResponsePklFile:      %s - responsePklFile,
		ProjectDir:           %s - projectDir,

`, agentDir, actionDir, requestPklFile, responsePklFile, projectDir)

	return dependencyResolver, nil
}

func (dr *DependencyResolver) LoadResourceEntries() error {
	workflowDir := filepath.Join(dr.AgentDir, "resources")
	if err := afero.Walk(dr.Fs, workflowDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Errorf("Error walking through files: %s - %s", workflowDir, err)
			return err
		}

		// Check if the file has a .pkl extension
		if !info.IsDir() && filepath.Ext(path) == ".pkl" {
			// Load the resource file
			pklRes, err := resource.LoadResource(*dr.Context, path)
			if err != nil {
				fmt.Errorf("Error loading .pkl file "+path, err)
			}

			dr.Resources = append(dr.Resources, ResourceNodeEntry{
				Id:   pklRes.Id,
				File: path,
			})

			if pklRes.Requires != nil {
				dr.ResourceDependencies[pklRes.Id] = *pklRes.Requires
			} else {
				dr.ResourceDependencies[pklRes.Id] = nil
			}
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (dr *DependencyResolver) PrependDynamicImports(res ResourceNodeEntry) error {
	content, err := afero.ReadFile(dr.Fs, res.File)
	if err != nil {
		return err
	}

	// Define a regular expression to match "{{value}}"
	re := regexp.MustCompile(`\@\((.*)\)`)

	importCheck := map[string]string{
		dr.RequestPklFile: "",
		filepath.Join(dr.ActionDir, "/llm/llm_response.pkl"):       "llm",
		filepath.Join(dr.ActionDir, "/client/client_response.pkl"): "client",
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
				localVarLine := fmt.Sprintf("local %s = %s_responses\n", variable, importName)
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
			err = afero.WriteFile(dr.Fs, res.File, []byte(newContent), 0644)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func WaitForFile(fs afero.Fs, filepath string) error {
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
				return nil
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

func (dr *DependencyResolver) HandleRunAction() error {
	defer func() {
		if r := recover(); r != nil {
			logging.Error("Recovered from panic:", r)
			dr.handleAPIErrorResponse(500, "Server panic occurred")
		}
	}()

	visited := make(map[string]bool)
	actionId := dr.Workflow.Action

	logging.Info("Processing resources...")
	if err := dr.LoadResourceEntries(); err != nil {
		return dr.handleAPIErrorResponse(500, err.Error())
	}

	stack := dr.Graph.BuildDependencyStack(actionId, visited)
	for _, resNode := range stack {
		for _, res := range dr.Resources {
			if res.Id == resNode {
				logging.Info("Executing resource: ", res.Id)

				if err := dr.PrependDynamicImports(res); err != nil {
					return dr.handleAPIErrorResponse(500, err.Error())
				}

				rsc, err := pklRes.LoadFromPath(*dr.Context, res.File)
				if err != nil {
					return dr.handleAPIErrorResponse(500, err.Error())
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
								return dr.handleAPIErrorResponse(
									runBlock.PreflightCheck.Error.Code,
									fmt.Sprintf("%s: %s", runBlock.PreflightCheck.Error.Message, res.Id))
							}

							return dr.handleAPIErrorResponse(500, "Preflight check failed for resource: "+res.Id)
						}
					}

					// Process the resource...

					// Handle Postflight Check
					if runBlock.PostflightCheck != nil && runBlock.PostflightCheck.Validations != nil {
						if !AllConditionsMet(runBlock.PostflightCheck.Validations) {
							if runBlock.PostflightCheck.Error != nil {
								return dr.handleAPIErrorResponse(
									runBlock.PostflightCheck.Error.Code,
									fmt.Sprintf("%s: %s", runBlock.PostflightCheck.Error.Message, res.Id))
							}

							logging.Error("Postflight check not met, failing:", res.Id)
							return dr.handleAPIErrorResponse(500, "Postflight check failed for resource: "+res.Id)
						}
					}

					// API Response
					if dr.ApiServerMode && runBlock.ApiResponse != nil {
						if err := dr.CreateResponsePklFile(runBlock.ApiResponse); err != nil {
							return dr.handleAPIErrorResponse(500, err.Error())
						}
					}
				}
			}
		}
	}

	logging.Info("All resources finished processing")
	return nil
}

// Helper function to handle API error responses
func (dr *DependencyResolver) handleAPIErrorResponse(code int, message string) error {
	if dr.ApiServerMode {
		errorResponse := NewAPIServerResponse(false, nil, code, message)
		if err := dr.CreateResponsePklFile(&errorResponse); err != nil {
			logging.Error("Failed to create error response file:", err)
			return err
		}
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
			responseData[i] = fmt.Sprintf("%v", v) // Convert each item to a string
		}
	}

	// Format the response block as "response { data { ... } }"
	var responseStr string
	if len(responseData) > 0 {
		responseStr = fmt.Sprintf(`
response {
  data {
    "%s"
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
