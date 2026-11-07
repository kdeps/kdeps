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

// Package parallel is the one scheduler both modes use to run independent
// work at once: workflow resources, component calls, subagents, judges and
// agent tool calls.
package parallel

import "sync/atomic"

// defaultLimit is used until the "parallel" event asset sets the cap.
const defaultLimit = 4

//nolint:gochecknoglobals // process-wide cap, set once from the event registry
var limit atomic.Int64

// SetLimit sets how many tasks run at once. 1 runs them one at a time.
func SetLimit(n int) { limit.Store(int64(n)) }

// Limit is the cap in effect.
func Limit() int {
	if n := limit.Load(); n > 0 {
		return int(n)
	}
	return defaultLimit
}

type result struct {
	index int
	err   error
	panic any
}

type scheduler struct {
	run      func(i int) error
	deps     [][]int
	limit    int
	done     []bool
	started  []bool
	running  int
	results  chan result
	failed   int
	failure  error
	panicked any
}

func (s *scheduler) ready(i int) bool {
	if i >= len(s.deps) {
		return true
	}
	for _, j := range s.deps[i] {
		if !s.done[j] {
			return false
		}
	}
	return true
}

func (s *scheduler) launch() {
	for i := range s.done {
		if s.running >= s.limit {
			return
		}
		if s.started[i] || !s.ready(i) {
			continue
		}
		s.started[i] = true
		s.running++
		go s.runOne(i)
	}
}

func (s *scheduler) runOne(i int) {
	res := result{index: i}
	defer func() {
		if p := recover(); p != nil {
			res.panic = p
		}
		s.results <- res
	}()
	res.err = s.run(i)
}

func (s *scheduler) record(res result) {
	s.running--
	s.done[res.index] = true
	switch {
	case res.panic != nil:
		if s.panicked == nil {
			s.panicked = res.panic
		}
	case res.err != nil && (s.failed < 0 || res.index < s.failed):
		s.failed, s.failure = res.index, res.err
	}
}

// Run runs tasks 0..n-1 with run, starting each once the earlier tasks
// deps[i] names have finished (a nil or short deps means none), at most
// Limit() at once. After the first error or panic it starts nothing new and
// waits for the running tasks. It returns the error of the earliest task in
// order and re-panics a task's panic on the caller's goroutine.
func Run(n int, deps [][]int, run func(i int) error) error {
	return RunN(n, Limit(), deps, run)
}

// RunN is Run with an explicit cap. A cap of 1 or less runs the tasks in order
// on the caller's goroutine.
func RunN(n, limit int, deps [][]int, run func(i int) error) error {
	if limit <= 1 || n <= 1 {
		for i := range n {
			if err := run(i); err != nil {
				return err
			}
		}
		return nil
	}
	s := &scheduler{
		run:     run,
		deps:    deps,
		limit:   limit,
		done:    make([]bool, n),
		started: make([]bool, n),
		results: make(chan result),
		failed:  -1,
	}
	s.launch()
	for s.running > 0 {
		s.record(<-s.results)
		if s.failed < 0 && s.panicked == nil {
			s.launch()
		}
	}
	if s.panicked != nil {
		panic(s.panicked)
	}
	return s.failure
}
