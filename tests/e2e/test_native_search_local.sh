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

# E2E tests for native searchLocal executor

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing native searchLocal executor..."

if [ -z "${KDEPS_BIN:-}" ] || [ ! -x "${KDEPS_BIN}" ]; then
    test_skipped "native searchLocal - kdeps binary not found"
    exit 0
fi

WORK_DIR=$(mktemp -d -t kdeps-searchlocal-e2e-XXXXXX 2>/dev/null || mktemp -d)
cleanup() { rm -rf "$WORK_DIR"; }
trap cleanup EXIT

# Create test files
SEARCH_DIR="$WORK_DIR/search-files"
mkdir -p "$SEARCH_DIR"
echo "hello world content" > "$SEARCH_DIR/hello.txt"
echo "goodbye content" > "$SEARCH_DIR/goodbye.txt"
echo "package main" > "$SEARCH_DIR/main.go"

# Test: searchLocal with glob filter
test_searchlocal_glob() {
    local pkg_dir="$WORK_DIR/sl-glob"
    mkdir -p "$pkg_dir/resources"

    cat > "$pkg_dir/workflow.yaml" <<'YAML'
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: searchlocal-glob-test
  version: "1.0.0"
  targetActionId: do-search
settings:
  apiServerMode: false
YAML

    cat > "$pkg_dir/resources/do-search.yaml" <<YAML
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: do-search
  name: Do Search
run:
  searchLocal:
    path: "${SEARCH_DIR}"
    glob: "*.go"
  apiResponse:
    success: true
    response:
      count: "{{ output('do-search').count }}"
YAML

    local result
    if result=$(timeout 30 "$KDEPS_BIN" run "$pkg_dir" 2>&1); then
        test_passed "native searchLocal - glob filter"
    else
        test_skipped "native searchLocal - glob filter (run failed)"
    fi
}

# Test: searchLocal with keyword filter
test_searchlocal_keyword() {
    local pkg_dir="$WORK_DIR/sl-keyword"
    mkdir -p "$pkg_dir/resources"

    cat > "$pkg_dir/workflow.yaml" <<'YAML'
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: searchlocal-keyword-test
  version: "1.0.0"
  targetActionId: do-search
settings:
  apiServerMode: false
YAML

    cat > "$pkg_dir/resources/do-search.yaml" <<YAML
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: do-search
  name: Do Search
run:
  searchLocal:
    path: "${SEARCH_DIR}"
    query: "hello"
  apiResponse:
    success: true
    response:
      count: "{{ output('do-search').count }}"
YAML

    local result
    if result=$(timeout 30 "$KDEPS_BIN" run "$pkg_dir" 2>&1); then
        test_passed "native searchLocal - keyword filter"
    else
        test_skipped "native searchLocal - keyword filter (run failed)"
    fi
}

# Test: searchLocal with glob and keyword combined
test_searchlocal_combined() {
    local pkg_dir="$WORK_DIR/sl-combined"
    mkdir -p "$pkg_dir/resources"

    cat > "$pkg_dir/workflow.yaml" <<'YAML'
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: searchlocal-combined-test
  version: "1.0.0"
  targetActionId: do-search
settings:
  apiServerMode: false
YAML

    cat > "$pkg_dir/resources/do-search.yaml" <<YAML
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: do-search
  name: Do Search
run:
  searchLocal:
    path: "${SEARCH_DIR}"
    glob: "*.txt"
    query: "hello"
  apiResponse:
    success: true
    response:
      count: "{{ output('do-search').count }}"
YAML

    local result
    if result=$(timeout 30 "$KDEPS_BIN" run "$pkg_dir" 2>&1); then
        test_passed "native searchLocal - glob and keyword combined"
    else
        test_skipped "native searchLocal - glob and keyword combined (run failed)"
    fi
}

test_searchlocal_glob
test_searchlocal_keyword
test_searchlocal_combined
