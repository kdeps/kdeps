package config

import (
	"context"
	"fmt"

	"github.com/kdeps/kdeps/pkg"
	"github.com/kdeps/kdeps/pkg/logging"
	apiserver "github.com/kdeps/schema/gen/api_server"
	webserver "github.com/kdeps/schema/gen/web_server"
	pklWf "github.com/kdeps/schema/gen/workflow"
)

// ProcessedConfiguration contains all configuration values with their sources
type ProcessedConfiguration struct {
	// API Server Configuration
	APIServerMode   pkg.ConfigurationValue[bool]
	APIServerHostIP pkg.ConfigurationValue[string]
	APIServerPort   pkg.ConfigurationValue[uint16]
	APIServerCORS   *apiserver.CORS
	APIServerRoutes *[]*apiserver.APIServerRoutes

	// Web Server Configuration
	WebServerMode   pkg.ConfigurationValue[bool]
	WebServerHostIP pkg.ConfigurationValue[string]
	WebServerPort   pkg.ConfigurationValue[uint16]
	WebServerRoutes *[]*webserver.WebServerRoutes

	// General Settings
	RateLimitMax pkg.ConfigurationValue[int]
	Environment  pkg.ConfigurationValue[string]

	// Agent Settings
	InstallAnaconda  pkg.ConfigurationValue[bool]
	Timezone         pkg.ConfigurationValue[string]
	OllamaTagVersion pkg.ConfigurationValue[string]
	Packages         *[]string
	Repositories     *[]string
	PythonPackages   *[]string
	CondaPackages    *map[string]map[string]string
	Args             *map[string]string
	Env              *map[string]string
	ExposedPorts     *[]string
}

// ConfigurationProcessor handles the processing of workflow configuration with PKL-first priority
type ConfigurationProcessor struct {
	configManager *pkg.ConfigurationManager
	logger        *logging.Logger
}

// NewConfigurationProcessor creates a new configuration processor
func NewConfigurationProcessor(logger *logging.Logger) *ConfigurationProcessor {
	return &ConfigurationProcessor{
		configManager: pkg.NewConfigurationManager(logger),
		logger:        logger,
	}
}

// ProcessWorkflowConfiguration processes workflow configuration with PKL-first priority
func (cp *ConfigurationProcessor) ProcessWorkflowConfiguration(_ context.Context, workflow pklWf.Workflow) (*ProcessedConfiguration, error) {
	cp.logger.Info("processing workflow configuration with PKL-first priority")

	settings := workflow.GetSettings()
	if settings == nil {
		cp.logger.Warn("no settings found in workflow, using all defaults")
		return cp.CreateDefaultConfiguration(), nil
	}

	config := &ProcessedConfiguration{}

	// Process API Server Configuration
	config.APIServerMode = cp.configManager.GetBoolWithPKLPriority(
		settings.APIServerMode,
		pkg.DefaultAPIServerMode,
		"APIServerMode",
	)

	if settings.APIServer != nil {
		config.APIServerHostIP = cp.configManager.GetStringWithPKLPriority(
			settings.APIServer.HostIP,
			pkg.DefaultHostIP,
			"APIServer.HostIP",
		)
		config.APIServerPort = cp.configManager.GetUint16WithPKLPriority(
			settings.APIServer.PortNum,
			pkg.DefaultPortNum,
			"APIServer.PortNum",
		)
		config.APIServerCORS = settings.APIServer.CORS
		config.APIServerRoutes = settings.APIServer.Routes
	} else {
		// Use defaults for API server settings
		config.APIServerHostIP = cp.configManager.GetStringWithPKLPriority(nil, pkg.DefaultHostIP, "APIServer.HostIP")
		config.APIServerPort = cp.configManager.GetUint16WithPKLPriority(nil, pkg.DefaultPortNum, "APIServer.PortNum")
		cp.logger.Debug("no APIServer configuration found, using defaults")
	}

	// Process Web Server Configuration
	config.WebServerMode = cp.configManager.GetBoolWithPKLPriority(
		settings.WebServerMode,
		pkg.DefaultWebServerMode,
		"WebServerMode",
	)

	if settings.WebServer != nil {
		config.WebServerHostIP = cp.configManager.GetStringWithPKLPriority(
			settings.WebServer.HostIP,
			pkg.DefaultHostIP,
			"WebServer.HostIP",
		)
		config.WebServerPort = cp.configManager.GetUint16WithPKLPriority(
			settings.WebServer.PortNum,
			pkg.DefaultAPIPortNum,
			"WebServer.PortNum",
		)
		config.WebServerRoutes = settings.WebServer.Routes
	} else {
		// Use defaults for web server settings
		config.WebServerHostIP = cp.configManager.GetStringWithPKLPriority(nil, pkg.DefaultHostIP, "WebServer.HostIP")
		config.WebServerPort = cp.configManager.GetUint16WithPKLPriority(nil, pkg.DefaultAPIPortNum, "WebServer.PortNum")
		cp.logger.Debug("no WebServer configuration found, using defaults")
	}

	// Process General Settings
	config.RateLimitMax = cp.configManager.GetIntWithPKLPriority(
		settings.RateLimitMax,
		pkg.DefaultRateLimitMax,
		"RateLimitMax",
	)

	// Handle Environment field which is of type *buildenv.BuildEnv
	var environmentStr *string
	if settings.Environment != nil {
		// Convert BuildEnv to string - assuming it has a String() method or similar
		envStr := fmt.Sprintf("%v", *settings.Environment)
		environmentStr = &envStr
	}

	config.Environment = cp.configManager.GetStringWithPKLPriority(
		environmentStr,
		pkg.DefaultEnvironment,
		"Environment",
	)

	// Process Agent Settings
	if settings.AgentSettings != nil {
		config.InstallAnaconda = cp.configManager.GetBoolWithPKLPriority(
			settings.AgentSettings.InstallAnaconda,
			pkg.DefaultInstallAnaconda,
			"AgentSettings.InstallAnaconda",
		)

		config.Timezone = cp.configManager.GetStringWithPKLPriority(
			settings.AgentSettings.Timezone,
			pkg.DefaultTimezone,
			"AgentSettings.Timezone",
		)

		config.OllamaTagVersion = cp.configManager.GetStringWithPKLPriority(
			settings.AgentSettings.OllamaTagVersion,
			pkg.DefaultOllamaTagVersion,
			"AgentSettings.OllamaTagVersion",
		)

		// Direct assignments for complex types (these come from PKL if present)
		config.Packages = settings.AgentSettings.Packages
		config.Repositories = settings.AgentSettings.Repositories
		config.PythonPackages = settings.AgentSettings.PythonPackages
		config.CondaPackages = settings.AgentSettings.CondaPackages
		config.Args = settings.AgentSettings.Args
		config.Env = settings.AgentSettings.Env
		config.ExposedPorts = settings.AgentSettings.ExposedPorts

		if config.Packages != nil {
			cp.logger.Debug("using PKL packages configuration", "packages", *config.Packages)
		}
		if config.Repositories != nil {
			cp.logger.Debug("using PKL repositories configuration", "repositories", *config.Repositories)
		}
		if config.PythonPackages != nil {
			cp.logger.Debug("using PKL Python packages configuration", "python_packages", *config.PythonPackages)
		}
		if config.ExposedPorts != nil {
			cp.logger.Debug("using PKL exposed ports configuration", "exposed_ports", *config.ExposedPorts)
		}
	} else {
		// Use defaults for agent settings
		config.InstallAnaconda = cp.configManager.GetBoolWithPKLPriority(nil, pkg.DefaultInstallAnaconda, "AgentSettings.InstallAnaconda")
		config.Timezone = cp.configManager.GetStringWithPKLPriority(nil, pkg.DefaultTimezone, "AgentSettings.Timezone")
		config.OllamaTagVersion = cp.configManager.GetStringWithPKLPriority(nil, pkg.DefaultOllamaTagVersion, "AgentSettings.OllamaTagVersion")
		cp.logger.Debug("no AgentSettings configuration found, using defaults")
	}

	// Log configuration summary
	cp.logConfigurationSummary(config)

	return config, nil
}

// CreateDefaultConfiguration creates a configuration with all default values
func (cp *ConfigurationProcessor) CreateDefaultConfiguration() *ProcessedConfiguration {
	cp.logger.Info("creating configuration with all default values")

	return &ProcessedConfiguration{
		// API Server Configuration
		APIServerMode:   cp.configManager.GetBoolWithPKLPriority(nil, pkg.DefaultAPIServerMode, "APIServerMode"),
		APIServerHostIP: cp.configManager.GetStringWithPKLPriority(nil, pkg.DefaultHostIP, "APIServer.HostIP"),
		APIServerPort:   cp.configManager.GetUint16WithPKLPriority(nil, pkg.DefaultPortNum, "APIServer.PortNum"),

		// Web Server Configuration
		WebServerMode:   cp.configManager.GetBoolWithPKLPriority(nil, pkg.DefaultWebServerMode, "WebServerMode"),
		WebServerHostIP: cp.configManager.GetStringWithPKLPriority(nil, pkg.DefaultHostIP, "WebServer.HostIP"),
		WebServerPort:   cp.configManager.GetUint16WithPKLPriority(nil, pkg.DefaultAPIPortNum, "WebServer.PortNum"),

		// General Settings
		RateLimitMax: cp.configManager.GetIntWithPKLPriority(nil, pkg.DefaultRateLimitMax, "RateLimitMax"),
		Environment:  cp.configManager.GetStringWithPKLPriority(nil, pkg.DefaultEnvironment, "Environment"),

		// Agent Settings
		InstallAnaconda:  cp.configManager.GetBoolWithPKLPriority(nil, pkg.DefaultInstallAnaconda, "AgentSettings.InstallAnaconda"),
		Timezone:         cp.configManager.GetStringWithPKLPriority(nil, pkg.DefaultTimezone, "AgentSettings.Timezone"),
		OllamaTagVersion: cp.configManager.GetStringWithPKLPriority(nil, pkg.DefaultOllamaTagVersion, "AgentSettings.OllamaTagVersion"),
	}
}

// logConfigurationSummary logs a summary of the configuration sources
func (cp *ConfigurationProcessor) logConfigurationSummary(config *ProcessedConfiguration) {
	pklCount := 0
	defaultCount := 0

	// Count PKL vs Default configurations
	configs := []pkg.ConfigurationValue[any]{
		{Value: config.APIServerMode.Value, Source: config.APIServerMode.Source},
		{Value: config.APIServerHostIP.Value, Source: config.APIServerHostIP.Source},
		{Value: config.APIServerPort.Value, Source: config.APIServerPort.Source},
		{Value: config.WebServerMode.Value, Source: config.WebServerMode.Source},
		{Value: config.WebServerHostIP.Value, Source: config.WebServerHostIP.Source},
		{Value: config.WebServerPort.Value, Source: config.WebServerPort.Source},
		{Value: config.RateLimitMax.Value, Source: config.RateLimitMax.Source},
		{Value: config.Environment.Value, Source: config.Environment.Source},
		{Value: config.InstallAnaconda.Value, Source: config.InstallAnaconda.Source},
		{Value: config.Timezone.Value, Source: config.Timezone.Source},
		{Value: config.OllamaTagVersion.Value, Source: config.OllamaTagVersion.Source},
	}

	for _, cfg := range configs {
		if cfg.Source == pkg.SourcePKL {
			pklCount++
		} else {
			defaultCount++
		}
	}

	// Count complex configurations
	if config.APIServerCORS != nil {
		pklCount++
	}
	if config.APIServerRoutes != nil {
		pklCount++
	}
	if config.WebServerRoutes != nil {
		pklCount++
	}
	if config.Packages != nil {
		pklCount++
	}
	if config.Repositories != nil {
		pklCount++
	}
	if config.PythonPackages != nil {
		pklCount++
	}
	if config.CondaPackages != nil {
		pklCount++
	}
	if config.Args != nil {
		pklCount++
	}
	if config.Env != nil {
		pklCount++
	}

	cp.logger.Info("configuration processing complete",
		"pkl_configs", pklCount,
		"default_configs", defaultCount,
		"total_configs", pklCount+defaultCount,
		"api_server_mode", config.APIServerMode.Value,
		"web_server_mode", config.WebServerMode.Value,
		"environment", config.Environment.Value,
	)
}

// ValidateConfiguration validates the processed configuration
func (cp *ConfigurationProcessor) ValidateConfiguration(config *ProcessedConfiguration) error {
	cp.logger.Debug("validating configuration")

	// Validate port numbers - allow port 0 (let OS choose free port)
	if config.APIServerPort.Value == 0 {
		cp.logger.Debug("API server port is 0, will let OS choose free port")
	}
	if config.WebServerPort.Value == 0 {
		cp.logger.Debug("Web server port is 0, will let OS choose free port")
	}

	// Validate environment
	validEnvironments := map[string]bool{"dev": true, "development": true, "prod": true, "production": true}
	if !validEnvironments[config.Environment.Value] {
		return fmt.Errorf("invalid environment: %s (must be dev, development, prod, or production)", config.Environment.Value)
	}

	// Validate rate limit
	if config.RateLimitMax.Value < 0 {
		return fmt.Errorf("invalid rate limit: %d (must be non-negative)", config.RateLimitMax.Value)
	}

	cp.logger.Debug("configuration validation passed")
	return nil
}
