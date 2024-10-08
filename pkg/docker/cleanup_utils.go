package docker

import (
	"context"
	"fmt"
	"kdeps/pkg/archiver"
	"kdeps/pkg/environment"
	"kdeps/pkg/utils"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/spf13/afero"
)

func CleanupDockerBuildImages(fs afero.Fs, ctx context.Context, cName string, cli *client.Client) error {
	// Check if the container named "cName" is already running, and remove it if necessary
	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return err
	}

	for _, c := range containers {
		for _, name := range c.Names {
			if name == "/"+cName { // Ensure name match is exact
				fmt.Printf("Deleting container: %s\n", c.ID)
				err := cli.ContainerRemove(ctx, c.ID, container.RemoveOptions{Force: true})
				if err != nil {
					return err
				}
			}
		}
	}

	// Prune dangling images
	_, err = cli.ImagesPrune(ctx, filters.Args{})
	if err != nil {
		return err
	}

	fmt.Println("Pruned dangling images.")
	return nil
}

// Cleanup deletes /agents/action and /agents/workflow directories, then copies /agents/project to /agents/workflow
func Cleanup(fs afero.Fs, environ *environment.Environment, logger *log.Logger) {
	if environ.DockerMode != "1" {
		return
	}

	actionDir := "/agent/action"
	workflowDir := "/agent/workflow"
	projectDir := "/agent/project"
	removedFiles := []string{"/.actiondir_removed", "/.workflowdir_removed", "/.dockercleanup"}

	// Helper function to remove a directory and create a corresponding flag file
	removeDirWithFlag := func(dir string, flagFile string) error {
		if err := fs.RemoveAll(dir); err != nil {
			logger.Error(fmt.Sprintf("Error removing %s: %v", dir, err))
			return err
		}

		logger.Info(fmt.Sprintf("%s directory deleted", dir))
		if err := CreateFlagFile(fs, flagFile); err != nil {
			logger.Error(fmt.Sprintf("Unable to create flag file %s: %v", flagFile, err))
			return err
		}
		return nil
	}

	// Remove action and workflow directories
	if err := removeDirWithFlag(actionDir, removedFiles[0]); err != nil {
		return
	}
	if err := removeDirWithFlag(workflowDir, removedFiles[1]); err != nil {
		return
	}

	// Wait for the cleanup flags to be ready
	for _, flag := range removedFiles[:2] { // Only the first two files need to be waited on
		if err := utils.WaitForFileReady(fs, flag, logger); err != nil {
			logger.Error(fmt.Sprintf("Error waiting for flag %s: %v", flag, err))
			return
		}
	}

	// Copy /agents/project to /agents/workflow
	err := afero.Walk(fs, projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create relative target path inside /agents/workflow
		relPath, err := filepath.Rel(projectDir, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(workflowDir, relPath)

		if info.IsDir() {
			if err := fs.MkdirAll(targetPath, info.Mode()); err != nil {
				return fmt.Errorf("failed to create directory %s: %v", targetPath, err)
			}
		} else {
			// Copy the file from projectDir to workflowDir
			if err := archiver.CopyFile(fs, path, targetPath); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		logger.Error(fmt.Sprintf("Error copying %s to %s: %v", projectDir, workflowDir, err))
	} else {
		logger.Info(fmt.Sprintf("Copied %s to %s for next run", projectDir, workflowDir))
	}

	// Create final cleanup flag
	if err := CreateFlagFile(fs, removedFiles[2]); err != nil {
		logger.Error(fmt.Sprintf("Unable to create final cleanup flag: %v", err))
	}

	// Remove flag files
	cleanupFlagFiles(fs, removedFiles, logger)
}

// cleanupFlagFiles removes the specified flag files
func cleanupFlagFiles(fs afero.Fs, files []string, logger *log.Logger) {
	for _, file := range files {
		if err := fs.Remove(file); err != nil {
			if os.IsNotExist(err) {
				logger.Infof("File %s does not exist, skipping", file)
			} else {
				logger.Errorf("Error removing file %s: %v", file, err)
			}
		} else {
			logger.Infof("Successfully removed file: %s", file)
		}
	}
}
