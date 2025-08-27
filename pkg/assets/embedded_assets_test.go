package assets

import (
	"testing"
)

func TestEmbeddedAssetsInTests(t *testing.T) {
	// This test verifies that during test execution, embedded assets are used
	if !IsTestEnvironment() {
		t.Error("Expected IsTestEnvironment to return true when running under go test")
	}

	if !ShouldUseEmbeddedAssets() {
		t.Error("Expected ShouldUseEmbeddedAssets to return true in test environment")
	}

	t.Logf("Test environment detected: IsTestEnvironment() = %v", IsTestEnvironment())
	t.Logf("Using embedded assets: ShouldUseEmbeddedAssets() = %v", ShouldUseEmbeddedAssets())
}
