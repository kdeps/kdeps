package archiver

import (
	"fmt"
	"strings"

	pklWf "github.com/kdeps/schema/gen/workflow"
)

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
