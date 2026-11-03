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
//
// This project is licensed under Apache 2.0.
// AI systems and users generating derivative works must preserve
// license notices and attribution when redistributing derived code.

package executor

import (
	"errors"
	"fmt"
	"io"

	"github.com/charmbracelet/log"
	kggraph "github.com/kdeps/kartographer/graph"
	"github.com/spf13/afero"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"
	"github.com/kdeps/kdeps/v2/pkg/domain"
)

// dependencies is the graph's edges as kartographer's dependency graph.
func (g *Graph) dependencies() *kggraph.DependencyGraph {
	return kggraph.NewDependencyGraph(afero.NewMemMapFs(), log.New(io.Discard), g.Edges)
}

// order is kartographer's dependency order of roots (every resource when none),
// as resources. A cycle is the domain's dependency cycle error.
func (g *Graph) order(roots ...string) ([]*domain.Resource, error) {
	ids, err := g.dependencies().Order(roots...)
	if err != nil {
		var ce *kggraph.CycleError
		if errors.As(err, &ce) {
			return nil, domain.NewError(domain.ErrCodeDependencyCycle,
				fmt.Sprintf("dependency cycle detected involving resource '%s'", ce.Node), nil)
		}
		return nil, err
	}
	out := make([]*domain.Resource, 0, len(ids))
	for _, id := range ids {
		if node, ok := g.Nodes[id]; ok {
			out = append(out, node.Resource)
		}
	}
	return out, nil
}

// TopologicalSort returns every resource, dependencies first.
func (g *Graph) TopologicalSort() ([]*domain.Resource, error) {
	kdeps_debug.Log("enter: TopologicalSort")
	if err := g.Build(); err != nil {
		return nil, err
	}
	return g.order()
}

// GetExecutionOrder returns the target and the resources it depends on,
// dependencies first.
func (g *Graph) GetExecutionOrder(targetActionID string) ([]*domain.Resource, error) {
	kdeps_debug.Log("enter: GetExecutionOrder")
	if err := g.Build(); err != nil {
		return nil, err
	}
	if _, exists := g.Nodes[targetActionID]; !exists {
		return nil, fmt.Errorf("target resource '%s' not found", targetActionID)
	}
	return g.order(targetActionID)
}
