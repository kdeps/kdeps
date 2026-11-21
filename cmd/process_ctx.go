package cmd

import "context"

// processRootCtx is the process's one root context; context.Background is
// only created here. Every command's work derives from it, directly or through
// the cobra command's context.
//
//nolint:gochecknoglobals // the single process root
var processRootCtx = context.Background()

// processRoot returns the process root context.
func processRoot() context.Context { return processRootCtx }

// processParent returns parent, or the process root when the command has none.
func processParent(parent context.Context) context.Context {
	if parent == nil {
		return processRoot()
	}
	return parent
}
