package enforcer

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/spf13/afero"
)

var runBlockRegexes = []*regexp.Regexp{
	regexp.MustCompile(`\s*Exec\s*{`),
	regexp.MustCompile(`\s*Python\s*{`),
	regexp.MustCompile(`\s*Chat\s*{`),
	regexp.MustCompile(`\s*HTTPClient\s*{`),
	regexp.MustCompile(`\s*APIResponse\s*{`),
}

type pklFileInfo struct {
	pklType     string
	expectedPkl string
}

var validPklFiles = map[string]pklFileInfo{
	"Kdeps.pkl":    {"configuration file", ".kdeps.pkl"},
	"Workflow.pkl": {"workflow file", "workflow.pkl"},
	"Resource.pkl": {},
}

func compareVersions(v1, v2 string, logger *logging.Logger) (int, error) {
	v1Parts := strings.Split(v1, ".")
	v2Parts := strings.Split(v2, ".")

	maxLen := len(v1Parts)
	if len(v2Parts) > maxLen {
		maxLen = len(v2Parts)
	}

	for i := range maxLen {
		var v1Part, v2Part int
		var err error

		if i < len(v1Parts) {
			// Handle version suffixes like "-dev", "+build", "-alpha+1000", etc.
			numPart := v1Parts[i]
			minIndex := len(numPart)

			if hyphenIndex := strings.Index(numPart, "-"); hyphenIndex != -1 && hyphenIndex < minIndex {
				minIndex = hyphenIndex
			}
			if plusIndex := strings.Index(numPart, "+"); plusIndex != -1 && plusIndex < minIndex {
				minIndex = plusIndex
			}

			if minIndex < len(numPart) {
				numPart = numPart[:minIndex]
			}

			v1Part, err = strconv.Atoi(numPart)
			if err != nil {
				logger.Error("invalid version format")
				return 0, errors.New("invalid version format")
			}
		}

		if i < len(v2Parts) {
			// Handle version suffixes like "-dev", "+build", "-alpha+1000", etc.
			numPart := v2Parts[i]
			minIndex := len(numPart)

			if hyphenIndex := strings.Index(numPart, "-"); hyphenIndex != -1 && hyphenIndex < minIndex {
				minIndex = hyphenIndex
			}
			if plusIndex := strings.Index(numPart, "+"); plusIndex != -1 && plusIndex < minIndex {
				minIndex = plusIndex
			}

			if minIndex < len(numPart) {
				numPart = numPart[:minIndex]
			}

			v2Part, err = strconv.Atoi(numPart)
			if err != nil {
				logger.Error("invalid version format")
				return 0, errors.New("invalid version format")
			}
		}

		if v1Part < v2Part {
			return -1, nil
		}
		if v1Part > v2Part {
			return 1, nil
		}
	}
	return 0, nil
}

func EnforceSchemaURL(ctx context.Context, line, filePath string, logger *logging.Logger) error {
	const amendErr = "the pkl file does not start with 'amends'"
	const schemaErr = "the pkl file does not contain 'schema.kdeps.com/core'"

	if !strings.HasPrefix(line, "amends") {
		logger.Error(amendErr, "file", filePath)
		return errors.New(amendErr)
	}

	if !strings.Contains(line, "schema.kdeps.com/core") {
		logger.Error(schemaErr, "file", filePath)
		return errors.New(schemaErr)
	}
	return nil
}

func EnforcePklVersion(ctx context.Context, line, filePath, schemaVersion string, logger *logging.Logger) error {
	start := strings.Index(line, "@")
	end := strings.Index(line, "#")
	if start == -1 || end == -1 || start >= end {
		err := errors.New("invalid version format in the amends line")
		logger.Error(err.Error())
		return err
	}

	version := line[start+1 : end]
	comparison, err := compareVersions(version, schemaVersion, logger)
	if err != nil {
		logger.Error("version comparison error", "error", err)
		return err
	}

	switch comparison {
	case -1:
		logger.Warn("version in amends line is lower than schema version. Run 'kdeps upgrade' to update your schema versions.",
			"version", version, "latestSchemaVersion(ctx)", schemaVersion, "file", filePath)
	case 1:
		logger.Debug("version in amends line is higher than schema version",
			"version", version, "schemaVersion", schemaVersion, "file", filePath)
	}
	return nil
}

func EnforcePklFilename(ctx context.Context, line string, filePath string, logger *logging.Logger) error {
	filename := strings.ToLower(filepath.Base(filePath))
	start := strings.Index(line, "#/")
	if start == -1 {
		err := errors.New("invalid format: could not extract .pkl filename")
		logger.Error(err.Error())
		return err
	}

	pklFilename := strings.Trim(line[start+2:], `"`)
	logger.Debug("checking pkl filename", "line", line, "filePath", filePath, "pklFilename", pklFilename)

	info, exists := validPklFiles[pklFilename]
	if !exists {
		expected := make([]string, 0, len(validPklFiles))
		for k := range validPklFiles {
			expected = append(expected, k)
		}
		logger.Error("invalid .pkl file in amends line", "expected", expected, "found", pklFilename)
		return errors.New("invalid .pkl file in amends line")
	}

	if pklFilename == "Resource.pkl" && (filename == ".kdeps.pkl" || filename == "workflow.pkl") {
		logger.Error("invalid filename for Resource.pkl", "filename", filename, "pklFilename", pklFilename)
		return errors.New("invalid filename for Resource.pkl")
	}

	if pklFilename != "Resource.pkl" && info.expectedPkl != filename {
		logger.Error("invalid .pkl filename", "expected", info.expectedPkl, "found", filename, "type", info.pklType)
		return fmt.Errorf("invalid .pkl filename for a %s", info.pklType)
	}
	return nil
}

func EnforceFolderStructure(fs afero.Fs, ctx context.Context, filePath string, logger *logging.Logger) error {
	const expectedFile = "workflow.pkl"
	expectedFolders := map[string]bool{"resources": false, "data": false}
	ignoredFiles := map[string]bool{".kdeps.pkl": true}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		logger.Error("error getting absolute path", "filePath", filePath, "error", err)
		return err
	}

	fileInfo, err := fs.Stat(absPath)
	if err != nil {
		logger.Error("error reading file info", "filePath", filePath, "error", err)
		return err
	}

	absTargetDir := absPath
	if !fileInfo.IsDir() {
		absTargetDir = filepath.Dir(absPath)
	}

	files, err := afero.ReadDir(fs, absTargetDir)
	if err != nil {
		logger.Error("error reading directory contents", "dir", absTargetDir, "error", err)
		return err
	}

	for _, file := range files {
		if ignoredFiles[file.Name()] {
			logger.Debug("ignored file found", "file", file.Name())
			continue
		}

		if file.IsDir() {
			if _, ok := expectedFolders[file.Name()]; !ok {
				logger.Error("unexpected folder found", "folder", file.Name())
				return fmt.Errorf("unexpected folder found: %s", file.Name())
			}
			expectedFolders[file.Name()] = true

			if file.Name() == "resources" {
				if err := enforceResourcesFolder(ctx, fs, filepath.Join(absTargetDir, "resources"), logger); err != nil {
					return err
				}
			}
		} else if file.Name() != expectedFile {
			logger.Error("unexpected file found", "file", file.Name())
			return fmt.Errorf("unexpected file found: %s", file.Name())
		}
	}

	for folder, found := range expectedFolders {
		if !found {
			logger.Warn("folder does not exist", "folder", folder)
		}
	}
	return nil
}

func EnforceResourceRunBlock(ctx context.Context, fs afero.Fs, file string, logger *logging.Logger) error {
	pklData, err := afero.ReadFile(fs, file)
	if err != nil {
		logger.Error("failed to read .pkl file", "file", file, "error", err)
		return err
	}

	count := 0
	content := string(pklData)
	for _, re := range runBlockRegexes {
		if re.MatchString(content) {
			count++
		}
	}

	if count > 1 {
		err := fmt.Errorf("resources can only contain one run block type. Found %d in file: %s", count, file)
		logger.Error(err.Error())
		return err
	}

	logger.Debug("run block validated successfully", "file", file)
	return nil
}

func enforceResourcesFolder(ctx context.Context, fs afero.Fs, resourcesPath string, logger *logging.Logger) error {
	files, err := afero.ReadDir(fs, resourcesPath)
	if err != nil {
		logger.Error("error reading resources folder", "path", resourcesPath, "error", err)
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			if file.Name() == "external" {
				continue
			}
			logger.Error("unexpected directory in resources folder", "dir", file.Name())
			return fmt.Errorf("unexpected directory found in resources folder: %s", file.Name())
		}

		if ext := filepath.Ext(file.Name()); ext != ".pkl" {
			logger.Error("unexpected file found in resources folder", "file", file.Name())
			return fmt.Errorf("unexpected file found in resources folder: %s", file.Name())
		}

		fullPath := filepath.Join(resourcesPath, file.Name())
		if err := EnforceResourceRunBlock(ctx, fs, fullPath, logger); err != nil {
			logger.Error("failed to process .pkl file", "file", fullPath, "error", err)
			return err
		}
	}
	return nil
}

func EnforcePklTemplateAmendsRules(fs afero.Fs, filePath string, ctx context.Context, logger *logging.Logger) error {
	file, err := fs.Open(filePath)
	if err != nil {
		logger.Error("failed to open file", "filePath", filePath, "error", err)
		return err
	}
	defer file.Close()

	if ext := filepath.Ext(filePath); ext != ".pkl" {
		logger.Error("unexpected file type", "file", filePath)
		return fmt.Errorf("unexpected file type: %s", filePath)
	}

	scanner := bufio.NewScanner(file)
	amendsCount := 0
	validAmendsFound := false
	validFileTypeFound := false
	filename := strings.ToLower(filepath.Base(filePath))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Check if this line is an amends statement
		if strings.HasPrefix(line, "amends") {
			amendsCount++
			logger.Debug("processing amends line", "line", line, "count", amendsCount)

			if err := EnforceSchemaURL(ctx, line, filePath, logger); err != nil {
				return fmt.Errorf("schema URL validation failed for amends statement %d: %w", amendsCount, err)
			}

			if err := EnforcePklVersion(ctx, line, filePath, schema.SchemaVersion(ctx), logger); err != nil {
				return fmt.Errorf("version validation failed for amends statement %d: %w", amendsCount, err)
			}

			// Check if this amends statement is valid for the file type
			if err := EnforcePklFilename(ctx, line, filePath, logger); err == nil {
				validFileTypeFound = true
				logger.Debug("amends statement valid for file type", "line", line, "count", amendsCount)
			} else {
				logger.Debug("amends statement not valid for file type, but continuing validation", "line", line, "count", amendsCount, "error", err)
			}

			validAmendsFound = true
			logger.Debug("amends statement validated successfully", "line", line, "count", amendsCount)
		}
	}

	if err := scanner.Err(); err != nil {
		logger.Error("error while scanning the file", "filePath", filePath, "error", err)
		return err
	}

	if !validAmendsFound {
		logger.Error("no valid 'amends' line found in the file", "filePath", filePath)
		return errors.New("no valid 'amends' line found")
	}

	if !validFileTypeFound {
		logger.Error("no amends statement valid for file type", "filePath", filePath, "filename", filename)
		return fmt.Errorf("no amends statement valid for file type %s", filename)
	}

	logger.Debug("all amends statements validated successfully", "filePath", filePath, "amendsCount", amendsCount)
	return nil
}
