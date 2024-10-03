package archiver

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"kdeps/pkg/enforcer"
	"kdeps/pkg/workflow"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	pklWf "github.com/kdeps/schema/gen/workflow"
	"github.com/spf13/afero"
)

type KdepsPackage struct {
	PkgFilePath string                         `json:"pkgFilePath"` // THe path to the kdeps package file
	Md5sum      string                         `json:"md5sum"`      // The package.kdeps md5sum signature
	Workflow    string                         `json:"workflow"`    // Absolute path to workflow.pkl
	Resources   []string                       `json:"resources"`   // Absolute paths to resource files
	Data        map[string]map[string][]string `json:"data"`        // Data[agentName][version] -> slice of absolute file paths for a specific agent version
}

func ExtractPackage(fs afero.Fs, ctx context.Context, kdepsDir string, kdepsPackage string, logger *log.Logger) (*KdepsPackage, error) {
	logger.Info("Starting extraction of package", "package", kdepsPackage)

	// Create a temporary directory for extraction
	tempDir, err := afero.TempDir(fs, "", "kdeps")
	if err != nil {
		logger.Error("Failed to create temporary directory for package extraction", "package", kdepsPackage)
		return nil, err
	}

	// Ensure the temporary directory exists
	err = fs.MkdirAll(tempDir, 0777)
	if err != nil {
		logger.Error("Failed to create temporary directory", "directory", tempDir, "error", err)
		return nil, fmt.Errorf("failed to create temporary directory: %w", err)
	}

	// Open the tar.gz file
	file, err := fs.Open(kdepsPackage)
	if err != nil {
		logger.Error("Failed to open tar.gz file", "error", err)
		return nil, fmt.Errorf("failed to open tar.gz file: %w", err)
	}
	defer file.Close()

	// Create a gzip reader
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		logger.Error("Failed to create gzip reader", "error", err)
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	// Create a tar reader
	tarReader := tar.NewReader(gzipReader)

	// Extract the contents into the tempDir
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			logger.Error("Failed to read tar header", "error", err)
			return nil, fmt.Errorf("failed to read tar header: %w", err)
		}

		// Construct the full absolute path for the file in the temp directory
		targetPath := filepath.Join(tempDir, header.Name)
		parentDir := filepath.Dir(targetPath)

		switch header.Typeflag {
		case tar.TypeDir:
			// Create directories
			err = fs.MkdirAll(targetPath, 0777)
			if err != nil {
				logger.Error("Failed to create directory", "directory", targetPath, "error", err)
				return nil, fmt.Errorf("failed to create directory: %w", err)
			}
		case tar.TypeReg:
			// Create parent directories
			err = fs.MkdirAll(parentDir, 0777)
			if err != nil {
				logger.Error("Failed to create parent directories", "directory", parentDir, "error", err)
				return nil, fmt.Errorf("failed to create parent directories: %w", err)
			}

			// Extract the file
			outFile, err := fs.Create(targetPath)
			if err != nil {
				logger.Error("Failed to create file", "file", targetPath, "error", err)
				return nil, fmt.Errorf("failed to create file: %w", err)
			}
			defer outFile.Close()

			// Copy the file contents
			_, err = io.Copy(outFile, tarReader)
			if err != nil {
				logger.Error("Failed to copy file contents", "file", targetPath, "error", err)
				return nil, fmt.Errorf("failed to copy file contents: %w", err)
			}

			// Set file permissions
			err = fs.Chmod(targetPath, 0666)
			if err != nil {
				logger.Error("Failed to set file permissions", "file", targetPath, "error", err)
				return nil, fmt.Errorf("failed to set file permissions: %w", err)
			}
		}
	}

	// Load the workflow configuration file (assumed to be in tempDir)
	wfTmpFile := filepath.Join(tempDir, "workflow.pkl")
	wfConfig, err := workflow.LoadWorkflow(ctx, wfTmpFile, logger)
	if err != nil {
		logger.Error("Failed to load the workflow file", "file", wfTmpFile, "error", err)
		return nil, fmt.Errorf("failed to load workflow file: %w", err)
	}

	// Extract the workflow name and version
	agentName := wfConfig.Name
	agentVersion := wfConfig.Version

	// Move the extracted files from the temporary directory to the permanent location
	extractBasePath := filepath.Join(kdepsDir, "agents", agentName, agentVersion)
	if err := MoveFolder(fs, tempDir, extractBasePath); err != nil {
		logger.Error("Failed to move extracted package to kdeps system directory", "kdepsDir", kdepsDir, "extractBasePath", extractBasePath)
		return nil, err
	}

	baseFilename := filepath.Base(kdepsPackage)

	// Copy the kdepsPackage to kdepsDir/packages
	packageDir := filepath.Join(kdepsDir, "packages")
	err = fs.MkdirAll(packageDir, 0777)
	if err != nil {
		logger.Error("Failed to create packages directory", "directory", packageDir, "error", err)
		return nil, fmt.Errorf("failed to create packages directory: %w", err)
	}

	destinationFile := filepath.Join(packageDir, baseFilename)
	sourceFile := kdepsPackage

	err = CopyFile(fs, sourceFile, destinationFile)
	if err != nil {
		logger.Error("Failed to copy kdeps package to packages directory", "error", err)
		return nil, fmt.Errorf("failed to copy kdeps package to packages directory: %w", err)
	}

	kdepsPackage = destinationFile

	runDir, err := PrepareRunDir(fs, wfConfig, kdepsDir, kdepsPackage, logger)
	if err != nil {
		logger.Error("Failed to prepare runtime directory", "runDir", runDir, "error", err)
		return nil, err
	}

	// Now walk the extractBasePath directory to populate the KdepsPackage
	kdeps := &KdepsPackage{
		Resources: []string{},
		Data:      make(map[string]map[string][]string),
	}

	err = afero.Walk(fs, extractBasePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logger.Error("Error walking through directory", "path", path, "error", err)
			return err
		}

		// Get the absolute path for each file
		absPath, err := filepath.Abs(path)
		if err != nil {
			logger.Error("Failed to get absolute path", "path", path, "error", err)
			return err
		}

		// Populate based on the file type and name
		relativePath, _ := filepath.Rel(extractBasePath, path)

		switch {
		case strings.HasSuffix(relativePath, "workflow.pkl"):
			kdeps.Workflow = absPath

		case strings.HasPrefix(relativePath, "resources/") && strings.HasSuffix(relativePath, ".pkl"):
			kdeps.Resources = append(kdeps.Resources, absPath)

		case strings.HasPrefix(relativePath, "data/"):
			parts := strings.Split(relativePath, string(os.PathSeparator))
			if len(parts) >= 3 {
				dataAgentName := parts[1]
				dataVersion := parts[2]

				// Initialize map for agent if not exists
				if kdeps.Data[dataAgentName] == nil {
					kdeps.Data[dataAgentName] = make(map[string][]string)
				}

				// Append the file to the corresponding agent version
				kdeps.Data[dataAgentName][dataVersion] = append(kdeps.Data[dataAgentName][dataVersion], absPath)
			}
		}

		return nil
	})

	if err != nil {
		logger.Error("Error populating KdepsPackage from directory walk", "extractBasePath", extractBasePath, "error", err)
		return nil, err
	}

	// Get the MD5 hash of the file
	md5Hash, err := getFileMD5(fs, kdepsPackage, 5)
	if err != nil {
		logger.Error("Error calculating MD5:", err)
		return nil, err
	}

	// Set additional fields in KdepsPackage
	kdeps.PkgFilePath = kdepsPackage
	kdeps.Md5sum = md5Hash

	logger.Info("Extraction and population completed successfully", "package", kdepsPackage)

	return kdeps, nil
}

// PackageProject compresses the contents of projectDir into a tar.gz file in kdepsDir
func PackageProject(fs afero.Fs, wf *pklWf.Workflow, kdepsDir, compiledProjectDir string, logger *log.Logger) (string, error) {
	// Enforce the folder structure
	if err := enforcer.EnforceFolderStructure(fs, compiledProjectDir, logger); err != nil {
		logger.Error("Failed to enforce folder structure", "error", err)
		return "", err
	}

	// Create the output filename for the package
	outFile := fmt.Sprintf("%s-%s.kdeps", wf.Name, wf.Version)
	packageDir := fmt.Sprintf("%s/packages", kdepsDir)

	if _, err := fs.Stat(packageDir); err != nil {
		if err := fs.MkdirAll(packageDir, 0777); err != nil {
			return "", fmt.Errorf("error creating the system packages folder: %s", packageDir)
		}
	}

	// Define the output file path for the tarball
	tarGzPath := filepath.Join(packageDir, outFile)

	// Check if the tar.gz file already exists, and if so, delete it
	exists, err := afero.Exists(fs, tarGzPath)
	if err != nil {
		logger.Error("Error checking if package exists", "path", tarGzPath, "error", err)
		return "", fmt.Errorf("error checking if package exists: %w", err)
	}

	if exists {
		if err := fs.Remove(tarGzPath); err != nil {
			logger.Error("Failed to remove existing package file", "path", tarGzPath, "error", err)
			return "", fmt.Errorf("failed to remove existing package file: %w", err)
		}
	}

	// Create the tar.gz file in kdepsDir
	tarFile, err := fs.Create(tarGzPath)
	if err != nil {
		logger.Error("Failed to create package file", "path", tarGzPath, "error", err)
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
	err = afero.Walk(fs, compiledProjectDir, func(file string, info os.FileInfo, err error) error {
		if err != nil {
			logger.Error("Error walking the file tree", "path", file, "error", err)
			return fmt.Errorf("error walking the file tree: %w", err)
		}

		// Skip directories, only process files
		if info.IsDir() {
			return nil
		}

		// Open the file to read its contents
		fileHandle, err := fs.Open(file)
		if err != nil {
			logger.Error("Failed to open file", "file", file, "error", err)
			return fmt.Errorf("failed to open file %s: %w", file, err)
		}
		defer fileHandle.Close()

		// Get relative path for tar header
		relPath, err := filepath.Rel(compiledProjectDir, file)
		if err != nil {
			logger.Error("Failed to get relative file path", "file", file, "error", err)
			return fmt.Errorf("failed to get relative file path: %w", err)
		}

		// Create tar header for the file
		header, err := tar.FileInfoHeader(info, relPath)
		if err != nil {
			logger.Error("Failed to create tar header", "file", file, "error", err)
			return fmt.Errorf("failed to create tar header for %s: %w", file, err)
		}
		header.Name = strings.ReplaceAll(relPath, "\\", "/") // Normalize path to use forward slashes

		// Write the header
		if err := tarWriter.WriteHeader(header); err != nil {
			logger.Error("Failed to write tar header", "file", file, "error", err)
			return fmt.Errorf("failed to write tar header for %s: %w", file, err)
		}

		// Copy the file data to the tar writer
		if _, err := io.Copy(tarWriter, fileHandle); err != nil {
			logger.Error("Failed to copy file contents", "file", file, "error", err)
			return fmt.Errorf("failed to copy file contents for %s: %w", file, err)
		}

		return nil
	})

	if err != nil {
		logger.Error("Error packaging project", "error", err)
		return "", fmt.Errorf("error packaging project: %w", err)
	}

	// Log successful packaging
	logger.Info("Project packaged successfully", "path", tarGzPath)

	// Return the path to the generated tar.gz file
	return tarGzPath, nil
}
