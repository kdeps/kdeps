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
func Cleanup(fs afero.Fs, ctx context.Context, environ *environment.Environment, logger *logging.Logger) {
	if environ.DockerMode != "1" {
		return
	}

	actionDir := "/agent/action"
	workflowDir := "/agent/workflow"
	projectDir := "/agent/project"
	removedFiles := []string{"/.actiondir_removed", "/.dockercleanup"}

	// Helper function to remove a directory and create a corresponding flag file
	removeDirWithFlag := func(ctx context.Context, dir string, flagFile string) error {
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

	// Remove action and workflow directories
	if err := removeDirWithFlag(ctx, actionDir, removedFiles[0]); err != nil {
		return
	}

	// Wait for the cleanup flags to be ready
	for _, flag := range removedFiles[:2] { // Correcting to wait for the first two files
		if err := utils.WaitForFileReady(fs, ctx, flag, logger); err != nil {
			logger.Error(fmt.Sprintf("Error waiting for flag %s: %v", flag, err))
			return
		}
	}

	// Copy /agent/project to /agent/workflow
	err := afero.Walk(fs, projectDir, func(path string, info os.FileInfo, err error) error {
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
		logger.Error(fmt.Sprintf("Error copying %s to %s: %v", projectDir, workflowDir, err))
	} else {
		logger.Debug(fmt.Sprintf("Copied %s to %s for next run", projectDir, workflowDir))
	}

	// Create final cleanup flag
	if err := CreateFlagFile(fs, ctx, removedFiles[1]); err != nil {
		logger.Error(fmt.Sprintf("Unable to create final cleanup flag: %v", err))
	}

	// Remove flag files
	cleanupFlagFiles(fs, ctx, removedFiles, logger)
}

// cleanupFlagFiles removes the specified flag files.
func cleanupFlagFiles(fs afero.Fs, ctx context.Context, files []string, logger *logging.Logger) {
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
