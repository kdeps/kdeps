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

package selftest_test

import (
	"context"
	stdhttp "net/http"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v3/pkg/domain"
	"github.com/kdeps/kdeps/v3/pkg/judge"
	"github.com/kdeps/kdeps/v3/pkg/selftest"
)

// flaky answers "good" on odd calls and "bad" on even ones.
func flaky() stdhttp.Handler {
	var n atomic.Int32
	return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		if n.Add(1)%2 == 1 {
			_, _ = w.Write([]byte("good"))
			return
		}
		_, _ = w.Write([]byte("bad"))
	})
}

func graded(repeat int, threshold float64) domain.TestCase {
	return domain.TestCase{
		Name:      "graded",
		Request:   domain.TestRequest{Path: "/"},
		Judges:    []domain.Judge{{Name: "word", Graders: []domain.JudgeGrader{{Kind: "contains", Pattern: "good"}}}},
		Repeat:    repeat,
		Threshold: threshold,
	}
}

func TestEvals_RepeatAndThreshold(t *testing.T) {
	res := runCases(t, flaky(), []domain.TestCase{graded(4, 0.5)}, selftest.Options{})[0]
	assert.True(t, res.Passed(), "2 of 4 runs reach a 0.5 threshold")
	assert.Equal(t, 4, res.Runs)
	assert.Equal(t, 2, res.Passes)

	res = runCases(t, flaky(), []domain.TestCase{graded(4, 0.75)}, selftest.Options{})[0]
	require.Error(t, res.Err)
	assert.Contains(t, res.Err.Error(), "passed 2/4 runs, threshold 0.75")
	assert.Contains(t, res.Err.Error(), `judge word: contains: missing "good"`)

	res = runCases(t, flaky(), []domain.TestCase{graded(2, 0)}, selftest.Options{})[0]
	require.Error(t, res.Err, "the default threshold needs every run")

	res = runCases(t, jsonHandler("bad"), []domain.TestCase{graded(0, 0)}, selftest.Options{})[0]
	require.Error(t, res.Err)
	assert.Equal(t, 1, res.Runs, "repeat defaults to one run")
	assert.Contains(t, res.Err.Error(), "judge word")
}

func TestEvals_CriteriaJudgesUseReview(t *testing.T) {
	tc := domain.TestCase{
		Name:    "criteria",
		Request: domain.TestRequest{Path: "/"},
		Judges:  []domain.Judge{{Name: "tone", Criteria: "polite"}},
	}
	res := runCases(t, jsonHandler("hi"), []domain.TestCase{tc}, selftest.Options{})[0]
	require.Error(t, res.Err)
	assert.Contains(t, res.Err.Error(), "no judge model")

	var seen string
	review := func(_ context.Context, got domain.TestCase, spec domain.Judge, output string) judge.Verdict {
		seen = got.Name + "/" + spec.Name + "/" + output
		return judge.Verdict{Name: spec.Name, Approved: output == "hi"}
	}
	res = runCases(t, jsonHandler("hi"), []domain.TestCase{tc}, selftest.Options{Review: review})[0]
	require.NoError(t, res.Err)
	assert.Equal(t, "criteria/tone/hi", seen)

	res = runCases(t, jsonHandler("rude"), []domain.TestCase{tc}, selftest.Options{Review: review})[0]
	require.Error(t, res.Err)
}
