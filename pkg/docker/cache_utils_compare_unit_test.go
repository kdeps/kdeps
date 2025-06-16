package docker

import (
	"context"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompareVersionsUnit(t *testing.T) {
	ctx := context.Background()
	assert.True(t, CompareVersions(ctx, "1.2.3", "1.2.0"))
	assert.False(t, CompareVersions(ctx, "1.2.0", "1.2.3"))
	assert.False(t, CompareVersions(ctx, "1.2.3", "1.2.3"))
}

func TestGetCurrentArchitectureMappingUnit(t *testing.T) {
	ctx := context.Background()
	arch := GetCurrentArchitecture(ctx, "apple/pkl")
	switch runtime.GOARCH {
	case "amd64":
		assert.Equal(t, "amd64", arch)
	case "arm64":
		assert.Equal(t, "aarch64", arch)
	default:
		assert.Equal(t, runtime.GOARCH, arch)
	}
}
