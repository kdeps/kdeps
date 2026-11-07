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

package parallel

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// peak runs n tasks that each hold until want of them overlap (or 2s) and
// returns the most that ran at once.
func peak(t *testing.T, n, limit, want int, deps [][]int) (int, []int) {
	t.Helper()
	var mu sync.Mutex
	active, top := 0, 0
	var order []int
	err := RunN(n, limit, deps, func(i int) error {
		mu.Lock()
		active++
		top = max(top, active)
		order = append(order, i)
		mu.Unlock()
		for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); {
			mu.Lock()
			reached := top >= want
			mu.Unlock()
			if reached {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		mu.Lock()
		active--
		mu.Unlock()
		return nil
	})
	require.NoError(t, err)
	return top, order
}

func TestRunN_IndependentOverlapUpToLimit(t *testing.T) {
	top, _ := peak(t, 5, 3, 3, nil)
	assert.Equal(t, 3, top)
}

func TestRunN_DepsWait(t *testing.T) {
	top, order := peak(t, 3, 4, 2, [][]int{nil, nil, {0, 1}})
	assert.Equal(t, 2, top)
	assert.Equal(t, 2, order[2])
}

func TestRunN_LimitOneRunsInOrder(t *testing.T) {
	top, order := peak(t, 3, 1, 1, nil)
	assert.Equal(t, 1, top)
	assert.Equal(t, []int{0, 1, 2}, order)
}

func TestRunN_EarliestErrorStopsNewStarts(t *testing.T) {
	var mu sync.Mutex
	ran := map[int]bool{}
	err := RunN(4, 2, [][]int{nil, nil, {0, 1}, {0, 1}}, func(i int) error {
		mu.Lock()
		ran[i] = true
		mu.Unlock()
		if i == 0 || i == 1 {
			return errors.New("fail " + string(rune('0'+i)))
		}
		return nil
	})
	require.EqualError(t, err, "fail 0")
	assert.False(t, ran[2])
	assert.False(t, ran[3])
}

func TestRunN_SerialError(t *testing.T) {
	calls := 0
	err := RunN(3, 1, nil, func(i int) error {
		calls++
		if i == 1 {
			return errors.New("stop")
		}
		return nil
	})
	require.EqualError(t, err, "stop")
	assert.Equal(t, 2, calls)
}

func TestRunN_PanicReachesCaller(t *testing.T) {
	assert.PanicsWithValue(t, "boom", func() {
		_ = RunN(2, 2, nil, func(i int) error {
			if i == 0 {
				panic("boom")
			}
			return nil
		})
	})
}

func TestLimit(t *testing.T) {
	prev := Limit()
	t.Cleanup(func() { SetLimit(prev) })
	SetLimit(0)
	assert.Equal(t, defaultLimit, Limit())
	SetLimit(7)
	assert.Equal(t, 7, Limit())
	assert.NoError(t, Run(0, nil, func(int) error { return nil }))
}
