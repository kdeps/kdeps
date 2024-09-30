package archiver

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/md5"
	"encoding/hex"
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

type KdepsPackage struct {
	PkgFilePath string                         `json:"pkgFilePath"` // THe path to the kdeps package file
	Md5sum      string                         `json:"md5sum"`      // The package.kdeps md5sum signature
	Workflow    string                         `json:"workflow"`    // Absolute path to workflow.pkl
	Resources   []string                       `json:"resources"`   // Absolute paths to resource files
	Data        map[string]map[string][]string `json:"data"`        // Data[agentName][version] -> slice of absolute file paths for a specific agent version
}

func ExtractPackage(fs afero.Fs, ctx context.Context, kdepsDir string, kdepsPackage string) (*KdepsPackage, error) {
	logging.Info("Starting extraction of package", "package", kdepsPackage)

	// Create a temporary directory for extraction
	tempDir, err := afero.TempDir(fs, "", "kdeps")
	if err != nil {
		logging.Error("Failed to create temporary directory for package extraction", "package", kdepsPackage)
		return nil, err
	}

	// Ensure the temporary directory exists
	err = fs.MkdirAll(tempDir, 0777)
	if err != nil {
		logging.Error("Failed to create temporary directory", "directory", tempDir, "error", err)
		return nil, fmt.Errorf("failed to create temporary directory: %w", err)
	}

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

	// Extract the contents into the tempDir
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			logging.Error("Failed to read tar header", "error", err)
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
				logging.Error("Failed to create directory", "directory", targetPath, "error", err)
				return nil, fmt.Errorf("failed to create directory: %w", err)
			}
		case tar.TypeReg:
			// Create parent directories
			err = fs.MkdirAll(parentDir, 0777)
			if err != nil {
				logging.Error("Failed to create parent directories", "directory", parentDir, "error", err)
				return nil, fmt.Errorf("failed to create parent directories: %w", err)
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

			// Set file permissions
			err = fs.Chmod(targetPath, 0666)
			if err != nil {
				logging.Error("Failed to set file permissions", "file", targetPath, "error", err)
				return nil, fmt.Errorf("failed to set file permissions: %w", err)
			}
		}
	}

	// Load the workflow configuration file (assumed to be in tempDir)
	wfTmpFile := filepath.Join(tempDir, "workflow.pkl")
	wfConfig, err := workflow.LoadWorkflow(ctx, wfTmpFile)
	if err != nil {
		logging.Error("Failed to load the workflow file", "file", wfTmpFile, "error", err)
		return nil, fmt.Errorf("failed to load workflow file: %w", err)
	}

	// Extract the workflow name and version
	agentName := wfConfig.Name
	agentVersion := wfConfig.Version

	// Move the extracted files from the temporary directory to the permanent location
	extractBasePath := filepath.Join(kdepsDir, "agents", agentName, agentVersion)
	if err := MoveFolder(fs, tempDir, extractBasePath); err != nil {
		logging.Error("Failed to move extracted package to kdeps system directory", "kdepsDir", kdepsDir, "extractBasePath", extractBasePath)
		return nil, err
	}

	baseFilename := filepath.Base(kdepsPackage)

	// Copy the kdepsPackage to kdepsDir/packages
	packageDir := filepath.Join(kdepsDir, "packages")
	err = fs.MkdirAll(packageDir, 0777)
	if err != nil {
		logging.Error("Failed to create packages directory", "directory", packageDir, "error", err)
		return nil, fmt.Errorf("failed to create packages directory: %w", err)
	}

	destinationFile := filepath.Join(packageDir, baseFilename)
	sourceFile := kdepsPackage

	err = CopyFile(fs, sourceFile, destinationFile)
	if err != nil {
		logging.Error("Failed to copy kdeps package to packages directory", "error", err)
		return nil, fmt.Errorf("failed to copy kdeps package to packages directory: %w", err)
	}

	kdepsPackage = destinationFile

	runDir, err := PrepareRunDir(fs, wfConfig, kdepsDir, kdepsPackage)
	if err != nil {
		logging.Error("Failed to prepare runtime directory", "runDir", runDir, "error", err)
		return nil, err
	}

	// Now walk the extractBasePath directory to populate the KdepsPackage
	kdeps := &KdepsPackage{
		Resources: []string{},
		Data:      make(map[string]map[string][]string),
	}

	err = afero.Walk(fs, extractBasePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logging.Error("Error walking through directory", "path", path, "error", err)
			return err
		}

		// Get the absolute path for each file
		absPath, err := filepath.Abs(path)
		if err != nil {
			logging.Error("Failed to get absolute path", "path", path, "error", err)
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
		logging.Error("Error populating KdepsPackage from directory walk", "extractBasePath", extractBasePath, "error", err)
		return nil, err
	}

	// Get the MD5 hash of the file
	md5Hash, err := getFileMD5(fs, kdepsPackage, 5)
	if err != nil {
		logging.Error("Error calculating MD5:", err)
		return nil, err
	}

	// Set additional fields in KdepsPackage
	kdeps.PkgFilePath = kdepsPackage
	kdeps.Md5sum = md5Hash

	logging.Info("Extraction and population completed successfully", "package", kdepsPackage)

	return kdeps, nil
}

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

// Function to compare version numbers
func compareVersions(versions []string) string {
	logging.Info("Comparing versions", "versions", versions)
	sort.Slice(versions, func(i, j int) bool {
		// Split the version strings into parts
		v1 := strings.Split(versions[i], ".")
		v2 := strings.Split(versions[j], ".")

		// Compare each part of the version (major, minor, patch)
		for k := 0; k < len(v1); k++ {
			if v1[k] != v2[k] {
				result := v1[k] > v2[k]
				logging.Info("Version comparison result", "v1", v1, "v2", v2, "result", result)
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
			logging.Info("Found version directory", "directory", info.Name())
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

func PrepareRunDir(fs afero.Fs, wf *pklWf.Workflow, kdepsDir, pkgFilePath string) (string, error) {
	agentName, agentVersion := wf.Name, wf.Version

	runDir := filepath.Join(kdepsDir, "run/"+agentName+"/"+agentVersion+"/workflow")

	// Create the directory if it doesn't exist
	if err := fs.MkdirAll(runDir, 0755); err != nil {
		return "", err
	}

	if err := CopyFile(fs, "../../build/linux/arm64/kdeps", filepath.Join(runDir, "kdeps")); err != nil {
		return "", err
	}

	if _, err := fs.Stat(pkgFilePath); err != nil {
		logging.Error("Package not found!", "package", pkgFilePath)
		return "", err
	}

	file, err := os.Open(pkgFilePath)
	if err != nil {
		logging.Error("Error opening file: %v\n", err)
		return "", err
	}
	defer file.Close()

	// Open the gzip reader
	gzr, err := gzip.NewReader(file)
	if err != nil {
		logging.Error("Error creating gzip reader: %v\n", err)
		return "", err
	}
	defer gzr.Close()

	// Open the tar reader
	tarReader := tar.NewReader(gzr)

	// Extract all the files
	for {
		// Get the next header in the tar file
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			logging.Error("Error reading tar file: %v\n", err)
			return "", err
		}

		// Create the full path for the file to extract
		target := filepath.Join(runDir, header.Name)

		// Handle file types (file, directory, etc.)
		switch header.Typeflag {
		case tar.TypeDir:
			// Create directory
			if err := os.MkdirAll(target, os.ModePerm); err != nil {
				logging.Error("Error creating directory: %v\n", err)
				return "", err
			}
		case tar.TypeReg:
			// Extract file
			if err := os.MkdirAll(filepath.Dir(target), os.ModePerm); err != nil {
				logging.Error("Error creating file directory: %v\n", err)
				return "", err
			}
			outFile, err := os.Create(target)
			if err != nil {
				logging.Error("Error creating file: %v\n", err)
				return "", err
			}
			defer outFile.Close()

			// Copy file contents
			if _, err := io.Copy(outFile, tarReader); err != nil {
				logging.Error("Error writing file: %v\n", err)
				return "", err
			}
		default:
			logging.Error("Unknown type: %v in %s\n", header.Typeflag, header.Name)
		}
	}

	logging.Info("Extraction in runtime folder completed!", runDir)

	return runDir, nil
}

// PackageProject compresses the contents of projectDir into a tar.gz file in kdepsDir
func PackageProject(fs afero.Fs, wf *pklWf.Workflow, kdepsDir, compiledProjectDir string) (string, error) {
	// Enforce the folder structure
	if err := enforcer.EnforceFolderStructure(fs, compiledProjectDir); err != nil {
		logging.Error("Failed to enforce folder structure", "error", err)
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

// CompileWorkflow compiles a workflow file and updates the action field
func CompileWorkflow(fs afero.Fs, wf *pklWf.Workflow, kdepsDir, projectDir string) (string, error) {
	action := wf.Action

	if action == "" {
		logging.Error("No action specified in workflow!")
		return "", errors.New("Action is required! Please specify the default action in the workflow!")
	}

	var compiledAction string

	name := wf.Name
	version := wf.Version

	filePath := filepath.Join(projectDir, "workflow.pkl")
	agentDir := filepath.Join(kdepsDir, fmt.Sprintf("agents/%s/%s", name, version))
	resourcesDir := filepath.Join(agentDir, "resources")
	compiledFilePath := filepath.Join(agentDir, "workflow.pkl")

	re := regexp.MustCompile(`^@`)

	if !re.MatchString(action) {
		compiledAction = fmt.Sprintf("@%s/%s:%s", name, action, version)
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

// Move the original file to a new name with MD5 and copy the latest file
func CopyFile(fs afero.Fs, src, dst string) error {
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

	fmt.Println("File copied successfully:", src, "to", dst)
	return nil
}

func CopyDir(fs afero.Fs, wf *pklWf.Workflow, kdepsDir, projectDir, compiledProjectDir, agentName, agentVersion, agentAction string, processWorkflows bool) error {
	var srcDir, destDir string

	srcDir = filepath.Join(projectDir, "data")
	destDir = filepath.Join(compiledProjectDir, fmt.Sprintf("data/%s/%s", wf.Name, wf.Version))

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
					if err := CopyFile(fs, path, dstPath); err != nil {
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
			if err := CopyFile(fs, path, dstPath); err != nil {
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
func CompileProject(fs afero.Fs, ctx context.Context, wf *pklWf.Workflow, kdepsDir string, projectDir string) (string, string, error) {
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
	newWorkflow, err := workflow.LoadWorkflow(ctx, newWorkflowFile)
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
	if err := ProcessExternalWorkflows(fs, newWorkflow, kdepsDir, projectDir, compiledProjectDir); err != nil {
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
		if err := enforcer.EnforcePklTemplateAmendsRules(fs, pklFile); err != nil {
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
			logging.Info("Processing .pkl", "file", file)
			if err := processResourcePklFiles(fs, file, wf, resourcesDir); err != nil {
				logging.Error("Failed to process .pkl file", "file", file, "error", err)
				return err
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
	name, version := wf.Name, wf.Version

	readFile, err := fs.Open(file)
	if err != nil {
		logging.Error("Failed to open file", "file", file, "error", err)
		return err
	}
	defer readFile.Close()

	var fileBuffer bytes.Buffer
	scanner := bufio.NewScanner(readFile)

	// Define regex patterns for exec, chat, client with actionID, and id replacement
	idPattern := regexp.MustCompile(`(?i)^\s*id\s*=\s*"(.+)"`)
	// Pattern to capture lines like exec["actionID"], chat["actionID"], client["actionID"]
	actionIDPattern := regexp.MustCompile(`(?i)(stdout|stderr|chat|client)\["(.+)"\]`)

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
		} else if actionIDMatch := actionIDPattern.FindStringSubmatch(line); actionIDMatch != nil {
			// Extract the block type (exec, chat, client) and the actionID
			blockType := actionIDMatch[1]
			field := actionIDMatch[2]

			// Only modify if actionID does not already start with "@"
			if !strings.HasPrefix(field, "@") {
				// Prefix and append name and version to the actionID in the format @name/actionID:version
				modifiedField := fmt.Sprintf("%s[\"@%s/%s:%s\"]", blockType, name, field, version)
				// Replace the original field with the modified one
				newLine := strings.Replace(line, actionIDMatch[0], modifiedField, 1)
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
	name, version := wf.Name, wf.Version

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

// ProcessExternalWorkflows processes each workflow and copies directories as needed
func ProcessExternalWorkflows(fs afero.Fs, wf *pklWf.Workflow, kdepsDir, projectDir, compiledProjectDir string) error {
	if wf.Workflows == nil {
		logging.Info("No external workflows to process")
		return nil
	}

	for _, value := range wf.Workflows {
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

	logging.Info("Processed all external workflows")
	return nil
}
