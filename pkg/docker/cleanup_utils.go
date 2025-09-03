package docker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/kdeps/kdeps/pkg/archiver"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/ktx"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/utils"
	"github.com/spf13/afero"
)

// DockerPruneClient is a minimal interface for Docker operations used in CleanupDockerBuildImages.
type DockerPruneClient interface {
	ContainerList(ctx context.Context, options container.ListOptions) ([]container.Summary, error)
	ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error
	ImagesPrune(ctx context.Context, pruneFilters filters.Args) (image.PruneReport, error)
}

func CleanupDockerBuildImages(fs afero.Fs, ctx context.Context, cName string, cli DockerPruneClient) error {
	// Check if the container named "cName" is already running, and remove it if necessary
	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return fmt.Errorf("error listing containers: %w", err)
	}

	for _, c := range containers {
		for _, name := range c.Names {
			if name == "/"+cName { // Ensure name match is exact
				fmt.Printf("Deleting container: %s\n", c.ID) //nolint:forbidigo // Status reporting
				if err := cli.ContainerRemove(ctx, c.ID, container.RemoveOptions{Force: true}); err != nil {
					// Log error and continue
					fmt.Printf("Error removing container %s: %v\n", c.ID, err) //nolint:forbidigo // Error reporting
					continue
				}
			}
		}
	}

	// Prune dangling images
	if _, err := cli.ImagesPrune(ctx, filters.Args{}); err != nil {
		return fmt.Errorf("error pruning images: %w", err)
	}

	fmt.Println("Pruned dangling images.") //nolint:forbidigo // Status reporting
	return nil
}

// Cleanup deletes /agent/action and /agent/workflow directories, then copies /agent/project to /agent/workflow.
func Cleanup(fs afero.Fs, ctx context.Context, environ *environment.Environment, logger *logging.Logger) {
	if environ.DockerMode != "1" {
		return
	}

	graphID, actionDir := extractCleanupContext(ctx)
	cleanupConfig := setupCleanupDirectories(graphID, actionDir)

	if err := performCleanupOperations(ctx, fs, cleanupConfig, logger); err != nil {
		logger.Error("Cleanup operations failed", "error", err)
		return
	}

	if err := copyProjectToWorkflow(ctx, fs, cleanupConfig, logger); err != nil {
		logger.Error("Failed to copy project to workflow", "error", err)
	}

	finalizeCleanup(ctx, fs, cleanupConfig, logger)
}

func extractCleanupContext(ctx context.Context) (string, string) {
	var graphID, actionDir string

	contextKeys := map[*string]ktx.ContextKey{
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

	return graphID, actionDir
}

func setupCleanupDirectories(graphID, actionDir string) *cleanupConfig {
	return &cleanupConfig{
		workflowDir: "/agent/workflow",
		projectDir:  "/agent/project",
		removedFiles: []string{
			filepath.Join("/tmp", ".actiondir_removed_"+graphID),
			filepath.Join(actionDir, ".dockercleanup_"+graphID),
		},
		actionDir: actionDir,
		graphID:   graphID,
	}
}

type cleanupConfig struct {
	workflowDir  string
	projectDir   string
	removedFiles []string
	actionDir    string
	graphID      string
}

func performCleanupOperations(ctx context.Context, fs afero.Fs, config *cleanupConfig, logger *logging.Logger) error {
	// Remove action directory and create flag
	if err := removeDirWithFlag(ctx, fs, config.actionDir, config.removedFiles[0], logger); err != nil {
		return err
	}

	// Wait for cleanup flags to be ready
	for _, flag := range config.removedFiles[:2] {
		if err := utils.WaitForFileReady(fs, flag, logger); err != nil {
			logger.Error(fmt.Sprintf("Error waiting for flag %s: %v", flag, err))
			return err
		}
	}

	return nil
}

func removeDirWithFlag(ctx context.Context, fs afero.Fs, dir string, flagFile string, logger *logging.Logger) error {
	if err := fs.RemoveAll(dir); err != nil {
		logger.Error(fmt.Sprintf("Error removing %s: %v", dir, err))
		return err
	}

	logger.Debug(dir + " directory deleted")
	if err := CreateFlagFile(fs, ctx, flagFile); err != nil {
		logger.Error(fmt.Sprintf("Unable to create flag file %s: %v", flagFile, err))
		return err
	}
	return nil
}

func copyProjectToWorkflow(ctx context.Context, fs afero.Fs, config *cleanupConfig, logger *logging.Logger) error {
	err := afero.Walk(fs, config.projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create relative target path inside /agent/workflow
		relPath, err := filepath.Rel(config.projectDir, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(config.workflowDir, relPath)

		if info.IsDir() {
			if err := fs.MkdirAll(targetPath, info.Mode()); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
			}
		} else {
			// Copy the file from projectDir to workflowDir
			if err := archiver.CopyFile(fs, ctx, path, targetPath, logger); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("error copying %s to %s: %w", config.projectDir, config.workflowDir, err)
	}

	logger.Debug(fmt.Sprintf("Copied %s to %s for next run", config.projectDir, config.workflowDir))
	return nil
}

func finalizeCleanup(ctx context.Context, fs afero.Fs, config *cleanupConfig, logger *logging.Logger) {
	// Create final cleanup flag
	if err := CreateFlagFile(fs, ctx, config.removedFiles[1]); err != nil {
		logger.Error(fmt.Sprintf("Unable to create final cleanup flag: %v", err))
	}

	// Remove flag files
	cleanupFlagFiles(fs, config.removedFiles, logger)
}

// cleanupFlagFiles removes the specified flag files.
func cleanupFlagFiles(fs afero.Fs, files []string, logger *logging.Logger) {
	for _, file := range files {
		if err := fs.Remove(file); err != nil {
			if os.IsNotExist(err) {
				logger.Debugf("file %s does not exist, skipping", file)
			} else {
				logger.Errorf("error removing file %s: %v", file, err)
			}
		} else {
			logger.Debugf("successfully removed file: %s", file)
		}
	}
}
