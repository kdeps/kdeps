package archiver

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/messages"
	pklWf "github.com/kdeps/schema/gen/workflow"
	"github.com/spf13/afero"
)

// MoveFolder moves a directory by copying its contents and then deleting the original.
func MoveFolder(fs afero.Fs, src, dest string) error {
	err := afero.Walk(fs, src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(dest, relPath)

		if info.IsDir() {
			return fs.MkdirAll(destPath, info.Mode())
		}

		if err := copyFile(fs, path, destPath); err != nil {
			return err
		}

		return fs.Remove(path)
	})
	if err != nil {
		return err
	}

	return fs.RemoveAll(src)
}

// copyFile copies a file from src to dst using the provided filesystem.
func copyFile(fs afero.Fs, src, dst string) error {
	srcFile, err := fs.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := fs.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	return err
}

// GetFileMD5 calculates the MD5 hash of a file and returns a truncated version.
func GetFileMD5(fs afero.Fs, filePath string, length int) (string, error) {
	file, err := fs.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	hashInBytes := hash.Sum(nil)
	md5String := hex.EncodeToString(hashInBytes)

	if length > len(md5String) {
		length = len(md5String)
	}
	return md5String[:length], nil
}

// CopyFile copies a file from src to dst, handling existing files by creating backups.
func CopyFile(fs afero.Fs, ctx context.Context, src, dst string, logger *logging.Logger) error {
	exists, err := afero.Exists(fs, dst)
	if err != nil {
		return fmt.Errorf("failed to check destination existence: %w", err)
	}

	if exists {
		srcMD5, err := GetFileMD5(fs, src, 8)
		if err != nil {
			return fmt.Errorf("failed to calculate MD5 for source file: %w", err)
		}

		dstMD5, err := GetFileMD5(fs, dst, 8)
		if err != nil {
			return fmt.Errorf("failed to calculate MD5 for destination file: %w", err)
		}

		if srcMD5 == dstMD5 {
			logger.Info("files have the same MD5, skipping copy", "src", src, "dst", dst)
			return nil
		}

		backupPath := getBackupPath(dst, dstMD5)
		logger.Debug(messages.MsgMovingExistingToBackup, "backupPath", backupPath)
		if err := fs.Rename(dst, backupPath); err != nil {
			return fmt.Errorf("failed to move file to backup: %w", err)
		}
	}

	if err := performCopy(fs, src, dst); err != nil {
		return err
	}

	if err := setPermissions(fs, src, dst); err != nil {
		return err
	}

	logger.Debug(messages.MsgFileCopiedSuccessfully, "from", src, "to", dst)
	return nil
}

func getBackupPath(dst, dstMD5 string) string {
	ext := filepath.Ext(dst)
	baseName := strings.TrimSuffix(filepath.Base(dst), ext)
	return filepath.Join(filepath.Dir(dst), fmt.Sprintf("%s_%s%s", baseName, dstMD5, ext))
}

func performCopy(fs afero.Fs, src, dst string) error {
	srcFile, err := fs.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := fs.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	if _, err = io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}
	return nil
}

func setPermissions(fs afero.Fs, src, dst string) error {
	srcInfo, err := fs.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}

	if err = fs.Chmod(dst, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to change permissions on destination file: %w", err)
	}
	return nil
}

// CopyDataDir copies data directories, handling workflows and resources.
func CopyDataDir(fs afero.Fs, ctx context.Context, wf pklWf.Workflow, kdepsDir, projectDir, compiledProjectDir, agentName, agentVersion,
	agentAction string, processWorkflows bool, logger *logging.Logger,
) error {
	srcDir := filepath.Join(projectDir, "data")
	destDir := filepath.Join(compiledProjectDir, fmt.Sprintf("data/%s/%s", wf.GetName(), wf.GetVersion()))

	if processWorkflows {
		newSrcDir, newDestDir, err := ResolveAgentVersionAndCopyResources(fs, ctx, kdepsDir, compiledProjectDir, agentName, agentVersion, logger)
		if err != nil {
			return err
		}
		srcDir, destDir = newSrcDir, newDestDir
	}

	if _, err := fs.Stat(srcDir); err != nil {
		logger.Debug(messages.MsgNoDataFoundSkipping, "src", srcDir, "error", err)
		return nil
	}

	return CopyDir(fs, ctx, srcDir, destDir, logger)
}

func ResolveAgentVersionAndCopyResources(fs afero.Fs, ctx context.Context, kdepsDir, compiledProjectDir, agentName, agentVersion string, logger *logging.Logger) (string, string, error) {
	if agentVersion == "" {
		agentVersionPath := filepath.Join(kdepsDir, "agents", agentName)
		exists, err := afero.Exists(fs, agentVersionPath)
		if err != nil {
			return "", "", err
		}
		if exists {
			version, err := GetLatestVersion(agentVersionPath, logger)
			if err != nil {
				logger.Error("failed to get latest agent version", "agentVersionPath", agentVersionPath, "error", err)
				return "", "", err
			}
			agentVersion = version
		}
	}

	src := filepath.Join(kdepsDir, "agents", agentName, agentVersion, "resources")
	dst := filepath.Join(compiledProjectDir, "resources")

	exists, err := afero.Exists(fs, src)
	if err != nil {
		return "", "", err
	}
	if exists {
		if err := CopyDir(fs, ctx, src, dst, logger); err != nil {
			logger.Error("failed to copy resources", "src", src, "dst", dst, "error", err)
			return "", "", err
		}
	}

	newSrcDir := filepath.Join(kdepsDir, "agents", agentName, agentVersion, "data", agentName, agentVersion)
	newDestDir := filepath.Join(compiledProjectDir, fmt.Sprintf("data/%s/%s", agentName, agentVersion))
	return newSrcDir, newDestDir, nil
}

func CopyDir(fs afero.Fs, ctx context.Context, srcDir, destDir string, logger *logging.Logger) error {
	return afero.Walk(fs, srcDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			logger.Error("error walking source directory", "path", srcDir, "error", walkErr)
			return walkErr
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			logger.Error("failed to get relative path", "path", path, "error", err)
			return err
		}

		dstPath := filepath.Join(destDir, relPath)

		if info.IsDir() {
			if err := fs.MkdirAll(dstPath, info.Mode()); err != nil {
				logger.Error("failed to create directory", "path", dstPath, "error", err)
				return err
			}
		} else {
			if err := CopyFile(fs, ctx, path, dstPath, logger); err != nil {
				logger.Error("failed to copy file", "src", path, "dst", dstPath, "error", err)
				return err
			}
		}
		return nil
	})
}
