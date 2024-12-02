package resolver

import (
	"context"
	"fmt"
	"kdeps/pkg/environment"
	"kdeps/pkg/utils"
	"path/filepath"
	"time"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
	"github.com/kdeps/kartographer/graph"
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
	Workflow             pklWf.Workflow
	RequestId            string
	RequestPklFile       string
	ResponsePklFile      string
	ResponseTargetFile   string
	ProjectDir           string
	AgentDir             string
	ActionDir            string
	ApiServerMode        bool
	AnacondaInstalled    bool
}

type ResourceNodeEntry struct {
	Id   string `pkl:"id"`
	File string `pkl:"file"`
}

func NewGraphResolver(fs afero.Fs, logger *log.Logger, ctx context.Context, env *environment.Environment, agentDir string) (*DependencyResolver, error) {
	graphId := uuid.New().String()

	var actionDir, projectDir, pklWfFile, pklWfParentFile string

	if env.DockerMode == "1" {
		agentDir = filepath.Join(agentDir, "/workflow/")
		pklWfFile = filepath.Join(agentDir, "workflow.pkl")
		pklWfParentFile = filepath.Join(agentDir, "../workflow.pkl")

		// Check if "workflow.pkl" exists using afero.Exists
		exists, err := afero.Exists(fs, pklWfFile)
		if err != nil {
			return nil, fmt.Errorf("error checking %s: %v", pklWfFile, err)
		}

		if !exists {
			// If "workflow.pkl" doesn't exist, check for "../workflow.pkl"
			existsParent, errParent := afero.Exists(fs, pklWfParentFile)
			if errParent != nil {
				return nil, fmt.Errorf("error checking %s: %v", pklWfParentFile, errParent)
			}

			if !existsParent {
				return nil, fmt.Errorf("neither %s nor %s exist", pklWfFile, pklWfParentFile)
			}

			pklWfFile = pklWfParentFile
			agentDir = filepath.Join(agentDir, "../")
			projectDir = filepath.Join(agentDir, "/project/")
			actionDir = filepath.Join(agentDir, "/actions")
		} else {
			projectDir = filepath.Join(agentDir, "../project/")
			actionDir = filepath.Join(agentDir, "../actions")
		}

	}

	requestPklFile := filepath.Join(actionDir, "/api/"+graphId+"__request.pkl")
	responsePklFile := filepath.Join(actionDir, "/api/"+graphId+"__response.pkl")
	responseTargetFile := filepath.Join(actionDir, "/api/"+graphId+"__response.json")

	dependencyResolver := &DependencyResolver{
		Fs:                   fs,
		ResourceDependencies: make(map[string][]string),
		Logger:               logger,
		VisitedPaths:         make(map[string]bool),
		Context:              &ctx,
		AgentDir:             agentDir,
		ActionDir:            actionDir,
		RequestId:            graphId,
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
		dependencyResolver.ApiServerMode = workflowConfiguration.GetSettings().ApiServerMode
		agentSettings := workflowConfiguration.GetSettings().AgentSettings
		dependencyResolver.AnacondaInstalled = agentSettings.InstallAnaconda
	}

	dependencyResolver.Graph = graph.NewDependencyGraph(fs, logger, dependencyResolver.ResourceDependencies)
	if dependencyResolver.Graph == nil {
		return nil, fmt.Errorf("failed to initialize dependency graph")
	}

	return dependencyResolver, nil
}

func (dr *DependencyResolver) HandleRunAction() (bool, error) {
	defer func() {
		if r := recover(); r != nil {
			dr.Logger.Error("Recovered from panic:", r)
			dr.HandleAPIErrorResponse(500, "Server panic occurred", true)
		}
	}()

	visited := make(map[string]bool)
	actionId := dr.Workflow.GetAction()
	timeoutSeconds := 60 * time.Second
	dr.Logger.Debug("Processing resources...")
	if err := dr.LoadResourceEntries(); err != nil {
		return dr.HandleAPIErrorResponse(500, err.Error(), true)
	}

	stack := dr.Graph.BuildDependencyStack(actionId, visited)
	for _, resNode := range stack {
		for _, res := range dr.Resources {
			if res.Id == resNode {
				rsc, err := pklRes.LoadFromPath(*dr.Context, res.File)
				if err != nil {
					return dr.HandleAPIErrorResponse(500, err.Error(), true)
				}

				runBlock := rsc.Run
				if runBlock != nil {
					// Check Skip Condition
					if runBlock.SkipCondition != nil {
						if utils.ShouldSkip(runBlock.SkipCondition) {
							dr.Logger.Debug("Skip condition met, skipping:", res.Id)
							continue
						}
					}

					// Handle Preflight Check
					if runBlock.PreflightCheck != nil && runBlock.PreflightCheck.Validations != nil {
						if !utils.AllConditionsMet(runBlock.PreflightCheck.Validations) {
							dr.Logger.Error("Preflight check not met, failing:", res.Id)
							if runBlock.PreflightCheck.Error != nil {
								return dr.HandleAPIErrorResponse(
									runBlock.PreflightCheck.Error.Code,
									fmt.Sprintf("%s: %s", runBlock.PreflightCheck.Error.Message, res.Id), false)
							}
							dr.Logger.Error("Preflight check not met, failing:", res.Id)
							return dr.HandleAPIErrorResponse(500, "Preflight check failed for resource: "+res.Id, false)
						}
					}

					if runBlock.Exec != nil && runBlock.Exec.Command != "" {
						timestamp, err := dr.GetCurrentTimestamp(res.Id, "exec")
						if err != nil {
							dr.Logger.Error("Exec error:", res.Id)
							return dr.HandleAPIErrorResponse(500, fmt.Sprintf("Exec failed for resource: %s - %s", res.Id, err), false)
						}

						if err := dr.HandleExec(res.Id, runBlock.Exec); err != nil {
							dr.Logger.Error("Exec error:", res.Id)
							return dr.HandleAPIErrorResponse(500, fmt.Sprintf("Exec failed for resource: %s - %s", res.Id, err), false)
						}

						if runBlock.Exec.TimeoutSeconds != nil {
							timeoutSeconds = time.Duration(*runBlock.Exec.TimeoutSeconds) * time.Second
						}

						if err := dr.WaitForTimestampChange(res.Id, timestamp, timeoutSeconds, "exec"); err != nil {
							dr.Logger.Error("Exec error:", res.Id)
							return dr.HandleAPIErrorResponse(500, fmt.Sprintf("Exec timeout awaiting for output: %s - %s", res.Id, err), false)
						}
					}

					if runBlock.Python != nil && runBlock.Python.Script != "" {
						timestamp, err := dr.GetCurrentTimestamp(res.Id, "python")
						if err != nil {
							dr.Logger.Error("Python error:", res.Id)
							return dr.HandleAPIErrorResponse(500, fmt.Sprintf("Python script failed for resource: %s - %s", res.Id, err), false)
						}

						if err := dr.HandlePython(res.Id, runBlock.Python); err != nil {
							dr.Logger.Error("Python error:", res.Id)
							return dr.HandleAPIErrorResponse(500, fmt.Sprintf("Python script failed for resource: %s - %s", res.Id, err), false)
						}

						if runBlock.Python.TimeoutSeconds != nil {
							timeoutSeconds = time.Duration(*runBlock.Python.TimeoutSeconds) * time.Second
						}

						if err := dr.WaitForTimestampChange(res.Id, timestamp, timeoutSeconds, "python"); err != nil {
							dr.Logger.Error("Python error:", res.Id)
							return dr.HandleAPIErrorResponse(500, fmt.Sprintf("Python timeout awaiting for output: %s - %s", res.Id, err), false)
						}
					}

					if runBlock.Chat != nil && runBlock.Chat.Model != "" && runBlock.Chat.Prompt != "" {
						timestamp, err := dr.GetCurrentTimestamp(res.Id, "llm")
						if err != nil {
							dr.Logger.Error("LLM chat error:", res.Id)
							return dr.HandleAPIErrorResponse(500, fmt.Sprintf("LLM chat failed for resource: %s - %s", res.Id, err), false)
						}

						if err := dr.HandleLLMChat(res.Id, runBlock.Chat); err != nil {
							dr.Logger.Error("LLM chat error:", res.Id)
							return dr.HandleAPIErrorResponse(500, fmt.Sprintf("LLM chat failed for resource: %s - %s", res.Id, err), true)
						}

						if runBlock.Chat.TimeoutSeconds != nil {
							timeoutSeconds = time.Duration(*runBlock.Chat.TimeoutSeconds) * time.Second
						}

						if err := dr.WaitForTimestampChange(res.Id, timestamp, timeoutSeconds, "llm"); err != nil {
							dr.Logger.Error("LLM chat error:", res.Id)
							return dr.HandleAPIErrorResponse(500, fmt.Sprintf("LLM chat timeout awaiting for response: %s - %s", res.Id, err), false)
						}
					}

					if runBlock.HttpClient != nil && runBlock.HttpClient.Method != "" && runBlock.HttpClient.Url != "" {
						timestamp, err := dr.GetCurrentTimestamp(res.Id, "client")
						if err != nil {
							dr.Logger.Error("Http client error:", res.Id)
							return dr.HandleAPIErrorResponse(500, fmt.Sprintf("Http client failed for resource: %s - %s", res.Id, err), false)
						}

						if err := dr.HandleHttpClient(res.Id, runBlock.HttpClient); err != nil {
							dr.Logger.Error("Http client error:", res.Id)
							return dr.HandleAPIErrorResponse(500, fmt.Sprintf("Http client failed for resource: %s - %s", res.Id, err), false)
						}

						if runBlock.HttpClient.TimeoutSeconds != nil {
							timeoutSeconds = time.Duration(*runBlock.HttpClient.TimeoutSeconds) * time.Second
						}

						if err := dr.WaitForTimestampChange(res.Id, timestamp, timeoutSeconds, "client"); err != nil {
							dr.Logger.Error("Http client error:", res.Id)
							return dr.HandleAPIErrorResponse(500, fmt.Sprintf("Http client timeout awaiting for output: %s - %s", res.Id, err), false)
						}

					}

					// API Response
					if dr.ApiServerMode && runBlock.ApiResponse != nil {
						if err := dr.CreateResponsePklFile(*runBlock.ApiResponse); err != nil {
							return dr.HandleAPIErrorResponse(500, err.Error(), true)
						}
					}
				}
			}
		}
	}

	dr.Logger.Debug("All resources finished processing")
	return false, nil
}
