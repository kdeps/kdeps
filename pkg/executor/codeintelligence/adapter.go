//go:build !js

package codeintelligence

import (
	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
)

// Adapter adapts the codeIntelligence Executor to the ResourceExecutor interface.
type Adapter = executor.TypedAdapter[domain.CodeIntelligenceConfig]

// NewAdapter creates a new codeIntelligence executor adapter.
func NewAdapter() *Adapter {
	return executor.NewTypedAdapter[domain.CodeIntelligenceConfig]("codeIntelligence", NewExecutor())
}
