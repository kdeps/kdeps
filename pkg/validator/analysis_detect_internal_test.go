package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func TestBuildActionIDIndex(t *testing.T) {
	t.Parallel()
	wf := &domain.Workflow{
		Resources: []*domain.Resource{
			{ActionID: "step1"},
			{ActionID: "step2"},
		},
	}
	idx := buildActionIDIndex(wf)
	assert.True(t, idx["step1"])
	assert.True(t, idx["step2"])
	assert.False(t, idx["step3"])
}

func TestBuildComponentNameIndex(t *testing.T) {
	t.Parallel()
	wf := &domain.Workflow{
		Components: map[string]*domain.Component{
			"myComp": {},
			"other":  {},
		},
	}
	idx := buildComponentNameIndex(wf)
	assert.True(t, idx["myComp"])
	assert.True(t, idx["other"])
	assert.False(t, idx["missing"])
}

func TestDetectUnreachable_NoTarget(t *testing.T) {
	t.Parallel()
	wf := &domain.Workflow{
		Resources: []*domain.Resource{{ActionID: "a"}},
	}
	issues := detectUnreachable(wf)
	assert.Empty(t, issues)
}

func TestDetectUnreachable_AllReachable(t *testing.T) {
	t.Parallel()
	wf := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{TargetActionID: "b"},
		Resources: []*domain.Resource{
			{ActionID: "b", Requires: []string{"a"}},
			{ActionID: "a"},
		},
	}
	issues := detectUnreachable(wf)
	assert.Empty(t, issues)
}

func TestDetectUnreachable_HasUnreachable(t *testing.T) {
	t.Parallel()
	wf := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{TargetActionID: "a"},
		Resources: []*domain.Resource{
			{ActionID: "a"},
			{ActionID: "orphan"},
		},
	}
	issues := detectUnreachable(wf)
	assert.Len(t, issues, 1)
	assert.Equal(t, "orphan", issues[0].ActionID)
}

func TestIsKnownActionOrComponent(t *testing.T) {
	t.Parallel()
	actions := map[string]bool{"step1": true}
	comps := map[string]bool{"comp1": true}
	assert.True(t, isKnownActionOrComponent("step1", actions, comps))
	assert.True(t, isKnownActionOrComponent("comp1", actions, comps))
	assert.False(t, isKnownActionOrComponent("unknown", actions, comps))
}

func TestDetectMissingComponentInputs_None(t *testing.T) {
	t.Parallel()
	wf := &domain.Workflow{}
	issues := detectMissingComponentInputs(wf)
	assert.Empty(t, issues)
}
