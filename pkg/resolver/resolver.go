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
	Workflow             *pklWf.Workflow
	RequestId            string
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

type ResponseFileInfo struct {
	ResponseFlagFile string
	ResponseFileExt  string
	ContentType      string
	ResponseType     string
}

func NewGraphResolver(fs afero.Fs, logger *log.Logger, ctx context.Context, env *environment.Environment, agentDir string, responseFile *ResponseFileInfo) (*DependencyResolver, error) {
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
	responseFlag := filepath.Join(actionDir, "/api/"+graphId+"__"+responseFile.ResponseFlagFile)
	responseTargetFile := filepath.Join(actionDir, "/api/"+graphId+responseFile.ResponseFileExt)

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
		ResponseFlag:         responseFlag,
		ResponseType:         responseFile.ResponseType,
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

func (dr *DependencyResolver) HandleRunAction() error {
	defer func() {
		if r := recover(); r != nil {
			dr.Logger.Error("Recovered from panic:", r)
			dr.HandleAPIErrorResponse(500, "Server panic occurred")
		}
	}()

	visited := make(map[string]bool)
	actionId := dr.Workflow.Action
	timeoutSeconds := 60 * time.Second
	dr.Logger.Info("Processing resources...")
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
						if utils.ShouldSkip(runBlock.SkipCondition) {
							dr.Logger.Info("Skip condition met, skipping:", res.Id)
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
									fmt.Sprintf("%s: %s", runBlock.PreflightCheck.Error.Message, res.Id))
							}
							dr.Logger.Error("Preflight check not met, failing:", res.Id)
							return dr.HandleAPIErrorResponse(500, "Preflight check failed for resource: "+res.Id)
						}
					}

					if runBlock.Exec != nil && runBlock.Exec.Command != "" {
						timestamp, err := dr.GetCurrentTimestamp(res.Id, "exec")
						if err != nil {
							dr.Logger.Error("Exec error:", res.Id)
							return dr.HandleAPIErrorResponse(500, fmt.Sprintf("Exec failed for resource: %s - %s", res.Id, err))
						}

						if err := dr.HandleExec(res.Id, runBlock.Exec); err != nil {
							dr.Logger.Error("Exec error:", res.Id)
							return dr.HandleAPIErrorResponse(500, fmt.Sprintf("Exec failed for resource: %s - %s", res.Id, err))
						}

						if runBlock.Exec.TimeoutSeconds != nil {
							timeoutSeconds = time.Duration(*runBlock.Exec.TimeoutSeconds) * time.Second
						}

						if err := dr.WaitForTimestampChange(res.Id, timestamp, timeoutSeconds, "exec"); err != nil {
							dr.Logger.Error("Exec error:", res.Id)
							return dr.HandleAPIErrorResponse(500, fmt.Sprintf("Exec timeout awaiting for output: %s - %s", res.Id, err))
						}

					}

					if runBlock.Chat != nil && runBlock.Chat.Model != "" && runBlock.Chat.Prompt != "" {
						timestamp, err := dr.GetCurrentTimestamp(res.Id, "llm")
						if err != nil {
							dr.Logger.Error("LLM chat error:", res.Id)
							return dr.HandleAPIErrorResponse(500, fmt.Sprintf("LLM chat failed for resource: %s - %s", res.Id, err))
						}

						if err := dr.HandleLLMChat(res.Id, runBlock.Chat); err != nil {
							dr.Logger.Error("LLM chat error:", res.Id)
							return dr.HandleAPIErrorResponse(500, fmt.Sprintf("LLM chat failed for resource: %s - %s", res.Id, err))
						}

						if runBlock.Chat.TimeoutSeconds != nil {
							timeoutSeconds = time.Duration(*runBlock.Chat.TimeoutSeconds) * time.Second
						}

						if err := dr.WaitForTimestampChange(res.Id, timestamp, timeoutSeconds, "llm"); err != nil {
							dr.Logger.Error("LLM chat error:", res.Id)
							return dr.HandleAPIErrorResponse(500, fmt.Sprintf("LLM chat timeout awaiting for response: %s - %s", res.Id, err))
						}
					}

					if runBlock.HttpClient != nil && runBlock.HttpClient.Method != "" && runBlock.HttpClient.Url != "" {
						timestamp, err := dr.GetCurrentTimestamp(res.Id, "client")
						if err != nil {
							dr.Logger.Error("Http client error:", res.Id)
							return dr.HandleAPIErrorResponse(500, fmt.Sprintf("Http client failed for resource: %s - %s", res.Id, err))
						}

						if err := dr.HandleHttpClient(res.Id, runBlock.HttpClient); err != nil {
							dr.Logger.Error("Http client error:", res.Id)
							return dr.HandleAPIErrorResponse(500, fmt.Sprintf("Http client failed for resource: %s - %s", res.Id, err))
						}

						if runBlock.HttpClient.TimeoutSeconds != nil {
							timeoutSeconds = time.Duration(*runBlock.HttpClient.TimeoutSeconds) * time.Second
						}

						if err := dr.WaitForTimestampChange(res.Id, timestamp, timeoutSeconds, "client"); err != nil {
							dr.Logger.Error("Http client error:", res.Id)
							return dr.HandleAPIErrorResponse(500, fmt.Sprintf("Http client timeout awaiting for output: %s - %s", res.Id, err))
						}

					}

					// Handle Postflight Check
					if runBlock.PostflightCheck != nil && runBlock.PostflightCheck.Validations != nil {
						if !utils.AllConditionsMet(runBlock.PostflightCheck.Validations) {
							if runBlock.PostflightCheck.Error != nil {
								return dr.HandleAPIErrorResponse(
									runBlock.PostflightCheck.Error.Code,
									fmt.Sprintf("%s: %s", runBlock.PostflightCheck.Error.Message, res.Id))
							}

							dr.Logger.Error("Postflight check not met, failing:", res.Id)
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

	dr.Logger.Info("All resources finished processing")
	return nil
}
