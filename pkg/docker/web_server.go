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
	"github.com/kdeps/kdeps/pkg/logging"
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

	hostIP := wfWebServer.HostIP
	portNum := strconv.FormatUint(uint64(wfWebServer.PortNum), 10)
	hostPort := ":" + portNum

	router := gin.Default()

	setupWebRoutes(router, ctx, hostIP, wfTrustedProxies, wfWebServer.Routes, dr)

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
	fullPath := filepath.Join(dr.DataDir, route.PublicPath)

	// Log directory contents for debugging
	logDirectoryContents(dr, fullPath, logger)

	// Start app command if needed
	startAppCommand(ctx, route, fullPath, logger)

	return func(c *gin.Context) {
		switch route.ServerType {
		case webservertype.Static:
			handleStaticRequest(c, fullPath, route)
		case webservertype.App:
			handleAppRequest(c, hostIP, route, logger)
		default:
			logger.Error("unsupported server type", "type", route.ServerType)
			c.String(http.StatusInternalServerError, "500: Unsupported server type")
		}
	}
}

func logDirectoryContents(dr *resolver.DependencyResolver, fullPath string, logger *logging.Logger) {
	entries, err := afero.ReadDir(dr.Fs, fullPath)
	if err != nil {
		logger.Error("failed to read directory", "path", fullPath, "error", err)
		return
	}
	for _, entry := range entries {
		logger.Debug("found file", "name", entry.Name(), "isDir", entry.IsDir())
	}
}

func startAppCommand(ctx context.Context, route *webserver.WebServerRoutes, fullPath string, logger *logging.Logger) {
	if route.ServerType == webservertype.App && route.Command != nil {
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

func handleStaticRequest(c *gin.Context, fullPath string, route *webserver.WebServerRoutes) {
	// Use the standard file server, stripping the route prefix
	fileServer := http.StripPrefix(route.Path, http.FileServer(http.Dir(fullPath)))
	fileServer.ServeHTTP(c.Writer, c.Request)
}

func handleAppRequest(c *gin.Context, hostIP string, route *webserver.WebServerRoutes, logger *logging.Logger) {
	portNum := strconv.FormatUint(uint64(*route.AppPort), 10)
	if hostIP == "" || portNum == "" {
		logger.Error("proxy host or port not configured", "host", hostIP, "port", portNum)
		c.String(http.StatusInternalServerError, "500: Proxy host or port not configured")
		return
	}

	targetURL, err := url.Parse("http://" + net.JoinHostPort(hostIP, portNum))
	if err != nil {
		logger.Error("invalid proxy URL", "host", hostIP, "port", portNum, "error", err)
		c.String(http.StatusInternalServerError, "500: Invalid proxy URL")
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
		logger.Debug("proxying request", "url", req.URL.String())
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		logger.Error("failed to proxy request", "url", r.URL.String(), "error", err)
		c.String(http.StatusBadGateway, "502: Failed to reach app server")
	}

	proxy.ServeHTTP(c.Writer, c.Request)
}
