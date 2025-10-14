package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/utils"
	"github.com/kdeps/kdeps/pkg/version"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// UpgradeCommand creates the 'upgrade' command for upgrading schema versions in pkl files.
func UpgradeCommand(_ context.Context, fs afero.Fs, _ string, logger *logging.Logger) *cobra.Command {
	var targetVersion string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "upgrade [directory]",
		Short: "Upgrade schema versions and format in pkl files",
		Long: `Upgrade schema versions and format in pkl files within a directory.
		
This command scans for pkl files and performs two types of upgrades:
1. Schema version references (e.g., @0.2.44 -> @0.4.1-dev)
2. Schema format migration (e.g., lowercase -> capitalized attributes/blocks)

The format upgrade converts older lowercase PKL syntax to the new capitalized format:
- actionID -> ActionID, name -> Name, requires -> Requires, etc.
- Block names: run -> Run, chat -> Chat, exec -> Exec, etc.

Examples:
  kdeps upgrade                        # Upgrade current directory to default version
  kdeps upgrade ./my-agent            # Upgrade specific directory to default version  
  kdeps upgrade --version 0.4.1-dev .    # Upgrade to specific version
  kdeps upgrade --dry-run ./my-agent  # Preview changes without applying
		`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			// Determine target directory
			targetDir := "."
			if len(args) > 0 {
				targetDir = args[0]
			}

			// Determine target version
			if targetVersion == "" {
				targetVersion = version.DefaultSchemaVersion
			}

			// Validate target version
			if err := utils.ValidateSchemaVersion(targetVersion, version.MinimumSchemaVersion); err != nil {
				return fmt.Errorf("invalid target version: %w", err)
			}

			// Convert to absolute path
			absPath, err := filepath.Abs(targetDir)
			if err != nil {
				return fmt.Errorf("failed to resolve directory path: %w", err)
			}

			// Check if directory exists
			if exists, err := afero.DirExists(fs, absPath); err != nil {
				return fmt.Errorf("failed to check directory: %w", err)
			} else if !exists {
				return fmt.Errorf("directory does not exist: %s", absPath)
			}

			logger.Info("upgrading schema versions", "directory", absPath, "target_version", targetVersion, "dry_run", dryRun)

			// Perform the upgrade
			return upgradeSchemaVersions(fs, absPath, targetVersion, dryRun, logger)
		},
	}

	cmd.Flags().StringVarP(&targetVersion, "version", "v", "", "Target schema version (default: latest)")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview changes without applying them")

	return cmd
}

// upgradeSchemaVersions scans a directory for pkl files and upgrades schema versions.
func upgradeSchemaVersions(fs afero.Fs, dirPath, targetVersion string, dryRun bool, logger *logging.Logger) error {
	var filesProcessed int
	var filesUpdated int

	// Walk through directory to find pkl files
	err := afero.Walk(fs, dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-pkl files
		if info.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".pkl") {
			return nil
		}

		filesProcessed++
		logger.Debug("processing file", "path", path)

		// Read file content
		content, err := afero.ReadFile(fs, path)
		if err != nil {
			logger.Error("failed to read file", "path", path, "error", err)
			return nil // Continue processing other files
		}

		// Check if file contains schema version references
		updatedContent, changed, err := upgradeSchemaVersionInContent(string(content), targetVersion, logger)
		if err != nil {
			logger.Error("failed to upgrade schema version", "path", path, "error", err)
			return nil // Continue processing other files
		}

		if changed {
			filesUpdated++
			if dryRun {
				logger.Info("would update file", "path", path, "target_version", targetVersion)
			} else {
				// Write updated content back to file
				if err := afero.WriteFile(fs, path, []byte(updatedContent), info.Mode()); err != nil {
					logger.Error("failed to write updated file", "path", path, "error", err)
					return nil // Continue processing other files
				}
				logger.Info("updated file", "path", path, "target_version", targetVersion)
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("error walking directory: %w", err)
	}

	action := "updated"
	if dryRun {
		action = "would update"
	}

	logger.Info("schema upgrade complete",
		"files_processed", filesProcessed,
		"files_updated", filesUpdated,
		"action", action,
		"target_version", targetVersion)

	return nil
}

// upgradeSchemaVersionInContent upgrades schema version references and format in pkl file content.
func upgradeSchemaVersionInContent(content, targetVersion string, logger *logging.Logger) (string, bool, error) {
	logger.Debug("upgradeSchemaVersionInContent called", "targetVersion", targetVersion, "contentLength", len(content))
	updatedContent := content
	changed := false

	// Step 1: Upgrade schema version references
	logger.Debug("starting version upgrade step")
	versionChanged, err := upgradeVersionReferences(updatedContent, targetVersion, logger)
	if err != nil {
		logger.Debug("version upgrade failed", "error", err)
		return content, false, err
	}
	if versionChanged.changed {
		logger.Debug("version upgrade made changes")
		updatedContent = versionChanged.content
		changed = true
	} else {
		logger.Debug("version upgrade made no changes")
	}

	// Step 2: Upgrade schema format (lowercase to capitalized attributes/blocks)
	logger.Debug("starting format upgrade step")
	formatChanged, err := upgradeSchemaFormat(updatedContent, logger)
	if err != nil {
		logger.Debug("format upgrade failed", "error", err)
		return content, false, err
	}
	if formatChanged.changed {
		logger.Debug("format upgrade made changes")
		updatedContent = formatChanged.content
		changed = true
	} else {
		logger.Debug("format upgrade made no changes")
	}

	logger.Debug("upgradeSchemaVersionInContent finished", "totalChanged", changed, "finalContentLength", len(updatedContent))
	return updatedContent, changed, nil
}

type upgradeResult struct {
	content string
	changed bool
}

// upgradeVersionReferences upgrades schema version references in pkl file content.
func upgradeVersionReferences(content, targetVersion string, logger *logging.Logger) (upgradeResult, error) {
	logger.Debug("upgradeVersionReferences called", "targetVersion", targetVersion, "contentLength", len(content))

	// Regex patterns to match schema version references
	patterns := []string{
		// Match: amends "package://schema.kdeps.com/core@0.4.1-dev#/Workflow.pkl"
		`(amends\s+"package://schema\.kdeps\.com/core@)([^"#]+)(#/[^"]+")`,
		// Match: import "package://schema.kdeps.com/core@0.4.1-dev#/Resource.pkl"
		`(import\s+"package://schema\.kdeps\.com/core@)([^"#]+)(#/[^"]+")`,
		// Match other similar patterns
		`("package://schema\.kdeps\.com/core@)([^"#]+)(#/[^"]+")`,
	}

	updatedContent := content
	changed := false

	for i, pattern := range patterns {
		logger.Debug("testing pattern", "index", i, "pattern", pattern)
		re, err := regexp.Compile(pattern)
		if err != nil {
			return upgradeResult{}, fmt.Errorf("failed to compile regex pattern %d: %w", i, err)
		}
		matches := re.FindAllStringSubmatch(updatedContent, -1)
		logger.Debug("pattern matches", "index", i, "matchCount", len(matches))

		for j, match := range matches {
			logger.Debug("processing match", "patternIndex", i, "matchIndex", j, "match", match)
			if len(match) >= 4 {
				currentVersion := match[2]
				logger.Debug("found version", "currentVersion", currentVersion, "targetVersion", targetVersion)

				// Skip if already at target version
				if currentVersion == targetVersion {
					logger.Debug("skipping - already at target version")
					continue
				}

				// Skip validation - we want to upgrade FROM older versions TO newer versions
				// The whole point of upgrade is to handle versions below the minimum

				// Replace with target version
				oldRef := match[1] + currentVersion + match[3]
				newRef := match[1] + targetVersion + match[3]

				logger.Debug("performing replacement", "oldRef", oldRef, "newRef", newRef)

				beforeReplace := updatedContent
				updatedContent = strings.ReplaceAll(updatedContent, oldRef, newRef)
				afterReplace := updatedContent

				logger.Debug("replacement result",
					"beforeLength", len(beforeReplace),
					"afterLength", len(afterReplace),
					"contentChanged", beforeReplace != afterReplace)

				changed = true

				logger.Debug("upgrading schema version reference",
					"from", currentVersion,
					"to", targetVersion)
			}
		}
	}

	logger.Debug("upgradeVersionReferences finished", "changed", changed, "finalContentLength", len(updatedContent))
	return upgradeResult{content: updatedContent, changed: changed}, nil
}

// upgradeSchemaFormat upgrades PKL format from lowercase to capitalized attributes/blocks.
func upgradeSchemaFormat(content string, logger *logging.Logger) (upgradeResult, error) {
	updatedContent := content
	changed := false

	// Define attribute/block name mappings (lowercase -> capitalized)
	attributeMappings := map[string]string{
		// Common PKL attributes that were changed in schema migration
		"actionID":              "ActionID",
		"targetActionID":        "TargetActionID",
		"name":                  "AgentID",
		"description":           "Description",
		"category":              "Category",
		"version":               "Version",
		"requires":              "Requires",
		"items":                 "Items",
		"run":                   "Run",
		"settings":              "Settings",
		"workflows":             "Workflows",
		"model":                 "Model",
		"prompt":                "Prompt",
		"role":                  "Role",
		"scenario":              "Scenario",
		"tools":                 "Tools",
		"jsonResponse":          "JSONResponse",
		"jsonResponseKeys":      "JSONResponseKeys",
		"files":                 "Files",
		"timeoutDuration":       "TimeoutDuration",
		"timestamp":             "Timestamp",
		"skipCondition":         "SkipCondition",
		"preflightCheck":        "PreflightCheck",
		"validations":           "Validations",
		"error":                 "Error",
		"code":                  "Code",
		"message":               "Message",
		"expr":                  "Expr",
		"chat":                  "Chat",
		"exec":                  "Exec",
		"python":                "Python",
		"httpClient":            "HTTPClient",
		"apiResponse":           "APIResponse",
		"method":                "Method",
		"url":                   "Url",
		"body":                  "Body",
		"headers":               "Headers",
		"data":                  "Data",
		"response":              "Response",
		"meta":                  "Meta",
		"success":               "Success",
		"errors":                "Errors",
		"requestID":             "RequestID",
		"properties":            "Properties",
		"script":                "Script",
		"env":                   "Env",
		"command":               "Command",
		"stdout":                "Stdout",
		"stderr":                "Stderr",
		"exitCode":              "ExitCode",
		"restrictToHTTPMethods": "RestrictToHTTPMethods",
		"restrictToRoutes":      "RestrictToRoutes",
		"allowedHeaders":        "AllowedHeaders",
		"allowedParams":         "AllowedParams",
		"installAnaconda":       "InstallAnaconda",
		"condaPackages":         "CondaPackages",
		"pythonPackages":        "PythonPackages",
		"packages":              "Packages",
		"repositories":          "Repositories",
		"models":                "Models",
		"ollamaImageTag":        "OllamaImageTag",
		"args":                  "Args",
		"timezone":              "Timezone",
		"apiServerMode":         "APIServerMode",
		"apiServer":             "APIServer",
		"webServerMode":         "WebServerMode",
		"webServer":             "WebServer",
		"agentSettings":         "AgentSettings",
		"hostIP":                "HostIP",
		"portNum":               "PortNum",
		"trustedProxies":        "TrustedProxies",
		"routes":                "Routes",
		"path":                  "Path",
		"methods":               "Methods",
		"cors":                  "CORS",
		"enableCORS":            "EnableCORS",
		"allowOrigins":          "AllowOrigins",
		"allowMethods":          "AllowMethods",
		"allowCredentials":      "AllowCredentials",
		"exposeHeaders":         "ExposeHeaders",
		"maxAge":                "MaxAge",
	}

	// Apply attribute/block name transformations
	for oldName, newName := range attributeMappings {
		// Pattern 1: Attribute assignment (attribute = value)
		attributePattern, err := regexp.Compile(`\b` + regexp.QuoteMeta(oldName) + `\s*=`)
		if err != nil {
			return upgradeResult{}, fmt.Errorf("failed to compile attribute regex for %s: %w", oldName, err)
		}
		if attributePattern.MatchString(updatedContent) {
			updatedContent = attributePattern.ReplaceAllString(updatedContent, newName+" =")
			changed = true
			logger.Debug("upgraded attribute", "from", oldName, "to", newName)
		}

		// Pattern 2: Block definition (blockName {)
		blockPattern, err := regexp.Compile(`\b` + regexp.QuoteMeta(oldName) + `\s*\{`)
		if err != nil {
			return upgradeResult{}, fmt.Errorf("failed to compile block regex for %s: %w", oldName, err)
		}
		if blockPattern.MatchString(updatedContent) {
			updatedContent = blockPattern.ReplaceAllString(updatedContent, newName+" {")
			changed = true
			logger.Debug("upgraded block", "from", oldName, "to", newName)
		}
	}

	return upgradeResult{content: updatedContent, changed: changed}, nil
}
