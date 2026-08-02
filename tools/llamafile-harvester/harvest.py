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
    python3 harvest.py --write --gguf           # GGUF + Chinese labs (default with --gguf)
    python3 harvest.py --write --gguf --no-chinese-labs  # GGUF only, no lab pass

Chinese AI labs (DeepSeek, Qwen, Yi, GLM, InternLM, Kimi, …) are harvested into
the GGUF registry when --gguf is set (use --no-chinese-labs to skip). Nightly CI
must pass --gguf so labs stay in the daily crop.

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
import json
import os
import re
import subprocess
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

# HF pipeline_tag values that mean "not a text-generation chat model" even when
# the repo name matches a chat family's naming convention (e.g. the ASR repo
# "handy-computer/Qwen3-ASR-1.7B-gguf" contains "qwen" and would otherwise win
# the "qwen3:1.7b" alias by download count over the real chat GGUF). kdeps
# serves harvested models through a chat/completion API, so anything tagged as
# ASR, TTS, vision, or embedding-only can never actually work there.
NON_CHAT_PIPELINE_TAGS = frozenset({
    "automatic-speech-recognition",
    "text-to-speech",
    "audio-classification",
    "audio-to-audio",
    "voice-activity-detection",
    "image-classification",
    "image-segmentation",
    "object-detection",
    "image-to-text",
    "text-to-image",
    "image-to-image",
    "video-classification",
    "depth-estimation",
    "feature-extraction",
    "sentence-similarity",
    "token-classification",
    "fill-mask",
    "zero-shot-classification",
    "zero-shot-image-classification",
    "question-answering",
    "text-classification",
})

# Chinese AI labs to include in the daily GGUF harvest.
# Each entry is (name_filter, search_query) — name_filter is a case-insensitive
# substring matched against modelId; search_query finds GGUF variants on HF.
CHINESE_AI_LABS = [
    # (name_filter on modelId, HF search query)
    ("deepseek", "deepseek gguf"),
    ("qwen", "qwen gguf"),
    ("yi-", "yi gguf"),
    ("yi/", "01-ai yi gguf"),  # 01-ai/Yi-* repos
    ("glm", "glm gguf"),
    ("chatglm", "chatglm gguf"),
    ("internlm", "internlm gguf"),
    ("kimi", "kimi gguf"),
    ("moonshot", "moonshot gguf"),
    ("step-", "stepfun gguf"),
    ("stepfun", "stepfun gguf"),
    ("hunyuan", "hunyuan gguf"),
    ("baichuan", "baichuan gguf"),
    ("minimax", "minimax gguf"),
    ("ernie", "ernie gguf"),
    ("bge", "bge gguf"),
]

GGUF_QUANT_RE = re.compile(r"[.-](Q\d+(?:_[A-Z0-9]+)*)\.gguf$")
QUANT_RE = re.compile(r"[.-](Q\d+(?:_[A-Z0-9]+)*)\.llamafile$")
FAMILY_RE = re.compile(r"^([A-Za-z]+)[-_.]?v?(\d+(?:\.\d+)?)")
PARAMS_RE = re.compile(r"[-.](\d+(?:\.\d+)?)[bB](?=[-.]|$)")


def parse_filename(rfilename, quant_re):
    """Derive (base_alias, family_alias, quant, params, version) from a
    llamafile or GGUF filename."""
    quant_match = quant_re.search(rfilename)
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
        return None
    params = params_match.group(1)
    base_alias = f"{family_alias}:{params.lower()}b"
    params_label = f"{params.upper()}B"

    return base_alias, family_alias, quant, params_label, version


def parse_llamafile_name(rfilename):
    return parse_filename(rfilename, QUANT_RE)


def parse_gguf_name(rfilename):
    return parse_filename(rfilename, GGUF_QUANT_RE)


def quant_suffix(quant):
    """Map a GGUF quantization name to the alias suffix, e.g. Q4_K_M -> q4."""
    major = re.match(r"Q(\d+)", quant)
    return f"q{major.group(1)}" if major else quant.lower()


def quant_sort_key(quant):
    """Sort key preferring Q4 as the default, then ascending quant size."""
    major = re.match(r"Q(\d+)", quant)
    n = int(major.group(1)) if major else 99
    return (n != 4, n)


def _harvest_by_extension(candidates, api, args, extension, quant_re, parse_fn, limit=None):
    """Harvest models with a given file extension from HuggingFace.
    Returns (flat_entries, by_base_dict) following the same scheme as
    the llamafile harvesting logic.

    limit: max candidates to process. Defaults to args.limit. Pass 0 or a
    negative value to process every candidate (used for llmfit seeding).
    """
    if limit is None:
        limit = args.limit
    pool = candidates if limit is not None and limit <= 0 else candidates[:limit]

    seen_model_ids = set()
    by_base = OrderedDict()
    for m in pool:
        model_id = getattr(m, "modelId", None) or getattr(m, "id", None) or ""
        if not model_id or model_id in seen_model_ids:
            continue
        seen_model_ids.add(model_id)

        try:
            info = api.model_info(model_id, files_metadata=True)
        except Exception:
            continue

        downloads = getattr(m, "downloads", None) or 0
        pipeline_tag = getattr(m, "pipeline_tag", None) or ""
        if pipeline_tag in NON_CHAT_PIPELINE_TAGS:
            continue
        for s in info.siblings:
            if not s.rfilename.endswith(extension):
                continue
            if any(x in s.rfilename for x in SKIP_QUANT_MARKERS):
                continue
            parsed = parse_fn(s.rfilename)
            if not parsed:
                continue
            base_alias, family_alias, quant, params_label, version = parsed
            variants = by_base.setdefault(base_alias, OrderedDict())
            existing = variants.get(quant)
            if existing and (existing["downloads"] or 0) >= downloads:
                continue
            variants[quant] = {
                "family_alias": family_alias,
                "quant": quant,
                "params": params_label,
                "version": version,
                "url": f"https://huggingface.co/{model_id}/resolve/main/{s.rfilename}",
                "size_bytes": s.size or 0,
                "downloads": downloads,
                "pipeline_tag": pipeline_tag,
                "filename": s.rfilename,
                "repo": model_id,
            }

    return _build_entries(by_base), by_base


class _SeedModel:
    """Minimal HF model stand-in for seeding harvest from known repo ids."""

    def __init__(self, model_id, downloads=0, pipeline_tag=""):
        self.modelId = model_id
        self.downloads = downloads
        self.pipeline_tag = pipeline_tag


def load_llmfit_gguf_repos():
    """Return HuggingFace repo ids from `llmfit fit --json` gguf_sources.

    Returns an empty list when llmfit is not installed or fails. Seeding the
    GGUF harvest from llmfit keeps the catalog scoreable in the model picker.
    """
    try:
        out = subprocess.check_output(
            ["llmfit", "fit", "--json"],
            stderr=subprocess.DEVNULL,
            timeout=60,
        )
        data = json.loads(out)
    except (OSError, subprocess.SubprocessError, json.JSONDecodeError, ValueError):
        return []
    repos = []
    seen = set()
    for m in data.get("models") or []:
        for src in m.get("gguf_sources") or []:
            repo = (src.get("repo") or "").strip()
            if repo and repo not in seen:
                seen.add(repo)
                repos.append(repo)
    return repos


def _build_entries(by_base):
    """Convert a by_base dict into a flat entry list with family aliases."""
    entries = []
    family_smallest = {}
    for base_alias, variants in by_base.items():
        by_suffix = {}
        for quant, v in variants.items():
            suffix = quant_suffix(quant)
            best = by_suffix.get(suffix)
            if best is None or quant_preference(quant) < quant_preference(best["quant"]):
                by_suffix[suffix] = v

        quants = sorted((v["quant"] for v in by_suffix.values()), key=quant_sort_key)
        default_variant = by_suffix[quant_suffix(quants[0])]

        entries.append(make_entry(base_alias, default_variant))
        for quant in quants:
            entries.append(make_entry(f"{base_alias}-{quant_suffix(quant)}", by_suffix[quant_suffix(quant)]))

        params_sort = float(default_variant["params"][:-1]) if default_variant["params"] else 0.0
        family = default_variant["family_alias"]
        if family != base_alias:
            best = family_smallest.get(family)
            if best is None or params_sort < best[0]:
                family_smallest[family] = (params_sort, base_alias, default_variant)

    for family, (_, base_alias, variant) in family_smallest.items():
        entries.insert(0, make_entry(family, variant, note=" [default]"))
    return entries


def quant_preference(quant):
    """Rank quants that share an alias suffix (e.g. Q4_K_M vs Q4_0 -> q4).
    Lower is better; K_M sub-variants are the conventional default."""
    order = ["_K_M", "_K", "_0", "_K_S", "_K_L", "_1"]
    suffix = quant[re.match(r"Q\d+", quant).end():] if re.match(r"Q\d+", quant) else quant
    return order.index(suffix) if suffix in order else len(order)


def make_entry(alias, v, note=""):
    title = v["filename"].rsplit(".", 1)[0]
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


def yaml_entry_lines(entry):
    return [
        f"  - alias: {yaml_str(entry['alias'])}",
        f"    description: {yaml_str(entry['description'])}",
        f"    url: {yaml_str(entry['url'])}",
        f"    quantization: {yaml_str(entry['quantization'])}",
        f"    size_bytes: {entry['size_bytes']}",
        f"    llama_version: {yaml_str(entry['llama_version'])}",
        f"    params: {yaml_str(entry['params'])}",
        f"    downloads: {entry['downloads']}",
        f"    pipeline_tag: {yaml_str(entry['pipeline_tag'])}",
        f"    filename: {yaml_str(entry['filename'])}",
        f"    repo: {yaml_str(entry['repo'])}",
        "",
    ]


def _gen_go_source(varname, lines):
    chunks = []
    for line in lines:
        escaped = line.replace("\\", "\\\\").replace('"', '\\"').replace("\n", "\\n")
        chunks.append(f'\t\t"{escaped}\\n" +')
    return (
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
        f'// {varname} is the baked-in registry data.\n'
        '// It is kept in sync by running `make harvest-llamafiles`.\n'
        f'var {varname} = "" +\n'
        + '\n'.join(chunks) +
        '\n\t\t""\n'
    )


def main():
    parser = argparse.ArgumentParser(description="Harvest llamafile models from HuggingFace")
    parser.add_argument("--write", action="store_true", help="Write to source tree")
    parser.add_argument("--output", default=None, help="Output directory for generated files")
    parser.add_argument("--limit", type=int, default=40, help="Max top models to include")
    parser.add_argument("--include-others", action="store_true",
                        help="Include non-mozilla-ai models")
    parser.add_argument("--gguf", action="store_true",
                        help="Also harvest GGUF models from HuggingFace")
    parser.add_argument(
        "--chinese-labs",
        action=argparse.BooleanOptionalAction,
        default=None,
        help="Include Chinese AI labs (DeepSeek, Qwen, Yi, GLM, InternLM, …) in GGUF harvest. "
             "Default: on when --gguf is set. Use --no-chinese-labs to disable.",
    )
    args = parser.parse_args()
    # Nightly and local GGUF harvests must keep Chinese labs unless explicitly disabled.
    if args.chinese_labs is None:
        args.chinese_labs = bool(args.gguf)

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

    entries, _ = _harvest_by_extension(candidates, api, args, ".llamafile", QUANT_RE, parse_llamafile_name)

    # ── GGUF harvesting ──
    gguf_entries = []
    if args.gguf:
        gguf_models = list(api.list_models(search="gguf", sort="downloads", limit=200))
        gguf_models.sort(key=lambda m: m.downloads or 0, reverse=True)
        gguf_primary = [m for m in gguf_models]
        gguf_entries, _ = _harvest_by_extension(gguf_primary, api, args, ".gguf", GGUF_QUANT_RE, parse_gguf_name)

        seen_repos = {e["repo"] for e in gguf_entries}

        # ── llmfit gguf_sources seed ──
        # Top-download HF search currently surfaces brand-new models that
        # llmfit does not know. Seed the catalog with every GGUF repo llmfit
        # already scores so the model picker can show fit levels.
        llmfit_repos = load_llmfit_gguf_repos()
        if llmfit_repos:
            print(f"  llmfit: {len(llmfit_repos)} gguf_sources repos", file=sys.stderr)
            seed_models = [
                _SeedModel(r) for r in llmfit_repos if r not in seen_repos
            ]
            if seed_models:
                # limit=0: process every llmfit-sourced repo, not just top N.
                seed_entries, _ = _harvest_by_extension(
                    seed_models, api, args, ".gguf", GGUF_QUANT_RE, parse_gguf_name, limit=0,
                )
                added = 0
                for e in seed_entries:
                    if e["repo"] not in seen_repos:
                        seen_repos.add(e["repo"])
                        gguf_entries.append(e)
                        added += 1
                print(f"  llmfit: +{added} entries from gguf_sources", file=sys.stderr)
        else:
            print("  llmfit: not available — skip gguf_sources seed", file=sys.stderr)

        # ── Chinese AI labs GGUF harvest ──
        # Required for DeepSeek/Qwen/Yi/GLM/… coverage. Default on with --gguf.
        if not args.chinese_labs:
            print("  chinese-labs: skipped (--no-chinese-labs)", file=sys.stderr)
        if args.chinese_labs:
            for name_filter, query in CHINESE_AI_LABS:
                try:
                    lab_models = list(api.list_models(search=query, sort="downloads", limit=50))
                    lab_models.sort(key=lambda m: m.downloads or 0, reverse=True)
                    lab_filtered = [m for m in lab_models if name_filter in m.modelId.lower()]
                    print(f"  {name_filter}: {len(lab_models)} search results, {len(lab_filtered)} after name filter")
                    if lab_filtered:
                        lab_entries, _ = _harvest_by_extension(lab_filtered, api, args, ".gguf", GGUF_QUANT_RE, parse_gguf_name)
                        for e in lab_entries:
                            if e["repo"] not in seen_repos:
                                seen_repos.add(e["repo"])
                                gguf_entries.append(e)
                                print(f"    + {e['alias']} ({e['repo']})")
                except Exception as ex:
                    print(f"  {name_filter}: FAILED — {ex}")

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
        yaml_lines += yaml_entry_lines(e)

    if gguf_entries:
        yaml_lines.append("ggufs:")
        for e in gguf_entries:
            yaml_lines += yaml_entry_lines(e)

    yaml_text = "\n".join(yaml_lines)

    # ── Build Go sources ──
    go_source = _gen_go_source("defaultLlamafileVersionsYAML", yaml_lines)

    gguf_yaml_lines = []
    gguf_source = ""
    if gguf_entries:
        gguf_yaml_lines = [
            "# gguf_versions.yaml",
            "# Auto-generated by tools/llamafile-harvester/harvest.py",
            "# DO NOT EDIT BY HAND — run `make harvest-llamafiles` to regenerate.",
            f"# Generated: {datetime.now().strftime('%Y-%m-%dT%H:%M:%S')}",
            f"# Entries: {len(gguf_entries)}",
            "version: 1",
            "ggufs:",
        ]
        for e in gguf_entries:
            gguf_yaml_lines += yaml_entry_lines(e)
        gguf_source = _gen_go_source("defaultGGUFVersionsYAML", gguf_yaml_lines)

    # ── Write files ──
    if out_dir:
        os.makedirs(out_dir, exist_ok=True)

        go_path = os.path.join(out_dir, "llamafile_versions_data.go")
        with open(go_path, "w", encoding="utf-8") as f:
            f.write(go_source)
        print(f"Wrote {len(entries)} entries to {go_path}", file=sys.stderr)

        tools_yaml = os.path.normpath(os.path.join(HERE, "llamafile_versions.yaml"))
        with open(tools_yaml, "w", encoding="utf-8") as f:
            f.write(yaml_text)
        print(f"Wrote YAML to {tools_yaml}", file=sys.stderr)

        if gguf_source:
            gguf_go_path = os.path.join(out_dir, "gguf_versions_data.go")
            with open(gguf_go_path, "w", encoding="utf-8") as f:
                f.write(gguf_source)
            print(f"Wrote {len(gguf_entries)} GGUF entries to {gguf_go_path}", file=sys.stderr)

            gguf_yaml_path = os.path.normpath(os.path.join(HERE, "gguf_versions.yaml"))
            gguf_yaml_text = "\n".join(gguf_yaml_lines)
            with open(gguf_yaml_path, "w", encoding="utf-8") as f:
                f.write(gguf_yaml_text)
            print(f"Wrote GGUF YAML to {gguf_yaml_path}", file=sys.stderr)
    else:
        print(go_source)
        if gguf_source:
            print(gguf_source)


if __name__ == "__main__":
    main()
