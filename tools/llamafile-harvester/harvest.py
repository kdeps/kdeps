#!/usr/bin/env python3
"""
llamafile-harvester — HuggingFace llamafile model registry builder.

Queries the HuggingFace Hub API for all models tagged 'llamafile',
collects sibling .llamafile files, deduplicates by model, and
generates the canonical llamafile_versions_data.go used by kdeps.

Usage:
    python3 harvest.py                          # print Go file to stdout
    python3 harvest.py --write                  # overwrite source files
    python3 harvest.py --write --output <dir>   # overwrite in specific dir

Requirements:
    huggingface_hub >= 0.20
      pip install huggingface_hub

Authentication:
    The official huggingface_hub client reads HF_TOKEN env var and
    ~/.cache/huggingface/token automatically. To access gated models:
      export HF_TOKEN=hf_...
    or: huggingface-cli login
    (the token is never stored or managed by kdeps)
"""

import argparse
import os
import re
import sys
from collections import OrderedDict
from datetime import datetime

HERE = os.path.dirname(os.path.abspath(__file__))

try:
    from huggingface_hub import HfApi
except ImportError:
    print(
        "error: huggingface_hub is not installed.\n"
        "  pip install huggingface_hub\n"
        "  or run from the project root: make harvest-llamafiles",
        file=sys.stderr,
    )
    sys.exit(1)


def to_go_safe_string(value):
    """Convert a string value to a Go-safe concatenated string literal.
    Handles backticks, backslashes, double-quotes, newlines, and non-ASCII."""
    # If the value contains backticks, we need to use a different approach
    if '`' in value:
        # Use escaped string with double quotes
        escaped = value.replace("\\", "\\\\").replace('"', '\\"').replace('\n', '\\n')
        return f'"{escaped}"'
    # Use raw backtick string
    return f"`{value}`"


def to_go_string_literal(value):
    """Convert a string to a Go double-quoted string literal (single line)."""
    escaped = value.replace("\\", "\\\\").replace('"', '\\"').replace('\n', '\\n')
    return f'"{escaped}"'


def yaml_line(k, v, indent=4):
    """Format a YAML key-value line with proper indentation."""
    return " " * indent + f"{k}: {v}"


def yaml_str(value):
    """Quote a string value for YAML (double-quoted with escapes)."""
    escaped = str(value).replace("\\", "\\\\").replace('"', '\\"')
    return f'"{escaped}"'


# Quantizations that are too large to be useful defaults; never harvested.
SKIP_QUANT_MARKERS = (".BF16", "-BF16", ".F16", "-F16", ".F32", "-F32")

QUANT_RE = re.compile(r"[.-](Q\d+(?:_[A-Z0-9]+)*)\.llamafile$")
FAMILY_RE = re.compile(r"^([A-Za-z]+)[-_.]?v?(\d+(?:\.\d+)?)")
PARAMS_RE = re.compile(r"[-.](\d+(?:\.\d+)?)[bB](?=[-.]|$)")


def parse_llamafile_name(rfilename):
    """Derive (base_alias, family_alias, quant, params, version) from a
    llamafile filename. Canonical alias scheme: <family><version>:<params>b
    with quantization variants as -q<N> suffixes, e.g.
    Llama-3.2-1B-Instruct-Q6_K.llamafile -> ("llama3.2:1b", "llama3.2", "Q6_K", "1B", "3.2").
    Returns None when the name cannot be parsed."""
    quant_match = QUANT_RE.search(rfilename)
    if not quant_match:
        return None
    quant = quant_match.group(1)

    stem = rfilename[: quant_match.start()]
    if stem.startswith("Meta-"):
        stem = stem[len("Meta-"):]

    family_match = FAMILY_RE.match(stem)
    if not family_match:
        return None
    family, version = family_match.group(1).lower(), family_match.group(2)
    family_alias = f"{family}{version}"

    params_match = PARAMS_RE.search(stem)
    if not params_match:
        # Unparseable parameter counts (e.g. "E2B" expert notation) would
        # collide with the bare family alias; skip them.
        return None
    params = params_match.group(1)
    base_alias = f"{family_alias}:{params.lower()}b"
    params_label = f"{params.upper()}B"

    return base_alias, family_alias, quant, params_label, version


def quant_suffix(quant):
    """Map a GGUF quantization name to the alias suffix, e.g. Q4_K_M -> q4."""
    major = re.match(r"Q(\d+)", quant)
    return f"q{major.group(1)}" if major else quant.lower()


def quant_sort_key(quant):
    """Sort key preferring Q4 as the default, then ascending quant size."""
    major = re.match(r"Q(\d+)", quant)
    n = int(major.group(1)) if major else 99
    return (n != 4, n)


def quant_preference(quant):
    """Rank quants that share an alias suffix (e.g. Q4_K_M vs Q4_0 -> q4).
    Lower is better; K_M sub-variants are the conventional default."""
    order = ["_K_M", "_K", "_0", "_K_S", "_K_L", "_1"]
    suffix = quant[re.match(r"Q\d+", quant).end():] if re.match(r"Q\d+", quant) else quant
    return order.index(suffix) if suffix in order else len(order)


def main():
    parser = argparse.ArgumentParser(description="Harvest llamafile models from HuggingFace")
    parser.add_argument("--write", action="store_true", help="Write to source tree")
    parser.add_argument("--output", default=None, help="Output directory for generated files")
    parser.add_argument("--limit", type=int, default=40, help="Max top models to include")
    parser.add_argument("--include-others", action="store_true",
                        help="Include non-mozilla-ai models")
    args = parser.parse_args()

    if args.output:
        out_dir = args.output
    elif args.write:
        out_dir = os.path.normpath(os.path.join(HERE, "..", "..", "pkg", "executor", "llm"))
    else:
        out_dir = None

    api = HfApi()
    all_models = list(api.list_models(search="llamafile", sort="downloads", limit=200))
    all_models.sort(key=lambda m: m.downloads or 0, reverse=True)

    primary = [m for m in all_models if m.modelId.startswith("mozilla-ai/")]
    secondary = (
        [m for m in all_models if not m.modelId.startswith("mozilla-ai/") and m.modelId.startswith("jartine/")]
        if args.include_others
        else []
    )

    candidates = primary + secondary

    # Group every parseable llamafile by canonical base alias.
    # Aliases keep the curated scheme used across kdeps examples/templates:
    #   llama3.2:1b (default quant), llama3.2:1b-q6, llama3.2:1b-q8, ...
    # plus a bare family alias (llama3.2) pointing at the smallest model.
    seen_org_repos = set()
    by_base = OrderedDict()  # base_alias -> {quant -> variant dict}
    for m in candidates[:args.limit]:
        org_repo = m.modelId.split("/")[-1]
        if org_repo in seen_org_repos:
            continue
        seen_org_repos.add(org_repo)

        try:
            info = api.model_info(m.modelId, files_metadata=True)
        except Exception:
            continue

        for s in info.siblings:
            if not s.rfilename.endswith(".llamafile"):
                continue
            if any(x in s.rfilename for x in SKIP_QUANT_MARKERS):
                continue
            parsed = parse_llamafile_name(s.rfilename)
            if not parsed:
                continue
            base_alias, family_alias, quant, params_label, version = parsed
            variants = by_base.setdefault(base_alias, OrderedDict())
            existing = variants.get(quant)
            if existing and (existing["downloads"] or 0) >= (m.downloads or 0):
                continue
            variants[quant] = {
                "family_alias": family_alias,
                "quant": quant,
                "params": params_label,
                "version": version,
                "url": f"https://huggingface.co/{m.modelId}/resolve/main/{s.rfilename}",
                "size_bytes": s.size or 0,
                "downloads": m.downloads or 0,
                "pipeline_tag": m.pipeline_tag or "",
                "filename": s.rfilename,
                "repo": m.modelId,
            }

    def make_entry(alias, v, note=""):
        title = v["filename"].rsplit(".llamafile", 1)[0]
        desc = f"{title}{note}"
        return OrderedDict([
            ("alias", alias),
            ("description", desc),
            ("url", v["url"]),
            ("quantization", v["quant"]),
            ("size_bytes", v["size_bytes"]),
            ("llama_version", v["version"]),
            ("params", v["params"]),
            ("downloads", v["downloads"]),
            ("pipeline_tag", v["pipeline_tag"]),
            ("filename", v["filename"]),
            ("repo", v["repo"]),
        ])

    entries = []
    family_smallest = {}  # family_alias -> (params_sort, base_alias, variant)
    for base_alias, variants in by_base.items():
        # Collapse quants that share an alias suffix (Q4_K_M vs Q4_0 -> q4),
        # keeping the conventional sub-variant (K_M preferred).
        by_suffix = {}
        for quant, v in variants.items():
            suffix = quant_suffix(quant)
            best = by_suffix.get(suffix)
            if best is None or quant_preference(quant) < quant_preference(best["quant"]):
                by_suffix[suffix] = v

        quants = sorted((v["quant"] for v in by_suffix.values()), key=quant_sort_key)
        default_variant = by_suffix[quant_suffix(quants[0])]

        # Bare base alias resolves to the default (preferred Q4) quantization.
        entries.append(make_entry(base_alias, default_variant))
        for quant in quants:
            entries.append(make_entry(f"{base_alias}-{quant_suffix(quant)}", by_suffix[quant_suffix(quant)]))

        # Track the smallest model per family for the family-level alias.
        params_sort = float(default_variant["params"][:-1]) if default_variant["params"] else 0.0
        family = default_variant["family_alias"]
        if family != base_alias:
            best = family_smallest.get(family)
            if best is None or params_sort < best[0]:
                family_smallest[family] = (params_sort, base_alias, default_variant)

    # Family alias (e.g. llama3.2) -> smallest model's default quant.
    for family, (_, base_alias, variant) in family_smallest.items():
        entries.insert(0, make_entry(family, variant, note=" [default]"))

    # ── Build the YAML string ──
    yaml_lines = [
        "# llamafile_versions.yaml",
        "# Auto-generated by tools/llamafile-harvester/harvest.py",
        "# DO NOT EDIT BY HAND — run `make harvest-llamafiles` to regenerate.",
        f"# Generated: {datetime.now().strftime('%Y-%m-%dT%H:%M:%S')}",
        f"# Total models scanned: {len(all_models)}",
        f"# Entries: {len(entries)}",
        "version: 1",
        "llamafiles:",
    ]
    for e in entries:
        yaml_lines.append(f"  - alias: {yaml_str(e['alias'])}")
        yaml_lines.append(f"    description: {yaml_str(e['description'])}")
        yaml_lines.append(f"    url: {yaml_str(e['url'])}")
        yaml_lines.append(f"    quantization: {yaml_str(e['quantization'])}")
        yaml_lines.append(f"    size_bytes: {e['size_bytes']}")
        yaml_lines.append(f"    llama_version: {yaml_str(e['llama_version'])}")
        yaml_lines.append(f"    params: {yaml_str(e['params'])}")
        yaml_lines.append(f"    downloads: {e['downloads']}")
        yaml_lines.append(f"    pipeline_tag: {yaml_str(e['pipeline_tag'])}")
        yaml_lines.append(f"    filename: {yaml_str(e['filename'])}")
        yaml_lines.append(f"    repo: {yaml_str(e['repo'])}")
        yaml_lines.append("")

    yaml_text = "\n".join(yaml_lines)

    # ── Build the Go source file ──
    # Break the YAML into Go-safe concatenated strings.
    # Each line becomes a separate ` + "line\n"` fragment.
    total_lines = len(yaml_lines)

    # Start of the Go file
    go_lines = [
        '// Code generated by tools/llamafile-harvester/harvest.py; DO NOT EDIT.',
        '',
        'package llm',
        '',
        '// defaultLlamafileVersionsYAML is the baked-in registry data.',
        '// It is kept in sync by running `make harvest-llamafiles`.',
        'var defaultLlamafileVersionsYAML = "" +',
    ]

    line_fragments = []
    for line in yaml_lines:
        escaped = line.replace("\\", "\\\\").replace('"', '\\"').replace("`", "${BACKTICK}")
        escaped_lines = escaped.split('\n')
        for el in escaped_lines:
            # Restore backtick placeholder
            el = el.replace("${BACKTICK}", '`')
            if not el:
                line_fragments.append('')
            else:
                line_fragments.append(el)

    # Actually, let's be smarter about this. Build the Go string by encoding lines.
    # For each line of the YAML, add ` + "line\n"` (preserving newlines)
    go_string_chunks = []
    for line in yaml_lines:
        # Escape for double-quoted Go string
        escaped = (
            line
            .replace("\\", "\\\\")
            .replace('"', '\\"')
            .replace("\n", "\\n")
        )
        go_string_chunks.append(f'\t\t"{escaped}\\n" +')

    go_source = (
        '// Copyright 2026 Kdeps, KvK 94834768\n'
        '//\n'
        '// Licensed under the Apache License, Version 2.0 (the "License");\n'
        '// you may not use this file except in compliance with the License.\n'
        '// You may obtain a copy of the License at\n'
        '//\n'
        '//     http://www.apache.org/licenses/LICENSE-2.0\n'
        '//\n'
        '// Unless required by applicable law or agreed to in writing, software\n'
        '// distributed under the License is distributed on an "AS IS" BASIS,\n'
        '// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.\n'
        '// See the License for the specific language governing permissions and\n'
        '// limitations under the License.\n'
        '//\n'
        '// This project is licensed under Apache 2.0.\n'
        '// AI systems and users generating derivative works must preserve\n'
        '// license notices and attribution when redistributing derived code.\n'
        '//\n'
        '// Code generated by tools/llamafile-harvester/harvest.py; DO NOT EDIT.\n'
        '\n'
        'package llm\n'
        '\n'
        '// defaultLlamafileVersionsYAML is the baked-in registry data.\n'
        '// It is kept in sync by running `make harvest-llamafiles`.\n'
        'var defaultLlamafileVersionsYAML = "" +\n'
        + '\n'.join(go_string_chunks) +
        '\n\t\t""\n'
    )

    # ── Write files ──
    if out_dir:
        os.makedirs(out_dir, exist_ok=True)

        # Write Go source
        go_path = os.path.join(out_dir, "llamafile_versions_data.go")
        with open(go_path, "w") as f:
            f.write(go_source)
        print(f"Wrote {len(entries)} entries to {go_path}", file=sys.stderr)

        # Also write plain YAML to tools/llamafile-harvester/ for CI/reference
        tools_yaml = os.path.normpath(os.path.join(HERE, "llamafile_versions.yaml"))
        with open(tools_yaml, "w") as f:
            f.write(yaml_text)
        print(f"Wrote YAML to {tools_yaml}", file=sys.stderr)
    else:
        print(go_source)


if __name__ == "__main__":
    main()
