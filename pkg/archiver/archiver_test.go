package archiver

import (
	"errors"
	"fmt"
	"kdeps/pkg/enforcer"
	"kdeps/pkg/workflow"
	"path/filepath"
	"testing"

	"github.com/cucumber/godog"
	"github.com/spf13/afero"
)

var (
	testFs       = afero.NewOsFs()
	testingT     *testing.T
	aiAgentDir   string
	resourcesDir string
	workflowFile string
	resourceFile string
	kdepsDir     string
	packageDir   string
)

func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			ctx.Step(`^a kdeps archive "([^"]*)" is passed$`, aKdepsArchiveIsPassed)
			ctx.Step(`^an ai agent on "([^"]*)" folder exists$`, anAiAgentOnFolder)
			ctx.Step(`^it has a resource file with id property "([^"]*)"$`, itHasAResourceFileWithIdProperty)
			ctx.Step(`^it has a workflow file that has name property "([^"]*)" and version property "([^"]*)"$`, itHasAWorkflowFileThatHasNamePropertyAndVersionProperty)
			ctx.Step(`^the content of that archive file will be extracted to "([^"]*)"$`, theContentOfThatArchiveFileWillBeExtractedTo)
			ctx.Step(`^the pkl files is valid$`, thePklFilesIsValid)
			ctx.Step(`^the project is valid$`, theProjectIsValid)
			ctx.Step(`^the project will be archived to "([^"]*)"$`, theProjectWillBeArchivedTo)
			ctx.Step(`^the "([^"]*)" system folder exists$`, theSystemFolderExists)
			ctx.Step(`^theres a data file$`, theresADataFile)
			ctx.Step(`^the pkl files is invalid$`, thePklFilesIsInvalid)
			ctx.Step(`^the project is invalid$`, theProjectIsInvalid)
			ctx.Step(`^the project will not be archived to "([^"]*)"$`, theProjectWillNotBeArchivedTo)
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

func aKdepsArchiveIsPassed(arg1 string) error {
	return godog.ErrPending
}

func theSystemFolderExists(arg1 string) error {
	tempDir, err := afero.TempDir(testFs, "", arg1)
	if err != nil {
		return err
	}

	kdepsDir = tempDir

	packageDir = kdepsDir + "/packages"
	if err := testFs.MkdirAll(packageDir, 0755); err != nil {
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

func itHasAResourceFileWithIdProperty(arg1 string) error {
	resourcesDir = aiAgentDir + "/resources"
	if err := testFs.MkdirAll(resourcesDir, 0755); err != nil {
		return err
	}

	doc := fmt.Sprintf(`
amends "package://schema.kdeps.com/core@0.0.32#/Resource.pkl"

id = "%s"
`, arg1)

	file := filepath.Join(resourcesDir, "resource1.pkl")

	f, _ := testFs.Create(file)
	f.WriteString(doc)
	f.Close()

	resourceFile = file

	return nil
}

func itHasAWorkflowFileThatHasNamePropertyAndVersionProperty(arg1, arg2 string) error {
	doc := fmt.Sprintf(`
amends "package://schema.kdeps.com/core@0.0.32#/Workflow.pkl"

name = "%s"
description = "My awesome AI Agent"
version = "%s"
action = "helloWorld"
`, arg1, arg2)

	file := filepath.Join(aiAgentDir, "workflow.pkl")

	f, _ := testFs.Create(file)
	f.WriteString(doc)
	f.Close()

	workflowFile = file

	return nil
}

func theContentOfThatArchiveFileWillBeExtractedTo(arg1 string) error {
	return godog.ErrPending
}

func thePklFilesIsValid() error {
	if err := enforcer.EnforcePklTemplateAmendsRules(testFs, workflowFile, schemaVersionFilePath); err != nil {
		return err
	}

	if err := enforcer.EnforcePklTemplateAmendsRules(testFs, resourceFile, schemaVersionFilePath); err != nil {
		return err
	}

	return nil
}

func theProjectIsValid() error {
	if err := enforcer.EnforceFolderStructure(testFs, workflowFile); err != nil {
		return err
	}

	return nil
}

func theProjectWillBeArchivedTo(arg1 string) error {
	wf, err := workflow.LoadConfiguration(testFs, workflowFile)
	if err != nil {
		return err
	}

	fpath, err := PackageProject(testFs, wf, kdepsDir, aiAgentDir)
	if err != nil {
		return err
	}

	if _, err := testFs.Stat(fpath); err != nil {
		return err
	}

	fmt.Printf("Package file '%s' created!", fpath)

	return nil
}

func theresADataFile() error {
	dataDir := aiAgentDir + "/data"
	if err := testFs.MkdirAll(dataDir, 0755); err != nil {
		return err
	}

	doc := "THIS IS A TEXT FILE"

	file := filepath.Join(dataDir, "textfile.txt")

	f, _ := testFs.Create(file)
	f.WriteString(doc)
	f.Close()

	return nil
}

func thePklFilesIsInvalid() error {
	doc := `
name = "invalid agent"
description = "a not valid configuration"
version = "five"
action = "hello World"
`
	file := filepath.Join(aiAgentDir, "workflow1.pkl")

	f, _ := testFs.Create(file)
	f.WriteString(doc)
	f.Close()

	workflowFile = file

	if err := enforcer.EnforcePklTemplateAmendsRules(testFs, workflowFile, schemaVersionFilePath); err == nil {
		return errors.New("expected an error, but got nil")
	}

	return nil
}

func theProjectIsInvalid() error {
	if err := enforcer.EnforceFolderStructure(testFs, workflowFile); err == nil {
		return errors.New("expected an error, but got nil")
	}

	return nil
}

func theProjectWillNotBeArchivedTo(arg1 string) error {
	wf, err := workflow.LoadConfiguration(testFs, workflowFile)
	if err != nil {
		return err
	}

	fpath, err := PackageProject(testFs, wf, kdepsDir, aiAgentDir)
	if err == nil {
		return errors.New("expected an error, but got nil")
	}

	if _, err := testFs.Stat(fpath); err == nil {
		return errors.New("expected an error, but got nil")
	}

	return nil
}
