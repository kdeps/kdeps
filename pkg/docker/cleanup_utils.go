package docker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types"
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

// DockerPruneClient is a minimal interface for Docker operations used in CleanupDockerBuildImages
type DockerPruneClient interface {
	ContainerList(ctx context.Context, options container.ListOptions) ([]types.Container, error)
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

	workflowDir := "/agent/workflow"
	projectDir := "/agent/project"
	removedFiles := []string{filepath.Join("/tmp", ".actiondir_removed_"+graphID), filepath.Join(actionDir, ".dockercleanup_"+graphID)}

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
		if err := utils.WaitForFileReady(fs, flag, logger); err != nil {
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
	CleanupFlagFiles(fs, removedFiles, logger)
}

// CleanupFlagFiles removes the specified flag files.
func CleanupFlagFiles(fs afero.Fs, files []string, logger *logging.Logger) {
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
