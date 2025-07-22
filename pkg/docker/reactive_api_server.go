package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/reactive"
	"github.com/kdeps/kdeps/pkg/resolver"
)

// ReactiveAPIServer provides a reactive API server
type ReactiveAPIServer struct {
	resolver     *resolver.DependencyResolver
	logger       *logging.Logger
	engine       *gin.Engine
	ctx          context.Context
	cancel       context.CancelFunc
	subscription reactive.Subscription
}

// ReactiveAPIResponse represents API responses with reactive data
type ReactiveAPIResponse struct {
	Success   bool              `json:"success"`
	Timestamp int64             `json:"timestamp"`
	Data      interface{}       `json:"data,omitempty"`
	State     reactive.AppState `json:"state,omitempty"`
	Error     string            `json:"error,omitempty"`
	RequestID string            `json:"requestId,omitempty"`
}

// NewReactiveAPIServer creates a new reactive API server
func NewReactiveAPIServer(resolver *resolver.DependencyResolver, logger *logging.Logger) *ReactiveAPIServer {
	ctx, cancel := context.WithCancel(context.Background())

	// Set up Gin router
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())

	// CORS configuration for reactive endpoints
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	engine.Use(cors.New(corsConfig))

	server := &ReactiveAPIServer{
		resolver: resolver,
		logger:   logger,
		engine:   engine,
		ctx:      ctx,
		cancel:   cancel,
	}

	server.setupReactiveRoutes()
	server.subscribeToReactiveState()

	return server
}

// setupReactiveRoutes sets up reactive API endpoints
func (ras *ReactiveAPIServer) setupReactiveRoutes() {
	api := ras.engine.Group("/api/v1")

	// State endpoints
	api.GET("/state", ras.handleGetState)
	api.GET("/state/stream", ras.handleStateStream)

	// Operations endpoints
	api.GET("/operations", ras.handleGetOperations)
	api.GET("/operations/:id", ras.handleGetOperation)
	api.POST("/operations", ras.handleCreateOperation)
	api.PUT("/operations/:id", ras.handleUpdateOperation)
	api.DELETE("/operations/:id", ras.handleDeleteOperation)

	// Resources endpoints
	api.GET("/resources", ras.handleGetResources)
	api.GET("/resources/:id", ras.handleGetResource)
	api.POST("/resources", ras.handleCreateResource)
	api.PUT("/resources/:id", ras.handleUpdateResource)
	api.DELETE("/resources/:id", ras.handleDeleteResource)

	// Logs endpoints
	api.GET("/logs", ras.handleGetLogs)
	api.GET("/logs/stream", ras.handleLogsStream)
	api.POST("/logs", ras.handleCreateLog)

	// Events endpoints
	api.GET("/events/operations", ras.handleOperationEvents)
	api.GET("/events/resources", ras.handleResourceEvents)
	api.GET("/events/errors", ras.handleErrorEvents)

	// Metrics endpoint
	api.GET("/metrics", ras.handleGetMetrics)
	api.GET("/metrics/stream", ras.handleMetricsStream)

	// Health check
	api.GET("/health", ras.handleHealth)
}

// subscribeToReactiveState subscribes to reactive state changes
func (ras *ReactiveAPIServer) subscribeToReactiveState() {
	ras.subscription = ras.resolver.Subscribe(ras.ctx, reactive.ObserverFunc[reactive.AppState]{
		NextFunc: func(state reactive.AppState) {
			ras.logger.Debug("API server received state update",
				"operations", len(state.Operations),
				"resources", len(state.Resources),
				"logs", len(state.Logs))
		},
		ErrorFunc: func(err error) {
			ras.logger.Error("API server state error", "error", err)
		},
		CompleteFunc: func() {
			ras.logger.Info("API server state stream completed")
		},
	})
}

// Route handlers

func (ras *ReactiveAPIServer) handleGetState(c *gin.Context) {
	state := ras.resolver.GetState()

	response := ReactiveAPIResponse{
		Success:   true,
		Timestamp: time.Now().Unix(),
		State:     state,
		RequestID: c.Request.Header.Get("X-Request-ID"),
	}

	c.JSON(http.StatusOK, response)
}

func (ras *ReactiveAPIServer) handleStateStream(c *gin.Context) {
	// SSE endpoint for streaming state changes
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// Create a subscription for this client
	clientCtx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	subscription := ras.resolver.Subscribe(clientCtx, reactive.ObserverFunc[reactive.AppState]{
		NextFunc: func(state reactive.AppState) {
			data, _ := json.Marshal(state)
			fmt.Fprintf(c.Writer, "data: %s\n\n", data)
			c.Writer.Flush()
		},
		ErrorFunc: func(err error) {
			fmt.Fprintf(c.Writer, "event: error\ndata: %s\n\n", err.Error())
			c.Writer.Flush()
		},
		CompleteFunc: func() {
			fmt.Fprintf(c.Writer, "event: complete\ndata: stream ended\n\n")
			c.Writer.Flush()
		},
	})
	defer subscription.Unsubscribe()

	// Keep connection alive
	<-clientCtx.Done()
}

func (ras *ReactiveAPIServer) handleGetOperations(c *gin.Context) {
	state := ras.resolver.GetState()

	response := ReactiveAPIResponse{
		Success:   true,
		Timestamp: time.Now().Unix(),
		Data:      state.Operations,
		RequestID: c.Request.Header.Get("X-Request-ID"),
	}

	c.JSON(http.StatusOK, response)
}

func (ras *ReactiveAPIServer) handleGetOperation(c *gin.Context) {
	operationID := c.Param("id")
	state := ras.resolver.GetState()

	if operation, exists := state.Operations[operationID]; exists {
		response := ReactiveAPIResponse{
			Success:   true,
			Timestamp: time.Now().Unix(),
			Data:      operation,
			RequestID: c.Request.Header.Get("X-Request-ID"),
		}
		c.JSON(http.StatusOK, response)
	} else {
		response := ReactiveAPIResponse{
			Success:   false,
			Timestamp: time.Now().Unix(),
			Error:     fmt.Sprintf("Operation %s not found", operationID),
			RequestID: c.Request.Header.Get("X-Request-ID"),
		}
		c.JSON(http.StatusNotFound, response)
	}
}

func (ras *ReactiveAPIServer) handleCreateOperation(c *gin.Context) {
	var operation reactive.Operation
	if err := c.ShouldBindJSON(&operation); err != nil {
		response := ReactiveAPIResponse{
			Success:   false,
			Timestamp: time.Now().Unix(),
			Error:     err.Error(),
			RequestID: c.Request.Header.Get("X-Request-ID"),
		}
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Set creation time
	operation.StartTime = time.Now().Unix()
	if operation.Status == "" {
		operation.Status = "created"
	}

	// Dispatch to reactive system
	ras.resolver.EmitOperationCreated(operation, nil)

	response := ReactiveAPIResponse{
		Success:   true,
		Timestamp: time.Now().Unix(),
		Data:      operation,
		RequestID: c.Request.Header.Get("X-Request-ID"),
	}

	c.JSON(http.StatusCreated, response)
}

func (ras *ReactiveAPIServer) handleUpdateOperation(c *gin.Context) {
	operationID := c.Param("id")

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		response := ReactiveAPIResponse{
			Success:   false,
			Timestamp: time.Now().Unix(),
			Error:     err.Error(),
			RequestID: c.Request.Header.Get("X-Request-ID"),
		}
		c.JSON(http.StatusBadRequest, response)
		return
	}

	state := ras.resolver.GetState()
	operation, exists := state.Operations[operationID]
	if !exists {
		response := ReactiveAPIResponse{
			Success:   false,
			Timestamp: time.Now().Unix(),
			Error:     fmt.Sprintf("Operation %s not found", operationID),
			RequestID: c.Request.Header.Get("X-Request-ID"),
		}
		c.JSON(http.StatusNotFound, response)
		return
	}

	// Apply updates
	if status, ok := updates["status"].(string); ok {
		operation.Status = status
	}
	if progress, ok := updates["progress"].(float64); ok {
		operation.Progress = progress
	}
	if operation.Status == "completed" || operation.Status == "error" {
		operation.EndTime = time.Now().Unix()
	}

	// Dispatch update
	ras.resolver.EmitOperationUpdated(operation, updates)

	response := ReactiveAPIResponse{
		Success:   true,
		Timestamp: time.Now().Unix(),
		Data:      operation,
		RequestID: c.Request.Header.Get("X-Request-ID"),
	}

	c.JSON(http.StatusOK, response)
}

func (ras *ReactiveAPIServer) handleDeleteOperation(c *gin.Context) {
	operationID := c.Param("id")
	state := ras.resolver.GetState()

	if operation, exists := state.Operations[operationID]; exists {
		ras.resolver.EmitOperationRemoved(operation, nil)

		response := ReactiveAPIResponse{
			Success:   true,
			Timestamp: time.Now().Unix(),
			Data:      map[string]string{"deleted": operationID},
			RequestID: c.Request.Header.Get("X-Request-ID"),
		}
		c.JSON(http.StatusOK, response)
	} else {
		response := ReactiveAPIResponse{
			Success:   false,
			Timestamp: time.Now().Unix(),
			Error:     fmt.Sprintf("Operation %s not found", operationID),
			RequestID: c.Request.Header.Get("X-Request-ID"),
		}
		c.JSON(http.StatusNotFound, response)
	}
}

func (ras *ReactiveAPIServer) handleGetResources(c *gin.Context) {
	state := ras.resolver.GetState()

	response := ReactiveAPIResponse{
		Success:   true,
		Timestamp: time.Now().Unix(),
		Data:      state.Resources,
		RequestID: c.Request.Header.Get("X-Request-ID"),
	}

	c.JSON(http.StatusOK, response)
}

func (ras *ReactiveAPIServer) handleGetResource(c *gin.Context) {
	resourceID := c.Param("id")
	state := ras.resolver.GetState()

	if resource, exists := state.Resources[resourceID]; exists {
		response := ReactiveAPIResponse{
			Success:   true,
			Timestamp: time.Now().Unix(),
			Data:      resource,
			RequestID: c.Request.Header.Get("X-Request-ID"),
		}
		c.JSON(http.StatusOK, response)
	} else {
		response := ReactiveAPIResponse{
			Success:   false,
			Timestamp: time.Now().Unix(),
			Error:     fmt.Sprintf("Resource %s not found", resourceID),
			RequestID: c.Request.Header.Get("X-Request-ID"),
		}
		c.JSON(http.StatusNotFound, response)
	}
}

func (ras *ReactiveAPIServer) handleCreateResource(c *gin.Context) {
	var resource reactive.Resource
	if err := c.ShouldBindJSON(&resource); err != nil {
		response := ReactiveAPIResponse{
			Success:   false,
			Timestamp: time.Now().Unix(),
			Error:     err.Error(),
			RequestID: c.Request.Header.Get("X-Request-ID"),
		}
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Dispatch to reactive system
	ras.resolver.EmitResourceCreated(resource, nil)

	response := ReactiveAPIResponse{
		Success:   true,
		Timestamp: time.Now().Unix(),
		Data:      resource,
		RequestID: c.Request.Header.Get("X-Request-ID"),
	}

	c.JSON(http.StatusCreated, response)
}

func (ras *ReactiveAPIServer) handleUpdateResource(c *gin.Context) {
	resourceID := c.Param("id")

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		response := ReactiveAPIResponse{
			Success:   false,
			Timestamp: time.Now().Unix(),
			Error:     err.Error(),
			RequestID: c.Request.Header.Get("X-Request-ID"),
		}
		c.JSON(http.StatusBadRequest, response)
		return
	}

	state := ras.resolver.GetState()
	resource, exists := state.Resources[resourceID]
	if !exists {
		response := ReactiveAPIResponse{
			Success:   false,
			Timestamp: time.Now().Unix(),
			Error:     fmt.Sprintf("Resource %s not found", resourceID),
			RequestID: c.Request.Header.Get("X-Request-ID"),
		}
		c.JSON(http.StatusNotFound, response)
		return
	}

	// Apply updates
	if status, ok := updates["status"].(string); ok {
		resource.Status = status
	}
	if data, ok := updates["data"]; ok {
		resource.Data = data
	}

	// Dispatch update
	ras.resolver.EmitResourceUpdated(resource, updates)

	response := ReactiveAPIResponse{
		Success:   true,
		Timestamp: time.Now().Unix(),
		Data:      resource,
		RequestID: c.Request.Header.Get("X-Request-ID"),
	}

	c.JSON(http.StatusOK, response)
}

func (ras *ReactiveAPIServer) handleDeleteResource(c *gin.Context) {
	resourceID := c.Param("id")
	state := ras.resolver.GetState()

	if resource, exists := state.Resources[resourceID]; exists {
		ras.resolver.EmitResourceRemoved(resource, nil)

		response := ReactiveAPIResponse{
			Success:   true,
			Timestamp: time.Now().Unix(),
			Data:      map[string]string{"deleted": resourceID},
			RequestID: c.Request.Header.Get("X-Request-ID"),
		}
		c.JSON(http.StatusOK, response)
	} else {
		response := ReactiveAPIResponse{
			Success:   false,
			Timestamp: time.Now().Unix(),
			Error:     fmt.Sprintf("Resource %s not found", resourceID),
			RequestID: c.Request.Header.Get("X-Request-ID"),
		}
		c.JSON(http.StatusNotFound, response)
	}
}

func (ras *ReactiveAPIServer) handleGetLogs(c *gin.Context) {
	state := ras.resolver.GetState()

	// Optional query parameters for filtering
	level := c.Query("level")
	source := c.Query("source")
	limit := c.DefaultQuery("limit", "100")

	logs := state.Logs

	// Apply filters
	if level != "" || source != "" {
		var filteredLogs []reactive.LogEntry
		for _, log := range logs {
			if level != "" && log.Level != level {
				continue
			}
			if source != "" && log.Source != source {
				continue
			}
			filteredLogs = append(filteredLogs, log)
		}
		logs = filteredLogs
	}

	// Apply limit
	if limitInt, err := strconv.Atoi(limit); err == nil && limitInt > 0 && len(logs) > limitInt {
		logs = logs[len(logs)-limitInt:]
	}

	response := ReactiveAPIResponse{
		Success:   true,
		Timestamp: time.Now().Unix(),
		Data:      logs,
		RequestID: c.Request.Header.Get("X-Request-ID"),
	}

	c.JSON(http.StatusOK, response)
}

func (ras *ReactiveAPIServer) handleLogsStream(c *gin.Context) {
	// SSE endpoint for streaming logs
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	clientCtx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	subscription := ras.resolver.MonitorLogs().Subscribe(clientCtx, reactive.ObserverFunc[reactive.LogEvent]{
		NextFunc: func(logEvent reactive.LogEvent) {
			data, _ := json.Marshal(logEvent.Entry)
			fmt.Fprintf(c.Writer, "data: %s\n\n", data)
			c.Writer.Flush()
		},
		ErrorFunc: func(err error) {
			fmt.Fprintf(c.Writer, "event: error\ndata: %s\n\n", err.Error())
			c.Writer.Flush()
		},
	})
	defer subscription.Unsubscribe()

	<-clientCtx.Done()
}

func (ras *ReactiveAPIServer) handleCreateLog(c *gin.Context) {
	var logEntry reactive.LogEntry
	if err := c.ShouldBindJSON(&logEntry); err != nil {
		response := ReactiveAPIResponse{
			Success:   false,
			Timestamp: time.Now().Unix(),
			Error:     err.Error(),
			RequestID: c.Request.Header.Get("X-Request-ID"),
		}
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Set timestamp if not provided
	if logEntry.Timestamp == 0 {
		logEntry.Timestamp = time.Now().Unix()
	}

	// Emit log entry
	ras.resolver.EmitLog(logEntry.Level, logEntry.Message, logEntry.Source, logEntry.Data)

	response := ReactiveAPIResponse{
		Success:   true,
		Timestamp: time.Now().Unix(),
		Data:      logEntry,
		RequestID: c.Request.Header.Get("X-Request-ID"),
	}

	c.JSON(http.StatusCreated, response)
}

func (ras *ReactiveAPIServer) handleOperationEvents(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	clientCtx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	subscription := ras.resolver.MonitorOperationChanges().Subscribe(clientCtx, reactive.ObserverFunc[reactive.OperationEvent]{
		NextFunc: func(event reactive.OperationEvent) {
			data, _ := json.Marshal(event)
			fmt.Fprintf(c.Writer, "data: %s\n\n", data)
			c.Writer.Flush()
		},
		ErrorFunc: func(err error) {
			fmt.Fprintf(c.Writer, "event: error\ndata: %s\n\n", err.Error())
			c.Writer.Flush()
		},
	})
	defer subscription.Unsubscribe()

	<-clientCtx.Done()
}

func (ras *ReactiveAPIServer) handleResourceEvents(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	clientCtx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	subscription := ras.resolver.MonitorResourceChanges().Subscribe(clientCtx, reactive.ObserverFunc[reactive.ResourceEvent]{
		NextFunc: func(event reactive.ResourceEvent) {
			data, _ := json.Marshal(event)
			fmt.Fprintf(c.Writer, "data: %s\n\n", data)
			c.Writer.Flush()
		},
		ErrorFunc: func(err error) {
			fmt.Fprintf(c.Writer, "event: error\ndata: %s\n\n", err.Error())
			c.Writer.Flush()
		},
	})
	defer subscription.Unsubscribe()

	<-clientCtx.Done()
}

func (ras *ReactiveAPIServer) handleErrorEvents(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	clientCtx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	subscription := ras.resolver.MonitorErrors().Subscribe(clientCtx, reactive.ObserverFunc[reactive.ErrorEvent]{
		NextFunc: func(event reactive.ErrorEvent) {
			data, _ := json.Marshal(event)
			fmt.Fprintf(c.Writer, "data: %s\n\n", data)
			c.Writer.Flush()
		},
		ErrorFunc: func(err error) {
			fmt.Fprintf(c.Writer, "event: error\ndata: %s\n\n", err.Error())
			c.Writer.Flush()
		},
	})
	defer subscription.Unsubscribe()

	<-clientCtx.Done()
}

func (ras *ReactiveAPIServer) handleGetMetrics(c *gin.Context) {
	// Get current metrics
	metricsObs := ras.resolver.CollectMetrics()

	// Get the latest metrics (this would be better implemented with caching)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 1*time.Second)
	defer cancel()

	metricsChan := make(chan map[string]interface{}, 1)
	subscription := metricsObs.Subscribe(ctx, reactive.ObserverFunc[map[string]interface{}]{
		NextFunc: func(metrics map[string]interface{}) {
			select {
			case metricsChan <- metrics:
			default:
			}
		},
	})
	defer subscription.Unsubscribe()

	select {
	case metrics := <-metricsChan:
		response := ReactiveAPIResponse{
			Success:   true,
			Timestamp: time.Now().Unix(),
			Data:      metrics,
			RequestID: c.Request.Header.Get("X-Request-ID"),
		}
		c.JSON(http.StatusOK, response)
	case <-ctx.Done():
		response := ReactiveAPIResponse{
			Success:   false,
			Timestamp: time.Now().Unix(),
			Error:     "Metrics collection timeout",
			RequestID: c.Request.Header.Get("X-Request-ID"),
		}
		c.JSON(http.StatusRequestTimeout, response)
	}
}

func (ras *ReactiveAPIServer) handleMetricsStream(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	clientCtx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	subscription := ras.resolver.CollectMetrics().Subscribe(clientCtx, reactive.ObserverFunc[map[string]interface{}]{
		NextFunc: func(metrics map[string]interface{}) {
			data, _ := json.Marshal(metrics)
			fmt.Fprintf(c.Writer, "data: %s\n\n", data)
			c.Writer.Flush()
		},
		ErrorFunc: func(err error) {
			fmt.Fprintf(c.Writer, "event: error\ndata: %s\n\n", err.Error())
			c.Writer.Flush()
		},
	})
	defer subscription.Unsubscribe()

	<-clientCtx.Done()
}

func (ras *ReactiveAPIServer) handleHealth(c *gin.Context) {
	state := ras.resolver.GetState()

	health := map[string]interface{}{
		"status":     "healthy",
		"timestamp":  time.Now().Unix(),
		"operations": len(state.Operations),
		"resources":  len(state.Resources),
		"logs":       len(state.Logs),
		"error":      state.Error,
		"loading":    state.Loading,
	}

	status := http.StatusOK
	if state.Error != "" {
		health["status"] = "degraded"
		status = http.StatusServiceUnavailable
	}

	response := ReactiveAPIResponse{
		Success:   status == http.StatusOK,
		Timestamp: time.Now().Unix(),
		Data:      health,
		RequestID: c.Request.Header.Get("X-Request-ID"),
	}

	c.JSON(status, response)
}

// Start starts the reactive API server
func (ras *ReactiveAPIServer) Start(port string) error {
	ras.logger.Info("Starting reactive API server", "port", port)
	return ras.engine.Run(":" + port)
}

// Stop stops the reactive API server
func (ras *ReactiveAPIServer) Stop() {
	if ras.subscription != nil {
		ras.subscription.Unsubscribe()
	}
	ras.cancel()
	ras.logger.Info("Reactive API server stopped")
}
