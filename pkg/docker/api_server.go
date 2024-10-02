package docker

import (
	"context"
	"fmt"
	"io/ioutil"
	"kdeps/pkg/environment"
	"kdeps/pkg/evaluator"
	"kdeps/pkg/logging"
	"kdeps/pkg/resolver"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	apiserver "github.com/kdeps/schema/gen/api_server"
	pklWf "github.com/kdeps/schema/gen/workflow"
	"github.com/spf13/afero"
)

func StartApiServerMode(fs afero.Fs, ctx context.Context, wfCfg *pklWf.Workflow, environ *environment.Environment, agentDir string) error {
	// Extracting workflow settings and API server config
	wfSettings := *wfCfg.Settings
	wfApiServer := wfSettings.ApiServer

	if wfApiServer == nil {
		return fmt.Errorf("API server configuration is missing")
	}

	portNum := wfApiServer.PortNum
	hostPort := ":" + strconv.FormatUint(uint64(portNum), 10) // Format port for ListenAndServe

	// Set up routes from the configuration
	routes := wfApiServer.Routes
	for _, route := range routes {
		http.HandleFunc(route.Path, ApiServerHandler(fs, ctx, route, environ, agentDir))
	}

	// Start the server
	log.Printf("Starting API server on port %s", hostPort)
	go func() error {
		if err := http.ListenAndServe(hostPort, nil); err != nil {
			// Return the error instead of log.Fatal to allow better error handling
			return fmt.Errorf("failed to start API server: %w", err)
		}
		return nil
	}()

	return nil
}

func ApiServerHandler(fs afero.Fs, ctx context.Context, route *apiserver.APIServerRoutes, env *environment.Environment, apiServerPath string) http.HandlerFunc {
	var responseFileExt string
	var contentType string
	var responseFlagFile string

	switch route.ResponseType {
	case "jsonnet":
		responseFlagFile = "response-jsonnet"
		responseFileExt = ".json"
		contentType = "application/json"
	case "textproto":
		responseFlagFile = "response-txtpb"
		responseFileExt = ".txtpb"
		contentType = "application/protobuf"
	case "yaml":
		responseFlagFile = "response-yaml"
		responseFileExt = ".yaml"
		contentType = "application/yaml"
	case "plist":
		responseFlagFile = "response-plist"
		responseFileExt = ".plist"
		contentType = "application/yaml"
	case "xml":
		responseFlagFile = "response-xml"
		responseFileExt = ".xml"
		contentType = "application/yaml"
	case "pcf":
		responseFlagFile = "response-pcf"
		responseFileExt = ".pcf"
		contentType = "application/yaml"
	default:
		responseFlagFile = "response-json"
		responseFileExt = ".json"
		contentType = "application/json"
	}

	responseFlag := filepath.Join(apiServerPath, responseFlagFile)
	responseFile := filepath.Join(apiServerPath, "response"+responseFileExt)
	requestPklFile := filepath.Join(apiServerPath, "request.pkl")

	allowedMethods := route.Methods

	var paramSection string
	var headerSection string
	var dataSection string
	var url string
	var method string

	dr, err := resolver.NewGraphResolver(fs, nil, ctx, env, "/agent")
	if err != nil {
		log.Fatal(err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if _, err := fs.Stat(responseFile); err == nil {
			if err := fs.RemoveAll(responseFile); err != nil {
				logging.Error("Unable to delete old response file", "response-file", responseFile)
				return
			}
		}
		if _, err := fs.Stat(responseFlag); err == nil {
			if err := fs.RemoveAll(responseFlag); err != nil {
				logging.Error("Unable to delete old response flag file", "response-flag", responseFlag)
				return
			}
		}

		url = fmt.Sprintf(`url = "%s"`, r.URL.Path)

		if r.Method == "" {
			r.Method = "GET"
		}

		for _, allowedMethod := range allowedMethods {
			if allowedMethod == r.Method {
				method = fmt.Sprintf(`method = "%s"`, allowedMethod)

				break
			}
		}

		if method == "" {
			http.Error(w, fmt.Sprintf(`HTTP method "%s" not allowed!`, r.Method), http.StatusBadRequest)

			return
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)

			return
		}
		defer r.Body.Close()
		dataSection = fmt.Sprintf(`data = "%s"`, string(body))
		var paramsLines []string
		var headersLines []string

		params := r.URL.Query()
		for param, values := range params {
			for _, value := range values {
				value = strings.TrimSpace(value) // Trim any leading/trailing whitespace
				paramsLines = append(paramsLines, fmt.Sprintf(`["%s"] = "%s"`, param, value))
			}
		}
		paramSection = "params {\n" + strings.Join(paramsLines, "\n") + "\n}"

		for name, values := range r.Header {
			for _, value := range values {
				value = strings.TrimSpace(value) // Trim any leading/trailing whitespace
				headersLines = append(headersLines, fmt.Sprintf(`["%s"] = "%s"`, name, value))
			}
		}
		headerSection = "headers {\n" + strings.Join(headersLines, "\n") + "\n}"

		sections := []string{url, method, headerSection, dataSection, paramSection}

		if err := evaluator.CreateAndProcessPklFile(fs, sections, requestPklFile, "APIServerRequest.pkl",
			nil, evaluator.EvalPkl); err != nil {
			return
		}

		if err = CreateFlagFile(fs, responseFlag); err != nil {
			return
		}

		// Wait for the file to exist before responding
		for {
			if err := dr.PrepareWorkflowDir(); err != nil {
				log.Fatal(err)
			}

			if err := dr.PrepareImportFiles(); err != nil {
				log.Fatal(err)
			}

			if err := dr.HandleRunAction(); err != nil {
				log.Fatal(err)
			}

			stdout, err := dr.EvalPklFormattedResponseFile()
			if err != nil {
				log.Fatal(fmt.Errorf(stdout, err))
			}

			logging.Info("Awaiting for response...")
			if err := resolver.WaitForFile(fs, dr.ResponseTargetFile); err != nil {
				log.Fatal(err)
			}

			// File exists, now respond with its contents
			content, err := afero.ReadFile(fs, dr.ResponseTargetFile)
			if err != nil {
				http.Error(w, "Failed to read file", http.StatusInternalServerError)
				return
			}

			// Write the content to the response
			w.Header().Set("Content-Type", contentType)
			w.WriteHeader(http.StatusOK)
			w.Write(content)

			return
		}
	}
}
