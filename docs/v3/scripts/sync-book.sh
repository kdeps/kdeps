#!/usr/bin/env bash
# Replace docs/v3 book pages from gitignored ./book/
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
SRC="$ROOT/book"
DST="$ROOT/docs/v3"
PUB="$DST/public"

if [[ ! -d "$SRC" ]]; then
  echo "missing $SRC" >&2
  exit 1
fi

# Remove previous book pages (keep VitePress tooling + home index)
find "$DST" -maxdepth 1 -type f \( \
  -name 'chapter-*.md' -o \
  -name 'appendix-*.md' -o \
  -name 'frontmatter.md' -o \
  -name 'backmatter.md' -o \
  -name 'Book.txt' -o \
  -name 'something-*.md' \
\) -delete

cp "$SRC/Book.txt" "$DST/"
cp "$SRC/frontmatter.md" "$SRC/backmatter.md" "$DST/"
cp "$SRC"/chapter-*.md "$DST/"
cp "$SRC"/appendix-*.md "$DST/"
cp "$SRC/cover.jpg" "$PUB/book-cover.jpg" 2>/dev/null || true
cp "$SRC/cover.jpg" "$SRC/cover2.jpg" "$PUB/" 2>/dev/null || true

python3 - <<'PY'
from pathlib import Path
dst = Path("docs/v3")
patterns = ["chapter-*.md", "appendix-*.md", "frontmatter.md", "backmatter.md"]
for pat in patterns:
    for p in dst.glob(pat):
        t = p.read_text()
        t2 = t.replace("{{", "&#123;&#123;").replace("}}", "&#125;&#125;")
        t2 = t2.replace("&#123;&#123;&#123;&#123;", "&#123;&#123;").replace(
            "&#125;&#125;&#125;&#125;", "&#125;&#125;"
        )
        p.write_text(t2)
print("docs/v3 replaced from ./book/")
PY
