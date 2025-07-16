package docker

import (
	"context"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kdeps/kdeps/pkg"
	"github.com/kdeps/kdeps/pkg/config"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/messages"
	"github.com/kdeps/kdeps/pkg/resolver"
	webserver "github.com/kdeps/schema/gen/web_server"
	"github.com/kdeps/schema/gen/web_server/webservertype"
	"github.com/spf13/afero"
)

// StartWebServerMode initializes and starts the Web server based on the provided workflow configuration.
// It validates the Web server configuration, sets up routes, and starts the server on the configured port.
func StartWebServerMode(ctx context.Context, dr *resolver.DependencyResolver) error {
	wfSettings := dr.Workflow.GetSettings()
	wfWebServer := wfSettings.WebServer
	var wfTrustedProxies []string
	if wfWebServer.TrustedProxies != nil {
		wfTrustedProxies = *wfWebServer.TrustedProxies
	}

	// Use the new configuration processor for PKL-first config
	processor := config.NewConfigurationProcessor(dr.Logger)
	processedConfig, err := processor.ProcessWorkflowConfiguration(ctx, dr.Workflow)
	if err != nil {
		return err // or appropriate error handling
	}

	// Validate configuration
	if err := processor.ValidateConfiguration(processedConfig); err != nil {
		return err // or appropriate error handling
	}

	// Use processedConfig for all config values
	webHostIP := processedConfig.WebServerHostIP.Value
	webPortNum := processedConfig.WebServerPort.Value
	hostPort := ":" + strconv.FormatUint(uint64(webPortNum), 10)

	router := gin.Default()

	var routes []*webserver.WebServerRoutes
	if wfWebServer.Routes != nil {
		routes = *wfWebServer.Routes
	}

	setupWebRoutes(router, ctx, webHostIP, wfTrustedProxies, routes, dr)

	dr.Logger.Printf("Starting Web server on port %s", hostPort)

	go func() {
		if err := router.Run(hostPort); err != nil {
			dr.Logger.Error("failed to start Web server", "error", err)
		}
	}()

	return nil
}

func setupWebRoutes(router *gin.Engine, ctx context.Context, hostIP string, wfTrustedProxies []string, routes []*webserver.WebServerRoutes, dr *resolver.DependencyResolver) {
	for _, route := range routes {
		if route == nil || route.Path == "" {
			dr.Logger.Error("route configuration is invalid", "route", route)
			continue
		}

		handler := WebServerHandler(ctx, hostIP, route, dr)

		if len(wfTrustedProxies) > 0 {
			dr.Logger.Printf("Found trusted proxies %v", wfTrustedProxies)

			router.ForwardedByClientIP = true
			if err := router.SetTrustedProxies(wfTrustedProxies); err != nil {
				dr.Logger.Error("unable to set trusted proxies")
			}
		}

		router.Any(route.Path+"/*filepath", handler)

		dr.Logger.Printf("Web server route configured: %s", route.Path)
	}
}

func WebServerHandler(ctx context.Context, hostIP string, route *webserver.WebServerRoutes, dr *resolver.DependencyResolver) gin.HandlerFunc {
	logger := dr.Logger.With("webserver", route.Path)

	// Handle PublicPath pointer using default fallback
	publicPath := pkg.DefaultPublicPath
	if route.PublicPath != nil {
		publicPath = *route.PublicPath
	}
	fullPath := filepath.Join(dr.DataDir, publicPath)

	// Log directory contents for debugging
	LogDirectoryContents(dr, fullPath, logger)

	// Start app command if needed
	startAppCommand(ctx, route, fullPath, logger)

	return func(c *gin.Context) {
		// Handle ServerType pointer
		var serverType webservertype.WebServerType
		if route.ServerType != nil {
			serverType = *route.ServerType
		} else {
			serverType = webservertype.Static // default
		}

		switch serverType {
		case webservertype.Static:
			HandleStaticRequest(c, fullPath, route)
		case webservertype.App:
			HandleAppRequest(c, hostIP, route, logger)
		default:
			logger.Error(messages.ErrUnsupportedServerType, "type", serverType)
			c.String(http.StatusInternalServerError, messages.RespUnsupportedServerType)
		}
	}
}

// Exported for testing
func HandleAppRequest(c *gin.Context, hostIP string, route *webserver.WebServerRoutes, logger *logging.Logger) {
	portNum := strconv.FormatUint(uint64(*route.AppPort), 10)
	if hostIP == "" || portNum == "" {
		logger.Error(messages.ErrProxyHostPortMissing, "host", hostIP, "port", portNum)
		c.String(http.StatusInternalServerError, messages.RespProxyHostPortMissing)
		return
	}

	targetURL, err := url.Parse("http://" + net.JoinHostPort(hostIP, portNum))
	if err != nil {
		logger.Error(messages.ErrInvalidProxyURL, "host", hostIP, "port", portNum, "error", err)
		c.String(http.StatusInternalServerError, messages.RespInvalidProxyURL)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.Transport = &http.Transport{
		ResponseHeaderTimeout: 30 * time.Second,
	}

	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = targetURL.Scheme
		req.URL.Host = targetURL.Host
		req.URL.Path = strings.TrimPrefix(c.Request.URL.Path, route.Path)
		req.URL.RawQuery = c.Request.URL.RawQuery
		req.Host = targetURL.Host
		for key, values := range c.Request.Header {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
		logger.Debug(messages.MsgProxyingRequest, "url", req.URL.String())
	}

	proxy.ErrorHandler = func(_ http.ResponseWriter, r *http.Request, err error) {
		logger.Error(messages.ErrFailedProxyRequest, "url", r.URL.String(), "error", err)
		c.String(http.StatusBadGateway, messages.RespFailedReachApp)
	}

	proxy.ServeHTTP(c.Writer, c.Request)
}

func LogDirectoryContents(dr *resolver.DependencyResolver, fullPath string, logger *logging.Logger) {
	entries, err := afero.ReadDir(dr.Fs, fullPath)
	if err != nil {
		logger.Error("failed to read directory", "path", fullPath, "error", err)
		return
	}
	for _, entry := range entries {
		logger.Debug(messages.MsgLogDirFoundFile, "name", entry.Name(), "isDir", entry.IsDir())
	}
}

func HandleStaticRequest(c *gin.Context, fullPath string, route *webserver.WebServerRoutes) {
	// Use the standard file server, stripping the route prefix
	fileServer := http.StripPrefix(route.Path, http.FileServer(http.Dir(fullPath)))
	fileServer.ServeHTTP(c.Writer, c.Request)
}

func startAppCommand(ctx context.Context, route *webserver.WebServerRoutes, fullPath string, logger *logging.Logger) {
	// Check if ServerType is App and Command is provided
	isApp := route.ServerType != nil && *route.ServerType == webservertype.App
	if isApp && route.Command != nil {
		_, _, _, err := KdepsExec(
			ctx,
			"sh", []string{"-c", *route.Command},
			fullPath,
			true,
			true,
			logger.With("webserver command", *route.Command),
		)
		if err != nil {
			logger.Error("failed to start app command", "error", err)
		}
	}
}

// Exported for testing
func StartAppCommand(ctx context.Context, route *webserver.WebServerRoutes, fullPath string, logger *logging.Logger) {
	startAppCommand(ctx, route, fullPath, logger)
}

// Exported for testing
func SetupWebRoutes(router *gin.Engine, ctx context.Context, hostIP string, wfTrustedProxies []string, routes []*webserver.WebServerRoutes, dr *resolver.DependencyResolver) {
	setupWebRoutes(router, ctx, hostIP, wfTrustedProxies, routes, dr)
}
