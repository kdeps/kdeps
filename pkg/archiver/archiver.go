package archiver

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"kdeps/pkg/enforcer"
	"kdeps/pkg/logging"
	"kdeps/pkg/workflow"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	pklWf "github.com/kdeps/schema/gen/workflow"

	"github.com/spf13/afero"
)

var schemaVersionFilePath = "../../SCHEMA_VERSION"

type KdepsPackage struct {
	Workflow  string                         `json:"workflow"`  // Absolute path to workflow.pkl
	Resources []string                       `json:"resources"` // Absolute paths to resource files
	Data      map[string]map[string][]string `json:"data"`      // Data[agentName][version] -> slice of absolute file paths for a specific agent version
}

func ExtractPackage(fs afero.Fs, kdepsDir string, kdepsPackage string) (*KdepsPackage, error) {
	logging.Info("Starting extraction of package", "package", kdepsPackage)

	// Enforce the filename convention using a regular expression
	filenamePattern := regexp.MustCompile(`^([a-zA-Z]+)-([\d]+\.[\d]+\.[\d]+)\.kdeps$`)
	baseFilename := filepath.Base(kdepsPackage)

	// Validate the filename and extract agent name and version
	matches := filenamePattern.FindStringSubmatch(baseFilename)
	if matches == nil {
		logging.Error("Invalid archive filename", "filename", baseFilename)
		return nil, fmt.Errorf("invalid archive filename: %s (expected format: name-version.kdeps)", baseFilename)
	}

	agentName := matches[1]
	version := matches[2]

	logging.Debug("Extracted agent name and version", "agentName", agentName, "version", version)

	// Define the base extraction path for this agent and version
	extractBasePath := filepath.Join(kdepsDir, "agents", agentName, version)

	// Open the tar.gz file
	file, err := fs.Open(kdepsPackage)
	if err != nil {
		logging.Error("Failed to open tar.gz file", "error", err)
		return nil, fmt.Errorf("failed to open tar.gz file: %w", err)
	}
	defer file.Close()

	// Create a gzip reader
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		logging.Error("Failed to create gzip reader", "error", err)
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	// Create a tar reader
	tarReader := tar.NewReader(gzipReader)

	// Initialize the KdepsPackage struct
	kdeps := &KdepsPackage{
		Resources: []string{},
		Data:      make(map[string]map[string][]string),
	}

	logging.Debug("Initialized KdepsPackage struct")

	// Iterate through the files in the tar archive
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			logging.Error("Failed to read tar header", "error", err)
			return nil, fmt.Errorf("failed to read tar header: %w", err)
		}

		// Construct the full absolute path for the file
		targetPath := filepath.Join(extractBasePath, header.Name)
		absPath, err := filepath.Abs(targetPath)
		if err != nil {
			logging.Error("Failed to get absolute path", "error", err)
			return nil, fmt.Errorf("failed to get absolute path: %w", err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			logging.Debug("Extracting directory", "directory", targetPath)

			// Recreate the directory (remove it if it exists, then create)
			if _, err := fs.Stat(targetPath); !os.IsNotExist(err) {
				err = fs.RemoveAll(targetPath)
				if err != nil {
					logging.Error("Failed to remove existing directory", "directory", targetPath, "error", err)
					return nil, fmt.Errorf("failed to remove existing directory: %w", err)
				}
			}
			// Create the directory with more permissive permissions
			err = fs.MkdirAll(targetPath, 0777)
			if err != nil {
				logging.Error("Failed to create directory", "directory", targetPath, "error", err)
				return nil, fmt.Errorf("failed to create directory: %w", err)
			}

		case tar.TypeReg:
			logging.Debug("Extracting file", "file", targetPath)

			// Create parent directories if they don't exist
			parentDir := filepath.Dir(targetPath)
			err = fs.MkdirAll(parentDir, 0777)
			if err != nil {
				logging.Error("Failed to create parent directories", "directory", parentDir, "error", err)
				return nil, fmt.Errorf("failed to create parent directories: %w", err)
			}

			// Handle workflow and resource files
			if strings.HasSuffix(header.Name, "workflow.pkl") {
				kdeps.Workflow = absPath
				logging.Info("Found workflow file", "path", absPath)
			} else if strings.HasPrefix(header.Name, "resources/") && strings.HasSuffix(header.Name, ".pkl") {
				kdeps.Resources = append(kdeps.Resources, absPath)
				logging.Info("Found resource file", "path", absPath)
			} else if strings.HasPrefix(header.Name, "data/") {
				// Extract agentName and version from the data file path
				parts := strings.Split(header.Name, "/")
				if len(parts) >= 3 {
					dataAgentName := parts[1]
					dataVersion := parts[2]

					// Initialize map for agent if not exists
					if kdeps.Data[dataAgentName] == nil {
						kdeps.Data[dataAgentName] = make(map[string][]string)
					}

					// Append the file to the corresponding agent version
					kdeps.Data[dataAgentName][dataVersion] = append(kdeps.Data[dataAgentName][dataVersion], absPath)
					logging.Debug("Added data file", "agentName", dataAgentName, "version", dataVersion, "path", absPath)
				}
			}

			// Extract the file
			outFile, err := fs.Create(targetPath)
			if err != nil {
				logging.Error("Failed to create file", "file", targetPath, "error", err)
				return nil, fmt.Errorf("failed to create file: %w", err)
			}
			defer outFile.Close()

			// Copy the file contents
			_, err = io.Copy(outFile, tarReader)
			if err != nil {
				logging.Error("Failed to copy file contents", "file", targetPath, "error", err)
				return nil, fmt.Errorf("failed to copy file contents: %w", err)
			}

			// Set file permissions to more permissive ones (e.g., 0666 or 0777)
			err = fs.Chmod(targetPath, 0666)
			if err != nil {
				logging.Error("Failed to set file permissions", "file", targetPath, "error", err)
				return nil, fmt.Errorf("failed to set file permissions: %w", err)
			}
		}
	}

	// Copy the kdepsPackage to kdepsDir/packages
	packageDir := filepath.Join(kdepsDir, "packages")
	err = fs.MkdirAll(packageDir, 0777)
	if err != nil {
		logging.Error("Failed to create packages directory", "directory", packageDir, "error", err)
		return nil, fmt.Errorf("failed to create packages directory: %w", err)
	}

	sourceFile, err := fs.Open(kdepsPackage)
	if err != nil {
		logging.Error("Failed to open source kdeps package", "error", err)
		return nil, fmt.Errorf("failed to open source kdeps package: %w", err)
	}
	defer sourceFile.Close()

	destinationFile, err := fs.Create(filepath.Join(packageDir, baseFilename))
	if err != nil {
		logging.Error("Failed to create destination kdeps package", "error", err)
		return nil, fmt.Errorf("failed to create destination kdeps package: %w", err)
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		logging.Error("Failed to copy kdeps package to packages directory", "error", err)
		return nil, fmt.Errorf("failed to copy kdeps package to packages directory: %w", err)
	}

	logging.Info("Extraction completed successfully", "package", kdepsPackage)
	return kdeps, nil
}

// Function to compare version numbers
func compareVersions(versions []string) string {
	logging.Debug("Comparing versions", "versions", versions)
	sort.Slice(versions, func(i, j int) bool {
		// Split the version strings into parts
		v1 := strings.Split(versions[i], ".")
		v2 := strings.Split(versions[j], ".")

		// Compare each part of the version (major, minor, patch)
		for k := 0; k < len(v1); k++ {
			if v1[k] != v2[k] {
				result := v1[k] > v2[k]
				logging.Debug("Version comparison result", "v1", v1, "v2", v2, "result", result)
				return result
			}
		}
		return false
	})

	// Return the first version (which will be the latest after sorting)
	latestVersion := versions[0]
	logging.Info("Latest version determined", "version", latestVersion)
	return latestVersion
}

func getLatestVersion(directory string) (string, error) {
	var versions []string

	// Walk through the directory to collect version names
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logging.Error("Error walking the path", "path", path, "error", err)
			return err
		}

		// Collect directory names that match the version pattern
		if info.IsDir() && strings.Count(info.Name(), ".") == 2 {
			versions = append(versions, info.Name())
			logging.Debug("Found version directory", "directory", info.Name())
		}
		return nil
	})

	if err != nil {
		logging.Error("Error while walking the directory", "directory", directory, "error", err)
		return "", err
	}

	// Check if versions were found
	if len(versions) == 0 {
		err = fmt.Errorf("no versions found")
		logging.Warn("No versions found", "directory", directory)
		return "", err
	}

	// Find the latest version
	latestVersion := compareVersions(versions)
	return latestVersion, nil
}

// PackageProject compresses the contents of projectDir into a tar.gz file in kdepsDir
func PackageProject(fs afero.Fs, wf *pklWf.Workflow, kdepsDir, compiledProjectDir string) (string, error) {
	// Enforce the folder structure
	if err := enforcer.EnforceFolderStructure(fs, compiledProjectDir); err != nil {
		logging.Error("Failed to enforce folder structure", "error", err)
		return "", err
	}

	// Create the output filename for the package
	outFile := fmt.Sprintf("%s-%s.kdeps", *wf.Name, *wf.Version)
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
		logging.Error("Error checking if package exists", "path", tarGzPath, "error", err)
		return "", fmt.Errorf("error checking if package exists: %w", err)
	}

	if exists {
		if err := fs.Remove(tarGzPath); err != nil {
			logging.Error("Failed to remove existing package file", "path", tarGzPath, "error", err)
			return "", fmt.Errorf("failed to remove existing package file: %w", err)
		}
	}

	// Create the tar.gz file in kdepsDir
	tarFile, err := fs.Create(tarGzPath)
	if err != nil {
		logging.Error("Failed to create package file", "path", tarGzPath, "error", err)
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
			logging.Error("Error walking the file tree", "path", file, "error", err)
			return fmt.Errorf("error walking the file tree: %w", err)
		}

		// Skip directories, only process files
		if info.IsDir() {
			return nil
		}

		// Open the file to read its contents
		fileHandle, err := fs.Open(file)
		if err != nil {
			logging.Error("Failed to open file", "file", file, "error", err)
			return fmt.Errorf("failed to open file %s: %w", file, err)
		}
		defer fileHandle.Close()

		// Get relative path for tar header
		relPath, err := filepath.Rel(compiledProjectDir, file)
		if err != nil {
			logging.Error("Failed to get relative file path", "file", file, "error", err)
			return fmt.Errorf("failed to get relative file path: %w", err)
		}

		// Create tar header for the file
		header, err := tar.FileInfoHeader(info, relPath)
		if err != nil {
			logging.Error("Failed to create tar header", "file", file, "error", err)
			return fmt.Errorf("failed to create tar header for %s: %w", file, err)
		}
		header.Name = strings.ReplaceAll(relPath, "\\", "/") // Normalize path to use forward slashes

		// Write the header
		if err := tarWriter.WriteHeader(header); err != nil {
			logging.Error("Failed to write tar header", "file", file, "error", err)
			return fmt.Errorf("failed to write tar header for %s: %w", file, err)
		}

		// Copy the file data to the tar writer
		if _, err := io.Copy(tarWriter, fileHandle); err != nil {
			logging.Error("Failed to copy file contents", "file", file, "error", err)
			return fmt.Errorf("failed to copy file contents for %s: %w", file, err)
		}

		return nil
	})

	if err != nil {
		logging.Error("Error packaging project", "error", err)
		return "", fmt.Errorf("error packaging project: %w", err)
	}

	// Log successful packaging
	logging.Info("Project packaged successfully", "path", tarGzPath)

	// Return the path to the generated tar.gz file
	return tarGzPath, nil
}

// CompileWorkflow compiles a workflow file and updates the action field
func CompileWorkflow(fs afero.Fs, wf *pklWf.Workflow, kdepsDir, projectDir string) (string, error) {
	action := wf.Action

	if action == nil {
		logging.Error("No action specified in workflow!")
		return "", errors.New("Action is required! Please specify the default action in the workflow!")
	}

	var compiledAction string

	name := *wf.Name
	version := *wf.Version

	filePath := filepath.Join(projectDir, "workflow.pkl")
	agentDir := filepath.Join(kdepsDir, fmt.Sprintf("agents/%s/%s", name, version))
	resourcesDir := filepath.Join(agentDir, "resources")
	compiledFilePath := filepath.Join(agentDir, "workflow.pkl")

	re := regexp.MustCompile(`^@`)

	if !re.MatchString(*action) {
		compiledAction = fmt.Sprintf("@%s/%s:%s", name, *action, version)
	}

	// Check if agentDir exists and remove it if it does
	exists, err := afero.DirExists(fs, agentDir)
	if err != nil {
		logging.Error("Error checking if agent directory exists", "path", agentDir, "error", err)
		return "", err
	}

	if exists {
		err := fs.RemoveAll(agentDir)
		if err != nil {
			logging.Error("Failed to remove existing agent directory", "path", agentDir, "error", err)
			return "", err
		}
		logging.Info("Removed existing agent directory", "path", agentDir)
	}

	// Recreate the folder
	err = fs.MkdirAll(resourcesDir, 0755) // Create the folder with read-write-execute permissions
	if err != nil {
		logging.Error("Failed to create resources directory", "path", resourcesDir, "error", err)
		return "", err
	}
	logging.Info("Created resources directory", "path", resourcesDir)

	searchPattern := `action\s*=\s*".*"`
	replaceLine := fmt.Sprintf("action = \"%s\"\n", compiledAction)

	inputFile, err := fs.Open(filePath)
	if err != nil {
		logging.Error("Failed to open workflow file", "path", filePath, "error", err)
		return "", err
	}
	defer inputFile.Close()

	var lines []string
	scanner := bufio.NewScanner(inputFile)

	// Compile the regular expression
	re = regexp.MustCompile(searchPattern)

	for scanner.Scan() {
		line := scanner.Text()

		// Check if the line matches the regular expression
		if re.MatchString(line) {
			line = replaceLine // Replace the line if it matches
			logging.Info("Updated action line", "line", line)
		}

		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		logging.Error("Error reading workflow file", "path", filePath, "error", err)
		return "", err
	}

	err = afero.WriteFile(fs, compiledFilePath, []byte(strings.Join(lines, "\n")), 0644)
	if err != nil {
		logging.Error("Failed to write compiled workflow file", "path", compiledFilePath, "error", err)
		return "", err
	}
	logging.Info("Compiled workflow file written", "path", compiledFilePath)

	compiledProjectDir := filepath.Dir(compiledFilePath)

	return compiledProjectDir, nil
}

func copyFile(fs afero.Fs, src, dst string) error {
	// Open the source file
	srcFile, err := fs.Open(src)
	if err != nil {
		logging.Error("Failed to open source file", "src", src, "error", err)
		return err
	}
	defer srcFile.Close()

	// Create the destination file
	dstFile, err := fs.Create(dst)
	if err != nil {
		logging.Error("Failed to create destination file", "dst", dst, "error", err)
		return err
	}
	defer dstFile.Close()

	// Copy the file contents from src to dst
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		logging.Error("Failed to copy file contents", "src", src, "dst", dst, "error", err)
		return err
	}

	// Optionally, you can copy the file permissions from the source
	srcInfo, err := fs.Stat(src)
	if err != nil {
		logging.Error("Failed to stat source file", "src", src, "error", err)
		return err
	}
	err = fs.Chmod(dst, srcInfo.Mode())
	if err != nil {
		logging.Error("Failed to change permissions on destination file", "dst", dst, "error", err)
		return err
	}

	logging.Info("File copied successfully", "src", src, "dst", dst)
	return nil
}

func CopyDir(fs afero.Fs, wf *pklWf.Workflow, kdepsDir, projectDir, compiledProjectDir, agentName, agentVersion, agentAction string, processWorkflows bool) error {
	var srcDir, destDir string

	srcDir = filepath.Join(projectDir, "data")
	destDir = filepath.Join(compiledProjectDir, fmt.Sprintf("data/%s/%s", *wf.Name, *wf.Version))

	if processWorkflows {
		// Helper function to copy resources
		copyResources := func(src, dst string) error {
			return afero.Walk(fs, src, func(path string, info os.FileInfo, walkErr error) error {
				if walkErr != nil {
					logging.Error("Error walking source directory", "path", src, "error", walkErr)
					return walkErr
				}

				// Determine the relative path for correct directory structure copying
				relPath, err := filepath.Rel(src, path)
				if err != nil {
					logging.Error("Failed to get relative path", "path", path, "error", err)
					return err
				}

				// Create the full destination path based on the relative path
				dstPath := filepath.Join(dst, relPath)

				if info.IsDir() {
					if err := fs.MkdirAll(dstPath, info.Mode()); err != nil {
						logging.Error("Failed to create directory", "path", dstPath, "error", err)
						return err
					}
				} else {
					if err := copyFile(fs, path, dstPath); err != nil {
						logging.Error("Failed to copy file", "src", path, "dst", dstPath, "error", err)
						return err
					}
				}
				return nil
			})
		}

		if agentVersion == "" {
			agentVersionPath := filepath.Join(kdepsDir, "agents", agentName)
			version, err := getLatestVersion(agentVersionPath)
			if err != nil {
				logging.Error("Failed to get latest agent version", "agentVersionPath", agentVersionPath, "error", err)
				return err
			}
			agentVersion = version
		}

		src := filepath.Join(kdepsDir, "agents", agentName, agentVersion, "resources")
		dst := filepath.Join(compiledProjectDir, "resources")

		srcDir = filepath.Join(kdepsDir, "agents", agentName, agentVersion, "data", agentName, agentVersion)
		destDir = filepath.Join(compiledProjectDir, fmt.Sprintf("data/%s/%s", agentName, agentVersion))

		if err := copyResources(src, dst); err != nil {
			logging.Error("Failed to copy resources", "src", src, "dst", dst, "error", err)
			return err
		}
	}

	if _, err := fs.Stat(srcDir); err != nil {
		logging.Error("No data found! Skipping!")
		return nil
	}

	// Final walk for data directory copying
	err := afero.Walk(fs, srcDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			logging.Error("Error walking source directory", "path", srcDir, "error", walkErr)
			return walkErr
		}

		// Determine the relative path from the source directory
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			logging.Error("Failed to get relative path", "path", path, "error", err)
			return err
		}

		// Create the destination path
		dstPath := filepath.Join(destDir, relPath)

		// If it's a directory, create the directory in the destination
		if info.IsDir() {
			if err := fs.MkdirAll(dstPath, info.Mode()); err != nil {
				logging.Error("Failed to create directory", "path", dstPath, "error", err)
				return err
			}
		} else {
			// If it's a file, copy the file
			if err := copyFile(fs, path, dstPath); err != nil {
				logging.Error("Failed to copy file", "src", path, "dst", dstPath, "error", err)
				return err
			}
		}

		return nil
	})

	if err != nil {
		logging.Error("Error copying directory", "srcDir", srcDir, "destDir", destDir, "error", err)
		return err
	}

	logging.Info("Directory copied successfully", "srcDir", srcDir, "destDir", destDir)
	return nil
}

// CompileProject orchestrates the compilation and packaging of a project
func CompileProject(fs afero.Fs, wf *pklWf.Workflow, kdepsDir string, projectDir string) (string, string, error) {
	// Compile the workflow
	compiledProjectDir, err := CompileWorkflow(fs, wf, kdepsDir, projectDir)
	if err != nil {
		logging.Error("Failed to compile workflow", "error", err)
		return "", "", err
	}

	// Check if the compiled project directory exists
	exists, err := afero.DirExists(fs, compiledProjectDir)
	if err != nil {
		logging.Error("Error checking if compiled project directory exists", "path", compiledProjectDir, "error", err)
		return "", "", err
	}
	if !exists {
		err = errors.New("Compiled project directory does not exist!")
		logging.Error("Compiled project directory does not exist", "path", compiledProjectDir)
		return "", "", err
	}

	// Verify the compiled workflow file
	newWorkflowFile := filepath.Join(compiledProjectDir, "workflow.pkl")
	if _, err := fs.Stat(newWorkflowFile); err != nil {
		if os.IsNotExist(err) {
			err = fmt.Errorf("No compiled workflow found at: %s", newWorkflowFile)
			logging.Error("Compiled workflow file does not exist", "path", newWorkflowFile, "error", err)
			return "", "", err
		}
		logging.Error("Error stating compiled workflow file", "path", newWorkflowFile, "error", err)
		return "", "", err
	}

	// Load the new workflow
	newWorkflow, err := workflow.LoadWorkflow(newWorkflowFile)
	if err != nil {
		logging.Error("Failed to load new workflow", "path", newWorkflowFile, "error", err)
		return "", "", err
	}

	// Compile resources
	resourcesDir := filepath.Join(compiledProjectDir, "resources")
	if err := CompileResources(fs, newWorkflow, resourcesDir, projectDir); err != nil {
		logging.Error("Failed to compile resources", "resourcesDir", resourcesDir, "projectDir", projectDir, "error", err)
		return "", "", err
	}

	// Copy the project directory
	if err := CopyDir(fs, newWorkflow, kdepsDir, projectDir, compiledProjectDir, "", "", "", false); err != nil {
		logging.Error("Failed to copy project directory", "compiledProjectDir", compiledProjectDir, "error", err)
		return "", "", err
	}

	// Process workflows
	if err := ProcessWorkflows(fs, newWorkflow, kdepsDir, projectDir, compiledProjectDir); err != nil {
		logging.Error("Failed to process workflows", "compiledProjectDir", compiledProjectDir, "error", err)
		return "", "", err
	}

	// Package the project
	packageFile, err := PackageProject(fs, newWorkflow, kdepsDir, compiledProjectDir)
	if err != nil {
		logging.Error("Failed to package project", "compiledProjectDir", compiledProjectDir, "error", err)
		return "", "", err
	}

	// Verify the package file
	if _, err := fs.Stat(packageFile); err != nil {
		if os.IsNotExist(err) {
			err = fmt.Errorf("No package file found at: %s", packageFile)
			logging.Error("Package file does not exist", "path", packageFile, "error", err)
			return "", "", err
		}
		logging.Error("Error stating package file", "path", packageFile, "error", err)
		return "", "", err
	}

	logging.Info("Kdeps package created", "package-file", packageFile)

	return compiledProjectDir, packageFile, nil
}

func CheckAndValidatePklFiles(fs afero.Fs, projectResourcesDir string) error {
	// Check if the project resources directory exists
	if _, err := fs.Stat(projectResourcesDir); err != nil {
		logging.Error("No resource directory found! Exiting!")
		return fmt.Errorf("AI agent needs to have at least 1 resource in the '%s' folder.", projectResourcesDir)
	}

	// Get the list of files in the directory
	files, err := afero.ReadDir(fs, projectResourcesDir)
	if err != nil {
		logging.Error("Error reading resource directory", "error", err)
		return fmt.Errorf("failed to read directory '%s': %v", projectResourcesDir, err)
	}

	// Filter for .pkl files
	var pklFiles []string
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".pkl" {
			pklFiles = append(pklFiles, filepath.Join(projectResourcesDir, file.Name()))
		}
	}

	// Exit if no .pkl files are found
	if len(pklFiles) == 0 {
		logging.Error("No .pkl files found in the directory! Exiting!")
		return fmt.Errorf("No .pkl files found in the '%s' folder.", projectResourcesDir)
	}

	// Validate each .pkl file
	for _, pklFile := range pklFiles {
		logging.Info("Validating .pkl file", "file", pklFile)
		if err := enforcer.EnforcePklTemplateAmendsRules(fs, pklFile, schemaVersionFilePath); err != nil {
			logging.Error("Validation failed for .pkl file", "file", pklFile, "error", err)
			return fmt.Errorf("validation failed for '%s': %v", pklFile, err)
		}
	}

	logging.Info("All .pkl files validated successfully!")
	return nil
}

// CompileResources processes .pkl files from the project directory and copies them to the resources directory
func CompileResources(fs afero.Fs, wf *pklWf.Workflow, resourcesDir string, projectDir string) error {
	projectResourcesDir := filepath.Join(projectDir, "resources")

	if err := CheckAndValidatePklFiles(fs, projectResourcesDir); err != nil {
		return err
	}

	// Walk through all files in the project directory
	err := afero.Walk(fs, projectResourcesDir, func(file string, info os.FileInfo, err error) error {
		if err != nil {
			logging.Error("Error walking project resources directory", "path", projectResourcesDir, "error", err)
			return err
		}

		// Only process .pkl files
		if filepath.Ext(file) == ".pkl" {
			logging.Info("Processing .pkl file", "file", file)
			if processErr := processResourcePklFiles(fs, file, wf, resourcesDir); processErr != nil {
				logging.Error("Failed to process .pkl file", "file", file, "error", processErr)
				return processErr
			}
		}
		return nil
	})

	if err != nil {
		logging.Error("Error compiling resources", "resourcesDir", resourcesDir, "projectDir", projectDir, "error", err)
		return err
	}

	logging.Info("Resources compiled successfully", "resourcesDir", resourcesDir, "projectDir", projectDir)
	return nil
}

// processResourcePklFiles processes a .pkl file and writes modifications to the resources directory
func processResourcePklFiles(fs afero.Fs, file string, wf *pklWf.Workflow, resourcesDir string) error {
	name, version := *wf.Name, *wf.Version

	readFile, err := fs.Open(file)
	if err != nil {
		logging.Error("Failed to open file", "file", file, "error", err)
		return err
	}
	defer readFile.Close()

	var fileBuffer bytes.Buffer
	scanner := bufio.NewScanner(readFile)

	// Define regex pattern for `id = "value"` lines
	idPattern := regexp.MustCompile(`(?i)^\s*id\s*=\s*"(.+)"`) // Matches lines with id = "value" (case-insensitive)

	inRequiresBlock := false
	var requiresBlockBuffer bytes.Buffer
	var action string

	// Read file line by line
	for scanner.Scan() {
		line := scanner.Text()

		if inRequiresBlock {
			// Check if we've reached the end of the `requires { ... }` block
			if strings.TrimSpace(line) == "}" {
				inRequiresBlock = false
				// Process the accumulated `requires` block
				modifiedBlock := handleRequiresBlock(requiresBlockBuffer.String(), wf)

				// Write the modified block and the closing `}` line
				fileBuffer.WriteString(modifiedBlock)
				fileBuffer.WriteString(line + "\n")
			} else {
				// Continue accumulating the `requires` block lines
				requiresBlockBuffer.WriteString(line + "\n")
			}
			continue
		}

		// Check if the line matches the `id = "value"` pattern
		if idMatch := idPattern.FindStringSubmatch(line); idMatch != nil {
			// Extract the action from the id
			action = idMatch[1]

			// If action doesn't already start with "@", prefix and append name and version
			if !strings.HasPrefix(action, "@") {
				newLine := strings.Replace(line, action, fmt.Sprintf("@%s/%s:%s", name, action, version), 1)
				fileBuffer.WriteString(newLine + "\n")
			} else {
				fileBuffer.WriteString(line + "\n")
			}
		} else if strings.HasPrefix(strings.TrimSpace(line), "requires {") {
			// Start of a `requires { ... }` block, set flag to accumulate lines
			inRequiresBlock = true
			requiresBlockBuffer.Reset()                  // Clear previous block data if any
			requiresBlockBuffer.WriteString(line + "\n") // Add the opening `requires {` line
		} else {
			// Write the line unchanged if no pattern matches
			fileBuffer.WriteString(line + "\n")
		}
	}

	// Write back to the file if modifications were made
	if scanner.Err() == nil {
		if action == "" {
			err = fmt.Errorf("no valid action found in file: %s", file)
			logging.Error("No valid action found in file", "file", file, "error", err)
			return err
		}
		fname := fmt.Sprintf("%s_%s-%s.pkl", name, action, version)
		err = afero.WriteFile(fs, filepath.Join(resourcesDir, fname), fileBuffer.Bytes(), os.FileMode(0644))
		if err != nil {
			logging.Error("Error writing file", "file", fname, "error", err)
			return fmt.Errorf("error writing file: %w", err)
		}
	}

	if err := scanner.Err(); err != nil {
		logging.Error("Error reading file", "file", file, "error", err)
		return err
	}

	logging.Info("Processed .pkl file", "file", file)
	return nil
}

// Handle the values inside the requires { ... } block
func handleRequiresBlock(blockContent string, wf *pklWf.Workflow) string {
	name, version := *wf.Name, *wf.Version

	// Split the block by newline and process each value
	lines := strings.Split(blockContent, "\n")
	var modifiedLines []string

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		// If the line contains a value and does not start with "@", modify it
		if strings.HasPrefix(trimmedLine, `"`) && !strings.HasPrefix(trimmedLine, `"@`) {
			// Extract the value between the quotes
			value := strings.Trim(trimmedLine, `"`)

			// Add "@" to the agent name, "/" before the value, and ":" before the version
			modifiedValue := fmt.Sprintf(`"@%s/%s:%s"`, name, value, version)

			// Append the modified value
			modifiedLines = append(modifiedLines, modifiedValue)
		} else {
			// Keep the line as is if it starts with "@" or does not match the pattern
			modifiedLines = append(modifiedLines, trimmedLine)
		}
	}

	// Join the modified lines back together with newlines
	return strings.Join(modifiedLines, "\n")
}

// ProcessWorkflows processes each workflow and copies directories as needed
func ProcessWorkflows(fs afero.Fs, wf *pklWf.Workflow, kdepsDir, projectDir, compiledProjectDir string) error {
	if wf.Workflows == nil {
		logging.Info("No workflows to process")
		return nil
	}

	for _, value := range *wf.Workflows {
		// Remove the "@" at the beginning if it exists
		value = strings.TrimPrefix(value, "@")

		// Check if the string contains ":"
		if strings.Contains(value, ":") {
			// Split into agentName and version by colon ":"
			parts := strings.SplitN(value, ":", 2)
			agentAndAction := strings.SplitN(parts[0], "/", 2) // Split the agent and action by "/"

			agentName := agentAndAction[0]
			version := parts[1]

			if len(agentAndAction) == 2 {
				action := agentAndAction[1]

				if err := CopyDir(fs, wf, kdepsDir, projectDir, compiledProjectDir, agentName, version, action, true); err != nil {
					logging.Error("Failed to copy directory", "agentName", agentName, "version", version, "action", action, "error", err)
					return err
				}
			} else {
				if err := CopyDir(fs, wf, kdepsDir, projectDir, compiledProjectDir, agentName, version, "", true); err != nil {
					logging.Error("Failed to copy directory", "agentName", agentName, "version", version, "error", err)
					return err
				}
			}

		} else {
			// No version present, check if there is an action
			agentAndAction := strings.SplitN(value, "/", 2)
			agentName := agentAndAction[0]

			if len(agentAndAction) == 2 {
				action := agentAndAction[1]
				if err := CopyDir(fs, wf, kdepsDir, projectDir, compiledProjectDir, agentName, "", action, true); err != nil {
					logging.Error("Failed to copy directory", "agentName", agentName, "action", action, "error", err)
					return err
				}
			} else {
				if err := CopyDir(fs, wf, kdepsDir, projectDir, compiledProjectDir, agentName, "", "", true); err != nil {
					logging.Error("Failed to copy directory", "agentName", agentName, "error", err)
					return err
				}
			}
		}
	}

	logging.Info("Processed all workflows")
	return nil
}
