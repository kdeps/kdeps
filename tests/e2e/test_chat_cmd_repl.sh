set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing REPL subcommands..."

TMP_HOME=$(mktemp -d)
trap 'rm -rf "$TMP_HOME"' EXIT

_repl() {
    printf '%s\n' "$@" '/quit' | HOME="$TMP_HOME" timeout 15 "$KDEPS_BIN" 2>&1 || true
}

# /help
OUTPUT=$(_repl '/help')
if output_grep_i "available commands" "$OUTPUT"; then
    test_passed "REPL /help - prints available commands"
else
    test_failed "REPL /help - expected command list" "Output: $OUTPUT"
fi
if output_grep_fixed "/model list" "$OUTPUT"; then
    test_passed "REPL /help - shows /model list"
else
    test_failed "REPL /help - missing /model list" "Output: $OUTPUT"
fi
if output_grep_fixed "/model ps" "$OUTPUT"; then
    test_passed "REPL /help - shows /model ps"
else
    test_failed "REPL /help - missing /model ps" "Output: $OUTPUT"
fi
if output_grep_fixed "/model hff" "$OUTPUT"; then
    test_passed "REPL /help - shows /model hff"
else
    test_failed "REPL /help - missing /model hff" "Output: $OUTPUT"
fi
if output_grep_fixed "  /models" "$OUTPUT"; then
    test_failed "REPL /help - old /models still present as top-level" "Output: $OUTPUT"
else
    test_passed "REPL /help - old /models removed"
fi
if output_grep_fixed "  /processes" "$OUTPUT"; then
    test_failed "REPL /help - old /processes still present as top-level" "Output: $OUTPUT"
else
    test_passed "REPL /help - old /processes removed"
fi
if output_grep_fixed "  /hff" "$OUTPUT"; then
    test_failed "REPL /help - old /hff still present as top-level" "Output: $OUTPUT"
else
    test_passed "REPL /help - old /hff removed"
fi

# /clear
OUTPUT=$(_repl '/clear')
if output_grep_i "panic|runtime error" "$OUTPUT"; then
    test_failed "REPL /clear - panicked" "Output: $OUTPUT"
else
    test_passed "REPL /clear - exits without error"
fi

# /skills
OUTPUT=$(_repl '/skills')
if output_grep_i "skill|no skills|loaded" "$OUTPUT"; then
    test_passed "REPL /skills - prints skill list or empty"
elif output_grep_i "panic|runtime error" "$OUTPUT"; then
    test_failed "REPL /skills - panicked" "Output: $OUTPUT"
else
    test_passed "REPL /skills - exits without error"
fi

# /prompts
OUTPUT=$(_repl '/prompts')
if output_grep_i "prompt|no prompt|template" "$OUTPUT"; then
    test_passed "REPL /prompts - prints prompt list or empty"
elif output_grep_i "panic|runtime error" "$OUTPUT"; then
    test_failed "REPL /prompts - panicked" "Output: $OUTPUT"
else
    test_passed "REPL /prompts - exits without error"
fi

# /history
OUTPUT=$(_repl '/history')
if output_grep_i "panic|runtime error" "$OUTPUT"; then
    test_failed "REPL /history - panicked" "Output: $OUTPUT"
else
    test_passed "REPL /history - exits without error"
fi

# /compact (no turns to compact)
OUTPUT=$(_repl '/compact')
if output_grep_i "panic|runtime error" "$OUTPUT"; then
    test_failed "REPL /compact - panicked" "Output: $OUTPUT"
else
    test_passed "REPL /compact - exits without error"
fi

# /thinking (show current)
OUTPUT=$(_repl '/thinking')
if output_grep_i "thinking|reasoning|off|auto|minimal|low|medium|high" "$OUTPUT"; then
    test_passed "REPL /thinking - shows thinking mode"
elif output_grep_i "panic|runtime error" "$OUTPUT"; then
    test_failed "REPL /thinking - panicked" "Output: $OUTPUT"
else
    test_passed "REPL /thinking - exits without error"
fi

# /thinking off
OUTPUT=$(_repl '/thinking off')
if output_grep_i "panic|runtime error" "$OUTPUT"; then
    test_failed "REPL /thinking off - panicked" "Output: $OUTPUT"
else
    test_passed "REPL /thinking off - exits without error"
fi

# /thinking auto
OUTPUT=$(_repl '/thinking auto')
if output_grep_i "panic|runtime error" "$OUTPUT"; then
    test_failed "REPL /thinking auto - panicked" "Output: $OUTPUT"
else
    test_passed "REPL /thinking auto - exits without error"
fi

# /reload
OUTPUT=$(_repl '/reload')
if output_grep_i "panic|runtime error" "$OUTPUT"; then
    test_failed "REPL /reload - panicked" "Output: $OUTPUT"
else
    test_passed "REPL /reload - exits without error"
fi

# /session list
OUTPUT=$(_repl '/session list')
if output_grep_i "session|no saved|saved sessions" "$OUTPUT"; then
    test_passed "REPL /session list - prints session list or empty"
elif output_grep_i "panic|runtime error" "$OUTPUT"; then
    test_failed "REPL /session list - panicked" "Output: $OUTPUT"
else
    test_passed "REPL /session list - exits without error"
fi

# /session (no subcommand)
OUTPUT=$(_repl '/session')
if output_grep_i "panic|runtime error" "$OUTPUT"; then
    test_failed "REPL /session - panicked" "Output: $OUTPUT"
else
    test_passed "REPL /session - exits without error"
fi

# /model (show current)
OUTPUT=$(_repl '/model')
if output_grep_i "model|llama|claude|gemini|gpt|picker" "$OUTPUT"; then
    test_passed "REPL /model - shows current model or picker"
elif output_grep_i "panic|runtime error" "$OUTPUT"; then
    test_failed "REPL /model - panicked" "Output: $OUTPUT"
else
    test_passed "REPL /model - exits without error"
fi

# /model default (show current default)
OUTPUT=$(_repl '/model default')
if output_grep_i "panic|runtime error" "$OUTPUT"; then
    test_failed "REPL /model default - panicked" "Output: $OUTPUT"
else
    test_passed "REPL /model default - exits without error"
fi

# /model list
OUTPUT=$(_repl '/model list')
if output_grep_i "available models|cloud|ollama|no models|provider" "$OUTPUT"; then
    test_passed "REPL /model list - prints model list"
elif output_grep_i "panic|runtime error" "$OUTPUT"; then
    test_failed "REPL /model list - panicked" "Output: $OUTPUT"
else
    test_passed "REPL /model list - exits without error"
fi

# /model ps (no running servers expected in CI)
OUTPUT=$(_repl '/model ps')
if output_grep_i "panic|runtime error" "$OUTPUT"; then
    test_failed "REPL /model ps - panicked" "Output: $OUTPUT"
elif output_grep_i "no local model|pid|port|backend" "$OUTPUT"; then
    test_passed "REPL /model ps - prints process list or 'no local model servers'"
else
    test_passed "REPL /model ps - exits without error"
fi

# /model ps kill (no model arg - should not crash)
OUTPUT=$(_repl '/model ps kill')
if output_grep_i "panic|runtime error" "$OUTPUT"; then
    test_failed "REPL /model ps kill (no arg) - panicked" "Output: $OUTPUT"
else
    test_passed "REPL /model ps kill (no arg) - exits without error"
fi

# /model ps switch (no model arg - should not crash)
OUTPUT=$(_repl '/model ps switch')
if output_grep_i "panic|runtime error" "$OUTPUT"; then
    test_failed "REPL /model ps switch (no arg) - panicked" "Output: $OUTPUT"
else
    test_passed "REPL /model ps switch (no arg) - exits without error"
fi

# /model hff (no subcommand - should print usage)
OUTPUT=$(_repl '/model hff')
if output_grep_i "usage.*model hff|model hff search|model hff info" "$OUTPUT"; then
    test_passed "REPL /model hff - prints usage"
elif output_grep_i "panic|runtime error" "$OUTPUT"; then
    test_failed "REPL /model hff - panicked" "Output: $OUTPUT"
else
    test_failed "REPL /model hff - expected usage hint" "Output: $OUTPUT"
fi

# /model hff search (no query - should print usage)
OUTPUT=$(_repl '/model hff search')
if output_grep_i "usage.*model hff search|query" "$OUTPUT"; then
    test_passed "REPL /model hff search (no query) - prints usage"
elif output_grep_i "panic|runtime error" "$OUTPUT"; then
    test_failed "REPL /model hff search (no query) - panicked" "Output: $OUTPUT"
else
    test_failed "REPL /model hff search (no query) - expected usage hint" "Output: $OUTPUT"
fi

# /model hff info (no repo - should print usage)
OUTPUT=$(_repl '/model hff info')
if output_grep_i "usage.*model hff info|repo" "$OUTPUT"; then
    test_passed "REPL /model hff info (no repo) - prints usage"
elif output_grep_i "panic|runtime error" "$OUTPUT"; then
    test_failed "REPL /model hff info (no repo) - panicked" "Output: $OUTPUT"
else
    test_failed "REPL /model hff info (no repo) - expected usage hint" "Output: $OUTPUT"
fi

# /model hff download (no repo - should print usage)
OUTPUT=$(_repl '/model hff download')
if output_grep_i "usage.*model hff download|repo|filename" "$OUTPUT"; then
    test_passed "REPL /model hff download (no repo) - prints usage"
elif output_grep_i "panic|runtime error" "$OUTPUT"; then
    test_failed "REPL /model hff download (no repo) - panicked" "Output: $OUTPUT"
else
    test_failed "REPL /model hff download (no repo) - expected usage hint" "Output: $OUTPUT"
fi

# /model hff bogus subcommand - should print unknown subcommand error
OUTPUT=$(_repl '/model hff bogus')
if output_grep_i "unknown.*model hff|bogus|search.*info.*download" "$OUTPUT"; then
    test_passed "REPL /model hff bogus - prints unknown subcommand error"
elif output_grep_i "panic|runtime error" "$OUTPUT"; then
    test_failed "REPL /model hff bogus - panicked" "Output: $OUTPUT"
else
    test_failed "REPL /model hff bogus - expected error message" "Output: $OUTPUT"
fi

# unknown command
OUTPUT=$(_repl '/boguscmd')
if output_grep_i "unknown command|boguscmd" "$OUTPUT"; then
    test_passed "REPL unknown command - prints error"
elif output_grep_i "panic|runtime error" "$OUTPUT"; then
    test_failed "REPL unknown command - panicked" "Output: $OUTPUT"
else
    test_passed "REPL unknown command - exits without error"
fi

# EOF on stdin exits cleanly
OUTPUT=$(printf '' | HOME="$TMP_HOME" timeout 10 "$KDEPS_BIN" 2>&1 || true)
if output_grep_i "panic|runtime error" "$OUTPUT"; then
    test_failed "REPL EOF on stdin - panicked" "Output: $OUTPUT"
else
    test_passed "REPL EOF on stdin - exits cleanly"
fi

# Ready banner must reference /model list not /models
OUTPUT=$(_repl '/quit')
if output_grep_fixed "/models to browse" "$OUTPUT"; then
    test_failed "REPL banner - still references old /models" "Output: $OUTPUT"
else
    test_passed "REPL banner - no stale /models reference"
fi

# /context - no args shows current window size
OUTPUT=$(_repl '/context')
if output_grep_i "context window" "$OUTPUT"; then
    test_passed "REPL /context (no args) - shows context window size"
elif output_grep_i "panic|runtime error" "$OUTPUT"; then
    test_failed "REPL /context (no args) - panicked" "Output: $OUTPUT"
else
    test_failed "REPL /context (no args) - expected 'context window'" "Output: $OUTPUT"
fi

# /context with invalid input shows usage error
OUTPUT=$(_repl '/context notanumber')
if output_grep_i "usage.*context|context.*size" "$OUTPUT"; then
    test_passed "REPL /context invalid - shows usage error"
elif output_grep_i "panic|runtime error" "$OUTPUT"; then
    test_failed "REPL /context invalid - panicked" "Output: $OUTPUT"
else
    test_failed "REPL /context invalid - expected usage error" "Output: $OUTPUT"
fi

# /context with k suffix (32k -> 32768): no crash, prints context message
OUTPUT=$(_repl '/context 32k')
if output_grep_i "panic|runtime error" "$OUTPUT"; then
    test_failed "REPL /context 32k - panicked" "Output: $OUTPUT"
elif output_grep_i "context.*32768|context.*managed|context window" "$OUTPUT"; then
    test_passed "REPL /context 32k - handled without crash"
else
    test_passed "REPL /context 32k - no crash"
fi

# /context with M suffix (1m -> 1048576): no crash
OUTPUT=$(_repl '/context 1m')
if output_grep_i "panic|runtime error" "$OUTPUT"; then
    test_failed "REPL /context 1m - panicked" "Output: $OUTPUT"
elif output_grep_i "context.*1048576|context.*managed|context window" "$OUTPUT"; then
    test_passed "REPL /context 1m - M suffix handled without crash"
else
    test_passed "REPL /context 1m - no crash"
fi

# /context with uppercase K suffix
OUTPUT=$(_repl '/context 64K')
if output_grep_i "panic|runtime error" "$OUTPUT"; then
    test_failed "REPL /context 64K - panicked" "Output: $OUTPUT"
else
    test_passed "REPL /context 64K - uppercase K suffix no crash"
fi

# /context with uppercase M suffix
OUTPUT=$(_repl '/context 2M')
if output_grep_i "panic|runtime error" "$OUTPUT"; then
    test_failed "REPL /context 2M - panicked" "Output: $OUTPUT"
else
    test_passed "REPL /context 2M - uppercase M suffix no crash"
fi

echo ""
echo "REPL subcommand E2E tests complete."
