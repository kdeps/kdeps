package archiver

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/cucumber/godog"
	"github.com/kdeps/kdeps/pkg/enforcer"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resource"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/kdeps/pkg/workflow"
	"github.com/kr/pretty"
	"github.com/spf13/afero"
)

var (
	testFs             = afero.NewOsFs()
	testingT           *testing.T
	aiAgentDir         string
	resourcesDir       string
	logger             *logging.Logger
	dataDir            string
	workflowFile       string
	resourceFile       string
	kdepsDir           string
	projectDir         string
	packageDir         string
	lastCreatedPackage string
	ctx                context.Context
)

func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			ctx.Step(`^a kdeps archive "([^"]*)" is opened$`, aKdepsArchiveIsOpened)
			ctx.Step(`^an ai agent on "([^"]*)" folder exists$`, anAiAgentOnFolder)
			ctx.Step(`^it has a workflow file that has name property "([^"]*)" and version property "([^"]*)" and default action "([^"]*)"$`, itHasAWorkflowFile)
			ctx.Step(`^the content of that archive file will be extracted to "([^"]*)"$`, theContentOfThatArchiveFileWillBeExtractedTo)
			ctx.Step(`^the pkl files is valid$`, thePklFilesIsValid)
			ctx.Step(`^the project is valid$`, theProjectIsValid)
			ctx.Step(`^the project will be archived to "([^"]*)"$`, theProjectWillBeArchivedTo)
			ctx.Step(`^the "([^"]*)" system folder exists$`, theSystemFolderExists)
			ctx.Step(`^theres a data file$`, theresADataFile)
			ctx.Step(`^the pkl files is invalid$`, thePklFilesIsInvalid)
			ctx.Step(`^the project is invalid$`, theProjectIsInvalid)
			ctx.Step(`^the project will not be archived to "([^"]*)"$`, theProjectWillNotBeArchivedTo)

			ctx.Step(`^it has a "([^"]*)" file with id property "([^"]*)" and dependent on "([^"]*)"$`, itHasAFileWithIDPropertyAndDependentOn)
			ctx.Step(`^it has a "([^"]*)" file with no dependency with id property "([^"]*)"$`, itHasAFileWithNoDependencyWithIDProperty)
			ctx.Step(`^it will be stored to "([^"]*)"$`, itWillBeStoredTo)
			ctx.Step(`^the project is compiled$`, theProjectIsCompiled)
			ctx.Step(`^the resource id for "([^"]*)" will be "([^"]*)" and dependency "([^"]*)"$`, theResourceIDForWillBeAndDependency)
			ctx.Step(`^the resource id for "([^"]*)" will be rewritten to "([^"]*)"$`, theResourceIDForWillBeRewrittenTo)
			ctx.Step(`^the workflow action configuration will be rewritten to "([^"]*)"$`, theWorkflowActionConfigurationWillBeRewrittenTo)
			ctx.Step(`^the resources and data folder exists$`, theResourcesAndDataFolderExists)
			ctx.Step(`^the data files will be copied to "([^"]*)"$`, theDataFilesWillBeCopiedTo)
			ctx.Step(`^the package file "([^"]*)" will be created$`, thePackageFileWillBeCreated)
			ctx.Step(`^it has a workflow file that has name property "([^"]*)" and version property "([^"]*)" and default action "([^"]*)" and workspaces "([^"]*)"$`, itHasAWorkflowFileDependencies)
			ctx.Step(`^the resource file "([^"]*)" exists in the "([^"]*)" agent "([^"]*)"$`, theResourceFileExistsInTheAgent)
			ctx.Step(`^it has a "([^"]*)" file with id property "([^"]*)" and dependent on "([^"]*)" with run block "([^"]*)" and is not null$`, itHasAFileWithIDPropertyAndDependentOnWithRunBlockAndIsNotNull)
			ctx.Step(`^it has a "([^"]*)" file with id property "([^"]*)" and dependent on "([^"]*)" with run block "([^"]*)" and is null$`, itHasAFileWithIDPropertyAndDependentOnWithRunBlockAndIsNull)
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/archiver"},
			TestingT: t, // Testing instance that will run subtests.
		},
	}

	testingT = t

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}

func aKdepsArchiveIsOpened(arg1 string) error {
	name, version := regexp.MustCompile(`^([a-zA-Z]+)-([\d]+\.[\d]+\.[\d]+)\.kdeps$`).FindStringSubmatch(arg1)[1], regexp.MustCompile(`^([a-zA-Z]+)-([\d]+\.[\d]+\.[\d]+)\.kdeps$`).FindStringSubmatch(arg1)[2]

	kdepsAgentPath := filepath.Join(kdepsDir, "agents/"+name+"/"+version)
	if _, err := testFs.Stat(kdepsAgentPath); err == nil {
		return errors.New("agent should not yet exists on system agents dir")
	}

	proj, err := ExtractPackage(testFs, ctx, kdepsDir, lastCreatedPackage, logger)
	if err != nil {
		return err
	}

	fmt.Printf("%# v", pretty.Formatter(proj))

	return nil
}

func theSystemFolderExists(arg1 string) error {
	logger = logging.NewTestLogger()
	tempDir, err := afero.TempDir(testFs, "", arg1)
	if err != nil {
		return err
	}

	kdepsDir = tempDir

	packageDir = kdepsDir + "/packages"
	if err := testFs.MkdirAll(packageDir, 0o755); err != nil {
		return err
	}

	return nil
}

func anAiAgentOnFolder(arg1 string) error {
	tempDir, err := afero.TempDir(testFs, "", arg1)
	if err != nil {
		return err
	}

	aiAgentDir = tempDir

	return nil
}

func itHasAFileWithIDPropertyAndDependentOn(arg1, arg2, arg3 string) error {
	// Check if arg3 is a CSV (contains commas)
	var requiresSection string
	if strings.Contains(arg3, ",") {
		// Split arg3 into multiple values if it's a CSV
		values := strings.Split(arg3, ",")
		var requiresLines []string
		for _, value := range values {
			value = strings.TrimSpace(value) // Trim any leading/trailing whitespace
			requiresLines = append(requiresLines, fmt.Sprintf(`  "%s"`, value))
		}
		requiresSection = "Requires {\n" + strings.Join(requiresLines, "\n") + "\n}"
	} else {
		// Single value case
		requiresSection = fmt.Sprintf(`Requires {
  "%s"
}`, arg3)
	}

	// Create the document with the id and requires block
	doc := fmt.Sprintf(`
amends "package://schema.kdeps.com/core@%s#/Resource.pkl"

ActionID = "%s"
%s
run {
  Exec {
  ["key"] = """
@(exec.stdout["anAction"])
@(exec.stdin["anAction2"])
@(exec.stderr["anAction2"])
@(http.client["anAction3"].response)
@(llm.chat["anAction4"].response)
"""
  }
}
`, schema.SchemaVersion(ctx), arg2, requiresSection)

	// Write to the file
	file := filepath.Join(resourcesDir, arg1)

	f, _ := testFs.Create(file)
	if _, err := f.WriteString(doc); err != nil {
		return err
	}

	f.Close()

	resourceFile = file

	return nil
}

func itWillBeStoredTo(arg1 string) error {
	workflowFile = filepath.Join(kdepsDir, arg1)

	if _, err := testFs.Stat(workflowFile); err != nil {
		return err
	}

	return nil
}

func theProjectIsCompiled() error {
	ctx = context.Background()
	wf, err := workflow.LoadWorkflow(ctx, workflowFile, logger)
	if err != nil {
		return err
	}

	env := &environment.Environment{
		Home: kdepsDir,
		Pwd:  resourcesDir,
	}

	projectDir, _, _ := CompileProject(testFs, ctx, wf, kdepsDir, aiAgentDir, env, logger)

	workflowFile = filepath.Join(projectDir, "workflow.pkl")

	return nil
}

func theResourceIDForWillBeAndDependency(arg1, arg2, arg3 string) error {
	resFile := filepath.Join(projectDir, "resources/"+arg1)
	if _, err := testFs.Stat(resFile); err == nil {
		res, err := resource.LoadResource(ctx, resFile, logger)
		if err != nil {
			return err
		}
		if res.ActionID != arg2 {
			return errors.New("should be equal!")
		}
		found := false
		for _, v := range *res.Requires {
			if v == arg3 {
				found = true
				break
			}
		}

		if !found {
			return errors.New("require found!")
		}
	}

	return nil
}

func theResourceIDForWillBeRewrittenTo(arg1, arg2 string) error {
	resFile := filepath.Join(projectDir, "resources/"+arg1)
	if _, err := testFs.Stat(resFile); err == nil {
		res, err := resource.LoadResource(ctx, resFile, logger)
		if err != nil {
			return err
		}

		if res.ActionID != arg2 {
			return errors.New("should be equal!")
		}
	}

	return nil
}

func theWorkflowActionConfigurationWillBeRewrittenTo(arg1 string) error {
	wf, err := workflow.LoadWorkflow(ctx, workflowFile, logger)
	if err != nil {
		return err
	}

	if wf.GetTargetActionID() != arg1 {
		return fmt.Errorf("%s = %s does not match!", wf.GetTargetActionID(), arg1)
	}

	return nil
}

func theResourcesAndDataFolderExists() error {
	resourcesDir = filepath.Join(aiAgentDir, "resources")
	if err := testFs.MkdirAll(resourcesDir, 0o755); err != nil {
		return err
	}

	dataDir = filepath.Join(aiAgentDir, "data")
	if err := testFs.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}

	return nil
}

func itHasAFileWithNoDependencyWithIDProperty(arg1, arg2 string) error {
	doc := fmt.Sprintf(`
amends "package://schema.kdeps.com/core@%s#/Resource.pkl"

ActionID = "%s"
run {
  Exec {
  ["key"] = """
@(exec.stdout["anAction"])
@(exec.stdin["anAction2"])
@(exec.stderr["anAction2"])
@(http.client["anAction3"].response)
@(llm.chat["anAction4"].response)
"""
  }
}
`, schema.SchemaVersion(ctx), arg2)

	file := filepath.Join(resourcesDir, arg1)

	f, _ := testFs.Create(file)
	if _, err := f.WriteString(doc); err != nil {
		return err
	}
	f.Close()

	resourceFile = file

	return nil
}

func itHasAWorkflowFile(arg1, arg2, arg3 string) error {
	doc := fmt.Sprintf(`
amends "package://schema.kdeps.com/core@%s#/Workflow.pkl"

TargetActionID = "%s"
AgentID = "%s"
Description = "My awesome AI Agent"
Version = "%s"
`, schema.SchemaVersion(ctx), arg3, arg1, arg2)

	file := filepath.Join(aiAgentDir, "workflow.pkl")

	f, _ := testFs.Create(file)
	if _, err := f.WriteString(doc); err != nil {
		return err
	}
	f.Close()

	workflowFile = file

	return nil
}

func theContentOfThatArchiveFileWillBeExtractedTo(arg1 string) error {
	fpath := filepath.Join(kdepsDir, arg1)
	if _, err := testFs.Stat(fpath); err != nil {
		return errors.New("there should be an agent dir present, but none was found")
	}

	return nil
}

func thePklFilesIsValid() error {
	if err := enforcer.EnforcePklTemplateAmendsRules(testFs, ctx, workflowFile, logger); err != nil {
		return err
	}

	return nil
}

func theProjectIsValid() error {
	if err := enforcer.EnforceFolderStructure(testFs, ctx, workflowFile, logger); err != nil {
		return err
	}

	return nil
}

func theProjectWillBeArchivedTo(arg1 string) error {
	wf, err := workflow.LoadWorkflow(ctx, workflowFile, logger)
	if err != nil {
		return err
	}

	fpath, err := PackageProject(testFs, ctx, wf, kdepsDir, aiAgentDir, logger)
	if err != nil {
		return err
	}

	if _, err := testFs.Stat(fpath); err != nil {
		return err
	}

	return nil
}

func theresADataFile() error {
	doc := "THIS IS A TEXT FILE: "

	for x := range 10 {
		num := strconv.Itoa(x)
		file := filepath.Join(dataDir, fmt.Sprintf("textfile-%s.txt", num))

		f, _ := testFs.Create(file)
		if _, err := f.WriteString(doc + num); err != nil {
			return err
		}
		f.Close()
	}

	return nil
}

func theDataFilesWillBeCopiedTo(arg1 string) error {
	file := filepath.Join(kdepsDir, arg1+"/textfile-1.txt")

	if _, err := testFs.Stat(file); err != nil {
		return err
	}

	return nil
}

func thePklFilesIsInvalid() error {
	doc := `
	AgentID = "invalid agent"
	Description = "a not valid configuration"
	Version = "five"
	TargetActionID = "hello World"
	`
	file := filepath.Join(aiAgentDir, "workflow1.pkl")

	f, _ := testFs.Create(file)
	if _, err := f.WriteString(doc); err != nil {
		return err
	}
	f.Close()

	workflowFile = file

	if err := enforcer.EnforcePklTemplateAmendsRules(testFs, ctx, workflowFile, logger); err == nil {
		return errors.New("expected an error, but got nil")
	}

	return nil
}

func theProjectIsInvalid() error {
	if err := enforcer.EnforceFolderStructure(testFs, ctx, workflowFile, logger); err == nil {
		return errors.New("expected an error, but got nil")
	}

	return nil
}

func theProjectWillNotBeArchivedTo(arg1 string) error {
	wf, err := workflow.LoadWorkflow(ctx, workflowFile, logger)
	if err != nil {
		return err
	}

	fpath, err := PackageProject(testFs, ctx, wf, kdepsDir, aiAgentDir, logger)
	if err == nil {
		return errors.New("expected an error, but got nil")
	}

	if _, err := testFs.Stat(fpath); err == nil {
		return errors.New("expected an error, but got nil")
	}

	return nil
}

func thePackageFileWillBeCreated(arg1 string) error {
	fpath := filepath.Join(packageDir, arg1)
	if _, err := testFs.Stat(fpath); err != nil {
		return errors.New("expected a package, but got none")
	}
	lastCreatedPackage = fpath

	return nil
}

func itHasAWorkflowFileDependencies(arg1, arg2, arg3, arg4 string) error {
	var workflowsSection string
	if strings.Contains(arg4, ",") {
		// Split arg3 into multiple values if it's a CSV
		values := strings.Split(arg4, ",")
		var workflowsLines []string
		for _, value := range values {
			value = strings.TrimSpace(value) // Trim any leading/trailing whitespace
			workflowsLines = append(workflowsLines, fmt.Sprintf(`  "%s"`, value))
		}
		workflowsSection = "Workflows {\n" + strings.Join(workflowsLines, "\n") + "\n}"
	} else {
		// Single value case
		workflowsSection = fmt.Sprintf(`Workflows {
  "%s"
}`, arg4)
	}

	doc := fmt.Sprintf(`
amends "package://schema.kdeps.com/core@%s#/Workflow.pkl"

TargetActionID = "%s"
AgentID = "%s"
Description = "My awesome AI Agent"
Version = "%s"
%s
`, schema.SchemaVersion(ctx), arg3, arg1, arg2, workflowsSection)

	file := filepath.Join(aiAgentDir, "workflow.pkl")

	f, _ := testFs.Create(file)
	if _, err := f.WriteString(doc); err != nil {
		return err
	}
	f.Close()

	workflowFile = file

	return nil
}

func theResourceFileExistsInTheAgent(arg1, arg2, arg3 string) error {
	fpath := filepath.Join(kdepsDir, "agents/"+arg2+"/1.0.0/resources/"+arg1)
	if _, err := testFs.Stat(fpath); err != nil {
		return errors.New("expected a package, but got none")
	}

	return nil
}

func itHasAFileWithIDPropertyAndDependentOnWithRunBlockAndIsNotNull(arg1, arg2, arg3, arg4 string) error {
	// Check if arg3 is a CSV (contains commas)
	var requiresSection string
	if strings.Contains(arg3, ",") {
		// Split arg3 into multiple values if it's a CSV
		values := strings.Split(arg3, ",")
		var requiresLines []string
		for _, value := range values {
			value = strings.TrimSpace(value) // Trim any leading/trailing whitespace
			requiresLines = append(requiresLines, fmt.Sprintf(`  "%s"`, value))
		}
		requiresSection = "Requires {\n" + strings.Join(requiresLines, "\n") + "\n}"
	} else {
		// Single value case
		requiresSection = fmt.Sprintf(`Requires {
  "%s"
}`, arg3)
	}

	var fieldSection string
	if strings.Contains(arg4, ",") {
		// Split arg3 into multiple values if it's a CSV
		values := strings.Split(arg4, ",")
		var fieldLines []string
		for _, value := range values {
			value = strings.TrimSpace(value) // Trim any leading/trailing whitespace
			value = strings.Title(value)     // Capitalize for new schema
			fieldLines = append(fieldLines, value+" {\n[\"key\"] = \"\"\"\n@(exec.stdout[\"anAction\"])\n@(exec.stdin[\"anAction2\"])\n@(exec.stderr[\"anAction2\"])\n@(http.client[\"anAction3\"].response)\n@(llm.chat[\"anAction4\"].response)\n\"\"\"\n}")
		}
		fieldSection = "Run {\n" + strings.Join(fieldLines, "\n") + "\n}"
	} else {
		// Single value case
		fieldSection = fmt.Sprintf(`Run {
  %s {
["key"] = """
@(exec.stdout["anAction"])
@(exec.stdin["anAction2"])
@(exec.stderr["anAction2"])
@(http.client["anAction3"].response)
@(llm.chat["anAction4"].response)
"""
  }
}`, strings.Title(arg4))
	}

	// Create the document with the id and requires block
	doc := fmt.Sprintf(`
amends "package://schema.kdeps.com/core@%s#/Resource.pkl"

ActionID = "%s"
%s
%s
`, schema.SchemaVersion(ctx), arg2, requiresSection, fieldSection)

	// Write to the file
	file := filepath.Join(resourcesDir, arg1)

	f, _ := testFs.Create(file)
	if _, err := f.WriteString(doc); err != nil {
		return err
	}
	f.Close()

	resourceFile = file

	return nil
}

func itHasAFileWithIDPropertyAndDependentOnWithRunBlockAndIsNull(arg1, arg2, arg3, arg4 string) error {
	// Check if arg3 is a CSV (contains commas)
	var requiresSection string
	if strings.Contains(arg3, ",") {
		// Split arg3 into multiple values if it's a CSV
		values := strings.Split(arg3, ",")
		var requiresLines []string
		for _, value := range values {
			value = strings.TrimSpace(value) // Trim any leading/trailing whitespace
			requiresLines = append(requiresLines, fmt.Sprintf(`  "%s"`, value))
		}
		requiresSection = "Requires {\n" + strings.Join(requiresLines, "\n") + "\n}"
	} else {
		// Single value case
		requiresSection = fmt.Sprintf(`Requires {
  "%s"
}`, arg3)
	}

	var fieldSection string
	if strings.Contains(arg4, ",") {
		// Split arg3 into multiple values if it's a CSV
		values := strings.Split(arg4, ",")
		var fieldLines []string
		for _, value := range values {
			value = strings.TrimSpace(value) // Trim any leading/trailing whitespace
			value = strings.Title(value)     // Capitalize for new schema
			fieldLines = append(fieldLines, value+"=null")
		}
		fieldSection = "Run {\n" + strings.Join(fieldLines, "\n") + "\n}"
	} else {
		// Single value case
		fieldSection = fmt.Sprintf(`Run {
  %s=null
}`, strings.Title(arg4))
	}

	// Create the document with the id and requires block
	doc := fmt.Sprintf(`
amends "package://schema.kdeps.com/core@%s#/Resource.pkl"

ActionID = "%s"
%s
%s
`, schema.SchemaVersion(ctx), arg2, requiresSection, fieldSection)

	// Write to the file
	file := filepath.Join(resourcesDir, arg1)

	f, _ := testFs.Create(file)
	if _, err := f.WriteString(doc); err != nil {
		return err
	}
	f.Close()

	resourceFile = file

	return nil
}
