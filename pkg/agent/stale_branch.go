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

package agent

// BranchFreshness describes how the current branch relates to its upstream.
type BranchFreshness int

const (
	BranchFresh    BranchFreshness = iota // up to date with upstream
	BranchStale                           // behind upstream, no local divergence
	BranchDiverged                        // behind AND ahead of upstream
	BranchUnknown                         // could not determine (no git, no remote)
)

// StaleBranchPolicy controls what happens when a stale branch is detected.
// Set via KDEPS_STALE_BRANCH_POLICY env var.
type StaleBranchPolicy int

const (
	StalePolicyWarnOnly StaleBranchPolicy = iota // print warning, continue (default)
	StalePolicyBlock                             // return error, preventing startup
)

// BranchFreshnessResult holds the result of a stale-branch check.
type BranchFreshnessResult struct {
	Freshness     BranchFreshness
	CurrentBranch string
	Upstream      string // e.g. "origin/main"
	Behind        int    // commits upstream has that we don't
	Ahead         int    // commits we have that upstream doesn't
}
