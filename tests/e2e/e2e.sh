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

# Main E2E test runner - orchestrates all test scenarios

set -uo pipefail

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "=========================================="
echo "KDeps E2E Test Suite"
echo "=========================================="
echo -e "${GREEN}Using kdeps binary: $KDEPS_BIN${NC}"
echo ""

# Every workflow run here merges the Python packages of the example
# components (KDEPS_COMPONENT_DIR) into one shared venv. Build it once, up
# front and without a test's short timeout: a build killed half-way by a
# server test's timeout is rebuilt by the next run, which a slow runner
# (Windows) can then also kill, and so on.
_warm=$(mktemp -d)
mkdir -p "$_warm/resources"
printf 'apiVersion: kdeps.io/v1\nkind: Workflow\nmetadata:\n  name: venv-warmup\n  version: "1.0.0"\n  targetActionId: hi\nsettings:\n  agentSettings:\n    pythonVersion: "3.12"\n' \
    > "$_warm/workflow.yaml"
printf 'actionId: hi\nname: Hi\ncomponent:\n  name: exec\n  with:\n    command: echo hi\n' > "$_warm/resources/hi.yaml"
echo "Building the shared Python environment once (up to 15 minutes)..."
if timeout 900 "$KDEPS_BIN" run "$_warm/workflow.yaml" >"$_warm/out.log" 2>&1; then
    grep -E "Python venv|Packages" "$_warm/out.log" || true
else
    echo "  (warm-up did not finish; runs build the environment as they need it)"
    tail -5 "$_warm/out.log"
fi
rm -rf "$_warm"
unset _warm
echo ""

# E2E_SHARD=k/n runs every n-th scenario starting at the k-th, so CI can
# split the suite across runners. Unset runs them all.
_scenario_idx=0
scenario() {
    local i=$_scenario_idx
    _scenario_idx=$((i + 1))
    if [ -n "${E2E_SHARD:-}" ] && [ $((i % ${E2E_SHARD#*/})) -ne $((${E2E_SHARD%/*} - 1)) ]; then
        return 0
    fi
    # shellcheck disable=SC1090
    source "$SCRIPT_DIR/$1"
}

# Run all test scenarios (source them so counters persist)
scenario test_e2e_helpers.sh
scenario test_validation.sh
scenario test_scaffolding.sh
scenario test_packaging.sh
scenario test_prepackage.sh
scenario test_build.sh
scenario test_docker_versions.sh
scenario test_docker_services.sh
# source "$SCRIPT_DIR/test_container_behavior.sh"
# source "$SCRIPT_DIR/test_container_multiarch.sh"
scenario test_execution.sh
scenario test_examples.sh

# Run feature tests
echo "Testing additional features..."
scenario test_features_file_upload.sh
scenario test_features_input_validation.sh
scenario test_features_session.sh
scenario test_features_error_handling.sh
scenario test_features_cors.sh
scenario test_features_health_check.sh
scenario test_features_api_auth.sh
scenario test_features_management_api.sh
scenario test_features_kdeps_api.sh
scenario test_features_memory_storage.sh
scenario test_features_python_executor.sh
scenario test_features_multi_resource.sh
scenario test_features_expression_eval.sh
scenario test_features_items_iteration.sh
scenario test_features_loop_iteration.sh
scenario test_features_workflow_metadata.sh
scenario test_features_route_methods.sh
scenario test_http_client.sh
scenario test_features_edge_cases.sh
scenario test_features_input_sources.sh
scenario test_features_bot_sources.sh
scenario test_features_tts.sh
scenario test_features_git.sh
scenario test_features_file_component.sh
scenario test_features_whatsapp_template.sh
scenario test_features_embedding.sh
scenario test_features_exec.sh
scenario test_features_calendar.sh
scenario test_features_search_local.sh
scenario test_features_search_local_graph_boost.sh
scenario test_features_memory.sh
scenario test_features_embedding_upsert.sh
scenario test_features_events.sh
scenario test_features_autopilot.sh
scenario test_features_instrument.sh
scenario test_features_search_local_executor.sh
scenario test_embedding_component.sh
scenario test_loader_component.sh
scenario test_vector_store_component.sh
scenario test_native_scraper.sh

# CLI subcommand E2E tests
scenario test_bundle_export.sh
scenario test_export_k8s.sh
scenario test_chat_cmd.sh
scenario test_chat_cmd_repl.sh
scenario test_agent_theme.sh
scenario test_agent_model_name.sh
scenario test_agent_paste.sh
scenario test_update.sh
scenario test_assets.sh
scenario test_asset_profiles.sh
scenario test_asset_sets.sh
scenario test_assets_defensive.sh
scenario test_knowledge.sh
scenario test_knowledge_session.sh
scenario test_knowledge_defensive.sh
scenario test_knowledge_folders.sh
scenario test_knowledge_git.sh
scenario test_declared_tools.sh
scenario test_agent_self_limit.sh
scenario test_agent_unbacked_claim.sh
scenario test_agent_judge_roster.sh
scenario test_agent_tool_budget.sh
scenario test_memory_limits.sh
scenario test_tool_aliases.sh
scenario test_agent_dsml.sh
scenario test_providers.sh
scenario test_publish_resources.sh
scenario test_agent_refine.sh
scenario test_agent_efficiency.sh
scenario test_agent_prompt_summary.sh
scenario test_agent_instruct.sh
scenario test_agent_commands.sh
scenario test_agent_edit_md5.sh
scenario test_agent_sessions.sh
scenario test_desktop.sh
scenario test_repl_model_ollama_server_url.sh
scenario test_completion_cmd.sh
scenario test_features_clone_info.sh
scenario test_examples_component_cli.sh
scenario test_registry_install_local.sh
scenario test_registry_uninstall_update.sh
scenario test_example_components.sh
scenario test_component_update.sh
scenario test_examples_component_input_source.sh
scenario test_examples_component_setup_teardown.sh
scenario test_examples_file_processor.sh
scenario test_examples_input_component.sh
scenario test_examples_auto_env.sh
scenario test_registry_verify.sh
scenario test_install_exec.sh
scenario test_global_config.sh
scenario test_config_validation.sh
scenario test_connection_prompt.sh
scenario test_model_resolution.sh
scenario test_doctor.sh
scenario test_llm_cli.sh
scenario test_m365_proxy.sh
scenario test_env_cmd.sh
scenario test_validate_doctor.sh
scenario test_test_cmd.sh
scenario test_component_version_pinning.sh

# Run llamafile (default file backend) live test (skips when model not cached)
scenario test_llamafile_e2e.sh

# Run Ollama LLM tests (will skip if Ollama not available)
scenario test_ollama_e2e.sh
scenario test_local_ollama_e2e.sh

# Summary
echo "=========================================="
echo "Test Summary"
echo "=========================================="
echo -e "${GREEN}Passed:${NC} $PASSED"
echo -e "${RED}Failed:${NC} $FAILED"
echo -e "${YELLOW}Skipped:${NC} $SKIPPED"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
fi