// Code generated from Pkl module `org.kdeps.pkl.API`. DO NOT EDIT.
package api

import (
	"context"

	"github.com/apple/pkl-go/pkl"
)

type API struct {
}

// LoadFromPath loads the pkl module at the given path and evaluates it into a API
func LoadFromPath(ctx context.Context, path string) (ret *API, err error) {
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

// Load loads the pkl module at the given source and evaluates it with the given evaluator into a API
func Load(ctx context.Context, evaluator pkl.Evaluator, source *pkl.ModuleSource) (*API, error) {
	var ret API
	if err := evaluator.EvaluateModule(ctx, source, &ret); err != nil {
		return nil, err
	}
	return &ret, nil
}
