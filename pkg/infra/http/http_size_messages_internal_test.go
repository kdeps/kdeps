package http

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequestBodyTooLargeMessage(t *testing.T) {
	t.Parallel()
	msg := requestBodyTooLargeMessage(1024)
	assert.Contains(t, msg, "1024")
	assert.Contains(t, msg, "limit")
}

func TestUploadBodyTooLargeMessage(t *testing.T) {
	t.Parallel()
	msg := uploadBodyTooLargeMessage(2048, 1024)
	assert.Contains(t, msg, "2048")
	assert.Contains(t, msg, "1024")
}

func TestFileTooLargeMessage(t *testing.T) {
	t.Parallel()
	msg := fileTooLargeMessage(5000, 4096)
	assert.Contains(t, msg, "5000")
	assert.Contains(t, msg, "4096")
}

func TestLabelExceedsMaxMessage(t *testing.T) {
	t.Parallel()
	msg := labelExceedsMaxMessage("MyField", 256)
	assert.Contains(t, msg, "MyField")
	assert.Contains(t, msg, "256")
}

func TestPackageFileSizeExceededMessage(t *testing.T) {
	t.Parallel()
	msg := packageFileSizeExceededMessage("archive.tar.gz", 10485760)
	assert.Contains(t, msg, "archive.tar.gz")
	assert.True(t, strings.Contains(msg, "10485760"))
}

func TestPackageTotalSizeExceededMessage(t *testing.T) {
	t.Parallel()
	msg := packageTotalSizeExceededMessage(52428800)
	assert.Contains(t, msg, "52428800")
	assert.Contains(t, msg, "package")
}

func TestPackageEntryCountExceededMessage(t *testing.T) {
	t.Parallel()
	msg := packageEntryCountExceededMessage(1000)
	assert.Contains(t, msg, "1000")
	assert.Contains(t, msg, "entry count")
}
