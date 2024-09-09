package evaluator

import (
	"context"
	"errors"
	"os/exec"

	"github.com/alexellis/go-execute/v2"
	"github.com/spf13/afero"
)

var schemaVersionFilePath = "../../SCHEMA_VERSION"

func FindPklBinary() {
	binaryName := "pkl"
	if _, err := exec.LookPath(binaryName); err != nil {
		panic("The binary 'pkl' does not exist in PATH. For more information, see: https://pkl-lang.org")
	}
}

func EvalPkl(fs afero.Fs, resourcePath string) (string, error) {
	FindPklBinary()

	cmd := execute.ExecTask{
		Command:     "pkl",
		Args:        []string{"eval", resourcePath},
		StreamStdio: false,
	}

	res, err := cmd.Execute(context.Background())
	if err != nil {
		return "", err
	}

	if res.ExitCode != 0 {
		return "", errors.New("Non-zero exit code: " + res.Stderr)
	}

	return res.Stdout, nil
}
