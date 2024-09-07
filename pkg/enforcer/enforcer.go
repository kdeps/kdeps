package enforcer

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// CompareVersions compares two version strings and returns:
// -1 if v1 < v2
// 0 if v1 == v2
// 1 if v1 > v2
func CompareVersions(v1, v2 string) (int, error) {
	v1Parts := strings.Split(v1, ".")
	v2Parts := strings.Split(v2, ".")

	for i := 0; i < len(v1Parts) && i < len(v2Parts); i++ {
		v1Part, err1 := strconv.Atoi(v1Parts[i])
		v2Part, err2 := strconv.Atoi(v2Parts[i])

		if err1 != nil || err2 != nil {
			return 0, errors.New("invalid version format")
		}

		if v1Part < v2Part {
			return -1, nil
		}
		if v1Part > v2Part {
			return 1, nil
		}
	}

	// If all parts compared are equal, return 0
	return 0, nil
}

// ValidateSchemaURL checks if the "amends" line contains the correct schema.kdeps.com/core URL
func ValidateSchemaURL(line string) error {
	if !strings.HasPrefix(line, "amends") {
		return errors.New("the pkl file does not start with 'amends'")
	}

	if !strings.Contains(line, "schema.kdeps.com/core") {
		return errors.New("the pkl file does not contain 'schema.kdeps.com/core'")
	}

	return nil
}

// ValidateVersion extracts the version from the "amends" line and compares it with the provided schema version
func ValidateVersion(line, schemaVersion string) error {
	// Extract the version number after '@' and before '#'
	start := strings.Index(line, "@")
	end := strings.Index(line, "#")
	if start == -1 || end == -1 || start >= end {
		return errors.New("invalid version format in the amends line")
	}
	version := line[start+1 : end]

	// Compare versions
	comparison, err := CompareVersions(version, schemaVersion)
	if err != nil {
		return err
	}

	if comparison == -1 {
		// Version in the amends line is lower than the schema version
		fmt.Printf("Warning: The version '%s' in the amends line is lower than the schema version '%s'.\n", version, schemaVersion)
	} else if comparison == 1 {
		// Version in the amends line is higher
		fmt.Printf("The version '%s' in the amends line is higher than the schema version '%s'.\n", version, schemaVersion)
	}

	return nil
}

// ValidatePklFile extracts the .pkl file name and checks if it matches the expected value based on the rules
func ValidatePklFile(line string, filePath string) error {
	// Extract the base filename from the file path (e.g., ".kdeps.pkl" from the full path)
	filename := strings.ToLower(filepath.Base(filePath))

	// Extract the .pkl file from the line
	start := strings.Index(line, "#/")
	if start == -1 {
		return errors.New("invalid format: could not extract .pkl filename")
	}
	pklFilename := line[start+2:]

	// Remove trailing double-quote if present
	pklFilename = strings.Trim(pklFilename, `"`)

	// Define valid .pkl file names
	validPklFiles := map[string]bool{
		"Kdeps.pkl":    true,
		"Workflow.pkl": true,
		"Resource.pkl": true,
	}

	// Check if the extracted .pkl filename is valid
	if !validPklFiles[pklFilename] {
		return fmt.Errorf("invalid .pkl file: amends template expected one of '%s', but found '%s'", strings.Join(validPklFilesKeys(validPklFiles), "', '"), pklFilename)
	}

	if pklFilename == "Resource.pkl" {
		if filename == ".kdeps.pkl" || filename == "workflow.pkl" {
			return fmt.Errorf("Invalid filename: filename '%s' is not valid for a '%s' .pkl file. Please choose a different filename.", filename, pklFilename)
		}
	}

	// Map the base filename to the expected .pkl file name
	var expectedPkl string
	var pklType string
	switch pklFilename {
	case "Kdeps.pkl":
		pklType = "configuration file"
		expectedPkl = ".kdeps.pkl"
	case "Workflow.pkl":
		pklType = "workflow file"
		expectedPkl = "workflow.pkl"
	}

	// Validate if the extracted filename from the line matches the expected .pkl file
	if expectedPkl != filename {
		return fmt.Errorf("invalid .pkl filename for a %s: expected '%s', but found '%s'", pklType, expectedPkl, filename)
	}

	return nil
}

// Helper function to get the keys of a map as a slice of strings
func validPklFilesKeys(validPklFiles map[string]bool) []string {
	keys := make([]string, 0, len(validPklFiles))
	for k := range validPklFiles {
		keys = append(keys, k)
	}
	return keys
}

// ValidateAmendsLine combines the three validations (schema URL, version, and .pkl file)
func ValidateAmendsLine(filePath, schemaVersionFilePath string) error {
	// Open the file containing the amends line
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Open the file containing the schema version
	versionFile, err := os.Open(schemaVersionFilePath)
	if err != nil {
		return err
	}
	defer versionFile.Close()

	// Read the schema version from the ../SCHEMA_VERSION file
	var schemaVersion string
	scannerVersion := bufio.NewScanner(versionFile)
	if scannerVersion.Scan() {
		schemaVersion = strings.TrimSpace(scannerVersion.Text())
	}
	if err := scannerVersion.Err(); err != nil {
		return err
	}

	// Create a new scanner to read the amends file line by line
	scanner := bufio.NewScanner(file)

	// Iterate over lines and skip empty or whitespace-only lines
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text()) // Remove leading and trailing whitespace
		if line == "" {
			continue // Skip empty lines
		}

		// Validate the line in stages
		if err := ValidateSchemaURL(line); err != nil {
			return err
		}

		if err := ValidateVersion(line, schemaVersion); err != nil {
			return err
		}

		if err := ValidatePklFile(line, filePath); err != nil {
			return err
		}

		// All checks passed
		return nil
	}

	// Check for any scanning error
	if err := scanner.Err(); err != nil {
		return err
	}

	// Return error if no valid amends line was found
	return errors.New("no valid 'amends' line found")
}
