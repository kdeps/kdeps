package archiver

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	pklWf "github.com/kdeps/schema/gen/workflow"
	"github.com/spf13/afero"
)

// Move a directory by copying and then deleting the original
func MoveFolder(fs afero.Fs, src string, dest string) error {
	// Walk through the source directory and handle files and directories
	err := afero.Walk(fs, src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get the relative path from the source directory
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		// Construct the destination path
		destPath := filepath.Join(dest, relPath)

		// Check if the path is a directory
		if info.IsDir() {
			// Create the destination directory with the same permissions
			return fs.MkdirAll(destPath, info.Mode())
		}

		// If it's a file, copy it to the destination
		srcFile, err := fs.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		destFile, err := fs.Create(destPath)
		if err != nil {
			return err
		}
		defer destFile.Close()

		_, err = io.Copy(destFile, srcFile)
		if err != nil {
			return err
		}

		// Remove the original file
		return fs.Remove(path)
	})

	if err != nil {
		return err
	}

	// Remove the source directory after everything is copied
	return fs.RemoveAll(src)
}

func getFileMD5(fs afero.Fs, filePath string, length int) (string, error) {
	// Open the file using afero's filesystem
	file, err := fs.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Create an MD5 hash object
	hash := md5.New()

	// Copy the file content into the hash
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	// Get the hash sum in bytes and convert to a hexadecimal string
	hashInBytes := hash.Sum(nil)
	md5String := hex.EncodeToString(hashInBytes)

	// Return the shortened version of the hash (truncate to 'length')
	if length > len(md5String) {
		length = len(md5String)
	}
	return md5String[:length], nil
}

// Move the original file to a new name with MD5 and copy the latest file
func CopyFile(fs afero.Fs, src, dst string, logger *log.Logger) error {
	// Check if the destination file exists
	if _, err := fs.Stat(dst); err == nil {
		// Calculate MD5 for both source and destination files
		srcMD5, err := getFileMD5(fs, src, 8)
		if err != nil {
			return fmt.Errorf("failed to calculate MD5 for source file: %w", err)
		}

		dstMD5, err := getFileMD5(fs, dst, 8)
		if err != nil {
			return fmt.Errorf("failed to calculate MD5 for destination file: %w", err)
		}

		// If MD5 is the same, skip copying
		if srcMD5 == dstMD5 {
			fmt.Println("Files have the same MD5, skipping copy")
			return nil
		}

		// If MD5 is different, move the original destination file to a new name with MD5
		ext := filepath.Ext(dst)
		baseName := strings.TrimSuffix(filepath.Base(dst), ext)
		newName := fmt.Sprintf("%s_%s%s", baseName, dstMD5, ext)
		backupPath := filepath.Join(filepath.Dir(dst), newName)

		fmt.Println("Moving existing file to:", backupPath)
		if err := fs.Rename(dst, backupPath); err != nil {
			return fmt.Errorf("failed to move file to new name: %w", err)
		}
	}

	// Open the source file
	srcFile, err := fs.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Create the destination file
	dstFile, err := fs.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	// Copy the file contents from src to dst
	if _, err = io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	// Optionally, you can copy the file permissions from the source
	srcInfo, err := fs.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}
	if err = fs.Chmod(dst, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to change permissions on destination file: %w", err)
	}

	logger.Debug("File copied successfully:", "from", src, "to", dst)
	return nil
}

func CopyDataDir(fs afero.Fs, wf pklWf.Workflow, kdepsDir, projectDir, compiledProjectDir, agentName, agentVersion,
	agentAction string, processWorkflows bool, logger *log.Logger) error {
	var srcDir, destDir string

	srcDir = filepath.Join(projectDir, "data")
	destDir = filepath.Join(compiledProjectDir, fmt.Sprintf("data/%s/%s", wf.GetName(), wf.GetVersion()))

	if processWorkflows {
		// Helper function to copy resources
		copyResources := func(src, dst string) error {
			return afero.Walk(fs, src, func(path string, info os.FileInfo, walkErr error) error {
				if walkErr != nil {
					logger.Error("Error walking source directory", "path", src, "error", walkErr)
					return walkErr
				}

				// Determine the relative path for correct directory structure copying
				relPath, err := filepath.Rel(src, path)
				if err != nil {
					logger.Error("Failed to get relative path", "path", path, "error", err)
					return err
				}

				// Create the full destination path based on the relative path
				dstPath := filepath.Join(dst, relPath)

				if info.IsDir() {
					if err := fs.MkdirAll(dstPath, info.Mode()); err != nil {
						logger.Error("Failed to create directory", "path", dstPath, "error", err)
						return err
					}
				} else {
					if err := CopyFile(fs, path, dstPath, logger); err != nil {
						logger.Error("Failed to copy file", "src", path, "dst", dstPath, "error", err)
						return err
					}
				}
				return nil
			})
		}

		if agentVersion == "" {
			agentVersionPath := filepath.Join(kdepsDir, "agents", agentName)
			exists, err := afero.Exists(fs, agentVersionPath)
			if err != nil {
				return err
			}

			if exists {
				version, err := getLatestVersion(agentVersionPath, logger)
				if err != nil {
					logger.Error("Failed to get latest agent version", "agentVersionPath", agentVersionPath, "error", err)
					return err
				}
				agentVersion = version
			}
		}

		src := filepath.Join(kdepsDir, "agents", agentName, agentVersion, "resources")
		dst := filepath.Join(compiledProjectDir, "resources")

		srcDir = filepath.Join(kdepsDir, "agents", agentName, agentVersion, "data", agentName, agentVersion)
		destDir = filepath.Join(compiledProjectDir, fmt.Sprintf("data/%s/%s", agentName, agentVersion))

		exists, err := afero.Exists(fs, src)
		if err != nil {
			return err
		}

		if exists {
			if err := copyResources(src, dst); err != nil {
				logger.Error("Failed to copy resources", "src", src, "dst", dst, "error", err)
				return err
			}
		}
	}

	if _, err := fs.Stat(srcDir); err != nil {
		logger.Debug("No data found! Skipping!", "src", srcDir)
		return nil
	}

	// Final walk for data directory copying
	err := afero.Walk(fs, srcDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			logger.Error("Error walking source directory", "path", srcDir, "error", walkErr)
			return walkErr
		}

		// Determine the relative path from the source directory
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			logger.Error("Failed to get relative path", "path", path, "error", err)
			return err
		}

		pathParts := strings.Split(relPath, string(os.PathSeparator))
		if len(pathParts) >= 2 {
			// Adjust destDir if agent data already exists from src path
			destDir = filepath.Join(kdepsDir, "agents", wf.GetName(), wf.GetVersion(), "data")
		}

		// Create the destination path
		dstPath := filepath.Join(destDir, relPath)

		// If it's a directory, create the directory in the destination
		if info.IsDir() {
			if err := fs.MkdirAll(dstPath, info.Mode()); err != nil {
				logger.Error("Failed to create directory", "path", dstPath, "error", err)
				return err
			}
		} else {
			// If it's a file, copy the file
			if err := CopyFile(fs, path, dstPath, logger); err != nil {
				logger.Error("Failed to copy file", "src", path, "dst", dstPath, "error", err)
				return err
			}
		}

		return nil
	})

	if err != nil {
		logger.Error("Error copying directory", "srcDir", srcDir, "destDir", destDir, "error", err)
		return err
	}

	logger.Debug("Directory copied successfully", "srcDir", srcDir, "destDir", destDir)
	return nil
}
