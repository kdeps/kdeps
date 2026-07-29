#!/usr/bin/env bash
# Copy gitignored ./book/ into docs/v3/book (main docs source of truth for deploy).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
SRC="$ROOT/book"
DST="$ROOT/docs/v3/book"
PUB="$ROOT/docs/v3/public"

if [[ ! -d "$SRC" ]]; then
  echo "missing $SRC (local book tree)" >&2
  exit 1
fi

mkdir -p "$DST"
cp "$SRC"/Book.txt "$DST/" 2>/dev/null || true
cp "$SRC"/frontmatter.md "$SRC"/backmatter.md "$DST/"
cp "$SRC"/chapter-*.md "$SRC"/appendix-*.md "$DST/"
cp "$SRC"/cover.jpg "$SRC"/cover2.jpg "$DST/" 2>/dev/null || true
cp "$SRC"/cover.jpg "$PUB/book-cover.jpg" 2>/dev/null || true

# Vue must not interpret mustache expressions in book markdown
python3 - <<'PY'
from pathlib import Path
dst = Path("docs/v3/book")
for p in dst.glob("*.md"):
    t = p.read_text()
    t2 = t.replace("{{", "&#123;&#123;").replace("}}", "&#125;&#125;")
    # avoid double-escape on re-sync
    t2 = t2.replace("&#123;&#123;&#123;&#123;", "&#123;&#123;").replace("&#125;&#125;&#125;&#125;", "&#125;&#125;")
    p.write_text(t2)
print("synced book -> docs/v3/book")
PY
