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
	"github.com/kdeps/kdeps/pkg/bus"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/ktx"
	"github.com/kdeps/kdeps/pkg/logging"
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

	// Helper function to remove a directory and publish an event
	removeDirWithEvent := func(ctx context.Context, dir string, eventType string) error {
		if err := fs.RemoveAll(dir); err != nil {
			logger.Error(fmt.Sprintf("Error removing %s: %v", dir, err))
			return err
		}

		logger.Debug(dir + " directory deleted")
		bus.PublishGlobalEvent(eventType, graphID)
		logger.Debug(fmt.Sprintf("Published %s event", eventType), "graphID", graphID)
		return nil
	}

	// Remove action directory and publish event
	if err := removeDirWithEvent(ctx, actionDir, "actiondir_removed"); err != nil {
		return
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

	// Publish final dockercleanup event instead of creating flag file
	bus.PublishGlobalEvent("dockercleanup", graphID)
	logger.Debug("Published dockercleanup event", "graphID", graphID)
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
