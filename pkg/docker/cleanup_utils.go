package docker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/kdeps/kdeps/pkg/archiver"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/ktx"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/utils"
	"github.com/spf13/afero"
)

func CleanupDockerBuildImages(fs afero.Fs, ctx context.Context, cName string, cli *client.Client) error {
	// Check if the container named "cName" is already running, and remove it if necessary
	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return fmt.Errorf("error listing containers: %w", err)
	}

	for _, c := range containers {
		for _, name := range c.Names {
			if name == "/"+cName { // Ensure name match is exact
				fmt.Printf("Deleting container: %s\n", c.ID)
				if err := cli.ContainerRemove(ctx, c.ID, container.RemoveOptions{Force: true}); err != nil {
					// Log error and continue
					fmt.Printf("Error removing container %s: %v\n", c.ID, err)
					continue
				}
			}
		}
	}

	// Prune dangling images
	if _, err := cli.ImagesPrune(ctx, filters.Args{}); err != nil {
		return fmt.Errorf("error pruning images: %w", err)
	}

	fmt.Println("Pruned dangling images.")
	return nil
}

// Cleanup deletes /agent/action and /agent/workflow directories, then copies /agent/project to /agent/workflow.
func Cleanup(fs afero.Fs, ctx context.Context, env *environment.Environment, logger *logging.Logger) {
	actionDirValue, actionExists := ktx.ReadContext(ctx, ktx.CtxKeyActionDir)
	agentDirValue, agentExists := ktx.ReadContext(ctx, ktx.CtxKeyAgentDir)

	if !actionExists || !agentExists {
		logger.Warn("Missing directory context, skipping cleanup")
		return
	}

	actionDir, actionOk := actionDirValue.(string)
	agentDir, agentOk := agentDirValue.(string)

	if !actionOk || !agentOk || actionDir == "" || agentDir == "" {
		logger.Warn("Invalid directory context types, skipping cleanup")
		return
	}

	projectDir := filepath.Join(agentDir, "/project")
	workflowDir := filepath.Join(agentDir, "/workflow")

	removedFiles := []string{"/.actiondir_removed", "/.dockercleanup"}

	// Initialize bus manager for cleanup signaling
	busManager, err := utils.NewBusIPCManager(logger)
	if err != nil {
		logger.Warn("Bus not available, using file-based cleanup signaling", "error", err)
		busManager = nil
	}
	defer func() {
		if busManager != nil {
			busManager.Close()
		}
	}()

	// Helper function to remove a directory and signal via bus or create flag file
	removeDirWithSignal := func(ctx context.Context, dir string, flagFile string, signalType string) error {
		if err := fs.RemoveAll(dir); err != nil {
			logger.Error(fmt.Sprintf("Error removing %s: %v", dir, err))
			return err
		}

		logger.Debug(dir + " directory deleted")

		if busManager != nil {
			// Signal via bus
			if err := busManager.SignalCleanup(signalType, fmt.Sprintf("Directory %s removed", dir), map[string]interface{}{
				"directory": dir,
				"operation": "remove",
			}); err != nil {
				logger.Warn("Failed to signal cleanup via bus, creating flag file", "error", err)
				// Fallback to file creation
				if err := CreateFlagFile(fs, ctx, flagFile); err != nil {
					logger.Error(fmt.Sprintf("Unable to create flag file %s: %v", flagFile, err))
					return err
				}
			}
		} else {
			// Fallback to file creation
			if err := CreateFlagFile(fs, ctx, flagFile); err != nil {
				logger.Error(fmt.Sprintf("Unable to create flag file %s: %v", flagFile, err))
				return err
			}
		}
		return nil
	}

	// Remove action and workflow directories
	if err := removeDirWithSignal(ctx, actionDir, removedFiles[0], "action"); err != nil {
		return
	}

	// Signal docker cleanup completion
	if busManager != nil {
		if err := busManager.SignalCleanup("docker", "Docker cleanup completed", map[string]interface{}{
			"operation": "cleanup_complete",
		}); err != nil {
			logger.Warn("Failed to signal docker cleanup completion via bus, creating flag file", "error", err)
			// Fallback to file creation
			if err := CreateFlagFile(fs, ctx, removedFiles[1]); err != nil {
				logger.Error(fmt.Sprintf("Unable to create flag file %s: %v", removedFiles[1], err))
				return
			}
		}
	} else {
		// Fallback to file creation
		if err := CreateFlagFile(fs, ctx, removedFiles[1]); err != nil {
			logger.Error(fmt.Sprintf("Unable to create flag file %s: %v", removedFiles[1], err))
			return
		}
	}

	// Wait for cleanup signals or files - prioritize bus over files
	if busManager != nil {
		// Wait for cleanup signals via bus with short timeout
		if err := busManager.WaitForCleanup(3); err != nil {
			logger.Debug("Bus cleanup wait failed, falling back to file waiting", "error", err)
			// Fallback to file waiting
			for _, flag := range removedFiles[:2] {
				if err := utils.WaitForFileReady(fs, flag, logger); err != nil {
					logger.Error(fmt.Sprintf("Error waiting for flag %s: %v", flag, err))
					return
				}
			}
		}
	} else {
		// Wait for the cleanup flags to be ready
		for _, flag := range removedFiles[:2] {
			if err := utils.WaitForFileReady(fs, flag, logger); err != nil {
				logger.Error(fmt.Sprintf("Error waiting for flag %s: %v", flag, err))
				return
			}
		}
	}

	// Copy /agent/project to /agent/workflow
	err = afero.Walk(fs, projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create relative target path inside /agent/workflow
		relPath, err := filepath.Rel(projectDir, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(workflowDir, relPath)

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
		logger.Error("Error copying project directory to workflow directory", "error", err)
	}
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
