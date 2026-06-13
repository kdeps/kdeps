//go:build !js

package git

import (
	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
)

// Adapter adapts the git Executor to the ResourceExecutor interface.
type Adapter = executor.TypedAdapter[domain.GitResourceConfig]

// NewAdapter creates a new git executor adapter.
func NewAdapter() *Adapter {
	return executor.NewTypedAdapter[domain.GitResourceConfig]("git", NewExecutor())
}
