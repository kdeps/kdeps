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
	"fmt"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

// stagePermission decides whether a primary resource kind (exec, python,
// httpClient, ...) may run in this process. The agent package installs it at
// startup, so it applies to every run. Nil allows every stage.
//
//nolint:gochecknoglobals // set once at startup by the agent package
var stagePermission func(kind string) error

// SetStagePermission installs the check every resource runs through.
func SetStagePermission(check func(kind string) error) {
	stagePermission = check
}

// checkStagePermission refuses a resource whose primary kind the installed
// check denies. Response blocks (apiResponse) are not primary kinds and pass.
func checkStagePermission(resource *domain.Resource) error {
	if stagePermission == nil {
		return nil
	}
	for _, entry := range domain.ResourceExecCatalog() {
		if entry.ResponseBlock || !entry.PresentResource(resource) {
			continue
		}
		if err := stagePermission(entry.Name); err != nil {
			return fmt.Errorf("stage %s: %w", resource.ActionID, err)
		}
	}
	return nil
}
