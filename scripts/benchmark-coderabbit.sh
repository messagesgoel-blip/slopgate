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
# Requires: gh (authenticated), jq, slopgate (on PATH or SLOPGATE_BIN)
set -euo pipefail

SLOPGATE_BIN="${SLOPGATE_BIN:-$(which slopgate 2>/dev/null || echo /srv/storage/shared/tools/bin/slopgate)}"
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
    --output) OUTPUT_FILE="$2"; shift 2 ;;
    --base) BASE_REF="$2"; shift 2 ;;
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

echo "=== Slopgate vs CodeRabbit Benchmark ==="
echo "Repo:       $OWNER_REPO"
echo "PR:         #$PR_NUMBER"
echo "Base:       $BASE_REF"
echo ""

# --- Step 1: Run slopgate ---
echo "Running slopgate..." >&2
SLOPGATE_JSON=$("$SLOPGATE_BIN" --base "$BASE_REF" --format json -C "$REPO_DIR" 2>/dev/null || echo '{"findings":[],"summary":{"total":0,"block":0,"warn":0,"info":0}}')

SLOPGATE_COUNT="$(echo "$SLOPGATE_JSON" | jq '.summary.total // 0')"
SLOPGATE_BLOCK="$(echo "$SLOPGATE_JSON" | jq '.summary.block // 0')"
SLOPGATE_WARN="$(echo "$SLOPGATE_JSON" | jq '.summary.warn // 0')"
SLOPGATE_INFO="$(echo "$SLOPGATE_JSON" | jq '.summary.info // 0')"

echo "Slopgate: $SLOPGATE_COUNT findings ($SLOPGATE_BLOCK block, $SLOPGATE_WARN warn, $SLOPGATE_INFO info)"

# --- Step 2: Fetch CodeRabbit comments ---
echo "Fetching CodeRabbit comments..." >&2
CR_COMMENTS_RAW="$(gh api "repos/$OWNER_REPO/pulls/$PR_NUMBER/comments" --paginate 2>/dev/null || echo '[]')"
# Handle case where --paginate outputs multiple concatenated JSON arrays
if echo "$CR_COMMENTS_RAW" | jq -e . >/dev/null 2>&1; then
  CR_COMMENTS_RAW="$(echo "$CR_COMMENTS_RAW" | jq '.')"
fi

# Filter to CodeRabbit comments only
CR_COMMENTS="$(echo "$CR_COMMENTS_RAW" | jq '[.[] | select(.user.login | test("coderabbit|code-rabbit"; "i"))]')"
CR_COUNT="$(echo "$CR_COMMENTS" | jq 'length')"

echo "CodeRabbit: $CR_COUNT review comments"
echo ""

# --- Step 3: Use Python for fuzzy line matching comparison ---
# Write normalized data to temp files, then use python for the comparison
# since awk fuzzy matching was unreliable with TSV containing colons in messages.

TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

echo "$SLOPGATE_JSON" | jq -c '.findings[] | {file, line, rule_id, severity, message}' > "$TMPDIR/slopgate.json"
echo "$CR_COMMENTS" | jq -c '.[] | {path, line: (.line // .original_line // 0), body: (.body | split("\n")[0][0:120]), id}' > "$TMPDIR/cr.json"

# Python comparison script
python3 << PYEOF > "$TMPDIR/report.txt" 2>/dev/null
import json, sys

fuzzy = $FUZZY_LINE_RANGE

with open("$TMPDIR/slopgate.json") as f:
    sg_findings = [json.loads(line) for line in f if line.strip()]

with open("$TMPDIR/cr.json") as f:
    cr_comments = [json.loads(line) for line in f if line.strip()]

# Match by file + line proximity
overlap = []
sg_only = list(sg_findings)  # copy, will remove matches
cr_matched = set()

for sf in sg_findings:
    matched_cr = None
    for i, cc in enumerate(cr_comments):
        if sf["file"] == cc["path"] and abs(sf["line"] - cc["line"]) <= fuzzy:
            matched_cr = cc
            cr_matched.add(i)
            break
    if matched_cr:
        overlap.append({"sg": sf, "cr": matched_cr})
        sg_only.remove(sf)

cr_only = [cc for i, cc in enumerate(cr_comments) if i not in cr_matched]

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
from collections import Counter
rule_counts = Counter((f["rule_id"], f["severity"]) for f in sg_findings)
print("--- Slopgate per-rule breakdown ---")
for (rule, sev), count in rule_counts.most_common():
    print(f"  {rule:10} [{sev:5}] {count} findings")
print()

# JSON output
result = {
    "repo": "$OWNER_REPO",
    "pr": $PR_NUMBER,
    "base": "$BASE_REF",
    "slopgate": {"total": len(sg_findings), "block": sum(1 for f in sg_findings if f["severity"] == "block"), "warn": sum(1 for f in sg_findings if f["severity"] == "warn"), "info": sum(1 for f in sg_findings if f["severity"] == "info")},
    "coderabbit": {"total": len(cr_comments)},
    "comparison": {"overlap": len(overlap), "slopgate_only": len(sg_only), "cr_only": len(cr_only)},
    "overlap_details": [{"file": o["sg"]["file"], "line": o["sg"]["line"], "rule_id": o["sg"]["rule_id"], "cr_summary": o["cr"]["body"][:120]} for o in overlap],
    "cr_only_details": [{"file": cc["path"], "line": cc["line"], "summary": cc["body"][:120]} for cc in cr_only],
    "sg_only_details": [{"file": sf["file"], "line": sf["line"], "rule_id": sf["rule_id"], "severity": sf["severity"], "message": sf["message"]} for sf in sg_only]
}
with open("$TMPDIR/result.json", "w") as f:
    json.dump(result, f, indent=2)
PYEOF

cat "$TMPDIR/report.txt"

if [[ -n "$OUTPUT_FILE" ]]; then
  cp "$TMPDIR/result.json" "$OUTPUT_FILE"
  echo "JSON report written to $OUTPUT_FILE"
fi