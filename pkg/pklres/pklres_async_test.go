package pklres

import (
	"encoding/json"
	"net/url"
	"testing"
	"time"

	"github.com/spf13/afero"
)

func TestAsyncDependencyResolution(t *testing.T) {
	// Initialize pklres reader
	fs := afero.NewMemMapFs()
	reader, err := InitializePklResource("test-graph", "test-agent", "1.0.0", "/tmp", fs)
	if err != nil {
		t.Fatalf("Failed to initialize pklres reader: %v", err)
	}

	// Test execution order and dependencies
	executionOrder := []string{"resource1", "resource2", "resource3"}
	resourceDependencies := map[string][]string{
		"resource1": {},
		"resource2": {"resource1"},
		"resource3": {"resource2"},
	}

	// Pre-resolve dependencies
	err = reader.PreResolveDependencies(executionOrder, resourceDependencies)
	if err != nil {
		t.Fatalf("Failed to pre-resolve dependencies: %v", err)
	}

	// Check that all resources are in pending status initially
	statusSummary := reader.GetDependencyStatusSummary()
	if len(statusSummary) != 3 {
		t.Errorf("Expected 3 resources in status summary, got %d", len(statusSummary))
	}

	for actionID, status := range statusSummary {
		if status != "pending" {
			t.Errorf("Expected status 'pending' for %s, got %s", actionID, status)
		}
	}

	// Check dependency relationships
	depData1, err := reader.GetDependencyData("resource1")
	if err != nil {
		t.Fatalf("Failed to get dependency data for resource1: %v", err)
	}
	if len(depData1.Dependencies) != 0 {
		t.Errorf("Expected resource1 to have no dependencies, got %d", len(depData1.Dependencies))
	}

	depData2, err := reader.GetDependencyData("resource2")
	if err != nil {
		t.Fatalf("Failed to get dependency data for resource2: %v", err)
	}
	if len(depData2.Dependencies) != 1 || depData2.Dependencies[0] != "resource1" {
		t.Errorf("Expected resource2 to depend on resource1, got %v", depData2.Dependencies)
	}

	// Test dependency readiness
	if reader.IsDependencyReady("resource1") {
		t.Error("Expected resource1 to not be ready initially")
	}

	// Update resource1 to completed
	err = reader.UpdateDependencyStatus("resource1", "completed", "result1", nil)
	if err != nil {
		t.Fatalf("Failed to update resource1 status: %v", err)
	}

	if !reader.IsDependencyReady("resource1") {
		t.Error("Expected resource1 to be ready after completion")
	}

	// Test waiting for dependencies
	go func() {
		time.Sleep(100 * time.Millisecond)
		reader.UpdateDependencyStatus("resource2", "completed", "result2", nil)
	}()

	err = reader.WaitForDependencies("resource3", 1*time.Second)
	if err != nil {
		t.Errorf("Failed to wait for dependencies: %v", err)
	}

	// Test that resources not in dependency graph are ignored
	uri := url.URL{
		Scheme:   "pklres",
		RawQuery: "op=get&collection=unknownResource&key=test",
	}

	data, err := reader.Read(uri)
	if err != nil {
		t.Fatalf("Failed to read from pklres: %v", err)
	}

	var result interface{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if result != nil {
		t.Errorf("Expected null result for unknown resource, got %v", result)
	}
}

func TestDependencyGraphValidation(t *testing.T) {
	// Initialize pklres reader
	fs := afero.NewMemMapFs()
	reader, err := InitializePklResource("test-graph", "test-agent", "1.0.0", "/tmp", fs)
	if err != nil {
		t.Fatalf("Failed to initialize pklres reader: %v", err)
	}

	// Test with empty execution order
	err = reader.PreResolveDependencies([]string{}, map[string][]string{})
	if err != nil {
		t.Errorf("Expected no error with empty execution order, got %v", err)
	}

	// Test dependency graph validation - should return true when no graph is set up (backward compatibility)
	if !reader.IsInDependencyGraph("nonexistent") {
		t.Error("Expected nonexistent resource to be allowed when no dependency graph is set up")
	}

	// Add a resource to the graph
	executionOrder := []string{"testResource"}
	resourceDependencies := map[string][]string{"testResource": {}}

	err = reader.PreResolveDependencies(executionOrder, resourceDependencies)
	if err != nil {
		t.Fatalf("Failed to pre-resolve dependencies: %v", err)
	}

	if !reader.IsInDependencyGraph("testResource") {
		t.Error("Expected testResource to be in dependency graph")
	}

	// Test that nonexistent resources are rejected when graph is set up
	if reader.IsInDependencyGraph("nonexistent") {
		t.Error("Expected nonexistent resource to not be in dependency graph when graph is set up")
	}
}
