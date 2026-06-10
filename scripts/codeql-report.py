#!/usr/bin/env python3
"""Parse codeql-results.sarif, filter noise, print alerts and exit 1 if any remain."""
import json, sys, os

# Directories scanned by CodeQL that are not part of the active codebase.
IGNORE_PREFIXES = ("kdeps-io.old/", "private/")

# Known false positives suppressed via MaD neutral model on GitHub Actions.
# Locally, the neutral model extension is not loaded, so these are listed here.
KNOWN_FALSE_POSITIVES = {
    # ResponseWriterWrapper.Write: html.EscapeString applied for browser types;
    # non-browser types (JSON, text/plain, binary) cannot cause XSS.
    ("go/reflected-xss", "pkg/infra/http/middleware.go"),
    # Session cookies are HTTP-only dev tokens; Secure is set when TLS is enabled.
    ("go/cookie-secure-not-set", "pkg/infra/http/http_session.go"),
}

# Workflow YAML intentionally supplies SQL, shell commands, and file paths at runtime.
SUPPRESSED_RULE_PREFIXES = (
    ("go/sql-injection", "pkg/executor/sql/"),
    ("go/command-injection", "pkg/executor/exec/"),
    ("go/command-injection", "pkg/executor/python/"),
    ("go/command-injection", "pkg/executor/llm/"),
    ("go/path-injection", "pkg/executor/llm/"),
    ("go/path-injection", "pkg/infra/python/"),
)

sarif_file = sys.argv[1] if len(sys.argv) > 1 else "codeql-results.sarif"
if not os.path.exists(sarif_file):
    print(f"Error: {sarif_file} not found", file=sys.stderr)
    sys.exit(2)

d = json.load(open(sarif_file))

def is_suppressed(r):
    uri = r["locations"][0]["physicalLocation"]["artifactLocation"]["uri"]
    if uri.startswith(IGNORE_PREFIXES):
        return True
    rule = r["ruleId"]
    if (rule, uri) in KNOWN_FALSE_POSITIVES:
        return True
    for alert_rule, prefix in SUPPRESSED_RULE_PREFIXES:
        if rule == alert_rule and uri.startswith(prefix):
            return True
    return False

alerts = [r for run in d.get("runs", []) for r in run.get("results", []) if not is_suppressed(r)]

if "--count" in sys.argv:
    print(len(alerts))
    sys.exit(0)

if not alerts:
    print("✓ CodeQL: PASSED (0 alerts)")
    sys.exit(0)

print(f"✗ CodeQL: FAILED ({len(alerts)} alert(s))")
for r in alerts:
    loc = r["locations"][0]["physicalLocation"]
    uri = loc["artifactLocation"]["uri"]
    line = loc["region"]["startLine"]
    msg = r["message"]["text"].replace("\n", " ")[:80]
    print(f"  [{r['ruleId']}] {uri}:{line} - {msg}")
sys.exit(1)
