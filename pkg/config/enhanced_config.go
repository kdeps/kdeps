package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kdeps/kdeps/pkg/cache"
)

// EnhancedConfig contains all configuration for the enhanced kdeps system
type EnhancedConfig struct {
	// Core system configuration
	System SystemConfig `json:"system" yaml:"system"`
	
	// Resource processing configuration
	Processing ProcessingConfig `json:"processing" yaml:"processing"`
	
	// Caching configuration
	Cache cache.CacheConfig `json:"cache" yaml:"cache"`
	
	// Metrics and monitoring configuration
	Metrics MetricsConfig `json:"metrics" yaml:"metrics"`
	
	// Logging configuration
	Logging LoggingConfig `json:"logging" yaml:"logging"`
	
	// Performance tuning configuration
	Performance PerformanceConfig `json:"performance" yaml:"performance"`
}

// SystemConfig contains core system settings
type SystemConfig struct {
	Environment     string        `json:"environment" yaml:"environment"`         // dev, staging, prod
	DebugMode       bool          `json:"debug_mode" yaml:"debug_mode"`
	RequestTimeout  time.Duration `json:"request_timeout" yaml:"request_timeout"`
	MaxMemoryUsage  int64         `json:"max_memory_usage" yaml:"max_memory_usage"` // bytes
	TempDirectory   string        `json:"temp_directory" yaml:"temp_directory"`
	ConfigDirectory string        `json:"config_directory" yaml:"config_directory"`
}

// ProcessingConfig contains resource processing settings
type ProcessingConfig struct {
	EnableConcurrency    bool          `json:"enable_concurrency" yaml:"enable_concurrency"`
	MaxConcurrentWorkers int           `json:"max_concurrent_workers" yaml:"max_concurrent_workers"`
	ProcessingTimeout    time.Duration `json:"processing_timeout" yaml:"processing_timeout"`
	RetryAttempts        int           `json:"retry_attempts" yaml:"retry_attempts"`
	RetryDelay           time.Duration `json:"retry_delay" yaml:"retry_delay"`
	
	// Resource-specific timeouts
	HTTPTimeout   time.Duration `json:"http_timeout" yaml:"http_timeout"`
	LLMTimeout    time.Duration `json:"llm_timeout" yaml:"llm_timeout"`
	PythonTimeout time.Duration `json:"python_timeout" yaml:"python_timeout"`
	ExecTimeout   time.Duration `json:"exec_timeout" yaml:"exec_timeout"`
}

// MetricsConfig contains metrics collection settings
type MetricsConfig struct {
	Enabled           bool          `json:"enabled" yaml:"enabled"`
	CollectionInterval time.Duration `json:"collection_interval" yaml:"collection_interval"`
	RetentionPeriod    time.Duration `json:"retention_period" yaml:"retention_period"`
	MaxDataPoints     int           `json:"max_data_points" yaml:"max_data_points"`
	
	// Export settings
	EnableExport    bool   `json:"enable_export" yaml:"enable_export"`
	ExportPath      string `json:"export_path" yaml:"export_path"`
	ExportFormat    string `json:"export_format" yaml:"export_format"` // json, csv, prometheus
	ExportInterval  time.Duration `json:"export_interval" yaml:"export_interval"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level          string `json:"level" yaml:"level"`                     // trace, debug, info, warn, error
	Format         string `json:"format" yaml:"format"`                   // json, text
	Output         string `json:"output" yaml:"output"`                   // stdout, stderr, file
	FilePath       string `json:"file_path" yaml:"file_path"`
	MaxFileSize    int64  `json:"max_file_size" yaml:"max_file_size"`     // bytes
	MaxFiles       int    `json:"max_files" yaml:"max_files"`
	EnableRotation bool   `json:"enable_rotation" yaml:"enable_rotation"`
	
	// Structured logging settings
	EnableStructured bool `json:"enable_structured" yaml:"enable_structured"`
	EnableContext    bool `json:"enable_context" yaml:"enable_context"`
	EnableTracing    bool `json:"enable_tracing" yaml:"enable_tracing"`
}

// PerformanceConfig contains performance tuning settings
type PerformanceConfig struct {
	// Memory management
	GCTargetPercent     int           `json:"gc_target_percent" yaml:"gc_target_percent"`
	MaxGoroutines       int           `json:"max_goroutines" yaml:"max_goroutines"`
	MemoryPressureLevel float64       `json:"memory_pressure_level" yaml:"memory_pressure_level"`
	
	// Resource pooling
	EnableResourcePooling bool `json:"enable_resource_pooling" yaml:"enable_resource_pooling"`
	HTTPClientPoolSize    int  `json:"http_client_pool_size" yaml:"http_client_pool_size"`
	LLMClientPoolSize     int  `json:"llm_client_pool_size" yaml:"llm_client_pool_size"`
	
	// Connection management
	MaxIdleConnections        int           `json:"max_idle_connections" yaml:"max_idle_connections"`
	IdleConnectionTimeout     time.Duration `json:"idle_connection_timeout" yaml:"idle_connection_timeout"`
	MaxConnectionsPerHost     int           `json:"max_connections_per_host" yaml:"max_connections_per_host"`
	
	// Request optimization
	EnableRequestBatching     bool          `json:"enable_request_batching" yaml:"enable_request_batching"`
	BatchSize                 int           `json:"batch_size" yaml:"batch_size"`
	BatchTimeout              time.Duration `json:"batch_timeout" yaml:"batch_timeout"`
	EnableRequestCompression  bool          `json:"enable_request_compression" yaml:"enable_request_compression"`
}

// LoadConfig loads configuration from a file
func LoadConfig(configPath string) (*EnhancedConfig, error) {
	// Start with default configuration
	config := DefaultConfig()
	
	// If no config file specified, return defaults
	if configPath == "" {
		return config, nil
	}
	
	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return config, fmt.Errorf("config file not found: %s", configPath)
	}
	
	// Read the file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	
	// Parse based on file extension
	ext := filepath.Ext(configPath)
	switch ext {
	case ".json":
		if err := json.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse JSON config: %w", err)
		}
	case ".yaml", ".yml":
		// For this example, we'll use JSON parsing since yaml isn't imported
		// In a real implementation, you'd use yaml.Unmarshal
		if err := json.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse YAML config: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported config file format: %s", ext)
	}
	
	// Validate the configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}
	
	return config, nil
}

// SaveConfig saves configuration to a file
func (c *EnhancedConfig) SaveConfig(configPath string) error {
	// Validate before saving
	if err := c.Validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}
	
	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	// Marshal to JSON
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	// Write to file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	return nil
}

// Validate validates the configuration
func (c *EnhancedConfig) Validate() error {
	// Validate system config
	if c.System.RequestTimeout <= 0 {
		return fmt.Errorf("system.request_timeout must be positive")
	}
	if c.System.MaxMemoryUsage <= 0 {
		return fmt.Errorf("system.max_memory_usage must be positive")
	}
	
	// Validate processing config
	if c.Processing.MaxConcurrentWorkers <= 0 {
		return fmt.Errorf("processing.max_concurrent_workers must be positive")
	}
	if c.Processing.ProcessingTimeout <= 0 {
		return fmt.Errorf("processing.processing_timeout must be positive")
	}
	
	// Validate cache config
	if c.Cache.MaxSize <= 0 {
		return fmt.Errorf("cache.max_size must be positive")
	}
	if c.Cache.DefaultTTL <= 0 {
		return fmt.Errorf("cache.default_ttl must be positive")
	}
	
	// Validate metrics config
	if c.Metrics.Enabled {
		if c.Metrics.CollectionInterval <= 0 {
			return fmt.Errorf("metrics.collection_interval must be positive when metrics are enabled")
		}
		if c.Metrics.MaxDataPoints <= 0 {
			return fmt.Errorf("metrics.max_data_points must be positive when metrics are enabled")
		}
	}
	
	// Validate logging config
	validLevels := map[string]bool{
		"trace": true, "debug": true, "info": true, "warn": true, "error": true,
	}
	if !validLevels[c.Logging.Level] {
		return fmt.Errorf("logging.level must be one of: trace, debug, info, warn, error")
	}
	
	validFormats := map[string]bool{"json": true, "text": true}
	if !validFormats[c.Logging.Format] {
		return fmt.Errorf("logging.format must be one of: json, text")
	}
	
	// Validate performance config
	if c.Performance.GCTargetPercent < 50 || c.Performance.GCTargetPercent > 500 {
		return fmt.Errorf("performance.gc_target_percent must be between 50 and 500")
	}
	if c.Performance.MaxGoroutines <= 0 {
		return fmt.Errorf("performance.max_goroutines must be positive")
	}
	
	return nil
}

// DefaultConfig returns a sensible default configuration
func DefaultConfig() *EnhancedConfig {
	return &EnhancedConfig{
		System: SystemConfig{
			Environment:     "dev",
			DebugMode:       false,
			RequestTimeout:  30 * time.Second,
			MaxMemoryUsage:  1024 * 1024 * 1024, // 1GB
			TempDirectory:   "/tmp/kdeps",
			ConfigDirectory: "./config",
		},
		Processing: ProcessingConfig{
			EnableConcurrency:    true,
			MaxConcurrentWorkers: 4,
			ProcessingTimeout:    5 * time.Minute,
			RetryAttempts:        3,
			RetryDelay:           1 * time.Second,
			HTTPTimeout:          30 * time.Second,
			LLMTimeout:           2 * time.Minute,
			PythonTimeout:        1 * time.Minute,
			ExecTimeout:          1 * time.Minute,
		},
		Cache: cache.DefaultCacheConfig(),
		Metrics: MetricsConfig{
			Enabled:            true,
			CollectionInterval: 10 * time.Second,
			RetentionPeriod:    24 * time.Hour,
			MaxDataPoints:      10000,
			EnableExport:       false,
			ExportPath:         "./metrics",
			ExportFormat:       "json",
			ExportInterval:     1 * time.Hour,
		},
		Logging: LoggingConfig{
			Level:            "info",
			Format:           "json",
			Output:           "stdout",
			FilePath:         "./logs/kdeps.log",
			MaxFileSize:      100 * 1024 * 1024, // 100MB
			MaxFiles:         10,
			EnableRotation:   true,
			EnableStructured: true,
			EnableContext:    true,
			EnableTracing:    false,
		},
		Performance: PerformanceConfig{
			GCTargetPercent:           100,
			MaxGoroutines:             1000,
			MemoryPressureLevel:       0.8,
			EnableResourcePooling:     true,
			HTTPClientPoolSize:        10,
			LLMClientPoolSize:         5,
			MaxIdleConnections:        100,
			IdleConnectionTimeout:     90 * time.Second,
			MaxConnectionsPerHost:     10,
			EnableRequestBatching:     false,
			BatchSize:                 10,
			BatchTimeout:              100 * time.Millisecond,
			EnableRequestCompression:  true,
		},
	}
}

// ApplyEnvironmentOverrides applies environment-specific configuration overrides
func (c *EnhancedConfig) ApplyEnvironmentOverrides() {
	switch c.System.Environment {
	case "dev":
		c.System.DebugMode = true
		c.Logging.Level = "debug"
		c.Cache.DefaultTTL = 5 * time.Minute // Shorter cache for development
		c.Metrics.CollectionInterval = 5 * time.Second
		
	case "staging":
		c.System.DebugMode = false
		c.Logging.Level = "info"
		c.Cache.DefaultTTL = 30 * time.Minute
		c.Metrics.CollectionInterval = 10 * time.Second
		
	case "prod":
		c.System.DebugMode = false
		c.Logging.Level = "warn"
		c.Cache.DefaultTTL = 1 * time.Hour
		c.Metrics.CollectionInterval = 30 * time.Second
		c.Performance.EnableRequestBatching = true
		c.Processing.MaxConcurrentWorkers = 8 // More workers for production
	}
}

// GetEnvironmentString returns environment-specific configuration as a string
func (c *EnhancedConfig) GetEnvironmentString() string {
	data, _ := json.MarshalIndent(c, "", "  ")
	return string(data)
}