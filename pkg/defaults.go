package pkg

import (
	"time"

	"github.com/kdeps/schema/gen/kdeps/path"
)

// Package pkg provides default values and utilities for the kdeps system.
// This package serves as the single point of truth for all sensible defaults
// used throughout the kdeps codebase, ensuring consistency and maintainability.
//
// The defaults system provides:
// 1. Centralized default constants for all configurable values
// 2. Helper functions to create pointers to default values
// 3. Fallback functions that apply defaults when configuration values are nil
// 4. Type-safe generic fallback functions for common patterns
//
// Usage Examples:
//
//	// Using default constants directly
//	port := pkg.DefaultPortNum // uint16(3000)
//
//	// Using pointer helper functions
//	portPtr := pkg.GetDefaultPortNum() // *uint16 pointing to 3000
//
//	// Using fallback functions with nil-safe access
//	actualPort := pkg.GetDefaultUint16OrFallback(config.Port, pkg.DefaultPortNum)
//
// This approach ensures that:
// - All defaults are defined in one place
// - Changes to defaults propagate throughout the system
// - Code is consistent in how it handles missing configuration
// - Type safety is maintained with generic fallback functions

// API Server Defaults
const (
	DefaultAllowCredentials = true
	DefaultAPIServerMode    = false
	DefaultAppPort          = uint16(8052)
	DefaultEnableCORS       = false
	DefaultHostIP           = "127.0.0.1"
	DefaultPortNum          = uint16(3000)
	DefaultAPIPortNum       = uint16(8080)
	DefaultPublicPath       = "/web"
	DefaultRateLimitMax     = 100
	DefaultRetry            = false
	DefaultRetryTimes       = 3
	DefaultServerType       = "static"
	DefaultWebServerMode    = false
	DefaultEnvironment      = "dev"
	DefaultMaxAge           = 12 * time.Hour
	DefaultTimeoutDuration  = 60 * time.Second
)

// Docker Defaults
const (
	DefaultExitCode         = 0
	DefaultInstallAnaconda  = false
	DefaultJSONResponse     = false
	DefaultOllamaTagVersion = "latest"
	DefaultRequired         = true
	DefaultTimezone         = "Etc/UTC"
	DefaultDockerGPU        = "cpu"
)

// API Response Defaults
const (
	DefaultSuccess = true
)

// Directory Defaults
const (
	DefaultKdepsDir  = ".kdeps"
	DefaultKdepsPath = "user"
)

// RunMode Defaults
const (
	DefaultMode = "docker"
)

// Helper functions to create pointers
func StringPtr(s string) *string {
	return &s
}

func BoolPtr(b bool) *bool {
	return &b
}

func IntPtr(i int) *int {
	return &i
}

func Uint16Ptr(u uint16) *uint16 {
	return &u
}

func PathPtr(p path.Path) *path.Path {
	return &p
}

func DurationPtr(d time.Duration) *time.Duration {
	return &d
}

// Default value getters
func GetDefaultAllowCredentials() *bool {
	return BoolPtr(DefaultAllowCredentials)
}

func GetDefaultAPIServerMode() *bool {
	return BoolPtr(DefaultAPIServerMode)
}

func GetDefaultAppPort() *uint16 {
	return Uint16Ptr(DefaultAppPort)
}

func GetDefaultEnableCORS() *bool {
	return BoolPtr(DefaultEnableCORS)
}

func GetDefaultHostIP() *string {
	return StringPtr(DefaultHostIP)
}

func GetDefaultPortNum() *uint16 {
	return Uint16Ptr(DefaultPortNum)
}

func GetDefaultAPIPortNum() *uint16 {
	return Uint16Ptr(DefaultAPIPortNum)
}

func GetDefaultPublicPath() *string {
	return StringPtr(DefaultPublicPath)
}

func GetDefaultRateLimitMax() *int {
	return IntPtr(DefaultRateLimitMax)
}

func GetDefaultRetry() *bool {
	return BoolPtr(DefaultRetry)
}

func GetDefaultRetryTimes() *int {
	return IntPtr(DefaultRetryTimes)
}

func GetDefaultServerType() *string {
	return StringPtr(DefaultServerType)
}

func GetDefaultWebServerMode() *bool {
	return BoolPtr(DefaultWebServerMode)
}

func GetDefaultEnvironment() *string {
	return StringPtr(DefaultEnvironment)
}

func GetDefaultMaxAge() *time.Duration {
	return DurationPtr(DefaultMaxAge)
}

func GetDefaultTimeoutDuration() *time.Duration {
	return DurationPtr(DefaultTimeoutDuration)
}

func GetDefaultExitCode() *int {
	return IntPtr(DefaultExitCode)
}

func GetDefaultInstallAnaconda() *bool {
	return BoolPtr(DefaultInstallAnaconda)
}

func GetDefaultJSONResponse() *bool {
	return BoolPtr(DefaultJSONResponse)
}

func GetDefaultOllamaTagVersion() *string {
	return StringPtr(DefaultOllamaTagVersion)
}

func GetDefaultRequired() *bool {
	return BoolPtr(DefaultRequired)
}

func GetDefaultTimezone() *string {
	return StringPtr(DefaultTimezone)
}

func GetDefaultDockerGPU() *string {
	return StringPtr(DefaultDockerGPU)
}

func GetDefaultSuccess() *bool {
	return BoolPtr(DefaultSuccess)
}

func GetDefaultKdepsDir() *string {
	return StringPtr(DefaultKdepsDir)
}

func GetDefaultKdepsPath() *string {
	return StringPtr(DefaultKdepsPath)
}

func GetDefaultMode() *string {
	return StringPtr(DefaultMode)
}

// ApplyDefaultsToWorkflowSettings applies default values to workflow settings when they are nil or missing
func ApplyDefaultsToWorkflowSettings(settings interface{}) interface{} {
	// This function would need to be implemented based on the actual PKL schema types
	// For now, we provide individual helper functions for specific settings
	return settings
}

// GetDefaultValueOrFallback returns the value if not nil, otherwise returns the default
func GetDefaultValueOrFallback[T any](value *T, defaultValue T) T {
	if value != nil {
		return *value
	}
	return defaultValue
}

// GetDefaultStringOrFallback returns the string value if not nil, otherwise returns the default
func GetDefaultStringOrFallback(value *string, defaultValue string) string {
	if value != nil {
		return *value
	}
	return defaultValue
}

// GetDefaultBoolOrFallback returns the bool value if not nil, otherwise returns the default
func GetDefaultBoolOrFallback(value *bool, defaultValue bool) bool {
	if value != nil {
		return *value
	}
	return defaultValue
}

// GetDefaultUint16OrFallback returns the uint16 value if not nil, otherwise returns the default
func GetDefaultUint16OrFallback(value *uint16, defaultValue uint16) uint16 {
	if value != nil {
		return *value
	}
	return defaultValue
}

// GetDefaultIntOrFallback returns the int value if not nil, otherwise returns the default
func GetDefaultIntOrFallback(value *int, defaultValue int) int {
	if value != nil {
		return *value
	}
	return defaultValue
}

// GetDefaultDurationOrFallback returns the duration value if not nil, otherwise returns the default
func GetDefaultDurationOrFallback(value *time.Duration, defaultValue time.Duration) time.Duration {
	if value != nil {
		return *value
	}
	return defaultValue
}
