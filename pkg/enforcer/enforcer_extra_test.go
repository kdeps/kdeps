package enforcer

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/stretchr/testify/require"
)

func TestEnforcePklVersion(t *testing.T) {
	logger := logging.NewTestLogger()
	ctx := context.Background()
	schemaVersion := "1.2.3"

	goodLine := "amends \"package://schema.kdeps.com/core@1.2.3#/Kdeps.pkl\""
	require.NoError(t, EnforcePklVersion(ctx, goodLine, "file.pkl", schemaVersion, logger))

	// lower version should warn but not error
	lowLine := "amends \"package://schema.kdeps.com/core@1.0.0#/Kdeps.pkl\""
	require.NoError(t, EnforcePklVersion(ctx, lowLine, "file.pkl", schemaVersion, logger))

	// higher version also no error
	highLine := "amends \"package://schema.kdeps.com/core@2.0.0#/Kdeps.pkl\""
	require.NoError(t, EnforcePklVersion(ctx, highLine, "file.pkl", schemaVersion, logger))

	// invalid version format should error
	badLine := "amends \"package://schema.kdeps.com/core@1.x#/Kdeps.pkl\""
	require.Error(t, EnforcePklVersion(ctx, badLine, "file.pkl", schemaVersion, logger))
}

func TestEnforcePklFilename(t *testing.T) {
	logger := logging.NewTestLogger()
	ctx := context.Background()

	// Good configuration .kdeps.pkl
	lineCfg := "amends \"package://schema.kdeps.com/core@1.0.0#/Kdeps.pkl\""
	require.NoError(t, EnforcePklFilename(ctx, lineCfg, "/path/to/.kdeps.pkl", logger))

	// Good workflow.pkl
	lineWf := "amends \"package://schema.kdeps.com/core@1.0.0#/Workflow.pkl\""
	require.NoError(t, EnforcePklFilename(ctx, lineWf, "/some/workflow.pkl", logger))

	// Resource.pkl must not have those filenames
	lineResource := "amends \"package://schema.kdeps.com/core@1.0.0#/Resource.pkl\""
	require.NoError(t, EnforcePklFilename(ctx, lineResource, "/path/to/resources/custom.pkl", logger))

	// Invalid file extension for config
	err := EnforcePklFilename(ctx, lineCfg, "/path/to/wrongname.txt", logger)
	require.Error(t, err)

	// Resource.pkl with forbidden filename
	err = EnforcePklFilename(ctx, lineResource, "/path/to/.kdeps.pkl", logger)
	require.Error(t, err)

	// Unknown pkl filename in amends line -> expect error
	unknownLine := "amends \"package://schema.kdeps.com/core@1.0.0#/Unknown.pkl\""
	err = EnforcePklFilename(ctx, unknownLine, "/path/to/unknown.pkl", logger)
	require.Error(t, err)
}
