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
	"kdeps/pkg/workflow"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/charmbracelet/log"
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
	// Enforce the filename convention using a regular expression
	filenamePattern := regexp.MustCompile(`^([a-zA-Z]+)-([\d]+\.[\d]+\.[\d]+)\.kdeps$`)
	baseFilename := filepath.Base(kdepsPackage)

	// Validate the filename and extract agent name and version
	matches := filenamePattern.FindStringSubmatch(baseFilename)
	if matches == nil {
		return nil, fmt.Errorf("invalid archive filename: %s (expected format: name-version.kdeps)", baseFilename)
	}

	agentName := matches[1]
	version := matches[2]

	// Define the base extraction path for this agent and version
	extractBasePath := filepath.Join(kdepsDir, "agents", agentName, version)

	// Open the tar.gz file
	file, err := fs.Open(kdepsPackage)
	if err != nil {
		return nil, fmt.Errorf("failed to open tar.gz file: %w", err)
	}
	defer file.Close()

	// Create a gzip reader
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
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

	// Iterate through the files in the tar archive
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tar header: %w", err)
		}

		// Construct the full absolute path for the file
		targetPath := filepath.Join(extractBasePath, header.Name)
		absPath, err := filepath.Abs(targetPath)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path: %w", err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			// Recreate the directory (remove it if it exists, then create)
			if _, err := fs.Stat(targetPath); !os.IsNotExist(err) {
				err = fs.RemoveAll(targetPath)
				if err != nil {
					return nil, fmt.Errorf("failed to remove existing directory: %w", err)
				}
			}
			// Create the directory with more permissive permissions
			err = fs.MkdirAll(targetPath, 0777)
			if err != nil {
				return nil, fmt.Errorf("failed to create directory: %w", err)
			}

		case tar.TypeReg:
			// Create parent directories if they don't exist
			parentDir := filepath.Dir(targetPath)
			err = fs.MkdirAll(parentDir, 0777)
			if err != nil {
				return nil, fmt.Errorf("failed to create parent directories: %w", err)
			}

			// Handle workflow and resource files
			if strings.HasSuffix(header.Name, "workflow.pkl") {
				kdeps.Workflow = absPath
			} else if strings.HasPrefix(header.Name, "resources/") && strings.HasSuffix(header.Name, ".pkl") {
				kdeps.Resources = append(kdeps.Resources, absPath)
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
				}
			}

			// Extract the file
			outFile, err := fs.Create(targetPath)
			if err != nil {
				return nil, fmt.Errorf("failed to create file: %w", err)
			}
			defer outFile.Close()

			// Copy the file contents
			_, err = io.Copy(outFile, tarReader)
			if err != nil {
				return nil, fmt.Errorf("failed to copy file contents: %w", err)
			}

			// Set file permissions to more permissive ones (e.g., 0666 or 0777)
			err = fs.Chmod(targetPath, 0666)
			if err != nil {
				return nil, fmt.Errorf("failed to set file permissions: %w", err)
			}
		}
	}

	// Copy the kdepsPackage to kdepsDir/packages
	packageDir := filepath.Join(kdepsDir, "packages")
	err = fs.MkdirAll(packageDir, 0777)
	if err != nil {
		return nil, fmt.Errorf("failed to create packages directory: %w", err)
	}

	sourceFile, err := fs.Open(kdepsPackage)
	if err != nil {
		return nil, fmt.Errorf("failed to open source kdeps package: %w", err)
	}
	defer sourceFile.Close()

	destinationFile, err := fs.Create(filepath.Join(packageDir, baseFilename))
	if err != nil {
		return nil, fmt.Errorf("failed to create destination kdeps package: %w", err)
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		return nil, fmt.Errorf("failed to copy kdeps package to packages directory: %w", err)
	}

	return kdeps, nil
}

// Function to compare version numbers
func compareVersions(versions []string) string {
	sort.Slice(versions, func(i, j int) bool {
		// Split the version strings into parts
		v1 := strings.Split(versions[i], ".")
		v2 := strings.Split(versions[j], ".")

		// Compare each part of the version (major, minor, patch)
		for k := 0; k < len(v1); k++ {
			if v1[k] != v2[k] {
				return v1[k] > v2[k]
			}
		}
		return false
	})

	// Return the first version (which will be the latest after sorting)
	return versions[0]
}

func getLatestVersion(directory string) (string, error) {
	var versions []string

	// Walk through the directory to collect version names
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Collect directory names that match the version pattern
		if info.IsDir() && strings.Count(info.Name(), ".") == 2 {
			versions = append(versions, info.Name())
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	// Check if versions were found
	if len(versions) == 0 {
		return "", fmt.Errorf("no versions found")
	}

	// Find the latest version
	latestVersion := compareVersions(versions)

	return latestVersion, nil
}

// PackageProject compresses the contents of projectDir into a tar.gz file in kdepsDir
func PackageProject(fs afero.Fs, wf *pklWf.Workflow, kdepsDir, compiledProjectDir string) (string, error) {
	// Enforce the folder structure
	if err := enforcer.EnforceFolderStructure(fs, compiledProjectDir); err != nil {
		return "", err
	}

	// Create the output filename for the package
	outFile := fmt.Sprintf("%s-%s.kdeps", *wf.Name, *wf.Version)
	packageDir := fmt.Sprintf("%s/packages", kdepsDir)

	// Define the output file path for the tarball
	tarGzPath := filepath.Join(packageDir, outFile)

	// Check if the tar.gz file already exists, and if so, delete it
	exists, err := afero.Exists(fs, tarGzPath)
	if err != nil {
		return "", fmt.Errorf("error checking if package exists: %w", err)
	}
	if exists {
		if err := fs.Remove(tarGzPath); err != nil {
			return "", fmt.Errorf("failed to remove existing package file: %w", err)
		}
	}

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
	err = afero.Walk(fs, compiledProjectDir, func(file string, info os.FileInfo, err error) error {
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
		relPath, err := filepath.Rel(compiledProjectDir, file)
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

func CompileWorkflow(fs afero.Fs, wf *pklWf.Workflow, kdepsDir, projectDir string) (string, error) {
	action := wf.Action

	if action == nil {
		return "", nil
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

	exists, err := afero.DirExists(fs, agentDir)
	if err != nil {
		return "", err
	}

	if exists {
		err := fs.RemoveAll(agentDir)
		if err != nil {
			return "", err
		}
	}

	// Step 3: Recreate the folder
	err = fs.MkdirAll(resourcesDir, 0755) // Create the folder with read-write-execute permissions
	if err != nil {
		return "", err
	}

	searchPattern := `action\s*=\s*".*"`
	replaceLine := fmt.Sprintf("action = \"%s\"\n", compiledAction)

	inputFile, err := fs.Open(filePath)
	if err != nil {
		return "", err
	}
	defer inputFile.Close()

	var lines []string
	scanner := bufio.NewScanner(inputFile)

	// Compile the regular expression
	re = regexp.MustCompile(searchPattern)

	for scanner.Scan() {
		line := scanner.Text()

		// Step 2: Check if the line matches the regular expression
		if re.MatchString(line) {
			line = replaceLine // Replace the line if it matches
		}

		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	err = afero.WriteFile(fs, compiledFilePath, []byte(strings.Join(lines, "\n")), 0644)
	if err != nil {
		return "", err
	}

	compiledProjectDir := filepath.Dir(compiledFilePath)

	return compiledProjectDir, nil
}

func copyFile(fs afero.Fs, src, dst string) error {
	// Open the source file
	srcFile, err := fs.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Create the destination file
	dstFile, err := fs.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// Copy the file contents from src to dst
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	// Optionally, you can copy the file permissions from the source
	srcInfo, err := fs.Stat(src)
	if err != nil {
		return err
	}
	err = fs.Chmod(dst, srcInfo.Mode())
	if err != nil {
		return err
	}

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
					return walkErr
				}

				// Determine the relative path for correct directory structure copying
				relPath, err := filepath.Rel(src, path)
				if err != nil {
					return err
				}

				// Create the full destination path based on the relative path
				dstPath := filepath.Join(dst, relPath)

				if info.IsDir() {
					if err := fs.MkdirAll(dstPath, info.Mode()); err != nil {
						return err
					}
				} else {
					if err := copyFile(fs, path, dstPath); err != nil {
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
				return err
			}
			agentVersion = version
		}

		src := filepath.Join(kdepsDir, "agents", agentName, agentVersion, "resources")
		dst := filepath.Join(compiledProjectDir, "resources")

		srcDir = filepath.Join(kdepsDir, "agents", agentName, agentVersion, "data", agentName, agentVersion)
		destDir = filepath.Join(compiledProjectDir, fmt.Sprintf("data/%s/%s", agentName, agentVersion))

		if err := copyResources(src, dst); err != nil {
			return err
		}
	}

	// Final walk for data directory copying
	err := afero.Walk(fs, srcDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		// Determine the relative path from the source directory
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		// Create the destination path
		dstPath := filepath.Join(destDir, relPath)

		// If it's a directory, create the directory in the destination
		if info.IsDir() {
			if err := fs.MkdirAll(dstPath, info.Mode()); err != nil {
				return err
			}
		} else {
			// If it's a file, copy the file
			if err := copyFile(fs, path, dstPath); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func CompileProject(fs afero.Fs, wf *pklWf.Workflow, kdepsDir string, projectDir string) (string, error) {
	compiledProjectDir, err := CompileWorkflow(fs, wf, kdepsDir, projectDir)
	if err != nil {
		return "", err
	}

	exists, err := afero.DirExists(fs, compiledProjectDir)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", errors.New("Compiled project directory does not exist!")
	}

	newWorkflowFile := filepath.Join(compiledProjectDir, "workflow.pkl")

	if _, err := fs.Stat(newWorkflowFile); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("No compiled workflow found at: %s", newWorkflowFile)
		}
		return "", err
	}

	newWorkflow, err := workflow.LoadWorkflow(newWorkflowFile)
	if err != nil {
		return "", err
	}

	resourcesDir := filepath.Join(compiledProjectDir, "resources")

	if err := CompileResources(fs, newWorkflow, resourcesDir, projectDir); err != nil {
		return "", err
	}

	if err := CopyDir(fs, newWorkflow, kdepsDir, projectDir, compiledProjectDir, "", "", "", false); err != nil {
		return "", err
	}

	if err := ProcessWorkflows(fs, newWorkflow, kdepsDir, projectDir, compiledProjectDir); err != nil {
		return "", err
	}

	packageFile, err := PackageProject(fs, newWorkflow, kdepsDir, compiledProjectDir)
	if err != nil {
		return "", err
	}

	if _, err := fs.Stat(packageFile); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("No package file found at: %s", packageFile)
		}
		return "", err
	}

	log.Info("Kdeps package created", "package-file", packageFile)

	return compiledProjectDir, nil
}

func CompileResources(fs afero.Fs, wf *pklWf.Workflow, resourcesDir string, projectDir string) error {
	// Walk through all files in the project directory
	err := afero.Walk(fs, filepath.Join(projectDir, "resources"), func(file string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only process .pkl files
		if filepath.Ext(file) == ".pkl" {
			if processErr := processResourcePklFiles(fs, file, wf, resourcesDir); processErr != nil {
				return processErr
			}
		}
		return nil
	})

	return err
}

func processResourcePklFiles(fs afero.Fs, file string, wf *pklWf.Workflow, resourcesDir string) error {
	name, version := *wf.Name, *wf.Version

	readFile, err := fs.Open(file)
	if err != nil {
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
			return fmt.Errorf("no valid action found in file: %s", file)
		}
		fname := fmt.Sprintf("%s_%s-%s.pkl", name, action, version)
		err = afero.WriteFile(fs, filepath.Join(resourcesDir, fname), fileBuffer.Bytes(), os.FileMode(0644))
		if err != nil {
			return fmt.Errorf("error writing file: %w", err)
		}
	}

	return scanner.Err()
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

func ProcessWorkflows(fs afero.Fs, wf *pklWf.Workflow, kdepsDir, projectDir, compiledProjectDir string) error {
	if wf.Workflows == nil {
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
					return err
				}
			} else {
				if err := CopyDir(fs, wf, kdepsDir, projectDir, compiledProjectDir, agentName, version,
					"", true); err != nil {
					return err
				}
			}

		} else {
			// No version present, check if there is an action
			agentAndAction := strings.SplitN(value, "/", 2)
			agentName := agentAndAction[0]

			if len(agentAndAction) == 2 {
				action := agentAndAction[1]
				if err := CopyDir(fs, wf, kdepsDir, projectDir, compiledProjectDir, agentName, "",
					action, true); err != nil {
					return err
				}
			} else {
				if err := CopyDir(fs, wf, kdepsDir, projectDir, compiledProjectDir, agentName, "", "", true); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
