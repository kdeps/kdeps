#!/usr/bin/env bash
# E2E: kdeps secret keeps secrets in an age vault; a workflow reads them
# with secret('NAME'), the project vault before ~/.kdeps/secrets.age.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing kdeps secret and secret()..."

SEC_DIR=$(mktemp -d)
SEC_HOME="$SEC_DIR/home"
SEC_PROJ="$SEC_DIR/proj"
mkdir -p "$SEC_HOME" "$SEC_PROJ/resources"

sec() {
    (cd "$SEC_PROJ" && HOME="$SEC_HOME" USERPROFILE="$(to_native_path "$SEC_HOME")" \
        KDEPS_VAULT_PASSWORD="${SEC_PW-e2e-pw}" "$KDEPS_BIN" "$@")
}

# Global vault: no workflow.yaml in the folder yet.
OUT=$(printf 'from-global' | sec secret set shared 2>&1)
OUT2=$(printf 'only-global' | sec secret set db.pass 2>&1)
if output_grep "Saved shared in .*home" "$OUT" && output_grep "Saved db.pass" "$OUT2" \
    && [ -f "$SEC_HOME/.kdeps/secrets.age" ]; then
    test_passed "secret set writes ~/.kdeps/secrets.age"
else
    test_failed "secret set global" "$OUT $OUT2"
fi

if ! grep -q "from-global" "$SEC_HOME/.kdeps/secrets.age" && grep -q "age-encryption.org" "$SEC_HOME/.kdeps/secrets.age"; then
    test_passed "secret vault is encrypted"
else
    test_failed "secret vault is encrypted" "plaintext found in the vault"
fi

cat > "$SEC_PROJ/workflow.yaml" <<'YAML'
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: secret-e2e
  version: "1.0.0"
  targetActionId: response
YAML
cat > "$SEC_PROJ/resources/response.yaml" <<'YAML'
actionId: response
name: Response
apiResponse:
  success: true
  response:
    shared: "{{ secret('shared') }}"
    db: "{{ secret('db.pass') }}"
YAML

# Project vault: the folder now holds workflow.yaml.
OUT=$(printf 'from-project' | sec secret set shared 2>&1)
if output_grep "Saved shared in secrets.age" "$OUT" && [ -f "$SEC_PROJ/secrets.age" ]; then
    test_passed "secret set next to workflow.yaml writes the project vault"
else
    test_failed "secret set project" "$OUT"
fi

OUT=$(sec secret list 2>&1)
if output_grep "shared" "$OUT" && ! output_grep "from-project" "$OUT"; then
    test_passed "secret list shows names, not values"
else
    test_failed "secret list" "$OUT"
fi

OUT=$(sec run workflow.yaml 2>&1)
if output_grep '"shared": "from-project"' "$OUT" && output_grep '"db": "only-global"' "$OUT"; then
    test_passed "secret() reads the project vault, then the global vault"
else
    test_failed "secret() in a workflow" "$OUT"
fi

OUT=$(SEC_PW="" sec run workflow.yaml </dev/null 2>&1)
if output_grep "vault is locked" "$OUT"; then
    test_passed "secret() with no password says the vault is locked"
else
    test_failed "secret() locked vault" "$OUT"
fi

OUT=$(SEC_PW="wrong" sec secret list 2>&1)
if output_grep "wrong password" "$OUT"; then
    test_passed "secret list with a wrong password fails"
else
    test_failed "secret list wrong password" "$OUT"
fi

OUT=$(sec secret rm shared 2>&1)
OUT2=$(sec run workflow.yaml 2>&1)
if output_grep "Removed shared" "$OUT" && output_grep '"shared": "from-global"' "$OUT2"; then
    test_passed "secret rm drops the project value; the global value is used"
else
    test_failed "secret rm" "$OUT $OUT2"
fi

rm -rf "$SEC_DIR"
echo ""
