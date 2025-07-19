package cmd

import (
	"context"
	"fmt"

	"github.com/kdeps/kdeps/pkg/archiver"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/ui"
	"github.com/kdeps/kdeps/pkg/workflow"
)

// extractRoutes extracts route information from the workflow
func extractRoutes(pkgProject *archiver.KdepsPackage, ctx context.Context, logger *logging.Logger) []ui.RouteInfo {
	var routes []ui.RouteInfo

	// Load workflow configuration
	wfCfg, err := workflow.LoadWorkflow(ctx, pkgProject.Workflow, logger)
	if err != nil {
		logger.Debug("Failed to load workflow for route extraction", "error", err)
		return routes
	}

	// Get settings
	wfSettings := wfCfg.GetSettings()
	if wfSettings == nil {
		return routes
	}

	// Extract API Server routes
	if wfSettings.APIServer != nil && wfSettings.APIServer.Routes != nil {
		apiRoutes := *wfSettings.APIServer.Routes
		for _, route := range apiRoutes {
			if route == nil || route.Path == "" {
				continue
			}

			routeInfo := ui.RouteInfo{
				Path:       route.Path,
				Methods:    route.Methods, // Methods is already []string, no dereferencing needed
				ServerType: "api",
			}

			routes = append(routes, routeInfo)
		}
	}

	// Extract Web Server routes
	if wfSettings.WebServer != nil && wfSettings.WebServer.Routes != nil {
		webRoutes := *wfSettings.WebServer.Routes
		for _, route := range webRoutes {
			if route == nil || route.Path == "" {
				continue
			}

			serverType := "static" // default
			if route.ServerType != nil {
				serverType = string(*route.ServerType)
			}

			routeInfo := ui.RouteInfo{
				Path:       route.Path,
				ServerType: serverType,
			}

			if route.AppPort != nil {
				routeInfo.AppPort = fmt.Sprintf("%d", *route.AppPort)
			}

			routes = append(routes, routeInfo)
		}
	}

	return routes
}