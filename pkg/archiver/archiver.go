package archiver

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"kdeps/pkg/enforcer"
	"os"
	"path/filepath"
	"strings"

	"github.com/kdeps/schema/gen/workflow"
	"github.com/spf13/afero"
)

var schemaVersionFilePath = "../../SCHEMA_VERSION"

// if the filesystem supports it, use Lstat, else use fs.Stat
func lstatIfPossible(fs afero.Fs, path string) (os.FileInfo, error) {
	if lfs, ok := fs.(afero.Lstater); ok {
		fi, _, err := lfs.LstatIfPossible(path)
		return fi, err
	}
	return fs.Stat(path)
}

func walk(fs afero.Fs, path string, info os.FileInfo, walkFn filepath.WalkFunc) error {
	err := walkFn(path, info, nil) // Call walkFn for the current file/directory
	if err != nil {
		if info.IsDir() && err == filepath.SkipDir {
			return nil // Skip directory if instructed
		}
		return err
	}

	// If it's not a directory, we are done with this file
	if !info.IsDir() {
		return nil
	}

	// If it's a directory, open it to read its contents
	dir, err := fs.Open(path)
	if err != nil {
		return walkFn(path, info, err)
	}
	defer dir.Close()

	// Read directory entries
	names, err := dir.Readdirnames(-1) // Read all directory entries
	if err != nil {
		return walkFn(path, info, err)
	}

	// Sort entries for deterministic output
	for _, name := range names {
		filename := filepath.Join(path, name)
		fileInfo, err := lstatIfPossible(fs, filename)
		if err != nil {
			if err := walkFn(filename, nil, err); err != nil && err != filepath.SkipDir {
				return err
			}
			continue
		}

		// Check if the file has a ".pkl" extension and enforce structure if needed
		if !fileInfo.IsDir() && filepath.Ext(filename) == ".pkl" {
			if err := enforcer.EnforcePklTemplateAmendsRules(fs, filename, schemaVersionFilePath); err != nil {
				return err
			}
		}

		// Recursively call walk for each file/directory in the current directory
		err = walk(fs, filename, fileInfo, walkFn)
		if err != nil {
			if !fileInfo.IsDir() || err != filepath.SkipDir {
				return err
			}
		}
	}
	return nil
}

// PackageProject compresses the contents of projectDir into a tar.gz file in kdepsDir
func PackageProject(fs afero.Fs, wf *workflow.Workflow, kdepsDir string, projectDir string) (string, error) {
	if err := enforcer.EnforceFolderStructure(fs, projectDir); err != nil {
		return "", err
	}

	outFile := fmt.Sprintf("%s-%s.kdeps", wf.Name, wf.Version)

	packageDir := fmt.Sprintf("%s/packages", kdepsDir)

	// Define the output file path for the tarball
	tarGzPath := filepath.Join(packageDir, outFile)

	// Create the tar.gz file in kdepsDir
	tarFile, err := fs.Create(tarGzPath)
	if err != nil {
		return "", fmt.Errorf("failed to create package file: %w", err)
	}
	defer tarFile.Close()

	// Create a gzip writer
	gzWriter := gzip.NewWriter(tarFile)
	defer gzWriter.Close()

	// Create a tar writer
	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// Walk through all the files in projectDir using Walk
	err = afero.Walk(fs, projectDir, func(file string, info os.FileInfo, err error) error {

		if err != nil {
			return fmt.Errorf("error walking the file tree: %w", err)
		}

		// Skip directories, only process files
		if info.IsDir() {
			return nil
		}

		// Open the file to read its contents
		fileHandle, err := fs.Open(file)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", file, err)
		}
		defer fileHandle.Close()

		// Get relative path for tar header
		relPath, err := filepath.Rel(projectDir, file)
		if err != nil {
			return fmt.Errorf("failed to get relative file path: %w", err)
		}

		// Create tar header for the file
		header, err := tar.FileInfoHeader(info, relPath)
		if err != nil {
			return fmt.Errorf("failed to create tar header for %s: %w", file, err)
		}
		header.Name = strings.ReplaceAll(relPath, "\\", "/") // Normalize path to use forward slashes

		// Write the header
		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write tar header for %s: %w", file, err)
		}

		// Copy the file data to the tar writer
		if _, err := io.Copy(tarWriter, fileHandle); err != nil {
			return fmt.Errorf("failed to copy file contents for %s: %w", file, err)
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("error packaging project: %w", err)
	}

	// Return the path to the generated tar.gz file
	return tarGzPath, nil
}
