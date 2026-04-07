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

# E2E tests for examples/auto-env
# Tests structure, validation, and auto-env scoping declarations.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing auto-env example..."

AE_DIR="$PROJECT_ROOT/examples/auto-env"
AE_WF="$AE_DIR/workflow.yaml"
AE_TRANS="$AE_DIR/components/translator/component.yaml"
AE_SUM="$AE_DIR/components/summarizer/component.yaml"

if [ ! -f "$AE_WF" ]; then
    test_skipped "auto-env (workflow.yaml not found)"
else
    # Test 1: workflow.yaml exists
    test_passed "auto-env - workflow.yaml exists"

    # Test 2: translator component exists
    if [ -f "$AE_TRANS" ]; then
        test_passed "auto-env - components/translator/component.yaml exists"
    else
        test_failed "auto-env - components/translator/component.yaml exists" "File not found: $AE_TRANS"
    fi

    # Test 3: summarizer component exists
    if [ -f "$AE_SUM" ]; then
        test_passed "auto-env - components/summarizer/component.yaml exists"
    else
        test_failed "auto-env - components/summarizer/component.yaml exists" "File not found: $AE_SUM"
    fi

    # Test 4: all resource files exist
    if [ -f "$AE_DIR/resources/01-translate.yaml" ] && \
       [ -f "$AE_DIR/resources/02-summarize.yaml" ] && \
       [ -f "$AE_DIR/resources/03-response.yaml" ]; then
        test_passed "auto-env - all 3 resource files exist"
    else
        test_failed "auto-env - all 3 resource files exist" "One or more resource files missing in $AE_DIR/resources/"
    fi

    # Test 5: README.md exists
    if [ -f "$AE_DIR/README.md" ]; then
        test_passed "auto-env - README.md exists"
    else
        test_failed "auto-env - README.md exists" "File not found: $AE_DIR/README.md"
    fi

    # Test 6: translator component has kind: Component
    if grep -q "kind: Component" "$AE_TRANS" 2>/dev/null; then
        test_passed "auto-env - translator has kind: Component"
    else
        test_failed "auto-env - translator has kind: Component" "kind: Component not found in $AE_TRANS"
    fi

    # Test 7: summarizer component has kind: Component
    if grep -q "kind: Component" "$AE_SUM" 2>/dev/null; then
        test_passed "auto-env - summarizer has kind: Component"
    else
        test_failed "auto-env - summarizer has kind: Component" "kind: Component not found in $AE_SUM"
    fi

    # Test 8: translator uses env('OPENAI_API_KEY') — demonstrates scoped override
    if grep -q "env('OPENAI_API_KEY')" "$AE_TRANS" 2>/dev/null; then
        test_passed "auto-env - translator uses env('OPENAI_API_KEY')"
    else
        test_failed "auto-env - translator uses env('OPENAI_API_KEY')" "env('OPENAI_API_KEY') not found in $AE_TRANS"
    fi

    # Test 9: summarizer uses env('OPENAI_API_KEY')
    if grep -q "env('OPENAI_API_KEY')" "$AE_SUM" 2>/dev/null; then
        test_passed "auto-env - summarizer uses env('OPENAI_API_KEY')"
    else
        test_failed "auto-env - summarizer uses env('OPENAI_API_KEY')" "env('OPENAI_API_KEY') not found in $AE_SUM"
    fi

    # Test 10: translate resource calls translator component
    if grep -q "name: translator" "$AE_DIR/resources/01-translate.yaml" 2>/dev/null; then
        test_passed "auto-env - translate resource calls translator component"
    else
        test_failed "auto-env - translate resource calls translator component" "name: translator not found in 01-translate.yaml"
    fi

    # Test 11: summarize resource calls summarizer component
    if grep -q "name: summarizer" "$AE_DIR/resources/02-summarize.yaml" 2>/dev/null; then
        test_passed "auto-env - summarize resource calls summarizer component"
    else
        test_failed "auto-env - summarize resource calls summarizer component" "name: summarizer not found in 02-summarize.yaml"
    fi

    # Test 12: response requires both translate and summarize
    if grep -q "translate" "$AE_DIR/resources/03-response.yaml" 2>/dev/null && \
       grep -q "summarize" "$AE_DIR/resources/03-response.yaml" 2>/dev/null; then
        test_passed "auto-env - response requires translate and summarize"
    else
        test_failed "auto-env - response requires translate and summarize" "translate or summarize not referenced in 03-response.yaml"
    fi

    # Test 13: README documents scoped env var naming convention
    if grep -q "TRANSLATOR_OPENAI_API_KEY" "$AE_DIR/README.md" 2>/dev/null; then
        test_passed "auto-env - README documents TRANSLATOR_OPENAI_API_KEY scoped override"
    else
        test_failed "auto-env - README documents TRANSLATOR_OPENAI_API_KEY" "TRANSLATOR_OPENAI_API_KEY not found in README.md"
    fi

    # Test 14: workflow validates
    if "$KDEPS_BIN" validate "$AE_WF" &>/dev/null; then
        test_passed "auto-env - workflow validates"
    else
        test_failed "auto-env - workflow validates" "Validation failed for $AE_WF"
    fi
fi

echo ""
