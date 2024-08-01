// Code generated from Pkl module `org.kdeps.pkl.Workflow`. DO NOT EDIT.
package workflow

import (
	"context"

	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/gen/settings"
)

type Workflow struct {
	Settings *settings.AppSettings `pkl:"settings"`

	Workflows []string `pkl:"workflows"`
}

// LoadFromPath loads the pkl module at the given path and evaluates it into a Workflow
func LoadFromPath(ctx context.Context, path string) (ret *Workflow, err error) {
	evaluator, err := pkl.NewEvaluator(ctx, pkl.PreconfiguredOptions)
	if err != nil {
		return nil, err
	}
	defer func() {
		cerr := evaluator.Close()
		if err == nil {
			err = cerr
		}
	}()
	ret, err = Load(ctx, evaluator, pkl.FileSource(path))
	return ret, err
}

// Load loads the pkl module at the given source and evaluates it with the given evaluator into a Workflow
func Load(ctx context.Context, evaluator pkl.Evaluator, source *pkl.ModuleSource) (*Workflow, error) {
	var ret Workflow
	if err := evaluator.EvaluateModule(ctx, source, &ret); err != nil {
		return nil, err
	}
	return &ret, nil
}
