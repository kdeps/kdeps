package resolver

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/google/uuid"
	"github.com/kdeps/kartographer/graph"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/utils"
	pklRes "github.com/kdeps/schema/gen/resource"
	pklWf "github.com/kdeps/schema/gen/workflow"
	"github.com/spf13/afero"
)

type DependencyResolver struct {
	Fs                   afero.Fs
	Logger               *logging.Logger
	Resources            []ResourceNodeEntry
	ResourceDependencies map[string][]string
	DependencyGraph      []string
	VisitedPaths         map[string]bool
	Context              context.Context
	Graph                *graph.DependencyGraph
	Environment          *environment.Environment
	Workflow             pklWf.Workflow
	RequestID            string
	RequestPklFile       string
	ResponsePklFile      string
	ResponseTargetFile   string
	ProjectDir           string
	AgentDir             string
	ActionDir            string
	FilesDir             string
	DataDir              string
	APIServerMode        bool
	AnacondaInstalled    bool
}

type ResourceNodeEntry struct {
	ActionID string `pkl:"actionID"`
	File     string `pkl:"file"`
}

func NewGraphResolver(fs afero.Fs, ctx context.Context, env *environment.Environment, agentDir, actionDir, graphID string, logger *logging.Logger) (*DependencyResolver, error) {
	var dataDir, filesDir, projectDir, pklWfFile, pklWfParentFile string

	if env.DockerMode == "1" {
		agentDir = filepath.Join(agentDir, "/workflow/")
		pklWfFile = filepath.Join(agentDir, "workflow.pkl")
		pklWfParentFile = filepath.Join(agentDir, "../workflow.pkl")

		// Check if "workflow.pkl" exists using afero.Exists
		exists, err := afero.Exists(fs, pklWfFile)
		if err != nil {
			return nil, fmt.Errorf("error checking %s: %w", pklWfFile, err)
		}

		if !exists {
			// If "workflow.pkl" doesn't exist, check for "../workflow.pkl"
			existsParent, errParent := afero.Exists(fs, pklWfParentFile)
			if errParent != nil {
				return nil, fmt.Errorf("error checking %s: %w", pklWfParentFile, errParent)
			}

			if !existsParent {
				return nil, fmt.Errorf("neither %s nor %s exist", pklWfFile, pklWfParentFile)
			}

			pklWfFile = pklWfParentFile
			agentDir = filepath.Join(agentDir, "../")
			projectDir = filepath.Join(agentDir, "/project/")
			dataDir = filepath.Join(projectDir, "/data/")
			filesDir = filepath.Join(actionDir, "/files/")
		} else {
			projectDir = filepath.Join(agentDir, "../project/")
			dataDir = filepath.Join(projectDir, "/data/")
			filesDir = filepath.Join(actionDir, "/files/")
		}

		// List of directories to create
		directories := []string{
			projectDir,
			actionDir,
			filesDir,
		}

		// Create directories
		if err := utils.CreateDirectories(fs, ctx, directories); err != nil {
			return nil, fmt.Errorf("error creating directory: %w", err)
		}

		// List of files to create (stamp file)
		files := []string{
			filepath.Join(actionDir, graphID),
		}

		if err := utils.CreateFiles(fs, ctx, files); err != nil {
			return nil, fmt.Errorf("error creating file: %w", err)
		}
	}

	requestPklFile := filepath.Join(actionDir, "/api/"+graphID+"__request.pkl")
	responsePklFile := filepath.Join(actionDir, "/api/"+graphID+"__response.pkl")
	responseTargetFile := filepath.Join(actionDir, "/api/"+graphID+"__response.json")

	dependencyResolver := &DependencyResolver{
		Fs:                   fs,
		ResourceDependencies: make(map[string][]string),
		Logger:               logger,
		VisitedPaths:         make(map[string]bool),
		Context:              ctx,
		Environment:          env,
		AgentDir:             agentDir,
		ActionDir:            actionDir,
		FilesDir:             filesDir,
		DataDir:              dataDir,
		RequestID:            graphID,
		RequestPklFile:       requestPklFile,
		ResponsePklFile:      responsePklFile,
		ResponseTargetFile:   responseTargetFile,
		ProjectDir:           projectDir,
	}

	workflowConfiguration, err := pklWf.LoadFromPath(ctx, pklWfFile)
	if err != nil {
		return nil, err
	}
	dependencyResolver.Workflow = workflowConfiguration
	if workflowConfiguration.GetSettings() != nil {
		dependencyResolver.APIServerMode = workflowConfiguration.GetSettings().APIServerMode
		agentSettings := workflowConfiguration.GetSettings().AgentSettings
		dependencyResolver.AnacondaInstalled = agentSettings.InstallAnaconda
	}

	dependencyResolver.Graph = graph.NewDependencyGraph(fs, logger.BaseLogger(), dependencyResolver.ResourceDependencies)
	if dependencyResolver.Graph == nil {
		return nil, errors.New("failed to initialize dependency graph")
	}

	return dependencyResolver, nil
}

// processResourceStep consolidates the pattern of: get timestamp, run a handler, adjust timeout (if provided),
// then wait for the timestamp change.
func (dr *DependencyResolver) processResourceStep(resourceID, step string, timeoutPtr *int, handler func() error) error {
	timestamp, err := dr.GetCurrentTimestamp(resourceID, step)
	if err != nil {
		return fmt.Errorf("%s error: %w", step, err)
	}

	timeout := 60 * time.Second
	if timeoutPtr != nil {
		timeout = time.Duration(*timeoutPtr) * time.Second
		dr.Logger.Infof("Timeout duration for '%s' is set to '%.0f' seconds", resourceID, timeout.Seconds())
	}

	if err := handler(); err != nil {
		return fmt.Errorf("%s error: %w", step, err)
	}

	if err := dr.WaitForTimestampChange(resourceID, timestamp, timeout, step); err != nil {
		return fmt.Errorf("%s timeout awaiting for output: %w", step, err)
	}
	return nil
}

// ensureRequestFileExists checks for the existence of the request stamp file and,
// if not found, creates a new request by updating RequestID and preparing directories/files.
func (dr *DependencyResolver) ensureRequestFileExists() error {
	requestFilePath := filepath.Join(dr.ActionDir, dr.RequestID)
	if _, err := os.Stat(requestFilePath); os.IsNotExist(err) {
		// File does not exist, generate a new graphID
		newGraphID := uuid.New().String()
		dr.Logger.Info("creating a new graph resolver ID", "requestID", newGraphID)
		dr.RequestID = newGraphID

		stampFile := filepath.Join(dr.ActionDir, dr.RequestID)
		if err := utils.CreateFiles(dr.Fs, dr.Context, []string{stampFile}); err != nil {
			dr.Logger.Error("error creating stamp file for new request", "file", stampFile, "err", err)

			return err
		}

		dr.RequestPklFile = filepath.Join(dr.ActionDir, "/api/"+dr.RequestID+"__request.pkl")
		dr.ResponsePklFile = filepath.Join(dr.ActionDir, "/api/"+dr.RequestID+"__response.pkl")
		dr.ResponseTargetFile = filepath.Join(dr.ActionDir, "/api/"+dr.RequestID+"__response.json")

		if err := dr.PrepareWorkflowDir(); err != nil {
			dr.Logger.Error("failed to prepare workflow directory", "err", err)

			return err
		}

		if err := dr.PrepareImportFiles(); err != nil {
			dr.Logger.Error("failed to prepare import files", "err", err)

			return err
		}
	} else if err != nil {
		dr.Logger.Errorf("failed to check if file exists: %v", err)

		return err
	}
	return nil
}

// HandleRunAction is the main entry point to process resource run blocks.
func (dr *DependencyResolver) HandleRunAction() (bool, error) {
	// Recover from panics in this function.
	defer func() {
		if r := recover(); r != nil {
			dr.Logger.Error("panic recovered in HandleRunAction", "panic", r)
			buf := make([]byte, 1<<16)
			stackSize := runtime.Stack(buf, false)
			dr.Logger.Error("stack trace", "stack", string(buf[:stackSize]))
		}
	}()

	// Ensure a valid request file exists
	if err := dr.ensureRequestFileExists(); err != nil {
		// If ensureRequestFileExists fails, the logger would have already recorded the fatal error.
		return false, err
	}
	requestFilePath := filepath.Join(dr.ActionDir, dr.RequestID)

	visited := make(map[string]bool)
	actionID := dr.Workflow.GetTargetActionID()
	dr.Logger.Debug("processing resources...")

	if err := dr.LoadResourceEntries(); err != nil {
		return dr.HandleAPIErrorResponse(500, err.Error(), true)
	}

	// Build dependency stack for the target action
	stack := dr.Graph.BuildDependencyStack(actionID, visited)

	// Process each resource in the dependency stack
	for _, nodeActionID := range stack {
		for _, res := range dr.Resources {
			if res.ActionID != nodeActionID {
				continue
			}

			rsc, err := pklRes.LoadFromPath(dr.Context, res.File)
			if err != nil {
				return dr.HandleAPIErrorResponse(500, err.Error(), true)
			}

			runBlock := rsc.Run
			if runBlock == nil {
				continue
			}

			// Skip condition
			if runBlock.SkipCondition != nil && utils.ShouldSkip(runBlock.SkipCondition) {
				dr.Logger.Infof("skip condition met, skipping: %s", res.ActionID)
				continue
			}

			// Preflight check
			if runBlock.PreflightCheck != nil && runBlock.PreflightCheck.Validations != nil &&
				!utils.AllConditionsMet(runBlock.PreflightCheck.Validations) {
				dr.Logger.Error("preflight check not met, failing:", res.ActionID)
				if runBlock.PreflightCheck.Error != nil {
					return dr.HandleAPIErrorResponse(
						runBlock.PreflightCheck.Error.Code,
						fmt.Sprintf("%s: %s", runBlock.PreflightCheck.Error.Message, res.ActionID), false)
				}
				return dr.HandleAPIErrorResponse(500, "Preflight check failed for resource: "+res.ActionID, false)
			}

			// Process Exec step, if defined
			if runBlock.Exec != nil && runBlock.Exec.Command != "" {
				if err := dr.processResourceStep(res.ActionID, "exec", runBlock.Exec.TimeoutDuration, func() error {
					return dr.HandleExec(res.ActionID, runBlock.Exec)
				}); err != nil {
					dr.Logger.Error("exec error:", res.ActionID)
					return dr.HandleAPIErrorResponse(500, fmt.Sprintf("Exec failed for resource: %s - %s", res.ActionID, err), false)
				}
			}

			// Process Python step, if defined
			if runBlock.Python != nil && runBlock.Python.Script != "" {
				if err := dr.processResourceStep(res.ActionID, "python", runBlock.Python.TimeoutDuration, func() error {
					return dr.HandlePython(res.ActionID, runBlock.Python)
				}); err != nil {
					dr.Logger.Error("python error:", res.ActionID)
					return dr.HandleAPIErrorResponse(500, fmt.Sprintf("Python script failed for resource: %s - %s", res.ActionID, err), false)
				}
			}

			// Process Chat (LLM) step, if defined
			if runBlock.Chat != nil && runBlock.Chat.Model != "" && runBlock.Chat.Prompt != "" {
				if err := dr.processResourceStep(res.ActionID, "llm", runBlock.Chat.TimeoutDuration, func() error {
					return dr.HandleLLMChat(res.ActionID, runBlock.Chat)
				}); err != nil {
					dr.Logger.Error("lLM chat error:", res.ActionID)
					return dr.HandleAPIErrorResponse(500, fmt.Sprintf("LLM chat failed for resource: %s - %s", res.ActionID, err), true)
				}
			}

			// Process HTTP Client step, if defined
			if runBlock.HTTPClient != nil && runBlock.HTTPClient.Method != "" && runBlock.HTTPClient.Url != "" {
				if err := dr.processResourceStep(res.ActionID, "client", runBlock.HTTPClient.TimeoutDuration, func() error {
					return dr.HandleHTTPClient(res.ActionID, runBlock.HTTPClient)
				}); err != nil {
					dr.Logger.Error("HTTP client error:", res.ActionID)
					return dr.HandleAPIErrorResponse(500, fmt.Sprintf("HTTP client failed for resource: %s - %s", res.ActionID, err), false)
				}
			}

			// API Response
			if dr.APIServerMode && runBlock.APIResponse != nil {
				if err := dr.CreateResponsePklFile(*runBlock.APIResponse); err != nil {
					return dr.HandleAPIErrorResponse(500, err.Error(), true)
				}
			}
		}
	}

	// Remove the request stamp file
	if err := dr.Fs.RemoveAll(requestFilePath); err != nil {
		dr.Logger.Error("failed to delete old requestID file", "file", requestFilePath, "error", err)
		return false, err
	}

	dr.Logger.Debug("all resources finished processing")
	return false, nil
}
