package template

import (
	"errors"
	"fmt"
	"kdeps/pkg/schema"
	"kdeps/pkg/texteditor"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
)

var lightBlue = lipgloss.NewStyle().Foreground(lipgloss.Color("#6495ED")).Bold(true)
var lightGreen = lipgloss.NewStyle().Foreground(lipgloss.Color("#90EE90")).Bold(true)

func printWithDots(message string) {
	fmt.Print(lightBlue.Render(message))
	fmt.Print("...")
	fmt.Println()
}

func GenerateAgent(fs afero.Fs, logger *log.Logger) error {
	var name string

	form := huh.NewInput().
		Title("Configure Your AI Agent").
		Prompt("Enter a name for your AI Agent (no spaces): ").
		Validate(func(input string) error {
			if strings.Contains(input, " ") {
				return errors.New("Agent name cannot contain spaces. Please enter a valid name.")
			}
			return nil
		}).
		Value(&name)

	err := form.Run()
	if err != nil {
		log.Fatal(err)
	}

	mainDir := fmt.Sprintf("./%s", name)
	printWithDots(fmt.Sprintf("Creating main directory: %s", lightGreen.Render(mainDir)))
	err = os.MkdirAll(mainDir, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}
	time.Sleep(80 * time.Millisecond)

	printWithDots(fmt.Sprintf("Creating '%s' and '%s' subfolders", lightGreen.Render("resources"), lightGreen.Render("data")))
	err = os.MkdirAll(mainDir+"/resources", os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}
	err = os.MkdirAll(mainDir+"/data", os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}
	time.Sleep(80 * time.Millisecond)

	gitkeepFilePath := mainDir + "/data/.gitkeep"
	printWithDots(fmt.Sprintf("Adding '.gitkeep' in %s", lightGreen.Render(gitkeepFilePath)))
	err = os.WriteFile(gitkeepFilePath, []byte{}, 0644)
	if err != nil {
		log.Fatal(err)
	}
	time.Sleep(80 * time.Millisecond)

	workflowFilePath := mainDir + "/workflow.pkl"
	workflowHeader := fmt.Sprintf(`
amends "package://schema.kdeps.com/core@%s#/Workflow.pkl"
`, schema.SchemaVersion)

	workflowContent := fmt.Sprintf(`%s
name = "%s"
description = "My AI Agent"
website = ""
authors {}
documentation = ""
repository = ""

// Version is Required
version = "1.0.0"

// This section defines the default resource action that will be executed
// when this API resource is called.
action = "responseResource1"

// Specify any external resources to use in this AI Agent.
// For example, you can refer to another agent with "@agentName".
workflows {}

settings {
	// When set to false, the agent runs in standalone mode, executing once
	// when the Docker container starts and then stops after all resources
	// have been processed.
	apiServerMode = true

	// The API server block contains settings related to the API configuration.
	//
	// You can access the incoming request details using the following helper functions:
	//
	// - "@(request.path())"
	// - "@(request.method())"
	// - "@(request.headers("HEADER"))"
	// - "@(request.data())"
	// - "@(request.params("PARAMS"))"
	//
	// And use the following functions for file upload related functions
	//
	// - "@(request.file("FILENAME"))"
	// - "@(request.filetype("FILENAME"))"
	// - "@(request.filepath("FILENAME"))"
	// - "@(request.filecount())"
	// - "@(request.files())"
	// - "@(request.filetypes())"
	// - "@(request.filesByType("image/jpeg"))"
	//
	// For example, to use these in your resource, you can define a local variable like this:
	//
	// local xApiHeader = "@(request.headers["X-API-HEADER"])"
	// You can then retrieve the value with "@(xApiHeader)".
	//
	// The "@(...)" syntax enables lazy evaluation, ensuring that values are
	// retrieved only after the result is ready.
	apiServer {
		// Set the host IP address and port number for the AI Agent.
		hostIP = "127.0.0.1"
		portNum = 3000

		// You can define multiple routes for this agent. Each route points to
		// the main action specified in the action setting, so you must define
		// your skip condition on the resources appropriately.
		routes {
			new {
				path = "/api/v1/whois"
				methods {
					"GET" // Allows retrieving data
					"POST" // Allows submitting data
				}
			}
		}
	}

	// This section contains the agent settings that will be used to build
	// the agent's Docker image.
	agentSettings {
		// Specify the custom Ubuntu PPA repos that would contain the packages available
		// for this image.
		ppa {}

		// Specify the Ubuntu packages that should be pre-installed when
		// building this image.
		packages {}

		// List the local Ollama LLM models that will be pre-installed.
		// You can specify multiple models here.
		models {
			// An example of a language model
			"llama3.1"
		}
	}
}
`, workflowHeader, name)
	printWithDots(fmt.Sprintf("Creating workflow file: %s", lightGreen.Render(workflowFilePath)))
	err = os.WriteFile(workflowFilePath, []byte(workflowContent), 0644)
	if err != nil {
		log.Fatal(err)
	}
	time.Sleep(80 * time.Millisecond)

	resourceHeader := fmt.Sprintf(`
amends "package://schema.kdeps.com/core@%s#/Resource.pkl"
`, schema.SchemaVersion)
	resourceFiles := map[string]string{
		"http.pkl": fmt.Sprintf(`%s
id = "httpClientResource1"
name = "HTTP Client"
description = "This resource allows for making API requests using an HTTP client."
category = ""
requires {
	// Define the ID of any dependency resource that must be executed before this resource.
}
run {
	skipCondition {
		// Conditions under which the execution of this resource should be skipped.
		// If any evaluated condition returns true, the resource execution will be bypassed.
		// "@(request.path)" != "/api/v1/whois" && "@(request.method)" != "GET"
	}
	preflightCheck {
		validations {
			// This section expects boolean validations.
			// If any validation returns false, an exception will be thrown before proceeding to the next step.
			// "@(request.header("X-API-KEY"))" != ""
		}
		// Custom error message and code to be used if the preflight check fails.
		error {
			code = 404
			message = "Header X-API-KEY not found in request!"
		}
	}

	// Initiates an HTTP client request for this resource.
	//
	// The HTTP resource provides the following helper functions:
	//
	// - "@(client.resource("ResourceID"))"
	// - "@(client.responseBody("ResourceID"))"
	// - "@(client.responseHeader("ResourceID", "HEADER"))"
	//
	// For example, to use these in your resource, you can define a local variable like this:
	//
	// local bearerToken = "@(client.responseHeader("ResourceID", "Bearer"))"
	// You can then access the value using "@(bearerToken)".
	//
	// The "@(...)" syntax enables lazy evaluation, ensuring that values are
	// retrieved only after the result is ready.
	//
	// Note: Each resource is restricted to a single dedicated action. Combining multiple
	// actions within the same resource is not allowed.
	httpClient {
		method = "GET"  // Specifies the HTTP method to be used for the request.
		url = ""        // The URL endpoint for the HTTP request.
		data {
			// Any data that will be sent with the HTTP request.
		}
		headers {
			// Headers to be included in the HTTP request.
			["X-API-KEY"] = "@(request.header("X-API-KEY"))"  // Example header.
		}
		// Timeout duration in seconds. This specifies when to terminate the request.
		timeoutSeconds = 60
	}
}
`, resourceHeader),
		"exec.pkl": fmt.Sprintf(`%s
id = "shellExecResource1"
name = "Exec Resource"
description = "This resource creates a shell session."
category = ""
requires {
	// Define the ID of any dependency resource that must be executed before this resource.
}
run {
	skipCondition {
		// Conditions under which the execution of this resource should be skipped.
		// If any evaluated condition returns true, the resource execution will be bypassed.
	}
	preflightCheck {
		validations {
			// This section expects boolean validations.
			// If any validation returns false, an exception will be thrown before proceeding to the next step.
			//
			// For example, this expects that the 'file.txt' is in the 'data' folder.
			// All data files are mapped from 'data/file.txt' to 'data/<agentName>/<agentVersion>/file.txt'.
			// read("file:/agent/workflow/data/%s/1.0.0/file.txt").text != "" && read("file:/agent/workflow/data/%s/1.0.0/file.txt").base64 != ""
		}
		// Custom error message and code to be used if the preflight check fails.
		error {
			code = 500
			message = "Data file file.txt not found!"
		}
	}

	// Initiates a shell session for executing commands within this resource. Any packages
	// defined in the workflow are accessible here.
	//
	// The exec resource provides the following helper functions:
	//
	// - "@(exec.resource("ResourceID"))"
	// - "@(exec.stderr("ResourceID"))"
	// - "@(exec.stdout("ResourceID"))"
	// - "@(exec.exitCode("ResourceID"))"
	//
	// To use these in your resource, you can define a local variable like this:
	//
	// local successExec = "@(exec.exitCode("ResourceID"))"
	// You can then reference the value using "@(successExec)".
	//
	// If you need to access a file in your resource, you can use PKL's read("file") API like this:
	// "@(read("file"))".
	//
	// The "@(...)" syntax enables lazy evaluation, ensuring that values are
	// retrieved only after the result is ready.
	//
	// Note: Each resource is restricted to a single dedicated action. Combining multiple
	// actions within the same resource is not allowed.
	exec {
		command = """
		# The command to be executed
		echo "hello world"
		"""
		env {
			// Environment variables that would be accessible inside the shell
			["ENVVAR"] = "XYZ"  // Example ENVVAR.
		}
		// Timeout duration in seconds. This specifies when to terminate the shell exec.
		timeoutSeconds = 60
	}
}
`, resourceHeader, name, name),
		"chat.pkl": fmt.Sprintf(`%s
id = "chatResource1"
name = "LLM Chat Resource"
description = "This resource creates a LLM chat session."
category = ""
requires {
	// Define the ID of any dependency resource that must be executed before this resource.
	// For example "@aiChatResource1"
}
run {
	skipCondition {
		// Conditions under which the execution of this resource should be skipped.
		// If any evaluated condition returns true, the resource execution will be bypassed.
	}
	preflightCheck {
		validations {
			// This section expects boolean validations.
			// If any validation returns false, an exception will be thrown before proceeding to the next step.
		}
		// Custom error message and code to be used if the preflight check fails.
		error {
			code = 0
			message = ""
		}
	}

	// Initializes a chat session with the LLM for this resource.
	//
	// This resource offers the following helper functions:
	//
	// - "@(llm.response("ResourceID"))"
	// - "@(llm.prompt("ResourceID"))"
	//
	// To use these in your resource, you can define a local variable as follows:
	//
	// local llmResponse = "@(llm.response("ResourceID"))"
	// You can then access the value with "@(llmResponse)".
	//
	// The "@(...)" syntax enables lazy evaluation, ensuring that values are
	// retrieved only after the result is ready.
	//
	// Note: Each resource is restricted to a single dedicated action. Combining multiple
	// actions within the same resource is not allowed.
	chat {
		model = "llama3.2" // This LLM model needs to be defined in the workflow
		prompt = "Who is @(request.data())?"

		// Specify if the LLM response should be a structured JSON
		jsonResponse = true

		// If jsonResponse is true, then the structured JSON data will need to have the
		// following keys.
		jsonResponseKeys {
			"first_name"
			"last_name"
			"parents"
			"address"
			"famous_quotes"
			"known_for"
		}

		// Timeout duration in seconds. This specifies when to terminate the llm session.
		timeoutSeconds = 60
	}
}
`, resourceHeader),
		"response.pkl": fmt.Sprintf(`%s
id = "responseResource1"
name = "API Response Resource"
description = "This resource creates a API response."
category = ""
requires {
	"chatResource1"
	// Define the ID of any dependency resource that must be executed before this resource.
	// For example "aiChatResource1"
}
run {
	skipCondition {
		// Conditions under which the execution of this resource should be skipped.
		// If any evaluated condition returns true, the resource execution will be bypassed.
	}
	preflightCheck {
		validations {
			// This section expects boolean validations.
			// If any validation returns false, an exception will be thrown before proceeding to the next step.
		}
		// Custom error message and code to be used if the preflight check fails.
		error {
			code = 0
			message = ""
		}
	}

	// Initializes an api response for this agent.
	//
	// This resource action is straightforward. It
	// creates a JSON response with the following shape
	//
	// {
	//   "success": true,
		//   "response": {
			//     "data": [],
			//   },
		//   "errors": {
			//     "code": 0,
			//     "message": ""
			//   }
		// }
	//
	apiResponse {
		success = true
		response {
			data {
				"@(llm.response("chatResource1"))"
			}
		}
		errors {
			code = 0
			message = ""
		}
	}
}
`, resourceHeader),
	}

	for fileName, content := range resourceFiles {
		filePath := fmt.Sprintf("%s/resources/%s", mainDir, fileName)
		printWithDots(fmt.Sprintf("Creating resource file: %s", lightGreen.Render(filePath)))
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			log.Fatal(err)
		}
		time.Sleep(80 * time.Millisecond)
	}

	var openWorkflow bool
	editorForm := huh.NewConfirm().
		Title("Edit the AI agent in Editor?").
		Affirmative("Yes").
		Negative("No").
		Value(&openWorkflow)

	err = editorForm.Run()
	if err != nil {
		log.Fatal(err)
	}

	if openWorkflow {
		if err := texteditor.EditPkl(fs, workflowFilePath, logger); err != nil {
			return fmt.Errorf("failed to edit workflow file: %w", err)
		}
	}

	printWithDots("Workflow generated successfully")
	return nil
}
