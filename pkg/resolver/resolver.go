package resolver

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
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
	ID   string `pkl:"id"`
	File string `pkl:"file"`
}

func NewGraphResolver(fs afero.Fs, ctx context.Context, env *environment.Environment, agentDir string, logger *logging.Logger) (*DependencyResolver, error) {
	graphID := uuid.New().String()

	var dataDir, actionDir, filesDir, projectDir, pklWfFile, pklWfParentFile string

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
			actionDir = filepath.Join(agentDir, "/actions")
			dataDir = filepath.Join(projectDir, "/data/")
			filesDir = filepath.Join(actionDir, "/files/")
		} else {
			projectDir = filepath.Join(agentDir, "../project/")
			dataDir = filepath.Join(projectDir, "/data/")
			actionDir = filepath.Join(agentDir, "../actions")
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
		} else {
			logger.Debug("directories created successfully")
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

func (dr *DependencyResolver) HandleRunAction() (bool, error) {
	// defer func() {
	//	if r := recover(); r != nil {
	//		dr.Logger.Error("recovered from panic:", r)
	//		dr.HandleAPIErrorResponse(500, "Server panic occurred", true)
	//	}
	// }()

	visited := make(map[string]bool)
	actionID := dr.Workflow.GetAction()
	timeoutDuration := 60 * time.Second
	dr.Logger.Debug("processing resources...")
	if err := dr.LoadResourceEntries(); err != nil {
		return dr.HandleAPIErrorResponse(500, err.Error(), true)
	}

	stack := dr.Graph.BuildDependencyStack(actionID, visited)
	for _, resNode := range stack {
		for _, res := range dr.Resources {
			if res.ID == resNode {
				rsc, err := pklRes.LoadFromPath(dr.Context, res.File)
				if err != nil {
					return dr.HandleAPIErrorResponse(500, err.Error(), true)
				}

				runBlock := rsc.Run
				if runBlock != nil {
					// Check Skip Condition
					if runBlock.SkipCondition != nil {
						if utils.ShouldSkip(runBlock.SkipCondition) {
							dr.Logger.Debug("skip condition met, skipping:", res.ID)
							continue
						}
					}

					// Handle Preflight Check
					if runBlock.PreflightCheck != nil && runBlock.PreflightCheck.Validations != nil {
						if !utils.AllConditionsMet(runBlock.PreflightCheck.Validations) {
							dr.Logger.Error("preflight check not met, failing:", res.ID)
							if runBlock.PreflightCheck.Error != nil {
								return dr.HandleAPIErrorResponse(
									runBlock.PreflightCheck.Error.Code,
									fmt.Sprintf("%s: %s", runBlock.PreflightCheck.Error.Message, res.ID), false)
							}
							dr.Logger.Error("preflight check not met, failing:", res.ID)
							return dr.HandleAPIErrorResponse(500, "Preflight check failed for resource: "+res.ID, false)
						}
					}

					if runBlock.Exec != nil && runBlock.Exec.Command != "" {
						timestamp, err := dr.GetCurrentTimestamp(res.ID, "exec")
						if err != nil {
							dr.Logger.Error("exec error:", res.ID)
							return dr.HandleAPIErrorResponse(500, fmt.Sprintf("Exec failed for resource: %s - %s", res.ID, err), false)
						}

						if err := dr.HandleExec(res.ID, runBlock.Exec); err != nil {
							dr.Logger.Error("exec error:", res.ID)
							return dr.HandleAPIErrorResponse(500, fmt.Sprintf("Exec failed for resource: %s - %s", res.ID, err), false)
						}

						if runBlock.Exec.TimeoutDuration != nil {
							timeoutDuration = time.Duration(*runBlock.Exec.TimeoutDuration) * time.Second
						}

						if err := dr.WaitForTimestampChange(res.ID, timestamp, timeoutDuration, "exec"); err != nil {
							dr.Logger.Error("exec error:", res.ID)
							return dr.HandleAPIErrorResponse(500, fmt.Sprintf("Exec timeout awaiting for output: %s - %s", res.ID, err), false)
						}
					}

					if runBlock.Python != nil && runBlock.Python.Script != "" {
						timestamp, err := dr.GetCurrentTimestamp(res.ID, "python")
						if err != nil {
							dr.Logger.Error("python error:", res.ID)
							return dr.HandleAPIErrorResponse(500, fmt.Sprintf("Python script failed for resource: %s - %s", res.ID, err), false)
						}

						if err := dr.HandlePython(res.ID, runBlock.Python); err != nil {
							dr.Logger.Error("python error:", res.ID)
							return dr.HandleAPIErrorResponse(500, fmt.Sprintf("Python script failed for resource: %s - %s", res.ID, err), false)
						}

						if runBlock.Python.TimeoutDuration != nil {
							timeoutDuration = time.Duration(*runBlock.Python.TimeoutDuration) * time.Second
						}

						if err := dr.WaitForTimestampChange(res.ID, timestamp, timeoutDuration, "python"); err != nil {
							dr.Logger.Error("python error:", res.ID)
							return dr.HandleAPIErrorResponse(500, fmt.Sprintf("Python timeout awaiting for output: %s - %s", res.ID, err), false)
						}
					}

					if runBlock.Chat != nil && runBlock.Chat.Model != "" && runBlock.Chat.Prompt != "" {
						timestamp, err := dr.GetCurrentTimestamp(res.ID, "llm")
						if err != nil {
							dr.Logger.Error("lLM chat error:", res.ID)
							return dr.HandleAPIErrorResponse(500, fmt.Sprintf("LLM chat failed for resource: %s - %s", res.ID, err), false)
						}

						if err := dr.HandleLLMChat(res.ID, runBlock.Chat); err != nil {
							dr.Logger.Error("lLM chat error:", res.ID)
							return dr.HandleAPIErrorResponse(500, fmt.Sprintf("LLM chat failed for resource: %s - %s", res.ID, err), true)
						}

						if runBlock.Chat.TimeoutDuration != nil {
							timeoutDuration = time.Duration(*runBlock.Chat.TimeoutDuration) * time.Second
						}

						if err := dr.WaitForTimestampChange(res.ID, timestamp, timeoutDuration, "llm"); err != nil {
							dr.Logger.Error("lLM chat error:", res.ID)
							return dr.HandleAPIErrorResponse(500, fmt.Sprintf("LLM chat timeout awaiting for response: %s - %s", res.ID, err), false)
						}
					}

					if runBlock.HTTPClient != nil && runBlock.HTTPClient.Method != "" && runBlock.HTTPClient.Url != "" {
						timestamp, err := dr.GetCurrentTimestamp(res.ID, "client")
						if err != nil {
							dr.Logger.Error("hTTP client error:", res.ID)
							return dr.HandleAPIErrorResponse(500, fmt.Sprintf("HTTP client failed for resource: %s - %s", res.ID, err), false)
						}

						if err := dr.HandleHTTPClient(res.ID, runBlock.HTTPClient); err != nil {
							dr.Logger.Error("hTTP client error:", res.ID)
							return dr.HandleAPIErrorResponse(500, fmt.Sprintf("HTTP client failed for resource: %s - %s", res.ID, err), false)
						}

						if runBlock.HTTPClient.TimeoutDuration != nil {
							timeoutDuration = time.Duration(*runBlock.HTTPClient.TimeoutDuration) * time.Second
						}

						if err := dr.WaitForTimestampChange(res.ID, timestamp, timeoutDuration, "client"); err != nil {
							dr.Logger.Error("hTTP client error:", res.ID)
							return dr.HandleAPIErrorResponse(500, fmt.Sprintf("HTTP client timeout awaiting for output: %s - %s", res.ID, err), false)
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
		}
	}

	dr.Logger.Debug("all resources finished processing")
	return false, nil
}
