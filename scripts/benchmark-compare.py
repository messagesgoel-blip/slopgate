#!/usr/bin/env python3
"""benchmark-compare.py — compare slopgate benchmark results across time or PRs.

Usage:
  # Compare all benchmarks for a repo
  benchmark-compare.py slopgate

  # Compare benchmarks for specific PRs
  benchmark-compare.py slopgate 16 20

  # Compare two specific benchmark files
  benchmark-compare.py --file bench1.json bench2.json

  # Show trend over time for all repos
  benchmark-compare.py --trend

Outputs a markdown table suitable for PR descriptions or reports.
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from typing import Any

BENCHMARK_DIR = Path("/srv/storage/shared/slopgate-benchmarks")


def load_benchmark(path: Path) -> dict[str, Any] | None:
    """Load a single benchmark JSON file."""
    try:
        with open(path) as f:
            data = json.load(f)
        if "slopgate" not in data:
            return None
        return data
    except (json.JSONDecodeError, KeyError):
        # Skip malformed benchmark files silently
        return None


def collect_benchmarks(repo: str | None = None, pr_nums: list[int] | None = None) -> list[dict[str, Any]]:
    """Collect benchmarks from the shared directory."""
    results = []
    if not BENCHMARK_DIR.exists():
        return results

    for f in sorted(BENCHMARK_DIR.glob("*.json")):
        data = load_benchmark(f)
        if data is None:
            continue
        if repo and data.get("repo") != repo:
            continue
        if pr_nums and data.get("pr") not in pr_nums:
            continue
        data["_source"] = str(f)
        results.append(data)

    return results


def format_table(benchmarks: list[dict[str, Any]], title: str = "") -> str:
    """Format benchmarks as a markdown table."""
    if not benchmarks:
        return f"## {title}\n\nNo benchmarks found.\n"

    lines = [f"## {title}\n" if title else ""]
    lines.append("| Repo | PR | Slopgate | CodeRabbit | Actionable | Sentry | Combined | Overlap | Ov% |")
    lines.append("|------|-----|----------|------------|------------|--------|----------|---------|-----|")

    for b in benchmarks:
        repo = b.get("repo", "?")
        pr = b.get("pr", "?")
        sg = b.get("slopgate", {}).get("total", 0)
        cr = b.get("coderabbit", {}).get("total", 0)
        cr_act = b.get("coderabbit_actionable", {}).get("total", 0)
        sentry = b.get("sentry", {}).get("total", 0)
        combined = b.get("actionable_plus_sentry", {}).get("total", 0)
        scores = b.get("scores", {})
        overlap = scores.get("overlap_all", 0)
        overlap_pct = scores.get("coverage_all_pct", 0)

        lines.append(f"| {repo} | #{pr} | {sg} | {cr} | {cr_act} | {sentry} | {combined} | {overlap} | {overlap_pct:.1f}% |")

    # Totals
    total_sg = sum(b.get("slopgate", {}).get("total", 0) for b in benchmarks)
    total_cr = sum(b.get("coderabbit", {}).get("total", 0) for b in benchmarks)
    total_act = sum(b.get("coderabbit_actionable", {}).get("total", 0) for b in benchmarks)
    total_sentry = sum(b.get("sentry", {}).get("total", 0) for b in benchmarks)
    total_combined = sum(b.get("actionable_plus_sentry", {}).get("total", 0) for b in benchmarks)
    total_overlap = sum(b.get("scores", {}).get("overlap_all", 0) for b in benchmarks)

    lines.append(f"| **Total** | | **{total_sg}** | **{total_cr}** | **{total_act}** | **{total_sentry}** | **{total_combined}** | **{total_overlap}** | |")
    lines.append("")

    return "\n".join(lines)


def show_trend() -> str:
    """Show benchmark trends over time."""
    all_benchmarks = collect_benchmarks()
    if not all_benchmarks:
        return "No benchmarks found."

    # Group by repo
    by_repo: dict[str, list[dict[str, Any]]] = {}
    for b in all_benchmarks:
        repo = b.get("repo", "unknown")
        by_repo.setdefault(repo, []).append(b)

    lines = ["# Slopgate Benchmark Trends\n"]

    for repo, benchmarks in sorted(by_repo.items()):
        # Sort by PR number
        benchmarks.sort(key=lambda x: x.get("pr", 0))
        lines.append(format_table(benchmarks, title=f"Repo: {repo}"))

    # Overall summary
    lines.append(format_table(all_benchmarks, title="All Benchmarks"))

    return "\n".join(lines)


def compare_files(file1: str, file2: str) -> str:
    """Compare two specific benchmark files."""
    b1 = load_benchmark(Path(file1))
    b2 = load_benchmark(Path(file2))

    if not b1 or not b2:
        return "Error: Could not load one or both benchmark files."

    lines = ["# Benchmark Comparison\n"]
    lines.append("| Metric | Before | After | Change |")
    lines.append("|--------|--------|-------|--------|")

    metrics = [
        ("Slopgate Total", "slopgate.total"),
        ("CodeRabbit Total", "coderabbit.total"),
        ("CodeRabbit Actionable", "coderabbit_actionable.total"),
        ("Sentry Total", "sentry.total"),
        ("Actionable+Sentry", "actionable_plus_sentry.total"),
        ("Overlap (all)", "scores.overlap_all"),
        ("Overlap (actionable)", "scores.overlap_actionable"),
        ("Overlap (actionable+sentry)", "scores.overlap_actionable_plus_sentry"),
        ("Coverage (all %)", "scores.coverage_all_pct"),
        ("Coverage (actionable %)", "scores.coverage_actionable_pct"),
    ]

    for label, path in metrics:
        v1 = _get_nested(b1, path, 0)
        v2 = _get_nested(b2, path, 0)
        if isinstance(v1, (int, float)) and isinstance(v2, (int, float)):
            diff = v2 - v1
            sign = "+" if diff > 0 else ""
            lines.append(f"| {label} | {v1} | {v2} | {sign}{diff} |")
        else:
            lines.append(f"| {label} | {v1} | {v2} | - |")

    lines.append("")
    return "\n".join(lines)


def _get_nested(data: dict[str, Any], path: str, default: Any = None) -> Any:
    """Get a nested value from a dict using dot notation."""
    keys = path.split(".")
    current = data
    for key in keys:
        if isinstance(current, dict):
            current = current.get(key, default)
        else:
            return default
    return current


def main() -> int:
    parser = argparse.ArgumentParser(description="Compare slopgate benchmark results")
    parser.add_argument("repo", nargs="?", help="Repo name to filter by")
    parser.add_argument("pr_nums", nargs="*", type=int, help="PR numbers to filter by")
    parser.add_argument("--file", nargs=2, metavar=("FILE1", "FILE2"), help="Compare two specific benchmark files")
    parser.add_argument("--trend", action="store_true", help="Show trend over time for all repos")
    parser.add_argument("--output", "-o", help="Write output to file instead of stdout")

    args = parser.parse_args()

    if args.file and len(args.file) >= 2:
        output = compare_files(args.file[0], args.file[1])
    elif args.trend:
        output = show_trend()
    else:
        benchmarks = collect_benchmarks(args.repo, args.pr_nums if args.pr_nums else None)
        title = "Slopgate Benchmarks"
        if args.repo:
            title += f" — {args.repo}"
        if args.pr_nums:
            title += f" (PRs: {', '.join(f'#{p}' for p in args.pr_nums)})"
        output = format_table(benchmarks, title=title)

    if args.output:
        Path(args.output).write_text(output)
        print(f"Written to {args.output}", file=sys.stderr)
    else:
        print(output)

    return 0


if __name__ == "__main__":
    sys.exit(main())
