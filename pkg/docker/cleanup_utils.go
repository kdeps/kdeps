package docker

import (
	"context"
	"fmt"
	"io"
	"kdeps/pkg/environment"
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

// cleanup deletes /agents/action and /agents/workflow directories, then copies /agents/project to /agents/workflow
func Cleanup(fs afero.Fs, environ *environment.Environment, logger *log.Logger) {
	if environ.DockerMode == "1" {
		actionDir := "/agent/action"
		workflowDir := "/agent/workflow"
		projectDir := "/agent/project"

		// Delete /agents/action directory
		if err := fs.RemoveAll(actionDir); err != nil {
			logger.Error(fmt.Sprintf("Error removing %s: %v", actionDir, err))
		} else {
			logger.Info(fmt.Sprintf("%s directory deleted", actionDir))
		}

		// Delete /agents/workflow directory
		if err := fs.RemoveAll(workflowDir); err != nil {
			logger.Error(fmt.Sprintf("Error removing %s: %v", workflowDir, err))
		} else {
			logger.Info(fmt.Sprintf("%s directory deleted", workflowDir))
		}

		// Copy /agents/project to /agents/workflow
		err := afero.Walk(fs, projectDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Create the relative target path inside /agents/workflow
			relPath, err := filepath.Rel(projectDir, path)
			if err != nil {
				return err
			}
			targetPath := filepath.Join(workflowDir, relPath)

			if info.IsDir() {
				// Create the directory in the destination
				if err := fs.MkdirAll(targetPath, info.Mode()); err != nil {
					return fmt.Errorf("failed to create directory %s: %v", targetPath, err)
				}
			} else {
				// Copy the file from projectDir to workflowDir
				srcFile, err := fs.Open(path)
				if err != nil {
					return fmt.Errorf("failed to open source file %s: %v", path, err)
				}
				defer srcFile.Close()

				destFile, err := fs.Create(targetPath)
				if err != nil {
					return fmt.Errorf("failed to create destination file %s: %v", targetPath, err)
				}
				defer destFile.Close()

				_, err = io.Copy(destFile, srcFile)
				if err != nil {
					return fmt.Errorf("failed to copy file from %s to %s: %v", path, targetPath, err)
				}

				// Set the same permissions as the source file
				if err := fs.Chmod(targetPath, info.Mode()); err != nil {
					return fmt.Errorf("failed to set file permissions on %s: %v", targetPath, err)
				}
			}

			return nil
		})

		if err != nil {
			logger.Error(fmt.Sprintf("Error copying %s to %s: %v", projectDir, workflowDir, err))
		} else {
			logger.Info(fmt.Sprintf("Copied %s to %s for next run", projectDir, workflowDir))
		}

		if err := CreateFlagFile(fs, "/.dockercleanup"); err != nil {
			logger.Error("Unable to create docker cleanup flag", err)
		}
	}
}
