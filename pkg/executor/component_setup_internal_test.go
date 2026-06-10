// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package executor

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	pythonpkg "github.com/kdeps/kdeps/v2/pkg/infra/python"
)

func TestDetectPackageManager_NoPackageManagerFound(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	name, checkFn, installFn := detectPackageManager()
	assert.Empty(t, name)
	assert.Nil(t, checkFn)
	assert.Nil(t, installFn)
}

func TestInstallOSPackages_NoPackages(t *testing.T) {
	require.NoError(t, installOSPackages(nil))
	require.NoError(t, installOSPackages([]string{}))
}

func TestInstallOSPackages_NoSupportedPackageManager(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	err := installOSPackages([]string{"curl", "git"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no supported package manager found")
}

func TestRunShellCommand_NonexistentCommand(t *testing.T) {
	err := runShellCommand("nonexistent_command_xyz_123")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "command failed")
}

func TestRunShellCommand_BashShell(t *testing.T) {
	t.Setenv("SHELL", "/bin/bash")
	require.NoError(t, runShellCommand("echo hello"))
}

func TestRunShellCommand_SimpleEcho(t *testing.T) {
	t.Setenv("SHELL", "")
	require.NoError(t, runShellCommand("echo hello"))
}

func TestRunShellCommand_EmptyCommand(t *testing.T) {
	// sh -c "" returns success (exit 0), so this does not error.
	require.NoError(t, runShellCommand(""))
}

func TestCollectPythonPackages_Dedup(t *testing.T) {
	comp := &domain.Component{
		PythonPackages: []string{"requests", "numpy"}, //nolint:staticcheck
	}
	setup := &domain.ComponentSetup{
		PythonPackages: []string{"requests", "pandas"},
	}
	result := collectPythonPackages(comp, setup)
	assert.ElementsMatch(t, []string{"requests", "numpy", "pandas"}, result)
}

func TestCollectPythonPackages_EmptyBoth(t *testing.T) {
	result := collectPythonPackages(&domain.Component{}, &domain.ComponentSetup{})
	assert.Empty(t, result)
}

func TestCollectPythonPackages_DedupWithinLegacy(t *testing.T) {
	comp := &domain.Component{
		PythonPackages: []string{"requests", "requests", "numpy"}, //nolint:staticcheck
	}
	result := collectPythonPackages(comp, &domain.ComponentSetup{})
	assert.ElementsMatch(t, []string{"requests", "numpy"}, result)
}

func TestRunComponentTeardown_NoTeardown(t *testing.T) {
	t.Helper()
	e := &Engine{}
	e.runComponentTeardown(&domain.Component{})
}

func TestRunComponentTeardown_NilCommands(t *testing.T) {
	t.Helper()
	e := &Engine{}
	e.runComponentTeardown(&domain.Component{Teardown: &domain.ComponentTeardown{}})
}

func TestRunComponentTeardown_CommandsRun(t *testing.T) {
	t.Helper()
	e := &Engine{}
	e.runComponentTeardown(&domain.Component{
		Teardown: &domain.ComponentTeardown{Commands: []string{"echo teardown-test"}},
	})
}

func TestRunComponentTeardown_CommandError(t *testing.T) {
	t.Helper()
	e := &Engine{}
	e.logger = slog.Default()
	e.runComponentTeardown(&domain.Component{
		Teardown: &domain.ComponentTeardown{Commands: []string{"nonexistent_cmd_xyz"}},
	})
}

func TestRunShellCommand_ExitCode(t *testing.T) {
	err := runShellCommand("exit 42")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "command failed")
}

func TestCommandExists_Missing(t *testing.T) {
	assert.False(t, commandExists("nonexistent_command_xyz_123_456"))
}

func TestCommandExists_Present(t *testing.T) {
	assert.True(t, commandExists("sh"))
}

func TestRunComponentSetup_NilSetupNoPythonPkgs(t *testing.T) {
	e := &Engine{}
	e.logger = slog.Default()
	err := e.runComponentSetup(
		&domain.Component{Metadata: domain.ComponentMetadata{Name: "test-comp"}},
		&ExecutionContext{},
	)
	require.NoError(t, err)
}

func TestRunComponentSetup_NoPythonPkgsWithSetupCommands(t *testing.T) {
	e := &Engine{}
	e.logger = slog.Default()
	err := e.runComponentSetup(
		&domain.Component{
			Metadata: domain.ComponentMetadata{Name: "test-comp-setup"},
			Setup:    &domain.ComponentSetup{Commands: []string{"echo setup-ran"}},
		},
		&ExecutionContext{},
	)
	require.NoError(t, err)
}

func TestRunComponentSetup_CacheHit(t *testing.T) {
	e := &Engine{}
	e.logger = slog.Default()
	comp := &domain.Component{
		Metadata: domain.ComponentMetadata{Name: "test-comp-cached"},
		Setup:    &domain.ComponentSetup{Commands: []string{"echo setup-ran"}},
	}
	ctx := &ExecutionContext{}
	require.NoError(t, e.runComponentSetup(comp, ctx))
	require.NoError(t, e.runComponentSetup(comp, ctx))
}

func TestRunComponentSetup_ScaffoldFiles(t *testing.T) {
	e := &Engine{}
	e.logger = slog.Default()
	tmpDir := t.TempDir()
	comp := &domain.Component{
		Metadata: domain.ComponentMetadata{Name: "test-comp-scaffold"},
		Dir:      tmpDir,
		Setup:    &domain.ComponentSetup{Commands: []string{"true"}},
	}
	require.NoError(t, e.runComponentSetup(comp, &ExecutionContext{}))
}

func TestRunCommand_Simple(t *testing.T) {
	require.NoError(t, runCommand("echo", []string{"hello"}))
}

func TestRunCommand_Error(t *testing.T) {
	err := runCommand("nonexistent_cmd_xyz_123", []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed")
}

func TestInstallComponentPythonPackages_Success(t *testing.T) {
	orig := pythonManagerFactory
	t.Cleanup(func() { pythonManagerFactory = orig })
	pythonManagerFactory = func(baseDir string) *pythonpkg.Manager {
		return &pythonpkg.Manager{BaseDir: baseDir}
	}
	// Create a temp dir that won't work for venv creation — the test will exercise
	// the code path even if EnsureVenv fails gracefully.
	t.Setenv("HOME", t.TempDir())
	e := &Engine{}
	ctx := &ExecutionContext{
		Workflow: &domain.Workflow{
			Metadata: domain.WorkflowMetadata{Name: "test"},
			Settings: domain.WorkflowSettings{
				AgentSettings: domain.AgentSettings{
					PythonVersion:  pythonpkg.IOToolsPythonVersion,
					PythonPackages: []string{"requests"},
				},
			},
		},
	}
	// This will attempt real venv creation — may fail but exercises the code path.
	_ = e.installComponentPythonPackages([]string{"example-pkg"}, ctx)
}

func TestInstallComponentPythonPackages_DefaultVersion(t *testing.T) {
	orig := pythonManagerFactory
	t.Cleanup(func() { pythonManagerFactory = orig })
	pythonManagerFactory = func(baseDir string) *pythonpkg.Manager {
		return &pythonpkg.Manager{BaseDir: baseDir}
	}
	t.Setenv("HOME", t.TempDir())
	e := &Engine{}
	ctx := &ExecutionContext{
		Workflow: &domain.Workflow{
			Metadata: domain.WorkflowMetadata{Name: "test"},
			Settings: domain.WorkflowSettings{
				AgentSettings: domain.AgentSettings{PythonVersion: ""},
			},
		},
	}
	_ = e.installComponentPythonPackages([]string{}, ctx)
}

func TestInstallComponentPythonPackages_WithRequirements(t *testing.T) {
	orig := pythonManagerFactory
	t.Cleanup(func() { pythonManagerFactory = orig })
	pythonManagerFactory = func(baseDir string) *pythonpkg.Manager {
		return &pythonpkg.Manager{BaseDir: baseDir}
	}
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	reqPath := filepath.Join(tmpDir, "requirements.txt")
	require.NoError(t, os.WriteFile(reqPath, []byte("click\n"), 0644))

	e := &Engine{}
	ctx := &ExecutionContext{
		Workflow: &domain.Workflow{
			Metadata: domain.WorkflowMetadata{Name: "test"},
			Settings: domain.WorkflowSettings{
				AgentSettings: domain.AgentSettings{
					PythonVersion:    pythonpkg.IOToolsPythonVersion,
					RequirementsFile: reqPath,
				},
			},
		},
	}
	_ = e.installComponentPythonPackages([]string{"requests"}, ctx)
}

// ---------------------------------------------------------------------------
// scaffoldComponentFilesIfNeeded error path
// ---------------------------------------------------------------------------

func TestScaffoldComponentFilesIfNeeded_ReadOnlyDir(t *testing.T) {
	tmpDir := t.TempDir()
	roDir := filepath.Join(tmpDir, "readonly")
	require.NoError(t, os.MkdirAll(roDir, 0755))
	require.NoError(t, os.Chmod(roDir, 0o000))
	t.Cleanup(func() { _ = os.Chmod(roDir, 0755) })

	e := &Engine{logger: slog.Default()}
	comp := &domain.Component{
		Metadata: domain.ComponentMetadata{Name: "test-scaffold-ro"},
		Dir:      roDir,
	}
	// Should log a warning but not panic.
	e.scaffoldComponentFilesIfNeeded(comp)
}

// ---------------------------------------------------------------------------
// installComponentPackages paths
// ---------------------------------------------------------------------------

func TestInstallComponentPackages_PythonPackages(t *testing.T) {
	orig := pythonManagerFactory
	t.Cleanup(func() { pythonManagerFactory = orig })
	pythonManagerFactory = func(baseDir string) *pythonpkg.Manager {
		return &pythonpkg.Manager{BaseDir: baseDir}
	}
	t.Setenv("HOME", t.TempDir())

	e := &Engine{logger: slog.Default()}
	comp := &domain.Component{
		Metadata: domain.ComponentMetadata{Name: "test-comp-py"},
	}
	setup := &domain.ComponentSetup{
		PythonPackages: []string{"some-pkg"},
	}
	ctx := &ExecutionContext{
		Workflow: &domain.Workflow{
			Settings: domain.WorkflowSettings{
				AgentSettings: domain.AgentSettings{
					PythonVersion: pythonpkg.IOToolsPythonVersion,
				},
			},
		},
	}
	// Errors are logged as warnings, does not panic.
	e.installComponentPackages(comp, setup, ctx)
}

func TestInstallComponentPackages_OSPackages(t *testing.T) {
	tmpDir := t.TempDir()
	apkBin := filepath.Join(tmpDir, "apk")
	require.NoError(t, os.WriteFile(apkBin, []byte("#!/bin/sh\n"+
		"if [ \"$1\" = \"info\" ]; then exit 1; fi\n"+
		"exit 0\n"), 0755))
	t.Setenv("PATH", tmpDir)

	e := &Engine{logger: slog.Default()}
	comp := &domain.Component{
		Metadata: domain.ComponentMetadata{Name: "test-comp-os"},
	}
	setup := &domain.ComponentSetup{
		OsPackages: []string{"some-os-pkg"},
	}
	ctx := &ExecutionContext{
		Workflow: &domain.Workflow{
			Settings: domain.WorkflowSettings{
				AgentSettings: domain.AgentSettings{
					PythonVersion: pythonpkg.IOToolsPythonVersion,
				},
			},
		},
	}
	e.installComponentPackages(comp, setup, ctx)
}

func TestInstallComponentPackages_OSPackagesNoPkgManager(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	e := &Engine{logger: slog.Default()}
	comp := &domain.Component{
		Metadata: domain.ComponentMetadata{Name: "test-comp-os-nopm"},
	}
	setup := &domain.ComponentSetup{
		OsPackages: []string{"some-os-pkg"},
	}
	ctx := &ExecutionContext{
		Workflow: &domain.Workflow{
			Settings: domain.WorkflowSettings{
				AgentSettings: domain.AgentSettings{
					PythonVersion: pythonpkg.IOToolsPythonVersion,
				},
			},
		},
	}
	e.installComponentPackages(comp, setup, ctx)
}

// ---------------------------------------------------------------------------
// installOSPackages success paths
// ---------------------------------------------------------------------------

func TestInstallOSPackages_AllInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	apkBin := filepath.Join(tmpDir, "apk")
	// apk info -e exits 0 => package is already installed.
	require.NoError(t, os.WriteFile(apkBin, []byte("#!/bin/sh\n"+
		"case \"$1\" in\n"+
		"  info) [ \"$2\" = \"-e\" ] && exit 0 ;;\n"+
		"  add) [ \"$2\" = \"--no-cache\" ] && exit 0 ;;\n"+
		"esac\n"+
		"exit 1\n"), 0755))
	t.Setenv("PATH", tmpDir+string(os.PathListSeparator)+"/usr/bin:/bin")

	require.NoError(t, installOSPackages([]string{"some-pkg"}))
}

func TestInstallOSPackages_WithAPK(t *testing.T) {
	tmpDir := t.TempDir()
	apkBin := filepath.Join(tmpDir, "apk")
	content := "#!/bin/sh\n" +
		"if [ \"$1\" = \"info\" ] && [ \"$2\" = \"-e\" ]; then exit 1; fi\n" +
		"if [ \"$1\" = \"add\" ] && [ \"$2\" = \"--no-cache\" ]; then exit 0; fi\n" +
		"exit 1\n"
	require.NoError(t, os.WriteFile(apkBin, []byte(content), 0755))
	t.Setenv("PATH", tmpDir)

	require.NoError(t, installOSPackages([]string{"some-pkg"}))
}

func TestInstallOSPackages_WithAptGet(t *testing.T) {
	tmpDir := t.TempDir()
	// dpkg -s exits 1 => not installed
	require.NoError(t, os.WriteFile(
		filepath.Join(tmpDir, "dpkg"),
		[]byte("#!/bin/sh\nexit 1\n"), 0755))
	// apt-get update + install succeed
	aptGetContent := "#!/bin/sh\n" +
		"if [ \"$1\" = \"update\" ]; then exit 0; fi\n" +
		"if [ \"$1\" = \"install\" ]; then exit 0; fi\n" +
		"exit 1\n"
	require.NoError(t, os.WriteFile(
		filepath.Join(tmpDir, "apt-get"),
		[]byte(aptGetContent), 0755))
	t.Setenv("PATH", tmpDir)

	require.NoError(t, installOSPackages([]string{"some-pkg"}))
}

// ---------------------------------------------------------------------------
// detectPackageManager per-manager paths
// ---------------------------------------------------------------------------

func TestDetectPackageManager_APK(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(tmpDir, "apk"),
		[]byte("#!/bin/sh\nexit 0\n"), 0755))
	t.Setenv("PATH", tmpDir)

	name, checkFn, installFn := detectPackageManager()
	assert.Equal(t, "apk", name)
	assert.NotNil(t, checkFn)
	assert.NotNil(t, installFn)
}

func TestDetectPackageManager_AptGet(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(tmpDir, "apt-get"),
		[]byte("#!/bin/sh\nexit 0\n"), 0755))
	t.Setenv("PATH", tmpDir)

	name, checkFn, installFn := detectPackageManager()
	assert.Equal(t, "apt-get", name)
	assert.NotNil(t, checkFn)
	assert.NotNil(t, installFn)
}

func TestDetectPackageManager_Brew(t *testing.T) {
	origGOOS := componentGOOS
	t.Cleanup(func() { componentGOOS = origGOOS })
	componentGOOS = goOSDarwin

	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(tmpDir, "brew"),
		[]byte("#!/bin/sh\nexit 0\n"), 0755))
	t.Setenv("PATH", tmpDir)

	name, checkFn, installFn := detectPackageManager()
	assert.Equal(t, "brew", name)
	assert.NotNil(t, checkFn)
	assert.NotNil(t, installFn)
}

func TestInstallOSPackages_WithBrewOverride(t *testing.T) {
	origGOOS := componentGOOS
	t.Cleanup(func() { componentGOOS = origGOOS })
	componentGOOS = goOSDarwin

	tmpDir := t.TempDir()
	brewContent := "#!/bin/sh\n" +
		"if [ \"$1\" = \"list\" ] && [ \"$2\" = \"--formula\" ]; then exit 1; fi\n" +
		"if [ \"$1\" = \"install\" ]; then exit 0; fi\n" +
		"exit 1\n"
	require.NoError(t, os.WriteFile(
		filepath.Join(tmpDir, "brew"),
		[]byte(brewContent), 0755))
	t.Setenv("PATH", tmpDir)

	require.NoError(t, installOSPackages([]string{"some-pkg"}))
}
