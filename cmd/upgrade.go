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
func UpgradeCommand(ctx context.Context, fs afero.Fs, _ string, logger *logging.Logger) *cobra.Command {
	var targetVersion string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "upgrade [directory]",
		Short: "Upgrade schema versions and Ollama image tags in pkl files",
		Long: `Upgrade schema versions and Ollama image tags in pkl files within a directory.
		
This command scans for pkl files containing schema version references and Ollama image tag 
versions, upgrading them to the specified schema version or latest default versions. It 
validates that the new schema version meets minimum requirements.

Examples:
  kdeps upgrade                        # Upgrade current directory to default versions
  kdeps upgrade ./my-agent            # Upgrade specific directory to default versions  
  kdeps upgrade --version 0.2.50 .    # Upgrade to specific schema version (Ollama tag uses latest)
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

			logger.Info("upgrading schema and Ollama versions", "directory", absPath, "target_schema_version", targetVersion, "target_ollama_tag", version.DefaultOllamaImageTag, "dry_run", dryRun)

			// Perform the upgrade
			return UpgradeSchemaVersions(ctx, fs, absPath, targetVersion, dryRun, logger)
		},
	}

	cmd.Flags().StringVarP(&targetVersion, "version", "v", "", "Target schema version (default: latest)")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview changes without applying them")

	return cmd
}

// UpgradeSchemaVersions scans a directory for pkl files and upgrades schema versions
func UpgradeSchemaVersions(_ context.Context, fs afero.Fs, dirPath, targetVersion string, dryRun bool, logger *logging.Logger) error {
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
		updatedContent, schemaChanged, err := UpgradeSchemaVersionInContent(string(content), targetVersion, logger)
		if err != nil {
			logger.Error("failed to upgrade schema version", "path", path, "error", err)
			return nil // Continue processing other files
		}

		// Check if file contains Ollama image tag references
		updatedContent, ollamaChanged, err := UpgradeOllamaTagVersionInContent(updatedContent, version.DefaultOllamaImageTag, logger)
		if err != nil {
			logger.Error("failed to upgrade Ollama tag version", "path", path, "error", err)
			return nil // Continue processing other files
		}

		changed := schemaChanged || ollamaChanged
		if changed {
			filesUpdated++
			if dryRun {
				changes := []string{}
				if schemaChanged {
					changes = append(changes, "schema_version="+targetVersion)
				}
				if ollamaChanged {
					changes = append(changes, "ollama_tag="+version.DefaultOllamaImageTag)
				}
				logger.Info("would update file", "path", path, "changes", strings.Join(changes, ", "))
			} else {
				// Write updated content back to file
				if err := afero.WriteFile(fs, path, []byte(updatedContent), info.Mode()); err != nil {
					logger.Error("failed to write updated file", "path", path, "error", err)
					return nil // Continue processing other files
				}
				changes := []string{}
				if schemaChanged {
					changes = append(changes, "schema_version="+targetVersion)
				}
				if ollamaChanged {
					changes = append(changes, "ollama_tag="+version.DefaultOllamaImageTag)
				}
				logger.Info("updated file", "path", path, "changes", strings.Join(changes, ", "))
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

	logger.Info("schema and Ollama upgrade complete",
		"files_processed", filesProcessed,
		"files_updated", filesUpdated,
		"action", action,
		"target_schema_version", targetVersion,
		"target_ollama_tag", version.DefaultOllamaImageTag)

	return nil
}

// UpgradeSchemaVersionInContent upgrades schema version references in pkl file content
func UpgradeSchemaVersionInContent(content, targetVersion string, logger *logging.Logger) (string, bool, error) {
	// Regex patterns to match schema version references
	patterns := []string{
		`(amends\s+"package://schema\.kdeps\.com/core@)([^\"]+)(#/[^"]+")`,
		`(import\s+"package://schema\.kdeps\.com/core@)([^\"]+)(#/[^"]+")`,
		`("package://schema\.kdeps\.com/core@)([^\"]+)(#/[^"]+")`,
	}

	updatedContent := content
	changed := false

	logger.Debug("upgrading schema version", "content", content, "target_version", targetVersion)

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		updatedContentNew := re.ReplaceAllStringFunc(updatedContent, func(match string) string {
			subs := re.FindStringSubmatch(match)
			if len(subs) < 4 {
				return match
			}
			currentVersion := subs[2]
			if currentVersion == targetVersion {
				return match
			}
			changed = true
			return subs[1] + targetVersion + subs[3]
		})
		updatedContent = updatedContentNew
	}

	return updatedContent, changed, nil
}

// UpgradeOllamaTagVersionInContent upgrades OllamaTagVersion references in pkl file content
func UpgradeOllamaTagVersionInContent(content, targetOllamaTag string, logger *logging.Logger) (string, bool, error) {
	// Regex pattern to match OllamaTagVersion assignments
	// Matches: OllamaTagVersion = "version"
	pattern := `(\s*OllamaTagVersion\s*=\s*")([^"]+)(")`

	re := regexp.MustCompile(pattern)

	updatedContent := content
	changed := false

	logger.Debug("upgrading Ollama tag version", "target_tag", targetOllamaTag)

	updatedContent = re.ReplaceAllStringFunc(content, func(match string) string {
		subs := re.FindStringSubmatch(match)
		if len(subs) < 4 {
			return match
		}
		currentTag := subs[2]
		if currentTag == targetOllamaTag {
			logger.Debug("Ollama tag already up to date", "current", currentTag, "target", targetOllamaTag)
			return match
		}
		logger.Debug("upgrading Ollama tag", "from", currentTag, "to", targetOllamaTag)
		changed = true
		return subs[1] + targetOllamaTag + subs[3]
	})

	return updatedContent, changed, nil
}
