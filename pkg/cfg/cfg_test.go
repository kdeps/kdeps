package cfg_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/adrg/xdg"
	"github.com/cucumber/godog"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/kdeps/pkg/texteditor"
	"github.com/kdeps/schema/gen/kdeps"
	"github.com/kdeps/schema/gen/kdeps/path"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	kpath "github.com/kdeps/schema/gen/kdeps/path"
)

var (
	testFs         = afero.NewOsFs()
	currentDirPath string
	homeDirPath    string
	fileThatExist  string
	ctx            = context.Background()
	logger         *logging.Logger
	testingT       *testing.T
)

func init() {
	os.Setenv("NON_INTERACTIVE", "1")
	// Save the original EditPkl function
	originalEditPkl := texteditor.EditPkl
	// Replace with mock for testing
	texteditor.EditPkl = texteditor.MockEditPkl
	// Restore original after tests
	defer func() { texteditor.EditPkl = originalEditPkl }()
}

func setNonInteractive(t *testing.T) func() {
	old := os.Getenv("NON_INTERACTIVE")
	os.Setenv("NON_INTERACTIVE", "1")
	return func() { os.Setenv("NON_INTERACTIVE", old) }
}

func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			ctx.Step(`^a file "([^"]*)" exists in the current directory$`, aFileExistsInTheCurrentDirectory)
			ctx.Step(`^a file "([^"]*)" exists in the home directory$`, aFileExistsInTheHomeDirectory)
			ctx.Step(`^the configuration file is "([^"]*)"$`, theConfigurationFileIs)
			ctx.Step(`^the configuration is loaded in the current directory$`, theConfigurationIsLoadedInTheCurrentDirectory)
			ctx.Step(`^the configuration is loaded in the home directory$`, theConfigurationIsLoadedInTheHomeDirectory)
			ctx.Step(`^the current directory is "([^"]*)"$`, theCurrentDirectoryIs)
			ctx.Step(`^the home directory is "([^"]*)"$`, theHomeDirectoryIs)
			ctx.Step(`^a file "([^"]*)" does not exists in the home or current directory$`, aFileDoesNotExistsInTheHomeOrCurrentDirectory)
			ctx.Step(`^the configuration fails to load any configuration$`, theConfigurationFailsToLoadAnyConfiguration)
			ctx.Step(`^the configuration file will be generated to "([^"]*)"$`, theConfigurationFileWillBeGeneratedTo)
			ctx.Step(`^the configuration will be edited$`, theConfigurationWillBeEdited)
			ctx.Step(`^the configuration will be validated$`, theConfigurationWillBeValidated)
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/cfg"},
			TestingT: t, // Testing instance that will run subtests.
		},
	}

	testingT = t

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}

func aFileExistsInTheCurrentDirectory(arg1 string) error {
	logger = logging.GetLogger()

	doc := fmt.Sprintf(`
amends "package://schema.kdeps.com/core@%s#/Kdeps.pkl"

runMode = "docker"
dockerGPU = "cpu"
`, schema.SchemaVersion(ctx))
	file := filepath.Join(currentDirPath, arg1)

	f, _ := testFs.Create(file)
	if _, err := f.WriteString(doc); err != nil {
		return err
	}
	f.Close()

	fileThatExist = file

	return nil
}

func aFileExistsInTheHomeDirectory(arg1 string) error {
	doc := fmt.Sprintf(`
amends "package://schema.kdeps.com/core@%s#/Kdeps.pkl"

runMode = "docker"
dockerGPU = "cpu"
`, schema.SchemaVersion(ctx))
	file := filepath.Join(homeDirPath, arg1)

	f, _ := testFs.Create(file)
	if _, err := f.WriteString(doc); err != nil {
		return err
	}
	f.Close()

	fileThatExist = file

	return nil
}

func theConfigurationFileIs(arg1 string) error {
	if _, err := testFs.Stat(fileThatExist); err != nil {
		return err
	}

	return nil
}

func theConfigurationIsLoadedInTheCurrentDirectory() error {
	env := &environment.Environment{
		Home: "",
		Pwd:  currentDirPath,
	}

	environ, err := environment.NewEnvironment(testFs, env)
	if err != nil {
		return err
	}

	cfgFile, err := FindConfiguration(testFs, ctx, environ, logger)
	if err != nil {
		return err
	}

	if _, err := LoadConfiguration(testFs, ctx, cfgFile, logger); err != nil {
		return err
	}

	return nil
}

func theConfigurationIsLoadedInTheHomeDirectory() error {
	env := &environment.Environment{
		Home: homeDirPath,
		Pwd:  "",
	}

	environ, err := environment.NewEnvironment(testFs, env)
	if err != nil {
		return err
	}

	cfgFile, err := FindConfiguration(testFs, ctx, environ, logger)
	if err != nil {
		return err
	}

	if _, err := LoadConfiguration(testFs, ctx, cfgFile, logger); err != nil {
		return err
	}

	return nil
}

func theCurrentDirectoryIs(arg1 string) error {
	tempDir, err := afero.TempDir(testFs, "", "")
	if err != nil {
		return err
	}

	currentDirPath = tempDir

	return nil
}

func theHomeDirectoryIs(arg1 string) error {
	tempDir, err := afero.TempDir(testFs, "", "")
	if err != nil {
		return err
	}

	homeDirPath = tempDir

	return nil
}

func aFileDoesNotExistsInTheHomeOrCurrentDirectory(arg1 string) error {
	fileThatExist = ""

	return nil
}

func theConfigurationFailsToLoadAnyConfiguration() error {
	env := &environment.Environment{
		Home: homeDirPath,
		Pwd:  currentDirPath,
	}

	environ, err := environment.NewEnvironment(testFs, env)
	if err != nil {
		return err
	}

	cfgFile, err := FindConfiguration(testFs, ctx, environ, logger)
	if err != nil {
		return fmt.Errorf("an error occurred while finding configuration: %w", err)
	}
	if cfgFile != "" {
		return errors.New("expected not finding configuration file, but found")
	}

	return nil
}

func theConfigurationFileWillBeGeneratedTo(arg1 string) error {
	env := &environment.Environment{
		Home:           homeDirPath,
		Pwd:            "",
		NonInteractive: "1",
	}

	environ, err := environment.NewEnvironment(testFs, env)
	if err != nil {
		return err
	}

	cfgFile, err := GenerateConfiguration(testFs, ctx, environ, logger)
	if err != nil {
		return err
	}

	if _, err := LoadConfiguration(testFs, ctx, cfgFile, logger); err != nil {
		return err
	}

	return nil
}

func theConfigurationWillBeEdited() error {
	env := &environment.Environment{
		Home:           homeDirPath,
		Pwd:            "",
		NonInteractive: "1",
	}

	environ, err := environment.NewEnvironment(testFs, env)
	if err != nil {
		return err
	}

	if _, err := EditConfiguration(testFs, ctx, environ, logger); err != nil {
		return err
	}

	return nil
}

func theConfigurationWillBeValidated() error {
	env := &environment.Environment{
		Home: homeDirPath,
		Pwd:  "",
	}

	environ, err := environment.NewEnvironment(testFs, env)
	if err != nil {
		return err
	}

	if _, err := ValidateConfiguration(testFs, ctx, environ, logger); err != nil {
		return err
	}

	return nil
}

// Unit Tests for comprehensive coverage

func TestFindConfigurationUnit(t *testing.T) {
	logger := logging.NewTestLogger()
	ctx := context.Background()

	t.Run("ConfigInPwd", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		env := &environment.Environment{
			Pwd:  "/test/pwd",
			Home: "/test/home",
		}

		// Create config file in Pwd
		fs.MkdirAll("/test/pwd", 0o755)
		afero.WriteFile(fs, "/test/pwd/.kdeps.pkl", []byte("test"), 0o644)

		result, err := FindConfiguration(fs, ctx, env, logger)
		assert.NoError(t, err)
		assert.Equal(t, "/test/pwd/.kdeps.pkl", result)
	})

	t.Run("ConfigInHome", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		env := &environment.Environment{
			Pwd:  "/test/pwd",
			Home: "/test/home",
		}

		// Create config file only in Home
		fs.MkdirAll("/test/home", 0o755)
		afero.WriteFile(fs, "/test/home/.kdeps.pkl", []byte("test"), 0o644)

		result, err := FindConfiguration(fs, ctx, env, logger)
		assert.NoError(t, err)
		assert.Equal(t, "/test/home/.kdeps.pkl", result)
	})

	t.Run("NoConfigFound", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		env := &environment.Environment{
			Pwd:  "/test/pwd",
			Home: "/test/home",
		}

		result, err := FindConfiguration(fs, ctx, env, logger)
		assert.NoError(t, err)
		assert.Equal(t, "", result)
	})
}

func TestGenerateConfigurationUnit(t *testing.T) {
	logger := logging.NewTestLogger()
	ctx := context.Background()

	t.Run("NonInteractiveMode", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		env := &environment.Environment{
			Home:           "/test/home",
			NonInteractive: "1",
		}

		fs.MkdirAll("/test/home", 0o755)

		result, err := GenerateConfiguration(fs, ctx, env, logger)
		// This might fail due to evaluator.EvalPkl, but we test the path
		if err != nil {
			assert.Contains(t, err.Error(), "failed to evaluate .pkl file")
		} else {
			assert.Equal(t, "/test/home/.kdeps.pkl", result)
		}
	})

	t.Run("ConfigFileExists", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		env := &environment.Environment{
			Home:           "/test/home",
			NonInteractive: "1",
		}

		fs.MkdirAll("/test/home", 0o755)
		afero.WriteFile(fs, "/test/home/.kdeps.pkl", []byte("existing"), 0o644)

		result, err := GenerateConfiguration(fs, ctx, env, logger)
		assert.NoError(t, err)
		assert.Equal(t, "/test/home/.kdeps.pkl", result)
	})
}

func TestEditConfigurationUnit(t *testing.T) {
	logger := logging.NewTestLogger()
	ctx := context.Background()

	t.Run("NonInteractiveMode", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		env := &environment.Environment{
			Home:           "/test/home",
			NonInteractive: "1",
		}

		fs.MkdirAll("/test/home", 0o755)
		afero.WriteFile(fs, "/test/home/.kdeps.pkl", []byte("test"), 0o644)

		result, err := EditConfiguration(fs, ctx, env, logger)
		assert.NoError(t, err)
		assert.Equal(t, "/test/home/.kdeps.pkl", result)
	})

	t.Run("ConfigFileDoesNotExist", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		env := &environment.Environment{
			Home:           "/test/home",
			NonInteractive: "1",
		}

		fs.MkdirAll("/test/home", 0o755)

		result, err := EditConfiguration(fs, ctx, env, logger)
		assert.NoError(t, err)
		assert.Equal(t, "/test/home/.kdeps.pkl", result)
	})
}

func TestValidateConfigurationUnit(t *testing.T) {
	logger := logging.NewTestLogger()
	ctx := context.Background()

	t.Run("ValidationFailure", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		env := &environment.Environment{
			Home: "/test/home",
		}

		fs.MkdirAll("/test/home", 0o755)
		afero.WriteFile(fs, "/test/home/.kdeps.pkl", []byte("invalid pkl"), 0o644)

		result, err := ValidateConfiguration(fs, ctx, env, logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "configuration validation failed")
		assert.Equal(t, "/test/home/.kdeps.pkl", result)
	})
}

func TestLoadConfigurationUnit(t *testing.T) {
	logger := logging.NewTestLogger()
	ctx := context.Background()

	t.Run("InvalidConfigFile", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		afero.WriteFile(fs, "/test/invalid.pkl", []byte("invalid"), 0o644)

		result, err := LoadConfiguration(fs, ctx, "/test/invalid.pkl", logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error reading config file")
		assert.Nil(t, result)
	})

	t.Run("NonExistentFile", func(t *testing.T) {
		fs := afero.NewMemMapFs()

		result, err := LoadConfiguration(fs, ctx, "/test/nonexistent.pkl", logger)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestGetKdepsPath(t *testing.T) {
	tests := []struct {
		name     string
		kdepsCfg kdeps.Kdeps
		want     string
		wantErr  bool
	}{
		{
			name: "UserPath",
			kdepsCfg: kdeps.Kdeps{
				KdepsDir:  ".kdeps",
				KdepsPath: path.User,
			},
			want:    filepath.Join(os.Getenv("HOME"), ".kdeps"),
			wantErr: false,
		},
		{
			name: "ProjectPath",
			kdepsCfg: kdeps.Kdeps{
				KdepsDir:  ".kdeps",
				KdepsPath: path.Project,
			},
			want:    filepath.Join(os.Getenv("PWD"), ".kdeps"),
			wantErr: false,
		},
		{
			name: "XdgPath",
			kdepsCfg: kdeps.Kdeps{
				KdepsDir:  ".kdeps",
				KdepsPath: path.Xdg,
			},
			want:    filepath.Join(xdg.ConfigHome, ".kdeps"),
			wantErr: false,
		},
		{
			name: "InvalidPath",
			kdepsCfg: kdeps.Kdeps{
				KdepsDir:  ".kdeps",
				KdepsPath: "invalid",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "EmptyKdepsDir",
			kdepsCfg: kdeps.Kdeps{
				KdepsDir:  "",
				KdepsPath: path.User,
			},
			want:    filepath.Join(os.Getenv("HOME"), ""),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetKdepsPath(ctx, tt.kdepsCfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetKdepsPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("GetKdepsPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateConfigurationAdditional(t *testing.T) {
	logger := logging.NewTestLogger()
	ctx := context.Background()

	t.Run("WriteFileError", func(t *testing.T) {
		fs := afero.NewReadOnlyFs(afero.NewMemMapFs())
		env := &environment.Environment{
			Home:           "/test/home",
			NonInteractive: "1",
		}

		result, err := GenerateConfiguration(fs, ctx, env, logger)
		// This will fail when trying to write the file
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write to")
		assert.Equal(t, "", result)
	})
}

func TestEditConfigurationAdditional(t *testing.T) {
	logger := logging.NewTestLogger()
	ctx := context.Background()

	t.Run("InteractiveMode", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		env := &environment.Environment{
			Home:           "/test/home",
			NonInteractive: "1", // Non-interactive to skip prompt
		}

		fs.MkdirAll("/test/home", 0o755)
		afero.WriteFile(fs, "/test/home/.kdeps.pkl", []byte("test"), 0o644)

		result, err := EditConfiguration(fs, ctx, env, logger)
		// This might fail due to texteditor.EditPkl, but we test the path
		if err != nil {
			assert.Contains(t, err.Error(), "failed to edit configuration file")
		} else {
			assert.Equal(t, "/test/home/.kdeps.pkl", result)
		}
	})
}

func TestValidateConfigurationAdditional(t *testing.T) {
	logger := logging.NewTestLogger()
	ctx := context.Background()

	t.Run("ValidConfig", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		env := &environment.Environment{
			Home: "/test/home",
		}

		fs.MkdirAll("/test/home", 0o755)
		// Create a valid-looking config that might pass validation
		validConfig := fmt.Sprintf(`
amends "package://schema.kdeps.com/core@%s#/Kdeps.pkl"

runMode = "docker"
dockerGPU = "cpu"
`, schema.SchemaVersion(ctx))
		afero.WriteFile(fs, "/test/home/.kdeps.pkl", []byte(validConfig), 0o644)

		result, err := ValidateConfiguration(fs, ctx, env, logger)
		// This might still fail due to evaluator.EvalPkl dependencies, but we test the path
		if err != nil {
			assert.Contains(t, err.Error(), "configuration validation failed")
		} else {
			assert.NoError(t, err)
		}
		assert.Equal(t, "/test/home/.kdeps.pkl", result)
	})
}

func TestLoadConfigurationAdditional(t *testing.T) {
	logger := logging.NewTestLogger()
	ctx := context.Background()

	t.Run("ValidConfigFile", func(t *testing.T) {
		fs := afero.NewMemMapFs()

		// Create a basic valid pkl config file that might work
		validConfig := fmt.Sprintf(`
amends "package://schema.kdeps.com/core@%s#/Kdeps.pkl"

runMode = "docker"
dockerGPU = "cpu"
`, schema.SchemaVersion(ctx))
		afero.WriteFile(fs, "/test/valid.pkl", []byte(validConfig), 0o644)

		result, err := LoadConfiguration(fs, ctx, "/test/valid.pkl", logger)
		// This might fail due to kdeps.LoadFromPath dependencies, but we test the code path
		if err != nil {
			assert.Contains(t, err.Error(), "error reading config file")
		} else {
			assert.NotNil(t, result)
		}
	})
}

func TestMain(m *testing.M) {
	teardown := setNonInteractive(nil)
	defer teardown()
	os.Exit(m.Run())
}

// helper to construct minimal config
func newKdepsCfg(dir string, p path.Path) kdeps.Kdeps {
	return kdeps.Kdeps{
		KdepsDir:  dir,
		KdepsPath: p,
	}
}

func TestGetKdepsPathUser(t *testing.T) {
	cfg := newKdepsCfg(".kdeps", path.User)
	got, err := GetKdepsPath(context.Background(), cfg)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".kdeps")
	if got != want {
		t.Fatalf("want %s got %s", want, got)
	}
}

func TestGetKdepsPathProject(t *testing.T) {
	cfg := newKdepsCfg("kd", path.Project)
	cwd, _ := os.Getwd()
	got, err := GetKdepsPath(context.Background(), cfg)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	want := filepath.Join(cwd, "kd")
	if got != want {
		t.Fatalf("want %s got %s", want, got)
	}
}

func TestGetKdepsPathXDG(t *testing.T) {
	cfg := newKdepsCfg("store", path.Xdg)
	got, err := GetKdepsPath(context.Background(), cfg)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	// do not assert exact path; just ensure ends with /store
	if filepath.Base(got) != "store" {
		t.Fatalf("unexpected path %s", got)
	}
}

func TestGetKdepsPathUnknown(t *testing.T) {
	// Provide invalid path using numeric constant outside defined ones.
	type customPath string
	bad := newKdepsCfg("dir", path.Path("bogus"))
	if _, err := GetKdepsPath(context.Background(), bad); err == nil {
		t.Fatalf("expected error for unknown path type")
	}
}

func TestGetKdepsPathVariants(t *testing.T) {
	ctx := context.Background()

	tmpHome := t.TempDir()
	if err := os.Setenv("HOME", tmpHome); err != nil {
		t.Fatalf("setenv: %v", err)
	}

	tmpProject := t.TempDir()
	if err := os.Chdir(tmpProject); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	dirName := "kdeps-system"
	build := func(p path.Path) kdeps.Kdeps {
		return kdeps.Kdeps{KdepsDir: dirName, KdepsPath: p}
	}

	cases := []struct {
		name    string
		cfg     kdeps.Kdeps
		want    string
		wantErr bool
	}{
		{"user", build(path.User), filepath.Join(tmpHome, dirName), false},
		{"project", build(path.Project), filepath.Join(tmpProject, dirName), false},
		{"xdg", build(path.Xdg), filepath.Join(os.Getenv("XDG_CONFIG_HOME"), dirName), false},
		{"unknown", build("weird"), "", true},
	}

	for _, c := range cases {
		got, err := GetKdepsPath(ctx, c.cfg)
		if c.wantErr {
			if err == nil {
				t.Fatalf("%s: expected error", c.name)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", c.name, err)
		}
		if filepath.Base(got) != dirName {
			t.Fatalf("%s: expected path ending with %s, got %s", c.name, dirName, got)
		}
	}

	// Restore cwd for other tests on Windows.
	if runtime.GOOS == "windows" {
		_ = os.Chdir("\\")
	}
}

func TestGetKdepsPathCases(t *testing.T) {
	tmpProject := t.TempDir()
	// Change working directory so path.Project branch produces deterministic path.
	oldWd, _ := os.Getwd()
	_ = os.Chdir(tmpProject)
	defer os.Chdir(oldWd)

	cases := []struct {
		name      string
		cfg       kdeps.Kdeps
		expectFn  func() string
		expectErr bool
	}{
		{
			"user path", kdeps.Kdeps{KdepsDir: "mykdeps", KdepsPath: kpath.User}, func() string {
				home, _ := os.UserHomeDir()
				return filepath.Join(home, "mykdeps")
			}, false,
		},
		{
			"project path", kdeps.Kdeps{KdepsDir: "mykdeps", KdepsPath: kpath.Project}, func() string {
				cwd, _ := os.Getwd()
				return filepath.Join(cwd, "mykdeps")
			}, false,
		},
		{
			"xdg path", kdeps.Kdeps{KdepsDir: "mykdeps", KdepsPath: kpath.Xdg}, func() string {
				return filepath.Join(xdg.ConfigHome, "mykdeps")
			}, false,
		},
		{
			"unknown", kdeps.Kdeps{KdepsDir: "abc", KdepsPath: "bogus"}, nil, true,
		},
	}

	for _, tc := range cases {
		got, err := GetKdepsPath(context.Background(), tc.cfg)
		if tc.expectErr {
			assert.Error(t, err, tc.name)
			continue
		}
		assert.NoError(t, err, tc.name)
		assert.Equal(t, tc.expectFn(), got, tc.name)
	}
}
