package archiver_test

import (
	"testing"

	. "github.com/kdeps/kdeps/pkg/archiver"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/stretchr/testify/assert"
)

func TestCompareVersions_SingleVersion(t *testing.T) {
	logger := logging.NewTestLogger()
	versions := []string{"1.0.0"}
	result := CompareVersions(versions, logger)
	assert.Equal(t, "1.0.0", result)
}

func TestCompareVersions_MultipleVersionsDifferentParts(t *testing.T) {
	logger := logging.NewTestLogger()
	versions := []string{"1.0.0", "2.0.0", "1.5.0", "1.0.1"}
	result := CompareVersions(versions, logger)
	assert.Equal(t, "2.0.0", result)
}

func TestCompareVersions_MultipleVersionsSameParts(t *testing.T) {
	logger := logging.NewTestLogger()
	versions := []string{"1.0.0", "1.0.0", "1.0.0"}
	result := CompareVersions(versions, logger)
	assert.Equal(t, "1.0.0", result)
}

func TestCompareVersions_MajorVersionComparison(t *testing.T) {
	logger := logging.NewTestLogger()
	versions := []string{"1.0.0", "2.0.0", "3.0.0"}
	result := CompareVersions(versions, logger)
	assert.Equal(t, "3.0.0", result)
}

func TestCompareVersions_MinorVersionComparison(t *testing.T) {
	logger := logging.NewTestLogger()
	versions := []string{"1.0.0", "1.1.0", "1.2.0"}
	result := CompareVersions(versions, logger)
	assert.Equal(t, "1.2.0", result)
}

func TestCompareVersions_PatchVersionComparison(t *testing.T) {
	logger := logging.NewTestLogger()
	versions := []string{"1.0.0", "1.0.1", "1.0.2"}
	result := CompareVersions(versions, logger)
	assert.Equal(t, "1.0.2", result)
}

func TestCompareVersions_MixedVersionTypes(t *testing.T) {
	logger := logging.NewTestLogger()
	versions := []string{"1.0.0", "2.1.0", "1.5.2", "2.0.1"}
	result := CompareVersions(versions, logger)
	assert.Equal(t, "2.1.0", result)
}

func TestCompareVersions_ReverseOrder(t *testing.T) {
	logger := logging.NewTestLogger()
	versions := []string{"3.0.0", "2.0.0", "1.0.0"}
	result := CompareVersions(versions, logger)
	assert.Equal(t, "3.0.0", result)
}
