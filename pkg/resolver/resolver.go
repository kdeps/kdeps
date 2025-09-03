package resolver

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/alexellis/go-execute/v2"
	"github.com/apple/pkl-go/pkl"
	"github.com/gin-gonic/gin"
	"github.com/kdeps/kartographer/graph"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/item"
	"github.com/kdeps/kdeps/pkg/ktx"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/memory"
	"github.com/kdeps/kdeps/pkg/messages"
	"github.com/kdeps/kdeps/pkg/session"
	"github.com/kdeps/kdeps/pkg/tool"
	"github.com/kdeps/kdeps/pkg/utils"
	"github.com/kdeps/kdeps/pkg/workflow"
	pklExec "github.com/kdeps/schema/gen/exec"
	pklHTTP "github.com/kdeps/schema/gen/http"
	pklLLM "github.com/kdeps/schema/gen/llm"
	pklPython "github.com/kdeps/schema/gen/python"
	pklResource "github.com/kdeps/schema/gen/resource"
	pklWf "github.com/kdeps/schema/gen/workflow"
	"github.com/spf13/afero"
	"github.com/tmc/langchaingo/llms/ollama"
)

type DependencyResolver struct {
	Fs                      afero.Fs
	Logger                  *logging.Logger
	Resources               []ResourceNodeEntry
	ResourceDependencies    map[string][]string
	DependencyGraph         []string
	VisitedPaths            map[string]bool
	Context                 context.Context // TODO: move this context into function params
	Graph                   *graph.DependencyGraph
	Environment             *environment.Environment
	Workflow                pklWf.Workflow
	Request                 *gin.Context
	MemoryReader            *memory.PklResourceReader
	MemoryDBPath            string
	SessionReader           *session.PklResourceReader
	SessionDBPath           string
	ToolReader              *tool.PklResourceReader
	ToolDBPath              string
	ItemReader              *item.PklResourceReader
	ItemDBPath              string
	DBs                     []*sql.DB // collection of DB connections used by the resolver
	AgentName               string
	RequestID               string
	RequestPklFile          string
	ResponsePklFile         string
	ResponseTargetFile      string
	ProjectDir              string
	WorkflowDir             string
	AgentDir                string
	ActionDir               string
	FilesDir                string
	DataDir                 string
	APIServerMode           bool
	AnacondaInstalled       bool
	FileRunCounter          map[string]int // Added to track run count per file
	DefaultTimeoutSec       int            // default timeout value in seconds
	CurrentResourceActionID string         // Track the currently processing resource actionID

	// Injectable helpers (overridable in tests)
	GetCurrentTimestampFn    func(string, string) (pkl.Duration, error)              `json:"-"`
	WaitForTimestampChangeFn func(string, pkl.Duration, time.Duration, string) error `json:"-"`

	// Additional injectable helpers for broader unit testing
	LoadResourceEntriesFn  func() error                                                               `json:"-"`
	LoadResourceFn         func(context.Context, string, ResourceType) (interface{}, error)           `json:"-"`
	BuildDependencyStackFn func(string, map[string]bool) []string                                     `json:"-"`
	ProcessRunBlockFn      func(ResourceNodeEntry, *pklResource.Resource, string, bool) (bool, error) `json:"-"`
	ClearItemDBFn          func() error                                                               `json:"-"`

	// Chat / HTTP injection helpers
	NewLLMFn               func(model string) (*ollama.LLM, error)                                                                                      `json:"-"`
	GenerateChatResponseFn func(context.Context, afero.Fs, *ollama.LLM, *pklLLM.ResourceChat, *tool.PklResourceReader, *logging.Logger) (string, error) `json:"-"`

	DoRequestFn func(*pklHTTP.ResourceHTTPClient) error `json:"-"`

	// Python / Conda execution injector
	ExecTaskRunnerFn func(context.Context, execute.ExecTask) (string, string, error) `json:"-"`

	// Import handling injectors
	PrependDynamicImportsFn func(string) error                              `json:"-"`
	AddPlaceholderImportsFn func(string) error                              `json:"-"`
	WalkFn                  func(afero.Fs, string, filepath.WalkFunc) error `json:"-"`
}

type ResourceNodeEntry struct {
	ActionID string `pkl:"actionID"`
	File     string `pkl:"file"`
}

// getResourceReaders returns all configured custom resource readers.
func (dr *DependencyResolver) getResourceReaders() []pkl.ResourceReader {
	readers := make([]pkl.ResourceReader, 0, 4)
	if dr.MemoryReader != nil {
		readers = append(readers, dr.MemoryReader)
	}
	if dr.SessionReader != nil {
		readers = append(readers, dr.SessionReader)
	}
	if dr.ToolReader != nil {
		readers = append(readers, dr.ToolReader)
	}
	if dr.ItemReader != nil {
		readers = append(readers, dr.ItemReader)
	}
	return readers
}

// (removed) createEvaluator: use pkg/evaluator.NewConfiguredEvaluator instead

func NewGraphResolver(fs afero.Fs, ctx context.Context, env *environment.Environment, req *gin.Context, logger *logging.Logger) (*DependencyResolver, error) {
	agentDir, graphID, actionDir, err := extractContextValues(ctx)
	if err != nil {
		return nil, err
	}

	paths, err := setupDirectoryPaths(agentDir, actionDir)
	if err != nil {
		return nil, err
	}

	if err := createDirectoriesAndFiles(fs, ctx, paths, graphID); err != nil {
		return nil, err
	}

	workflowConfiguration, err := loadWorkflowConfiguration(ctx, paths.workflowFile, logger)
	if err != nil {
		return nil, err
	}

	readers, err := initializeDatabaseReaders(fs, env, workflowConfiguration, logger)
	if err != nil {
		return nil, err
	}

	return createDependencyResolver(paths, workflowConfiguration, readers, req, logger)
}

func extractContextValues(ctx context.Context) (string, string, string, error) {
	var agentDir, graphID, actionDir string

	contextKeys := map[*string]ktx.ContextKey{
		&agentDir:  ktx.CtxKeyAgentDir,
		&graphID:   ktx.CtxKeyGraphID,
		&actionDir: ktx.CtxKeyActionDir,
	}

	for ptr, key := range contextKeys {
		if value, found := ktx.ReadContext(ctx, key); found {
			if strValue, ok := value.(string); ok {
				*ptr = strValue
			}
		}
	}

	if agentDir == "" || graphID == "" || actionDir == "" {
		return "", "", "", fmt.Errorf("missing required context values: agentDir=%s, graphID=%s, actionDir=%s", agentDir, graphID, actionDir)
	}

	return agentDir, graphID, actionDir, nil
}

type directoryPaths struct {
	workflowDir        string
	projectDir         string
	workflowFile       string
	dataDir            string
	filesDir           string
	requestPklFile     string
	responsePklFile    string
	responseTargetFile string
}

func setupDirectoryPaths(agentDir, actionDir string) (*directoryPaths, error) {
	workflowDir := filepath.Join(agentDir, "/workflow/")
	projectDir := filepath.Join(agentDir, "/project/")
	pklWfFile := filepath.Join(workflowDir, "workflow.pkl")

	dataDir := filepath.Join(projectDir, "/data/")
	filesDir := filepath.Join(actionDir, "/files/")

	requestPklFile := filepath.Join(actionDir, "/api/graphID__request.pkl")
	responsePklFile := filepath.Join(actionDir, "/api/graphID__response.pkl")
	responseTargetFile := filepath.Join(actionDir, "/api/graphID__response.json")

	return &directoryPaths{
		workflowDir:        workflowDir,
		projectDir:         projectDir,
		workflowFile:       pklWfFile,
		dataDir:            dataDir,
		filesDir:           filesDir,
		requestPklFile:     requestPklFile,
		responsePklFile:    responsePklFile,
		responseTargetFile: responseTargetFile,
	}, nil
}

func createDirectoriesAndFiles(fs afero.Fs, ctx context.Context, paths *directoryPaths, graphID string) error {
	directories := []string{
		paths.projectDir,
		paths.filesDir,
	}

	if err := utils.CreateDirectories(fs, ctx, directories); err != nil {
		return fmt.Errorf("error creating directory: %w", err)
	}

	files := []string{
		filepath.Join(paths.filesDir, graphID),
	}

	if err := utils.CreateFiles(fs, ctx, files); err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}

	return nil
}

func loadWorkflowConfiguration(ctx context.Context, workflowFile string, logger *logging.Logger) (pklWf.Workflow, error) {
	exists, err := afero.Exists(afero.NewOsFs(), workflowFile)
	if err != nil || !exists {
		return nil, fmt.Errorf("error checking %s: %w", workflowFile, err)
	}

	return workflow.LoadWorkflow(ctx, workflowFile, logger)
}

func initializeDatabaseReaders(fs afero.Fs, env *environment.Environment, workflowConfiguration pklWf.Workflow, logger *logging.Logger) (*databaseReaders, error) {
	agentName := workflowConfiguration.GetAgentID()

	kdepsBase := getKdepsBasePath(env)
	if err := fs.MkdirAll(kdepsBase, 0o777); err != nil {
		return nil, fmt.Errorf("failed to create kdeps base directory %s: %w", kdepsBase, err)
	}

	memoryReader, err := initializeMemoryReader(kdepsBase, agentName)
	if err != nil {
		return nil, err
	}

	sessionReader, err := initializeSessionReader()
	if err != nil {
		memoryReader.DB.Close()
		return nil, err
	}

	toolReader, err := initializeToolReader()
	if err != nil {
		memoryReader.DB.Close()
		sessionReader.DB.Close()
		return nil, err
	}

	itemReader, err := initializeItemReader()
	if err != nil {
		memoryReader.DB.Close()
		sessionReader.DB.Close()
		toolReader.DB.Close()
		return nil, err
	}

	return &databaseReaders{
		memoryReader:  memoryReader,
		sessionReader: sessionReader,
		toolReader:    toolReader,
		itemReader:    itemReader,
	}, nil
}

type databaseReaders struct {
	memoryReader  *memory.PklResourceReader
	sessionReader *session.PklResourceReader
	toolReader    *tool.PklResourceReader
	itemReader    *item.PklResourceReader
}

func getKdepsBasePath(env *environment.Environment) string {
	if kdepsBase := os.Getenv("KDEPS_VOLUME_PATH"); kdepsBase != "" {
		return kdepsBase
	}

	if env != nil && env.DockerMode == "1" {
		return "/agent/volume/"
	}
	return "/.kdeps/"
}

func initializeMemoryReader(kdepsBase, agentName string) (*memory.PklResourceReader, error) {
	memoryDBPath := filepath.Join(kdepsBase, agentName+"_memory.db")
	return memory.InitializeMemory(memoryDBPath)
}

func initializeSessionReader() (*session.PklResourceReader, error) {
	sessionDBPath := ":memory:"
	return session.InitializeSession(sessionDBPath)
}

func initializeToolReader() (*tool.PklResourceReader, error) {
	toolDBPath := ":memory:"
	return tool.InitializeTool(toolDBPath)
}

func initializeItemReader() (*item.PklResourceReader, error) {
	itemDBPath := ":memory:"
	return item.InitializeItem(itemDBPath, []string{})
}

func createDependencyResolver(paths *directoryPaths, workflowConfiguration pklWf.Workflow, readers *databaseReaders, req *gin.Context, logger *logging.Logger) (*DependencyResolver, error) {
	settings := workflowConfiguration.GetSettings()
	agentName := workflowConfiguration.GetAgentID()

	return &DependencyResolver{
		Fs:                afero.NewOsFs(),
		Workflow:          workflowConfiguration,
		Logger:            logger,
		AgentName:         agentName,
		DataDir:           paths.dataDir,
		ActionDir:         paths.filesDir,
		RequestID:         "graphID", // This should be passed in
		Request:           req,
		APIServerMode:     settings.APIServerMode,
		AnacondaInstalled: settings.AgentSettings.InstallAnaconda,
		MemoryReader:      readers.memoryReader,
		SessionReader:     readers.sessionReader,
		ToolReader:        readers.toolReader,
		ItemReader:        readers.itemReader,
		FileRunCounter:    make(map[string]int),
	}, nil
}

// ClearItemDB clears all contents of the item database.
func (dr *DependencyResolver) ClearItemDB() error {
	// Clear all records in the items table
	_, err := dr.ItemReader.DB.Exec("DELETE FROM items")
	if err != nil {
		return fmt.Errorf("failed to clear item database: %w", err)
	}
	dr.Logger.Info("cleared item database", "path", dr.ItemDBPath)
	return nil
}

// processResourceStep consolidates the pattern of: get timestamp, run a handler, adjust timeout (if provided),
// then wait for the timestamp change.
func (dr *DependencyResolver) processResourceStep(resourceID, step string, timeoutPtr *pkl.Duration, handler func() error) error {
	timestamp, err := dr.GetCurrentTimestampFn(resourceID, step)
	if err != nil {
		return fmt.Errorf("%s error: %w", step, err)
	}

	var timeout time.Duration
	switch {
	case dr.DefaultTimeoutSec > 0: // positive value overrides everything
		timeout = time.Duration(dr.DefaultTimeoutSec) * time.Second
	case dr.DefaultTimeoutSec == 0: // 0 => unlimited
		timeout = 0
	case timeoutPtr != nil: // negative or unset – fall back to resource value
		timeout = timeoutPtr.GoDuration()
	default:
		timeout = 60 * time.Second
	}

	if err := handler(); err != nil {
		return fmt.Errorf("%s error: %w", step, err)
	}

	if err := dr.WaitForTimestampChangeFn(resourceID, timestamp, timeout, step); err != nil {
		return fmt.Errorf("%s timeout awaiting for output: %w", step, err)
	}
	return nil
}

// validateRequestParams checks if params in request.params("header_id") are in AllowedParams.
func (dr *DependencyResolver) validateRequestParams(file string, allowedParams []string) error {
	if len(allowedParams) == 0 {
		return nil // Allow all if empty
	}

	// Split file into lines to check each line individually
	lines := strings.Split(file, "\n")
	re := regexp.MustCompile(`request\.params\("([^"]+)"\)`)

	for _, line := range lines {
		// Skip commented lines (lines that start with // after any whitespace)
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "//") {
			continue
		}

		// Find matches in non-commented lines only
		matches := re.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			param := match[1]
			if !utils.ContainsStringInsensitive(allowedParams, param) {
				return fmt.Errorf("param %s not in the allowed params", param)
			}
		}
	}
	return nil
}

// validateRequestHeaders checks if headers in request.header("header_id") are in AllowedHeaders.
func (dr *DependencyResolver) validateRequestHeaders(file string, allowedHeaders []string) error {
	if len(allowedHeaders) == 0 {
		return nil // Allow all if empty
	}

	// Split file into lines to check each line individually
	lines := strings.Split(file, "\n")
	re := regexp.MustCompile(`request\.header\("([^"]+)"\)`)

	for _, line := range lines {
		// Skip commented lines (lines that start with // after any whitespace)
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "//") {
			continue
		}

		// Find matches in non-commented lines only
		matches := re.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			header := match[1]
			if !utils.ContainsStringInsensitive(allowedHeaders, header) {
				return fmt.Errorf("header %s not in the allowed headers", header)
			}
		}
	}
	return nil
}

// validateRequestPath checks if the actual request path is in AllowedRoutes.
func (dr *DependencyResolver) validateRequestPath(req *gin.Context, allowedRoutes []string) error {
	if len(allowedRoutes) == 0 {
		return nil // Allow all if empty
	}

	actualPath := req.Request.URL.Path
	if !utils.ContainsStringInsensitive(allowedRoutes, actualPath) {
		return fmt.Errorf("path %s not in the allowed routes", actualPath)
	}
	return nil
}

// validateRequestMethod checks if the actual request method is in AllowedHTTPMethods.
func (dr *DependencyResolver) validateRequestMethod(req *gin.Context, allowedMethods []string) error {
	if len(allowedMethods) == 0 {
		return nil // Allow all if empty
	}

	actualMethod := req.Request.Method
	if !utils.ContainsStringInsensitive(allowedMethods, actualMethod) {
		return fmt.Errorf("method %s not in the allowed HTTP methods", actualMethod)
	}
	return nil
}

// HandleRunAction is the main entry point to process resource run blocks.
func (dr *DependencyResolver) HandleRunAction() (bool, error) {
	defer dr.setupPanicRecovery()

	requestFilePath := filepath.Join(dr.ActionDir, dr.RequestID)

	if err := dr.initializeAndProcessResources(); err != nil {
		return false, err
	}

	dr.finalizeProcessing(requestFilePath)
	return false, nil
}

func (dr *DependencyResolver) setupPanicRecovery() {
	defer func() {
		if r := recover(); r != nil {
			dr.Logger.Error("panic recovered in HandleRunAction", "panic", r)
			dr.closeAllDatabases()
			dr.cleanupSessionFiles()
			dr.logPanicStackTrace()
		}
	}()
}

func (dr *DependencyResolver) closeAllDatabases() {
	dr.MemoryReader.DB.Close()
	dr.SessionReader.DB.Close()
	dr.ToolReader.DB.Close()
	dr.ItemReader.DB.Close()
}

func (dr *DependencyResolver) cleanupSessionFiles() {
	if err := dr.Fs.RemoveAll(dr.SessionDBPath); err != nil {
		dr.Logger.Error("failed to delete the SessionDB file", "file", dr.SessionDBPath, "error", err)
	}
}

func (dr *DependencyResolver) logPanicStackTrace() {
	buf := make([]byte, 1<<16)
	stackSize := runtime.Stack(buf, false)
	dr.Logger.Error("stack trace", "stack", string(buf[:stackSize]))
}

func (dr *DependencyResolver) initializeAndProcessResources() error {
	visited := make(map[string]bool)
	targetActionID := dr.Workflow.GetTargetActionID()
	dr.Logger.Debug(messages.MsgProcessingResources)

	if err := dr.LoadResourceEntriesFn(); err != nil {
		_, apiErr := dr.HandleAPIErrorResponse(500, err.Error(), true)
		return apiErr
	}

	// Build dependency stack for the target action
	stack := dr.BuildDependencyStackFn(targetActionID, visited)

	// Process each resource in the dependency stack
	for _, nodeActionID := range stack {
		if err := dr.processResourceByActionID(nodeActionID); err != nil {
			return err
		}
	}
	return nil
}

func (dr *DependencyResolver) processResourceByActionID(nodeActionID string) error {
	for _, res := range dr.Resources {
		if res.ActionID != nodeActionID {
			continue
		}

		// Set the current resource actionID for error context
		dr.CurrentResourceActionID = res.ActionID

		// Load the resource with robust fallback
		resPkl, err := dr.loadResourceWithFallbackResolver(res.File)
		if err != nil {
			dr.Logger.Error("failed to load resource with fallback", "file", res.File, "error", err)
			_, apiErr := dr.HandleAPIErrorResponse(500, fmt.Sprintf("failed to load resource %s: %v", res.File, err), true)
			return apiErr
		}

		// Robustly cast to pklResource.Resource
		rsc, err := dr.castToResource(resPkl, res.File)
		if err != nil {
			dr.Logger.Error("failed to cast resource", "file", res.File, "error", err)
			_, apiErr := dr.HandleAPIErrorResponse(500, err.Error(), true)
			return apiErr
		}

		if err := dr.processResourceWithItems(res, rsc, nodeActionID); err != nil {
			return err
		}
	}
	return nil
}

func (dr *DependencyResolver) processResourceWithItems(res ResourceNodeEntry, rsc *pklResource.Resource, nodeActionID string) error {
	// Reinitialize item database with items, if any
	var items []string
	if rsc.Items != nil && len(*rsc.Items) > 0 {
		items = *rsc.Items
		if err := dr.reinitializeItemDatabase(items, nodeActionID); err != nil {
			return err
		}
	}

	// Process run block: once if no items, or once per item
	if len(items) == 0 {
		return dr.processRunBlockOnce(res, rsc, nodeActionID)
	}
	return dr.processRunBlockWithItems(res, rsc, items, nodeActionID)
}

func (dr *DependencyResolver) reinitializeItemDatabase(items []string, actionID string) error {
	dr.ItemReader.DB.Close()
	itemReader, err := item.InitializeItem(dr.ItemDBPath, items)
	if err != nil {
		_, apiErr := dr.HandleAPIErrorResponse(500, fmt.Sprintf("failed to reinitialize item DB with items: %v", err), true)
		return apiErr
	}
	dr.ItemReader = itemReader
	dr.Logger.Info("reinitialized item database with items", "actionID", actionID, "itemCount", len(items))
	return nil
}

func (dr *DependencyResolver) processRunBlockOnce(res ResourceNodeEntry, rsc *pklResource.Resource, nodeActionID string) error {
	dr.Logger.Info("no items specified, processing run block once", "actionID", res.ActionID)
	proceed, err := dr.ProcessRunBlockFn(res, rsc, nodeActionID, false)
	if err != nil {
		return err
	}
	_ = proceed // For resources with no items, we still want to process APIResponse
	return nil
}

func (dr *DependencyResolver) processRunBlockWithItems(res ResourceNodeEntry, rsc *pklResource.Resource, items []string, nodeActionID string) error {
	for _, itemValue := range items {
		if err := dr.processItemInResource(res, rsc, itemValue, nodeActionID); err != nil {
			return err
		}
	}

	if err := dr.ClearItemDBFn(); err != nil {
		dr.Logger.Error("failed to clear item database after iteration", "actionID", res.ActionID, "error", err)
		_, apiErr := dr.HandleAPIErrorResponse(500, fmt.Sprintf("failed to clear item database for resource %s: %v", res.ActionID, err), true)
		return apiErr
	}
	return nil
}

func (dr *DependencyResolver) processItemInResource(res ResourceNodeEntry, rsc *pklResource.Resource, itemValue, nodeActionID string) error {
	dr.Logger.Info("processing item", "actionID", res.ActionID, "item", itemValue)

	// Set the current item in the database
	query := url.Values{"op": []string{"set"}, "value": []string{itemValue}}
	uri := url.URL{Scheme: "item", RawQuery: query.Encode()}
	if _, err := dr.ItemReader.Read(uri); err != nil {
		dr.Logger.Error("failed to set item", "item", itemValue, "error", err)
		_, apiErr := dr.HandleAPIErrorResponse(500, fmt.Sprintf("failed to set item %s: %v", itemValue, err), true)
		return apiErr
	}

	// reload the resource with robust fallback
	resPkl, err := dr.loadResourceWithFallbackResolver(res.File)
	if err != nil {
		dr.Logger.Error("failed to reload resource with fallback", "file", res.File, "error", err)
		_, apiErr := dr.HandleAPIErrorResponse(500, fmt.Sprintf("failed to reload resource %s: %v", res.File, err), true)
		return apiErr
	}

	rsc, err = dr.castToResource(resPkl, res.File)
	if err != nil {
		dr.Logger.Error("failed to cast reloaded resource", "file", res.File, "error", err)
		_, apiErr := dr.HandleAPIErrorResponse(500, err.Error(), true)
		return apiErr
	}

	// Process runBlock for the current item
	_, err = dr.ProcessRunBlockFn(res, rsc, nodeActionID, true)
	return err
}

func (dr *DependencyResolver) finalizeProcessing(requestFilePath string) {
	dr.closeAllDatabases()
	dr.logFinalRunCounts()
	dr.Logger.Debug(messages.MsgAllResourcesProcessed)
}

func (dr *DependencyResolver) logFinalRunCounts() {
	for file, count := range dr.FileRunCounter {
		dr.Logger.Info("file run count", "file", file, "count", count)
	}
}

func (dr *DependencyResolver) waitForItemsDatabase(actionID string) error {
	const waitTimeout = 30 * time.Second
	const pollInterval = 500 * time.Millisecond
	deadline := time.Now().Add(waitTimeout)

	dr.Logger.Info("Waiting for items database to have a non-empty list", "actionID", actionID)
	for time.Now().Before(deadline) {
		query := url.Values{"op": []string{"list"}}
		uri := url.URL{Scheme: "item", RawQuery: query.Encode()}
		result, err := dr.ItemReader.Read(uri)
		if err != nil {
			dr.Logger.Error("Failed to read list from items database", "actionID", actionID, "error", err)
			_, apiErr := dr.HandleAPIErrorResponse(500, fmt.Sprintf("Failed to read list from items database for resource %s: %v", actionID, err), true)
			return apiErr
		}

		var items []string
		if len(result) > 0 {
			if err := json.Unmarshal(result, &items); err != nil {
				dr.Logger.Error("Failed to parse items database result as JSON array", "actionID", actionID, "error", err)
				_, apiErr := dr.HandleAPIErrorResponse(500, fmt.Sprintf("Failed to parse items database result for resource %s: %v", actionID, err), true)
				return apiErr
			}
		}

		if len(items) > 0 {
			dr.Logger.Info("Items database has a non-empty list", "actionID", actionID, "itemCount", len(items))
			return nil
		}

		dr.Logger.Debug(messages.MsgItemsDBEmptyRetry, "actionID", actionID)
		time.Sleep(pollInterval)
	}

	dr.Logger.Error("Timeout waiting for items database to have a non-empty list", "actionID", actionID)
	_, apiErr := dr.HandleAPIErrorResponse(500, "Timeout waiting for items database to have a non-empty list for resource "+actionID, true)
	return apiErr
}

func (dr *DependencyResolver) validateAPIServerMode(res ResourceNodeEntry, runBlock interface{}) (bool, error) {
	// Read the resource file content for validation
	fileContent, err := afero.ReadFile(dr.Fs, res.File)
	if err != nil {
		_, apiErr := dr.HandleAPIErrorResponse(500, fmt.Sprintf("failed to read resource file %s: %v", res.File, err), true)
		return false, apiErr
	}

	// Use reflection to access fields
	rv := reflect.ValueOf(runBlock)
	if rv.Kind() != reflect.Ptr && rv.Kind() != reflect.Interface {
		rv = rv.Addr()
	}
	if rv.Kind() == reflect.Interface {
		rv = rv.Elem()
	}

	// Validate request.params
	if allowedParamsField := rv.FieldByName("AllowedParams"); allowedParamsField.IsValid() {
		allowedParams := []string{}
		if !allowedParamsField.IsNil() {
			allowedParams = allowedParamsField.Elem().Interface().([]string)
		}
		if err := dr.validateRequestParams(string(fileContent), allowedParams); err != nil {
			dr.Logger.Error("request params validation failed", "actionID", res.ActionID, "error", err)
			_, apiErr := dr.HandleAPIErrorResponse(400, fmt.Sprintf("Request params validation failed for resource %s: %v", res.ActionID, err), true)
			return false, apiErr
		}
	}

	// Validate request.header
	if allowedHeadersField := rv.FieldByName("AllowedHeaders"); allowedHeadersField.IsValid() {
		allowedHeaders := []string{}
		if !allowedHeadersField.IsNil() {
			allowedHeaders = allowedHeadersField.Elem().Interface().([]string)
		}
		if err := dr.validateRequestHeaders(string(fileContent), allowedHeaders); err != nil {
			dr.Logger.Error("request headers validation failed", "actionID", res.ActionID, "error", err)
			_, apiErr := dr.HandleAPIErrorResponse(400, fmt.Sprintf("Request headers validation failed for resource %s: %v", res.ActionID, err), true)
			return false, apiErr
		}
	}

	// Validate request.path
	if restrictToRoutesField := rv.FieldByName("RestrictToRoutes"); restrictToRoutesField.IsValid() {
		allowedRoutes := []string{}
		if !restrictToRoutesField.IsNil() {
			allowedRoutes = restrictToRoutesField.Elem().Interface().([]string)
		}
		if err := dr.validateRequestPath(dr.Request, allowedRoutes); err != nil {
			dr.Logger.Info("skipping due to request path validation not allowed", "actionID", res.ActionID, "error", err)
			return true, nil
		}
	}

	// Validate request.method
	if restrictToHTTPMethodsField := rv.FieldByName("RestrictToHTTPMethods"); restrictToHTTPMethodsField.IsValid() {
		allowedMethods := []string{}
		if !restrictToHTTPMethodsField.IsNil() {
			allowedMethods = restrictToHTTPMethodsField.Elem().Interface().([]string)
		}
		if err := dr.validateRequestMethod(dr.Request, allowedMethods); err != nil {
			dr.Logger.Info("skipping due to request method validation not allowed", "actionID", res.ActionID, "error", err)
			return true, nil
		}
	}

	return false, nil
}

// processRunBlock handles the runBlock processing for a resource, excluding APIResponse.
func (dr *DependencyResolver) processRunBlock(res ResourceNodeEntry, rsc *pklResource.Resource, actionID string, hasItems bool) (bool, error) {
	dr.incrementRunCounter(res, actionID)

	runBlock := rsc.Run

	if err := dr.handlePrerequisites(res, runBlock, actionID, hasItems); err != nil {
		return false, err
	}

	if dr.shouldSkipRunBlock(res, runBlock) {
		return false, nil
	}

	if err := dr.validatePreflightCheck(res, runBlock); err != nil {
		return false, err
	}

	if dr.shouldSkipExpensiveOperations(res) {
		return true, nil
	}

	return dr.processAllRunBlockSteps(res, runBlock)
}

// loadResourceWithFallbackResolver tries to load a resource file with different resource types as fallback.
func (dr *DependencyResolver) loadResourceWithFallbackResolver(file string) (interface{}, error) {
	resourceTypes := []ResourceType{Resource, LLMResource, HTTPResource, PythonResource, ExecResource}

	for _, resourceType := range resourceTypes {
		res, err := dr.LoadResourceFn(dr.Context, file, resourceType)
		if err != nil {
			dr.Logger.Debug("failed to load resource with type", "file", file, "type", resourceType, "error", err)
			continue
		}

		dr.Logger.Debug("successfully loaded resource", "file", file, "type", resourceType)

		// If we successfully loaded as a specific resource type, try to convert it to Resource type
		if resourceType != Resource {
			// Try to load the same file as Resource type
			resourceRes, err := dr.LoadResourceFn(dr.Context, file, Resource)
			if err != nil {
				dr.Logger.Debug("failed to convert resource to Resource type", "file", file, "originalType", resourceType, "error", err)
				// Continue with the original loaded resource if conversion fails
			} else {
				return resourceRes, nil
			}
		}

		return res, nil
	}

	return nil, fmt.Errorf("failed to load resource with any type for file %s", file)
}

// castToResource robustly casts a loaded resource to pklResource.Resource
func (dr *DependencyResolver) incrementRunCounter(res ResourceNodeEntry, actionID string) {
	dr.FileRunCounter[res.File]++
	dr.Logger.Info("processing run block for file", "file", res.File, "runCount", dr.FileRunCounter[res.File], "actionID", actionID)
}

func (dr *DependencyResolver) handlePrerequisites(res ResourceNodeEntry, runBlock pklResource.ResourceAction, actionID string, hasItems bool) error {
	if hasItems {
		if err := dr.waitForItemsDatabase(actionID); err != nil {
			return err
		}
	}

	if dr.APIServerMode {
		if shouldSkip, err := dr.validateAPIServerMode(res, runBlock); err != nil {
			return err
		} else if shouldSkip {
			return fmt.Errorf("api server mode validation failed")
		}
	}

	return nil
}

func (dr *DependencyResolver) shouldSkipRunBlock(res ResourceNodeEntry, runBlock pklResource.ResourceAction) bool {
	if runBlock.SkipCondition != nil && utils.ShouldSkip(runBlock.SkipCondition) {
		dr.Logger.Infof("skip condition met, skipping: %s", res.ActionID)
		return true
	}
	return false
}

func (dr *DependencyResolver) validatePreflightCheck(res ResourceNodeEntry, runBlock pklResource.ResourceAction) error {
	if runBlock.PreflightCheck == nil || runBlock.PreflightCheck.Validations == nil {
		return nil
	}

	conditionsMet, failedConditions := utils.AllConditionsMetWithDetails(runBlock.PreflightCheck.Validations)
	if !conditionsMet {
		dr.Logger.Error("preflight check not met, collecting error and continuing to gather all errors:", res.ActionID, "failedConditions", failedConditions)

		errorMessage := dr.buildPreflightErrorMessage(res, runBlock, failedConditions)

		// Collect error but continue processing to gather ALL errors
		if runBlock.PreflightCheck.Error != nil {
			if _, err := dr.HandleAPIErrorResponse(runBlock.PreflightCheck.Error.Code, errorMessage, false); err != nil {
				dr.Logger.Error("failed to handle API error response", "error", err)
			}
		} else {
			if _, err := dr.HandleAPIErrorResponse(500, errorMessage, false); err != nil {
				dr.Logger.Error("failed to handle API error response", "error", err)
			}
		}
	}
	return nil
}

func (dr *DependencyResolver) buildPreflightErrorMessage(res ResourceNodeEntry, runBlock pklResource.ResourceAction, failedConditions []string) string {
	var errorMessage string
	if runBlock.PreflightCheck.Error != nil && runBlock.PreflightCheck.Error.Message != "" {
		errorMessage = runBlock.PreflightCheck.Error.Message
	} else {
		errorMessage = fmt.Sprintf("Validation failed for %s", res.ActionID)
	}

	if len(failedConditions) > 0 {
		if len(failedConditions) == 1 {
			errorMessage += fmt.Sprintf(" (%s)", failedConditions[0])
		} else {
			errorMessage += fmt.Sprintf(" (%s)", strings.Join(failedConditions, ", "))
		}
	}

	return errorMessage
}

func (dr *DependencyResolver) shouldSkipExpensiveOperations(res ResourceNodeEntry) bool {
	existingErrorsWithID := utils.GetRequestErrorsWithActionID(dr.RequestID)
	if len(existingErrorsWithID) > 0 {
		dr.Logger.Info("errors already accumulated, skipping expensive operations for fail-fast behavior", "actionID", res.ActionID, "errorCount", len(existingErrorsWithID))
		return true
	}
	return false
}

func (dr *DependencyResolver) processAllRunBlockSteps(res ResourceNodeEntry, runBlock pklResource.ResourceAction) (bool, error) {
	// Process Exec step, if defined
	if runBlock.Exec != nil && runBlock.Exec.Command != "" {
		if err := dr.processResourceStep(res.ActionID, "exec", runBlock.Exec.TimeoutDuration, func() error {
			return dr.HandleExec(res.ActionID, runBlock.Exec)
		}); err != nil {
			dr.Logger.Error("exec error:", res.ActionID)
			return dr.HandleAPIErrorResponse(500, fmt.Sprintf("Exec failed for resource: %s - %s", res.ActionID, err), true)
		}
	}

	// Process Python step, if defined
	if runBlock.Python != nil && runBlock.Python.Script != "" {
		if err := dr.processResourceStep(res.ActionID, "python", runBlock.Python.TimeoutDuration, func() error {
			return dr.HandlePython(res.ActionID, runBlock.Python)
		}); err != nil {
			dr.Logger.Error("python error:", res.ActionID)
			return dr.HandleAPIErrorResponse(500, fmt.Sprintf("Python script failed for resource: %s - %s", res.ActionID, err), true)
		}
	}

	// Process Chat (LLM) step, if defined
	if runBlock.Chat != nil && runBlock.Chat.Model != "" && (runBlock.Chat.Prompt != nil || runBlock.Chat.Scenario != nil) {
		dr.Logger.Info("Processing LLM chat step", "actionID", res.ActionID, "hasPrompt", runBlock.Chat.Prompt != nil, "hasScenario", runBlock.Chat.Scenario != nil)
		if runBlock.Chat.Scenario != nil {
			dr.Logger.Info("Scenario present", "length", len(*runBlock.Chat.Scenario))
		}
		if err := dr.processResourceStep(res.ActionID, "llm", runBlock.Chat.TimeoutDuration, func() error {
			return dr.HandleLLMChat(res.ActionID, runBlock.Chat)
		}); err != nil {
			dr.Logger.Error("LLM chat error", "actionID", res.ActionID, "error", err)
			return dr.HandleAPIErrorResponse(500, fmt.Sprintf("LLM chat failed for resource: %s - %s", res.ActionID, err), true)
		}
	} else {
		dr.Logger.Info("Skipping LLM chat step", "actionID", res.ActionID, "chatNil",
			runBlock.Chat == nil, "modelEmpty", runBlock.Chat == nil || runBlock.Chat.Model == "",
			"promptAndScenarioNil", runBlock.Chat != nil && runBlock.Chat.Prompt == nil &&
				runBlock.Chat.Scenario == nil)
	}

	// Process HTTP Client step, if defined
	if runBlock.HTTPClient != nil && runBlock.HTTPClient.Method != "" && runBlock.HTTPClient.Url != "" {
		if err := dr.processResourceStep(res.ActionID, "client", runBlock.HTTPClient.TimeoutDuration, func() error {
			return dr.HandleHTTPClient(res.ActionID, runBlock.HTTPClient)
		}); err != nil {
			dr.Logger.Error("HTTP client error:", res.ActionID)
			return dr.HandleAPIErrorResponse(500, fmt.Sprintf("HTTP client failed for resource: %s - %s", res.ActionID, err), true)
		}
	}

	// Check if any action was actually performed
	hasExec := runBlock.Exec != nil && runBlock.Exec.Command != ""
	hasPython := runBlock.Python != nil && runBlock.Python.Script != ""
	hasChat := runBlock.Chat != nil && runBlock.Chat.Model != "" && (runBlock.Chat.Prompt != nil || runBlock.Chat.Scenario != nil)
	hasHTTP := runBlock.HTTPClient != nil && runBlock.HTTPClient.Method != "" && runBlock.HTTPClient.Url != ""

	// If no actions were performed, return false to indicate no processing occurred
	if !hasExec && !hasPython && !hasChat && !hasHTTP {
		dr.Logger.Debug("No run actions defined, skipping resource processing", "actionID", res.ActionID)
		return false, nil
	}

	return true, nil
}

func (dr *DependencyResolver) castToResource(res interface{}, file string) (*pklResource.Resource, error) {
	// Try direct pointer cast first
	if ptr, ok := res.(*pklResource.Resource); ok {
		return ptr, nil
	}

	// Try value cast
	if resource, ok := res.(pklResource.Resource); ok {
		return &resource, nil
	}

	// Check if we loaded a specific resource type instead of Resource
	if _, ok := res.(*pklLLM.LLMImpl); ok {
		dr.Logger.Warn("loaded LLM resource as specific type, this may indicate a schema issue", "file", file)
		return nil, fmt.Errorf("resource loaded as LLM type but expected Resource type for file %s", file)
	}

	if _, ok := res.(*pklHTTP.HTTPImpl); ok {
		dr.Logger.Warn("loaded HTTP resource as specific type, this may indicate a schema issue", "file", file)
		return nil, fmt.Errorf("resource loaded as HTTP type but expected Resource type for file %s", file)
	}

	if _, ok := res.(*pklPython.PythonImpl); ok {
		dr.Logger.Warn("loaded Python resource as specific type, this may indicate a schema issue", "file", file)
		return nil, fmt.Errorf("resource loaded as Python type but expected Resource type for file %s", file)
	}

	if _, ok := res.(*pklExec.ExecImpl); ok {
		dr.Logger.Warn("loaded Exec resource as specific type, this may indicate a schema issue", "file", file)
		return nil, fmt.Errorf("resource loaded as Exec type but expected Resource type for file %s", file)
	}

	return nil, fmt.Errorf("failed to cast resource to pklResource.Resource for file %s (actual type: %T)", file, res)
}
