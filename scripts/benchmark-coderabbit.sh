#!/usr/bin/env bash
# benchmark-coderabbit.sh — Compare slopgate findings against CodeRabbit
# review comments on the same PR.
#
# Usage:
#   benchmark-coderabbit.sh REPO_PATH PR_NUMBER [BASE_REF] [--output FILE]
#
# REPO_PATH  — local path to the git repo (must have the PR branch checked out)
# PR_NUMBER  — GitHub PR number
# BASE_REF   — base branch for diff (default: main)
# --output   — optional: write JSON comparison to FILE
#
# Requires: gh (authenticated), jq, python3, slopgate (on PATH or SLOPGATE_BIN)
set -euo pipefail

# --- Resolve slopgate binary ---
SLOPGATE_BIN="${SLOPGATE_BIN:-$(command -v slopgate 2>/dev/null || true)}"
if [[ -z "$SLOPGATE_BIN" ]] || ! [[ -x "$SLOPGATE_BIN" ]]; then
  echo "Error: slopgate not found. Install it or set SLOPGATE_BIN." >&2
  exit 1
fi

FUZZY_LINE_RANGE="${BENCHMARK_FUZZY_RANGE:-2}"
OUTPUT_FILE=""

# --- Parse args ---
if [[ $# -lt 2 ]]; then
  echo "Usage: $0 REPO_PATH PR_NUMBER [BASE_REF] [--output FILE]" >&2
  exit 1
fi

REPO_PATH="$1"; shift
PR_NUMBER="$1"; shift
BASE_REF="main"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --output)
      if [[ $# -lt 2 ]] || [[ "$2" == -* ]]; then echo "Error: --output requires a non-option argument" >&2; exit 1; fi
      OUTPUT_FILE="$2"; shift 2 ;;
    --base)
      if [[ $# -lt 2 ]] || [[ "$2" == -* ]]; then echo "Error: --base requires a non-option argument" >&2; exit 1; fi
      BASE_REF="$2"; shift 2 ;;
    -*)
      echo "Error: unknown option $1" >&2; exit 1 ;;
    *) BASE_REF="$1"; shift ;;
  esac
done

# --- Resolve repo info ---
REPO_DIR="$(cd "$REPO_PATH" && git rev-parse --show-toplevel 2>/dev/null || echo "$REPO_PATH")"
REMOTE_URL="$(git -C "$REPO_DIR" remote get-url origin 2>/dev/null || true)"

# Handle both git@github.com:owner/repo.git and https://github.com/owner/repo.git
OWNER_REPO="$(echo "$REMOTE_URL" | sed -E 's|.*github.com[:/]||; s|\.git$||; s|/$||')"

if [[ -z "$OWNER_REPO" ]]; then
  echo "Error: could not determine OWNER/REPO from remote URL: $REMOTE_URL" >&2
  exit 1
fi

if ! [[ "$OWNER_REPO" =~ ^[^/]+/[^/]+$ ]]; then
  echo "Error: OWNER/REPO malformed (expected owner/repo): got '$OWNER_REPO' from '$REMOTE_URL'" >&2
  exit 1
fi

echo "=== Slopgate vs CodeRabbit Benchmark ==="
echo "Repo:       $OWNER_REPO"
echo "PR:         #$PR_NUMBER"
echo "Base:       $BASE_REF"
echo ""

# --- Step 1: Run slopgate (separate stdout from stderr) ---
echo "Running slopgate..." >&2
SLOPGATE_ERR_FILE="$(mktemp)"
SLOPGATE_JSON="$( "$SLOPGATE_BIN" --base "$BASE_REF" --format json -C "$REPO_DIR" 2>"$SLOPGATE_ERR_FILE" )" || {
  echo "Error: slopgate failed" >&2
  cat "$SLOPGATE_ERR_FILE" | head -5 >&2
  rm -f "$SLOPGATE_ERR_FILE"
  exit 1
}
SLOPGATE_ERR="$(cat "$SLOPGATE_ERR_FILE")"
rm -f "$SLOPGATE_ERR_FILE"
if [[ -n "$SLOPGATE_ERR" ]]; then
  echo "Warning: slopgate stderr:" >&2
  echo "$SLOPGATE_ERR" | head -3 >&2
fi

SLOPGATE_COUNT="$(echo "$SLOPGATE_JSON" | jq '.summary.total // 0')"
SLOPGATE_BLOCK="$(echo "$SLOPGATE_JSON" | jq '.summary.block // 0')"
SLOPGATE_WARN="$(echo "$SLOPGATE_JSON" | jq '.summary.warn // 0')"
SLOPGATE_INFO="$(echo "$SLOPGATE_JSON" | jq '.summary.info // 0')"

echo "Slopgate: $SLOPGATE_COUNT findings ($SLOPGATE_BLOCK block, $SLOPGATE_WARN warn, $SLOPGATE_INFO info)"

# --- Step 2: Fetch CodeRabbit comments ---
echo "Fetching CodeRabbit comments..." >&2
CR_ERR_FILE="$(mktemp)"
CR_COMMENTS_RAW="$(gh api "repos/$OWNER_REPO/pulls/$PR_NUMBER/comments" --paginate 2>"$CR_ERR_FILE")" || {
  echo "Error: gh api failed" >&2
  cat "$CR_ERR_FILE" | head -5 >&2
  rm -f "$CR_ERR_FILE"
  exit 1
}
rm -f "$CR_ERR_FILE"

# --paginate may output multiple JSON arrays; jq -s 'add' merges them.
CR_COMMENTS_RAW="$(echo "$CR_COMMENTS_RAW" | jq -s 'add')"

# Filter to CodeRabbit comments only
CR_COMMENTS="$(echo "$CR_COMMENTS_RAW" | jq '[.[] | select(.user.login | test("coderabbit|code-rabbit"; "i"))]')"
CR_COUNT="$(echo "$CR_COMMENTS" | jq 'length')"

echo "CodeRabbit: $CR_COUNT review comments"
echo ""

# --- Step 3: Normalize and compare using Python ---
TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

echo "$SLOPGATE_JSON" | jq -c '.findings[] | {file, line, rule_id, severity, message}' > "$TMPDIR/slopgate.json"
echo "$CR_COMMENTS" | jq -c '.[] | {path, line: (.line // .original_line), body: (.body | split("\n")[0][0:120]), id} | select(.line != null)' > "$TMPDIR/cr.json"

# Write Python script to a file to avoid shell interpolation issues
cat > "$TMPDIR/compare.py" << 'PYEOF'
import json, sys, os, re
from collections import Counter

fuzzy = int(os.environ.get("BENCHMARK_FUZZY_RANGE", "2"))
tmpdir = os.environ.get("BENCHMARK_TMPDIR", "/tmp")

with open(os.path.join(tmpdir, "slopgate.json")) as f:
    sg_findings = [json.loads(line) for line in f if line.strip()]

with open(os.path.join(tmpdir, "cr.json")) as f:
    cr_comments = [json.loads(line) for line in f if line.strip()]

# Maximum bipartite matching (DFS-based augmenting paths).
# A greedy first-fit loop can undercount overlaps when an early sg
# claims a cr that would be the only match for a later sg.
# Build bipartite graph: sg_idx -> [cr_idx, ...] where file matches
# and line proximity is within fuzzy range.

sys.setrecursionlimit(max(sys.getrecursionlimit(), len(sg_findings) * 2 + 100))

adj = {s: [] for s in range(len(sg_findings))}
for s, sf in enumerate(sg_findings):
    for c, cc in enumerate(cr_comments):
        if sf["file"] == cc["path"] and abs(sf["line"] - cc["line"]) <= fuzzy:
            adj[s].append(c)

match_cr = {}  # cr_idx -> sg_idx

def _dfs(s, seen):
    for c in adj[s]:
        if c in seen:
            continue
        seen.add(c)
        if c not in match_cr or _dfs(match_cr[c], seen):
            match_cr[c] = s
            return True
    return False

for s in range(len(sg_findings)):
    _dfs(s, set())

matched_sg = set(match_cr.values())
overlap = []
for c, s in match_cr.items():
    overlap.append({"sg": sg_findings[s], "cr": cr_comments[c]})

sg_only = [sf for i, sf in enumerate(sg_findings) if i not in matched_sg]
cr_only = [cc for i, cc in enumerate(cr_comments) if i not in match_cr]

# Output report
print(f"Overlap:       {len(overlap)} findings (both found)")
print(f"Slopgate-only: {len(sg_only)} findings")
print(f"CR-only:       {len(cr_only)} findings")
print()

if overlap:
    print("--- Overlap (both found) ---")
    for o in overlap:
        sf = o["sg"]
        cr = o["cr"]
        cr_msg = cr["body"][:80]
        print(f"  {sf['file']}:{sf['line']:<5} {sf['rule_id']:8} / CR: {cr_msg}")
    print()

if cr_only:
    print("--- CR-only (slopgate should investigate) ---")
    for cc in cr_only[:30]:
        print(f"  {cc['path']}:{cc['line']:<5} {cc['body'][:80]}")
    if len(cr_only) > 30:
        print(f"  ... ({len(cr_only) - 30} more)")
    print()

if sg_only:
    print("--- Slopgate-only (not in CR) ---")
    for sf in sg_only[:30]:
        print(f"  {sf['file']}:{sf['line']:<5} {sf['rule_id']:8} {sf['severity']:5} {sf['message'][:60]}")
    if len(sg_only) > 30:
        print(f"  ... ({len(sg_only) - 30} more)")
    print()

# Per-rule slopgate breakdown
rule_counts = Counter((f["rule_id"], f["severity"]) for f in sg_findings)
print("--- Slopgate per-rule breakdown ---")
for (rule, sev), count in rule_counts.most_common():
    print(f"  {rule:10} [{sev:5}] {count} findings")
print()

# JSON output — read repo/pr/base from environment to avoid injection
result = {
    "repo": os.environ.get("BENCHMARK_REPO", "unknown"),
    "pr": int(os.environ.get("BENCHMARK_PR", "0")),
    "base": os.environ.get("BENCHMARK_BASE", "main"),
    "slopgate": {
        "total": len(sg_findings),
        "block": sum(1 for f in sg_findings if f["severity"] == "block"),
        "warn": sum(1 for f in sg_findings if f["severity"] == "warn"),
        "info": sum(1 for f in sg_findings if f["severity"] == "info"),
    },
    "coderabbit": {"total": len(cr_comments)},
    "comparison": {"overlap": len(overlap), "slopgate_only": len(sg_only), "cr_only": len(cr_only)},
    "overlap_details": [{"file": o["sg"]["file"], "line": o["sg"]["line"], "rule_id": o["sg"]["rule_id"], "cr_summary": o["cr"]["body"][:120]} for o in overlap],
    "cr_only_details": [{"file": cc["path"], "line": cc["line"], "summary": cc["body"][:120]} for cc in cr_only],
    "sg_only_details": [{"file": sf["file"], "line": sf["line"], "rule_id": sf["rule_id"], "severity": sf["severity"], "message": sf["message"]} for sf in sg_only],
}
with open(os.path.join(tmpdir, "result.json"), "w") as f:
    json.dump(result, f, indent=2)
PYEOF

export BENCHMARK_TMPDIR="$TMPDIR"
export BENCHMARK_REPO="$OWNER_REPO"
export BENCHMARK_PR="$PR_NUMBER"
export BENCHMARK_BASE="$BASE_REF"
export BENCHMARK_FUZZY_RANGE="$FUZZY_LINE_RANGE"
if ! python3 "$TMPDIR/compare.py" > "$TMPDIR/report.txt"; then
  echo "Error: comparison script failed" >&2
  exit 1
fi

if [[ ! -f "$TMPDIR/report.txt" ]]; then
  echo "Error: report file not generated" >&2
  exit 1
fi

cat "$TMPDIR/report.txt"

if [[ -n "$OUTPUT_FILE" ]]; then
  if [[ ! -f "$TMPDIR/result.json" ]]; then
    echo "Error: JSON result file not generated" >&2
    exit 1
  fi
  cp "$TMPDIR/result.json" "$OUTPUT_FILE"
  echo "JSON report written to $OUTPUT_FILE"
fi