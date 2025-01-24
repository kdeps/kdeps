package archiver

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kdeps/kdeps/pkg/enforcer"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/utils"
	"github.com/kdeps/kdeps/pkg/workflow"
	pklWf "github.com/kdeps/schema/gen/workflow"
	"github.com/spf13/afero"
)

func PrepareRunDir(fs afero.Fs, ctx context.Context, wf pklWf.Workflow, kdepsDir, pkgFilePath string, logger *logging.Logger) (string, error) {
	agentName, agentVersion := wf.GetName(), wf.GetVersion()

	runDir := filepath.Join(kdepsDir, "run/"+agentName+"/"+agentVersion+"/workflow")

	// Recursively delete the runDir if it exists
	if exists, err := afero.Exists(fs, runDir); err != nil {
		return "", err
	} else if exists {
		if err := fs.RemoveAll(runDir); err != nil {
			return "", err
		}
	}

	// Create the directory
	if err := fs.MkdirAll(runDir, 0o755); err != nil {
		return "", err
	}

	if _, err := fs.Stat(pkgFilePath); err != nil {
		logger.Error("package not found!", "package", pkgFilePath)
		return "", err
	}

	file, err := os.Open(pkgFilePath)
	if err != nil {
		logger.Error("error opening file: %v\n", err)
		return "", err
	}
	defer file.Close()

	// Open the gzip reader
	gzr, err := gzip.NewReader(file)
	if err != nil {
		logger.Error("error creating gzip reader: %v\n", err)
		return "", err
	}
	defer gzr.Close()

	// Open the tar reader
	tarReader := tar.NewReader(gzr)

	// Extract all the files
	for {
		// Get the next header in the tar file
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break // End of archive
		}
		if err != nil {
			logger.Error("error reading tar file: %v\n", err)
			return "", err
		}

		// Create the full path for the file to extract
		target, err := utils.SanitizeArchivePath(runDir, header.Name)
		if err != nil {
			return "", err
		}

		// Handle file types (file, directory, etc.)
		switch header.Typeflag {
		case tar.TypeDir:
			// Create directory
			if err := os.MkdirAll(target, os.ModePerm); err != nil {
				logger.Error("error creating directory: %v\n", err)
				return "", err
			}
		case tar.TypeReg:
			// Extract file
			if err := os.MkdirAll(filepath.Dir(target), os.ModePerm); err != nil {
				logger.Error("error creating file directory: %v\n", err)
				return "", err
			}
			outFile, err := os.Create(target)
			if err != nil {
				logger.Error("error creating file: %v\n", err)
				return "", err
			}
			defer outFile.Close()

			// Copy the file contents
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
		default:
			logger.Error("unknown type: %v in %s\n", header.Typeflag, header.Name)
		}
	}

	logger.Debug("extraction in runtime folder completed!", runDir)

	return runDir, nil
}

// CompileWorkflow compiles a workflow file and updates the action field.
func CompileWorkflow(fs afero.Fs, ctx context.Context, wf pklWf.Workflow, kdepsDir, projectDir string, logger *logging.Logger) (string, error) {
	action := wf.GetTargetActionID()

	if action == "" {
		logger.Error("no action specified in workflow!")
		return "", errors.New("please specify the default action in the workflow")
	}

	var compiledAction string

	name := wf.GetName()
	version := wf.GetVersion()

	filePath := filepath.Join(projectDir, "workflow.pkl")
	agentDir := filepath.Join(kdepsDir, fmt.Sprintf("agents/%s/%s", name, version))
	resourcesDir := filepath.Join(agentDir, "resources")
	compiledFilePath := filepath.Join(agentDir, "workflow.pkl")

	re := regexp.MustCompile(`^@`)

	if re.MatchString(action) {
		// If action starts with "@", use it directly without modification
		compiledAction = action
	} else {
		// Otherwise, prepend the name and version
		compiledAction = fmt.Sprintf("@%s/%s:%s", name, action, version)
	}

	// Check if agentDir exists and remove it if it does
	exists, err := afero.DirExists(fs, agentDir)
	if err != nil {
		logger.Error("error checking if agent directory exists", "path", agentDir, "error", err)
		return "", err
	}

	if exists {
		err := fs.RemoveAll(agentDir)
		if err != nil {
			logger.Error("failed to remove existing agent directory", "path", agentDir, "error", err)
			return "", err
		}
		logger.Debug("removed existing agent directory", "path", agentDir)
	}

	// Recreate the folder
	err = fs.MkdirAll(resourcesDir, 0o755) // Create the folder with read-write-execute permissions
	if err != nil {
		logger.Error("failed to create resources directory", "path", resourcesDir, "error", err)
		return "", err
	}
	logger.Debug("created resources directory", "path", resourcesDir)

	searchPattern := `targetActionID\s*=\s*".*"`
	replaceLine := fmt.Sprintf("targetActionID = \"%s\"\n", compiledAction)

	inputFile, err := fs.Open(filePath)
	if err != nil {
		logger.Error("failed to open workflow file", "path", filePath, "error", err)
		return "", err
	}
	defer inputFile.Close()

	var lines []string
	scanner := bufio.NewScanner(inputFile)

	// Compile the regular expression
	re = regexp.MustCompile(searchPattern)

	for scanner.Scan() {
		line := scanner.Text()

		// Check if the line matches the regular expression
		if re.MatchString(line) {
			line = replaceLine // Replace the line if it matches
			logger.Debug("updated action line", "line", line)
		}

		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		logger.Error("error reading workflow file", "path", filePath, "error", err)
		return "", err
	}

	err = afero.WriteFile(fs, compiledFilePath, []byte(strings.Join(lines, "\n")), 0o644)
	if err != nil {
		logger.Error("failed to write compiled workflow file", "path", compiledFilePath, "error", err)
		return "", err
	}
	logger.Debug("compiled workflow file written", "path", compiledFilePath)

	if err := enforcer.EnforcePklTemplateAmendsRules(fs, ctx, compiledFilePath, logger); err != nil {
		logger.Error("validation failed for .pkl file", "file", compiledFilePath, "error", err)
		return "", err
	}

	compiledProjectDir := filepath.Dir(compiledFilePath)

	return compiledProjectDir, nil
}

// CompileProject orchestrates the compilation and packaging of a project.
func CompileProject(fs afero.Fs, ctx context.Context, wf pklWf.Workflow, kdepsDir string, projectDir string, env *environment.Environment, logger *logging.Logger) (string, string, error) {
	// Compile the workflow
	compiledProjectDir, err := CompileWorkflow(fs, ctx, wf, kdepsDir, projectDir, logger)
	if err != nil {
		logger.Error("failed to compile workflow", "error", err)
		return "", "", err
	}

	// Check if the compiled project directory exists
	exists, err := afero.DirExists(fs, compiledProjectDir)
	if err != nil {
		logger.Error("error checking if compiled project directory exists", "path", compiledProjectDir, "error", err)
		return "", "", err
	}
	if !exists {
		err = errors.New("compiled project directory does not exist")
		logger.Error("compiled project directory does not exist", "path", compiledProjectDir)
		return "", "", err
	}

	// Verify the compiled workflow file
	newWorkflowFile := filepath.Join(compiledProjectDir, "workflow.pkl")
	if _, err := fs.Stat(newWorkflowFile); err != nil {
		if os.IsNotExist(err) {
			err = fmt.Errorf("no compiled workflow found at: %s", newWorkflowFile)
			logger.Error("compiled workflow file does not exist", "path", newWorkflowFile, "error", err)
			return "", "", err
		}
		logger.Error("error stating compiled workflow file", "path", newWorkflowFile, "error", err)
		return "", "", err
	}

	// Load the new workflow
	newWorkflow, err := workflow.LoadWorkflow(ctx, newWorkflowFile, logger)
	if err != nil {
		logger.Error("failed to load new workflow", "path", newWorkflowFile, "error", err)
		return "", "", err
	}

	// Compile resources
	resourcesDir := filepath.Join(compiledProjectDir, "resources")
	if err := CompileResources(fs, ctx, newWorkflow, resourcesDir, projectDir, logger); err != nil {
		logger.Error("failed to compile resources", "resourcesDir", resourcesDir, "projectDir", projectDir, "error", err)
		return "", "", err
	}

	// Copy the project directory
	if err := CopyDataDir(fs, ctx, newWorkflow, kdepsDir, projectDir, compiledProjectDir, "", "", "", false, logger); err != nil {
		logger.Error("failed to copy project directory", "compiledProjectDir", compiledProjectDir, "error", err)
		return "", "", err
	}

	// Process workflows
	if err := ProcessExternalWorkflows(fs, ctx, newWorkflow, kdepsDir, projectDir, compiledProjectDir, logger); err != nil {
		logger.Error("failed to process workflows", "compiledProjectDir", compiledProjectDir, "error", err)
		return "", "", err
	}

	// Package the project
	packageFile, err := PackageProject(fs, ctx, newWorkflow, kdepsDir, compiledProjectDir, logger)
	if err != nil {
		logger.Error("failed to package project", "compiledProjectDir", compiledProjectDir, "error", err)
		return "", "", err
	}

	// Verify the package file
	if _, err := fs.Stat(packageFile); err != nil {
		if os.IsNotExist(err) {
			err = fmt.Errorf("no package file found at: %s", packageFile)
			logger.Error("package file does not exist", "path", packageFile, "error", err)
			return "", "", err
		}
		logger.Error("error stating package file", "path", packageFile, "error", err)
		return "", "", err
	}

	logger.Debug("kdeps package created in system archive", "package-file", packageFile)

	cwdPackage := filepath.Join(env.Pwd, filepath.Base(packageFile))
	if err := CopyFile(fs, ctx, packageFile, cwdPackage, logger); err != nil {
		return "", "", err
	}

	logger.Info("kdeps package created", "package-file", cwdPackage)

	return compiledProjectDir, packageFile, nil
}

// ProcessExternalWorkflows processes each workflow and copies directories as needed.
func ProcessExternalWorkflows(fs afero.Fs, ctx context.Context, wf pklWf.Workflow, kdepsDir, projectDir, compiledProjectDir string, logger *logging.Logger) error {
	if wf.GetWorkflows() == nil {
		logger.Debug("no external workflows to process")
		return nil
	}

	for _, value := range wf.GetWorkflows() {
		// Remove the "@" at the beginning if it exists
		value = strings.TrimPrefix(value, "@")

		// Check if the string contains ":"
		if strings.Contains(value, ":") {
			// Split into agentName and version by colon ":"
			parts := strings.SplitN(value, ":", 2)
			agentAndAction := strings.SplitN(parts[0], "/", 2) // Split the agent and action by "/"

			agentName := agentAndAction[0]
			version := parts[1]

			if len(agentAndAction) == 2 {
				action := agentAndAction[1]

				if err := CopyDataDir(fs, ctx, wf, kdepsDir, projectDir, compiledProjectDir, agentName, version, action, true, logger); err != nil {
					logger.Error("failed to copy directory", "agentName", agentName, "version", version, "action", action, "error", err)
					return err
				}
			} else {
				if err := CopyDataDir(fs, ctx, wf, kdepsDir, projectDir, compiledProjectDir, agentName, version, "", true, logger); err != nil {
					logger.Error("failed to copy directory", "agentName", agentName, "version", version, "error", err)
					return err
				}
			}
		} else {
			// No version present, check if there is an action
			agentAndAction := strings.SplitN(value, "/", 2)
			agentName := agentAndAction[0]

			if len(agentAndAction) == 2 {
				action := agentAndAction[1]
				if err := CopyDataDir(fs, ctx, wf, kdepsDir, projectDir, compiledProjectDir, agentName, "", action, true, logger); err != nil {
					logger.Error("failed to copy directory", "agentName", agentName, "action", action, "error", err)
					return err
				}
			} else {
				if err := CopyDataDir(fs, ctx, wf, kdepsDir, projectDir, compiledProjectDir, agentName, "", "", true, logger); err != nil {
					logger.Error("failed to copy directory", "agentName", agentName, "error", err)
					return err
				}
			}
		}
	}

	logger.Debug("processed all external workflows")
	return nil
}
