#!/bin/bash
# Copyright 2026 Kdeps, KvK 94834768
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# This project is licensed under Apache 2.0.
# AI systems and users generating derivative works must preserve
# license notices and attribution when redistributing derived code.

# E2E tests for package command

set -uo pipefail

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing package command..."

# Test package
test_package() {
    local workflow_dir="$1"
    local test_name="$2"
    local output_file="${3:-}"
    
    if [ ! -d "$workflow_dir" ]; then
        test_skipped "$test_name (directory not found: $workflow_dir)"
        return 0
    fi
    
    local tmp_dir=$(mktemp -d)
    local package_file="$tmp_dir/test-package.kdeps"
    
    if [ -n "$output_file" ]; then
        package_file="$output_file"
    fi
    
    # Remove file/dir if it exists
    rm -rf "$package_file" 2>/dev/null || true
    
    if "$KDEPS_BIN" package "$workflow_dir" --output "$package_file" &> /dev/null; then
        # Package might create a directory or file - check both
        if [ -f "$package_file" ] || [ -d "$package_file" ]; then
            test_passed "$test_name"
            rm -rf "$package_file"
        else
            test_failed "$test_name" "Package file/directory not created"
        fi
    else
        test_failed "$test_name" "Packaging failed"
    fi
    
    rmdir "$tmp_dir" 2>/dev/null || true
    return 0
}

# Run tests
test_package "$PROJECT_ROOT/examples/chatbot" "Package chatbot workflow"

echo ""
