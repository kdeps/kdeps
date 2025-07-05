package archiver

import (
	"strings"
	"testing"

	pklProject "github.com/kdeps/schema/gen/project"
	"github.com/stretchr/testify/require"
)

// stubWorkflow provides only the methods required by handleRequiresBlock tests.
type stubWorkflow struct {
	name    string
	version string
}

func (s stubWorkflow) GetAgentID() string { return s.name }
func (s stubWorkflow) GetName() string    { return s.name }
func (s stubWorkflow) GetVersion() string { return s.version }

// Below we satisfy the full interface with dummy methods so the compiler is happy.
func (s stubWorkflow) GetDescription() *string           { desc := ""; return &desc }
func (s stubWorkflow) GetWebsite() *string               { return nil }
func (s stubWorkflow) GetAuthors() *[]string             { return nil }
func (s stubWorkflow) GetDocumentation() *string         { return nil }
func (s stubWorkflow) GetRepository() *string            { return nil }
func (s stubWorkflow) GetHeroImage() *string             { return nil }
func (s stubWorkflow) GetAgentIcon() *string             { return nil }
func (s stubWorkflow) GetTargetActionID() string         { return "" }
func (s stubWorkflow) GetWorkflows() []string            { return nil }
func (s stubWorkflow) GetSettings() *pklProject.Settings { return nil }

func TestHandleRequiresBlock(t *testing.T) {
	wf := stubWorkflow{name: "chatBot", version: "1.2.3"}

	input := strings.Join([]string{
		"",                        // blank should be preserved
		"    \"\"",                // quoted empty
		"    \"@otherAgent/foo\"", // @-prefixed without version
		"    \"localAction\"",     // plain quoted value
		"    unquoted",            // unquoted retains verbatim
	}, "\n")

	got := handleRequiresBlock(input, wf)
	lines := strings.Split(got, "\n")

	require.Equal(t, "", lines[0], "blank line must stay blank")
	require.Equal(t, "\"\"", strings.TrimSpace(lines[1]))
	require.Equal(t, "\"@foo:1.2.3\"", strings.TrimSpace(lines[2]), "@otherAgent/foo should map to version only")
	require.Equal(t, "\"@chatBot/localAction:1.2.3\"", strings.TrimSpace(lines[3]))
	require.Equal(t, "unquoted", strings.TrimSpace(lines[4]))
}
