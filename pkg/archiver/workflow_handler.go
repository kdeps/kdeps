package archiver

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kdeps/kdeps/pkg/agent"
	"github.com/kdeps/kdeps/pkg/enforcer"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/evaluator"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/messages"
	"github.com/kdeps/kdeps/pkg/utils"
	"github.com/kdeps/kdeps/pkg/version"
	"github.com/kdeps/kdeps/pkg/workflow"
	pklWf "github.com/kdeps/schema/gen/workflow"
	"github.com/spf13/afero"
)

func PrepareRunDir(fs afero.Fs, ctx context.Context, wf pklWf.Workflow, kdepsDir, pkgFilePath string, logger *logging.Logger) (string, error) {
	agentName, agentVersion := wf.GetAgentID(), wf.GetVersion()
	runDir := filepath.Join(kdepsDir, "run/"+agentName+"/"+agentVersion+"/workflow")

	if exists, err := afero.Exists(fs, runDir); err != nil {
		return "", err
	} else if exists {
		if err := fs.RemoveAll(runDir); err != nil {
			return "", err
		}
	}

	if err := fs.MkdirAll(runDir, 0o755); err != nil {
		return "", err
	}

	file, err := os.Open(pkgFilePath)
	if err != nil {
		logger.Error("error opening file: %v\n", err)
		return "", err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		logger.Error("error creating gzip reader: %v\n", err)
		return "", err
	}
	defer gzr.Close()

	tarReader := tar.NewReader(gzr)

	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			logger.Error("error reading tar file: %v\n", err)
			return "", err
		}

		target, err := utils.SanitizeArchivePath(runDir, header.Name)
		if err != nil {
			return "", err
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := fs.MkdirAll(target, 0o755); err != nil {
				logger.Error("error creating directory: %v\n", err)
				return "", err
			}
		case tar.TypeReg:
			dir := filepath.Dir(target)
			if err := fs.MkdirAll(dir, 0o755); err != nil {
				logger.Error("error creating file directory: %v\n", err)
				return "", err
			}

			outFile, err := fs.Create(target)
			if err != nil {
				logger.Error("error creating file: %v\n", err)
				return "", err
			}

			for {
				_, err := io.CopyN(outFile, tarReader, 1024)
				if err != nil {
					if errors.Is(err, io.EOF) {
						break
					}
					logger.Error("error writing file: %v\n", err)
					return "", fmt.Errorf("failed to copy file: %w", err)
				}
			}
			outFile.Close()
		default:
			logger.Error("unknown type: %v in %s\n", header.Typeflag, header.Name)
		}
	}

	logger.Debug(messages.MsgExtractionRuntimeDone, runDir)
	return runDir, nil
}

func CompileWorkflow(ctx context.Context, fs afero.Fs, wf pklWf.Workflow, kdepsDir, projectDir string, logger *logging.Logger) (string, error) {
	action := wf.GetTargetActionID()
	if action == "" {
		return "", errors.New("please specify the default action in the workflow")
	}

	// Resolve the actionID canonically using the agent reader
	agentReader, err := agent.GetGlobalAgentReader(fs, "", wf.GetAgentID(), wf.GetVersion(), logger)
	if err != nil {
		return "", fmt.Errorf("failed to initialize agent reader: %w", err)
	}

	compiledAction := resolveActionIDCanonically(action, wf, agentReader)

	name, version := wf.GetAgentID(), wf.GetVersion()
	agentDir := filepath.Join(kdepsDir, fmt.Sprintf("agents/%s/%s", name, version))
	resourcesDir := filepath.Join(agentDir, "resources")
	compiledFilePath := filepath.Join(agentDir, "workflow.pkl")

	if exists, err := afero.DirExists(fs, agentDir); err != nil {
		logger.Error("error checking agent directory", "path", agentDir, "error", err)
		return "", err
	} else if exists {
		if err := fs.RemoveAll(agentDir); err != nil {
			logger.Error(messages.MsgRemovedAgentDirectory, "path", agentDir, "error", err)
			return "", err
		}
		logger.Debug(messages.MsgRemovedAgentDirectory, "path", agentDir)
	}

	if err := fs.MkdirAll(resourcesDir, 0o755); err != nil {
		logger.Error("failed to create resources directory", "path", resourcesDir, "error", err)
		return "", err
	}

	content, err := afero.ReadFile(fs, filepath.Join(projectDir, "workflow.pkl"))
	if err != nil {
		logger.Error("failed to read workflow file", "error", err)
		return "", err
	}

	// Replace TargetActionID using a more robust approach
	updatedContent, err := replaceTargetActionID(string(content), compiledAction)
	if err != nil {
		logger.Error("failed to replace TargetActionID", "error", err)
		return "", err
	}

	if err := afero.WriteFile(fs, compiledFilePath, []byte(updatedContent), 0o644); err != nil {
		logger.Error("failed to write compiled workflow", "path", compiledFilePath, "error", err)
		return "", err
	}

	if err := enforcer.EnforcePklTemplateAmendsRules(ctx, fs, compiledFilePath, logger); err != nil {
		logger.Error("validation failed for .pkl file", "file", compiledFilePath, "error", err)
		return "", err
	}

	return filepath.Dir(compiledFilePath), nil
}

// resolveActionIDCanonically resolves an actionID to its canonical form using the agent reader
func resolveActionIDCanonically(actionID string, wf pklWf.Workflow, agentReader *agent.PklResourceReader) string {
	// If the actionID is already in canonical form (@agent/action:version), return it
	if strings.HasPrefix(actionID, "@") {
		return actionID
	}

	// Create URI for agent ID resolution
	query := url.Values{}
	query.Set("op", "resolve")
	query.Set("agent", wf.GetAgentID())
	query.Set("version", wf.GetVersion())
	uri := url.URL{
		Scheme:   "agent",
		Path:     "/" + actionID,
		RawQuery: query.Encode(),
	}

	resolvedIDBytes, err := agentReader.Read(uri)
	if err != nil {
		// Fallback to default resolution if agent reader fails
		return fmt.Sprintf("@%s/%s:%s", wf.GetAgentID(), actionID, wf.GetVersion())
	}

	return string(resolvedIDBytes)
}

// replaceTargetActionID replaces the TargetActionID in workflow content using a more robust approach
func replaceTargetActionID(content, newActionID string) (string, error) {
	// Use a more precise regex pattern that matches the PKL syntax
	// This pattern looks for TargetActionID = "value" with proper PKL syntax
	targetActionIDPattern := regexp.MustCompile(`(?m)^(\s*TargetActionID\s*=\s*)"[^"]*"(\s*)$`)
	matches := targetActionIDPattern.FindStringSubmatch(content)
	if len(matches) >= 3 {
		// Replace with the new actionID
		replacement := matches[1] + `"` + newActionID + `"` + matches[2]
		return targetActionIDPattern.ReplaceAllString(content, replacement), nil
	}

	// Fallback to a more flexible pattern if the strict one doesn't match
	fallbackPattern := regexp.MustCompile(`(?i)(TargetActionID\s*=\s*)"[^"]*"`)
	matches = fallbackPattern.FindStringSubmatch(content)
	if len(matches) >= 2 {
		// Replace with the new actionID
		replacement := matches[1] + `"` + newActionID + `"`
		return fallbackPattern.ReplaceAllString(content, replacement), nil
	}

	return "", errors.New("TargetActionID not found in workflow content")
}

func CompileProject(ctx context.Context, fs afero.Fs, wf pklWf.Workflow, kdepsDir string, projectDir string, env *environment.Environment, logger *logging.Logger) (string, string, error) {
	compiledProjectDir, err := CompileWorkflow(ctx, fs, wf, kdepsDir, projectDir, logger)
	if err != nil {
		return "", "", fmt.Errorf("failed to compile workflow: %w", err)
	}

	if exists, err := afero.DirExists(fs, compiledProjectDir); !exists || err != nil {
		return "", "", fmt.Errorf("compiled project directory error: %w", err)
	}

	newWorkflowFile := filepath.Join(compiledProjectDir, "workflow.pkl")
	if _, err := fs.Stat(newWorkflowFile); err != nil {
		return "", "", fmt.Errorf("compiled workflow missing: %w", err)
	}

	newWorkflow, err := workflow.LoadWorkflow(ctx, newWorkflowFile, logger)
	if err != nil {
		return "", "", fmt.Errorf("failed to load workflow: %w", err)
	}

	resourcesDir := filepath.Join(compiledProjectDir, "resources")
	if err := CompileResources(ctx, fs, newWorkflow, resourcesDir, projectDir, logger); err != nil {
		return "", "", fmt.Errorf("failed to compile resources: %w", err)
	}

	if err := CopyDataDir(ctx, fs, newWorkflow, kdepsDir, projectDir, compiledProjectDir, "", "", "", false, logger); err != nil {
		return "", "", fmt.Errorf("failed to copy project: %w", err)
	}

	if err := ProcessExternalWorkflows(ctx, fs, newWorkflow, kdepsDir, projectDir, compiledProjectDir, logger); err != nil {
		return "", "", fmt.Errorf("failed to process workflows: %w", err)
	}

	// Evaluate all PKL files in the compiled project directory to test for any problems
	logger.Info("evaluating all PKL files for validation")

	// Skip PKL evaluation in local mode since local PKL files won't be available during packaging
	versionInfo := version.GetVersionInfo()
	if versionInfo.LocalMode == "1" {
		logger.Info("skipping PKL evaluation in local mode", "reason", "local PKL files not available during packaging")
	} else {
		if err := evaluator.EvaluateAllPklFilesInDirectory(fs, ctx, compiledProjectDir, logger); err != nil {
			return "", "", fmt.Errorf("failed to evaluate PKL files: %w", err)
		}
	}

	packageFile, err := PackageProject(fs, ctx, newWorkflow, kdepsDir, compiledProjectDir, logger)
	if err != nil {
		return "", "", fmt.Errorf("failed to package project: %w", err)
	}

	if _, err := fs.Stat(packageFile); err != nil {
		return "", "", fmt.Errorf("package file missing: %w", err)
	}

	cwdPackage := filepath.Join(env.Pwd, filepath.Base(packageFile))
	if err := CopyFile(ctx, fs, packageFile, cwdPackage, logger); err != nil {
		return "", "", err
	}

	logger.Info("kdeps package created", "package-file", cwdPackage)
	return compiledProjectDir, packageFile, nil
}

func parseWorkflowValue(value string) (string, string, string) {
	var agent, version, action string

	value = strings.TrimPrefix(value, "@")
	if parts := strings.SplitN(value, ":", 2); len(parts) > 1 {
		version = parts[1]
		agentAction := strings.SplitN(parts[0], "/", 2)
		agent = agentAction[0]
		if len(agentAction) > 1 {
			action = agentAction[1]
		}
	} else {
		agentAction := strings.SplitN(value, "/", 2)
		agent = agentAction[0]
		if len(agentAction) > 1 {
			action = agentAction[1]
		}
	}
	return agent, version, action
}

func ProcessExternalWorkflows(ctx context.Context, fs afero.Fs, wf pklWf.Workflow, kdepsDir, projectDir, compiledProjectDir string, logger *logging.Logger) error {
	if wf.GetWorkflows() == nil {
		return nil
	}

	for _, value := range wf.GetWorkflows() {
		agent, version, action := parseWorkflowValue(value)
		err := CopyDataDir(ctx, fs, wf, kdepsDir, projectDir, compiledProjectDir, agent, version, action, true, logger)
		if err != nil {
			logger.Error("failed to process workflow", "agent", agent, "version", version, "action", action, "error", err)
			return err
		}
	}
	return nil
}
