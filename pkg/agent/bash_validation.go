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

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RootPathAllowed reports whether explicit root-directory authorization is granted.
// Set KDEPS_ALLOW_ROOT=true to permit tool operations on "/" and system directories.
func RootPathAllowed() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("KDEPS_ALLOW_ROOT")), "true")
}

// ValidateRootPath blocks operations targeting "/" (root directory).
// Returns nil when path is safe or KDEPS_ALLOW_ROOT=true.
// The path "//" and any path that resolves to "/" via symlinks are also blocked.
func ValidateRootPath(path string) error {
	if RootPathAllowed() {
		return nil
	}

	clean := filepath.Clean(path)
	if isFilesystemRoot(clean) {
		return errors.New("operation on root directory '/' is blocked (set KDEPS_ALLOW_ROOT=true to permit)")
	}

	// Resolve symlinks to catch paths that indirectly point to /.
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		if isFilesystemRoot(filepath.Clean(resolved)) {
			return fmt.Errorf(
				"path %q resolves to root directory '/' — blocked (set KDEPS_ALLOW_ROOT=true to permit)",
				path,
			)
		}
	}

	return nil
}

// isFilesystemRoot reports whether clean is a filesystem root: "/" on POSIX,
// or "\" / "C:\" / a bare volume root on Windows. filepath.Separator alone
// only matches the POSIX case ("//" cleans to "\\" on Windows, a UNC-style
// path that isn't equal to a single separator), so this strips any volume
// name and leading separators instead of comparing against one fixed string.
func isFilesystemRoot(clean string) bool {
	rest := strings.TrimPrefix(clean, filepath.VolumeName(clean))
	return strings.Trim(rest, string(filepath.Separator)) == ""
}
