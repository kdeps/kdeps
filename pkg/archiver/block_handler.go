package archiver

import (
	"fmt"
	"strings"

	"github.com/kdeps/schema/gen/workflow"
)

// Handle the values inside the requires { ... } block.
func handleRequiresBlock(blockContent string, wf workflow.Workflow) string {
	name, version := wf.GetName(), wf.GetVersion()

	// Split the block by newline and process each line
	lines := strings.Split(blockContent, "\n")
	modifiedLines := make([]string, 0, len(lines)) // Preallocate the slice based on the number of lines

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			modifiedLines = append(modifiedLines, trimmedLine)
			continue
		}

		// Process quoted lines
		if strings.HasPrefix(trimmedLine, `"`) && strings.HasSuffix(trimmedLine, `"`) {
			value := strings.Trim(trimmedLine, `"`)

			if value == "" {
				modifiedLines = append(modifiedLines, `""`)
				continue
			}

			if strings.HasPrefix(value, "@") {
				parts := strings.Split(value, "/")
				if len(parts) == 2 && !strings.Contains(parts[1], ":") {
					modifiedLines = append(modifiedLines, fmt.Sprintf(`"@%s:%s"`, parts[1], version))
				} else {
					modifiedLines = append(modifiedLines, fmt.Sprintf(`"%s"`, value))
				}
			} else {
				modifiedLines = append(modifiedLines, fmt.Sprintf(`"@%s/%s:%s"`, name, value, version))
			}
			continue
		}

		// Retain unquoted lines
		modifiedLines = append(modifiedLines, trimmedLine)
	}

	return strings.Join(modifiedLines, "\n")
}
