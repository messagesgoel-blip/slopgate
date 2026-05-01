#!/usr/bin/env python3
"""benchmark_review.py — compare Slopgate findings against review streams.

This keeps the legacy CodeRabbit-vs-Slopgate benchmark output stable while
adding:
  - open-PR benchmarking against the actual PR head in an isolated worktree
  - actionable CodeRabbit scoring from unresolved review threads
  - optional Sentry-backed finding ingestion
"""

from __future__ import annotations

import argparse
import json
import os
import shutil
import subprocess
import sys
import tempfile
from dataclasses import dataclass
from pathlib import Path
from typing import Any


DEFAULT_SLOPGATE_BIN = os.environ.get("SLOPGATE_BIN", "/srv/storage/shared/tools/bin/slopgate")
DEFAULT_SENTRY_HELPER = os.environ.get("SLOPGATE_SENTRY_HELPER", "/srv/storage/shared/tools/bin/sentry-whimsy")
DEFAULT_FUZZY_RANGE = int(os.environ.get("BENCHMARK_FUZZY_RANGE", "2"))


class BenchmarkError(RuntimeError):
    pass


@dataclass
class ReviewFinding:
    path: str
    line: int
    body: str
    item_id: str
    source: str
    meta: dict[str, Any]

    def to_json(self) -> dict[str, Any]:
        data = {
            "path": self.path,
            "line": self.line,
            "body": self.body,
            "id": self.item_id,
            "source": self.source,
        }
        if self.meta:
            data.update(self.meta)
        return data


@dataclass
class WorktreeContext:
    repo_root: Path
    worktree_path: Path
    target_ref: str
    compare_base: str
    requested_base: str
    base_branch: str
    mode: str
    temp_ref: str | None = None

    def cleanup(self) -> None:
        subprocess.run(
            ["git", "-C", str(self.repo_root), "worktree", "remove", "--force", str(self.worktree_path)],
            check=False,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )
        if self.temp_ref:
            subprocess.run(
                ["git", "-C", str(self.repo_root), "update-ref", "-d", self.temp_ref],
                check=False,
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
            )
        shutil.rmtree(self.worktree_path, ignore_errors=True)


def run_cmd(cmd: list[str], *, cwd: Path | None = None, check: bool = True) -> subprocess.CompletedProcess[str]:
    proc = subprocess.run(
        cmd,
        cwd=str(cwd) if cwd else None,
        capture_output=True,
        text=True,
        check=False,
    )
    if check and proc.returncode != 0:
        raise BenchmarkError(
            f"command failed ({proc.returncode}): {' '.join(cmd)}\n"
            f"stdout:\n{proc.stdout}\n"
            f"stderr:\n{proc.stderr}"
        )
    return proc


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Compare Slopgate findings against CodeRabbit and Sentry review streams.")
    parser.add_argument("repo_path", help="Path to the target repository")
    parser.add_argument("pr_number", type=int, help="GitHub PR number")
    parser.add_argument("base_ref", nargs="?", default="", help="Base branch/ref override (defaults to PR base branch for open PRs)")
    parser.add_argument("--base", dest="base_opt", default="", help="Explicit base branch/ref override")
    parser.add_argument("--output", default="", help="Optional output JSON path (defaults to the system temp dir benchmark-<repo>-<pr>.json)")
    parser.add_argument("--fuzzy-range", type=int, default=DEFAULT_FUZZY_RANGE, help="Line matching tolerance")
    parser.add_argument("--sentry-project", action="append", default=[], help="Sentry project alias/slug to include (repeatable)")
    parser.add_argument("--sentry-stats-period", default="14d", help="Sentry issue stats period, e.g. 24h/7d/14d")
    parser.add_argument("--sentry-query", default="is:unresolved", help="Sentry issue search query")
    parser.add_argument("--sentry-helper", default=DEFAULT_SENTRY_HELPER, help="Path to sentry-whimsy helper")
    return parser.parse_args()


def trim_body(body: str) -> str:
    lines = (body or "").splitlines()
    first_line = lines[0].strip() if lines else ""
    return first_line[:120]


def parse_paginated_json(raw: str) -> Any:
    raw = raw.strip()
    if not raw:
        return []
    try:
        return json.loads(raw)
    except json.JSONDecodeError:
        parts: list[Any] = []
        for line in raw.splitlines():
            line = line.strip()
            if not line:
                continue
            parts.append(json.loads(line))
        if all(isinstance(p, list) for p in parts):
            merged: list[Any] = []
            for part in parts:
                merged.extend(part)
            return merged
        return parts


def repo_root(repo_path: str) -> Path:
    proc = run_cmd(["git", "-C", repo_path, "rev-parse", "--show-toplevel"])
    return Path(proc.stdout.strip())


def owner_repo(repo_root_path: Path) -> str:
    proc = run_cmd(["git", "-C", str(repo_root_path), "remote", "get-url", "origin"])
    remote = proc.stdout.strip()
    owner_repo_val = remote
    for token in ("github.com:", "github.com/"):
        before, separator, after = owner_repo_val.partition(token)
        if separator:
            owner_repo_val = after
            break
    owner_repo_val = owner_repo_val.removesuffix(".git").strip("/")
    if owner_repo_val.count("/") != 1:
        raise BenchmarkError(f"could not parse owner/repo from remote URL: {remote}")
    return owner_repo_val


def gh_api_json(args: list[str]) -> Any:
    proc = run_cmd(["gh", "api", *args])
    return parse_paginated_json(proc.stdout)


def gh_graphql_json(query: str, variables: dict[str, Any]) -> Any:
    cmd = ["gh", "api", "graphql", "-f", f"query={query}"]
    for key, value in variables.items():
        cmd.extend(["-F", f"{key}={value}"])
    proc = run_cmd(cmd)
    return json.loads(proc.stdout)


def fetch_pr_meta(owner_repo_name: str, pr_number: int) -> dict[str, Any]:
    return gh_api_json([f"repos/{owner_repo_name}/pulls/{pr_number}"])


def prepare_worktree(repo_root_path: Path, pr_meta: dict[str, Any], pr_number: int, requested_base: str) -> WorktreeContext:
    base_branch = pr_meta["base"]["ref"]
    merged = bool(pr_meta.get("merged"))
    worktree_path = Path(tempfile.mkdtemp(prefix=f"slopgate-benchmark-{pr_number}-"))

    def resolved_compare_base() -> str:
        if requested_base:
            verify_proc = run_cmd(
                ["git", "-C", str(repo_root_path), "rev-parse", "--verify", requested_base],
                check=False,
            )
            if verify_proc.returncode == 0:
                return requested_base
        run_cmd(["git", "-C", str(repo_root_path), "fetch", "origin", base_branch])
        return f"origin/{base_branch}"

    if merged:
        target_ref = pr_meta["merge_commit_sha"]
        if requested_base:
            compare_base = resolved_compare_base()
        else:
            compare_base = run_cmd(["git", "-C", str(repo_root_path), "rev-parse", f"{target_ref}^1"]).stdout.strip()
        mode = "merged_pr"
        temp_ref = None
    else:
        temp_ref = f"refs/slopgate-benchmark/pr-{pr_number}-{os.getpid()}"
        run_cmd(["git", "-C", str(repo_root_path), "fetch", "origin", f"refs/pull/{pr_number}/head:{temp_ref}"])
        target_ref = temp_ref
        compare_base = resolved_compare_base()
        mode = "open_pr_head"

    run_cmd(["git", "-C", str(repo_root_path), "worktree", "add", "--detach", str(worktree_path), target_ref])

    return WorktreeContext(
        repo_root=repo_root_path,
        worktree_path=worktree_path,
        target_ref=target_ref,
        compare_base=compare_base,
        requested_base=requested_base or base_branch,
        base_branch=base_branch,
        mode=mode,
        temp_ref=temp_ref,
    )


def run_slopgate(slug: str, worktree: WorktreeContext) -> tuple[dict[str, Any], str]:
    print("=== Slopgate vs Review Benchmark ===", file=sys.stderr)
    print(f"Repo:       {slug}", file=sys.stderr)
    print(f"Base:       {worktree.compare_base}", file=sys.stderr)
    if worktree.mode == "merged_pr":
        print(f"Merge SHA:  {worktree.target_ref}", file=sys.stderr)
    else:
        print(f"PR Head:    {worktree.target_ref}", file=sys.stderr)

    proc = run_cmd(
        [
            DEFAULT_SLOPGATE_BIN,
            "--base",
            worktree.compare_base,
            "--format",
            "json",
            "-C",
            str(worktree.worktree_path),
        ],
        check=False,
    )
    if proc.returncode not in (0, 1):
        raise BenchmarkError(f"slopgate failed with code {proc.returncode}\nstderr:\n{proc.stderr}")
    report = json.loads(proc.stdout or '{"findings":[],"summary":{"total":0,"block":0,"warn":0,"info":0}}')
    summary = report.get("summary", {})
    print(
        f"Slopgate: {summary.get('total', 0)} findings "
        f"({summary.get('block', 0)} block, {summary.get('warn', 0)} warn, {summary.get('info', 0)} info)",
        file=sys.stderr,
    )
    return report, proc.stderr.strip()


def collect_coderabbit_all(owner_repo_name: str, pr_number: int) -> list[ReviewFinding]:
    comments_raw = gh_api_json([f"repos/{owner_repo_name}/pulls/{pr_number}/comments", "--paginate"])
    findings: list[ReviewFinding] = []
    for item in comments_raw:
        login = ((item.get("user") or {}).get("login") or "").lower()
        if "coderabbit" not in login and "code-rabbit" not in login:
            continue
        line = item.get("line") or item.get("original_line")
        path = item.get("path")
        if not path or line is None:
            continue
        findings.append(
            ReviewFinding(
                path=path,
                line=int(line),
                body=trim_body(item.get("body", "")),
                item_id=str(item.get("id", "")),
                source="coderabbit_all",
                meta={"resolved": None},
            )
        )
    print(f"CodeRabbit: {len(findings)} review comments", file=sys.stderr)
    return findings


def collect_coderabbit_actionable(owner_repo_name: str, pr_number: int) -> list[ReviewFinding]:
    owner, repo = owner_repo_name.split("/", 1)
    query = """
query($owner:String!,$repo:String!,$pr:Int!,$cursor:String){
  repository(owner:$owner,name:$repo){
    pullRequest(number:$pr){
      reviewThreads(first:100, after:$cursor){
        pageInfo{hasNextPage endCursor}
        nodes{
          id
          isResolved
          comments(first:20){
            nodes{
              id
              path
              line
              originalLine
              body
              author{login}
            }
          }
        }
      }
    }
  }
}
"""
    cursor = None
    findings: list[ReviewFinding] = []

    while True:
        payload = gh_graphql_json(query, {"owner": owner, "repo": repo, "pr": pr_number, "cursor": cursor or ""})
        review_threads = payload["data"]["repository"]["pullRequest"]["reviewThreads"]
        for thread in review_threads["nodes"]:
            if thread.get("isResolved"):
                continue
            selected: ReviewFinding | None = None
            for comment in thread["comments"]["nodes"]:
                login = ((comment.get("author") or {}).get("login") or "").lower()
                if "coderabbit" not in login and "code-rabbit" not in login:
                    continue
                line = comment.get("line") or comment.get("originalLine")
                path = comment.get("path")
                if not path or line is None:
                    continue
                selected = ReviewFinding(
                    path=path,
                    line=int(line),
                    body=trim_body(comment.get("body", "")),
                    item_id=str(comment.get("id", "")),
                    source="coderabbit_actionable",
                    meta={"thread_id": thread.get("id", ""), "resolved": False},
                )
                break
            if selected:
                findings.append(selected)
        if not review_threads["pageInfo"]["hasNextPage"]:
            break
        cursor = review_threads["pageInfo"]["endCursor"]

    print(f"CodeRabbit actionable: {len(findings)} unresolved review threads", file=sys.stderr)
    return findings


def normalize_repo_path(raw_path: str, repo_files: list[str]) -> str | None:
    candidate = raw_path.replace("\\", "/").split("?", 1)[0]
    if candidate in repo_files:
        return candidate
    matches = [path for path in repo_files if candidate.endswith(path)]
    if not matches:
        return None
    return max(matches, key=len)


def iter_frame_candidates(node: Any) -> list[tuple[str, int]]:
    candidates: list[tuple[str, int]] = []
    if isinstance(node, dict):
        filename = node.get("filename") or node.get("abs_path")
        lineno = node.get("lineno")
        if filename and lineno and (node.get("in_app") is True or "/src/" in str(filename) or "\\src\\" in str(filename)):
            try:
                candidates.append((str(filename), int(lineno)))
            except (TypeError, ValueError):
                pass
        for value in node.values():
            candidates.extend(iter_frame_candidates(value))
    elif isinstance(node, list):
        for item in node:
            candidates.extend(iter_frame_candidates(item))
    return candidates


def collect_sentry_findings(
    helper_path: str,
    projects: list[str],
    stats_period: str,
    query: str,
    repo_root_path: Path,
) -> list[ReviewFinding]:
    if not projects:
        return []
    helper = Path(helper_path)
    if not helper.exists():
        raise BenchmarkError(f"sentry helper not found: {helper_path}")

    repo_files = [
        str(path.relative_to(repo_root_path)).replace("\\", "/")
        for path in repo_root_path.rglob("*")
        if path.is_file()
    ]
    findings: list[ReviewFinding] = []

    for project in projects:
        issues = json.loads(
            run_cmd(
                [
                    "python3",
                    str(helper),
                    "issues",
                    project,
                    "--query",
                    query,
                    "--stats-period",
                    stats_period,
                ]
            ).stdout
        )
        for issue in issues:
            issue_id = str(issue.get("id", ""))
            if not issue_id:
                continue
            events = json.loads(run_cmd(["python3", str(helper), "events", issue_id]).stdout)
            if not isinstance(events, list) or not events:
                continue
            first_event = events[0]
            event_id = str(first_event.get("id", ""))
            if not event_id:
                continue
            event = json.loads(run_cmd(["python3", str(helper), "event", issue_id, event_id]).stdout)
            location: tuple[str, int] | None = None
            for raw_path, line in iter_frame_candidates(event):
                normalized = normalize_repo_path(raw_path, repo_files)
                if normalized:
                    location = (normalized, line)
                    break
            if not location:
                continue

            title = issue.get("title") or issue.get("culprit") or issue.get("shortId") or "Sentry issue"
            location_path, location_line = location
            findings.append(
                ReviewFinding(
                    path=location_path,
                    line=int(location_line),
                    body=trim_body(str(title)),
                    item_id=issue_id,
                    source="sentry",
                    meta={
                        "project": project,
                        "event_id": event_id,
                        "short_id": issue.get("shortId", ""),
                    },
                )
            )

    print(f"Sentry:     {len(findings)} review findings", file=sys.stderr)
    return findings


def finding_to_compare_item(finding: dict[str, Any]) -> dict[str, Any]:
    return {
        "file": finding["file"],
        "line": int(finding["line"]),
        "rule_id": finding["rule_id"],
        "severity": finding["severity"],
        "message": finding["message"],
    }


def match_stream(
    sg_findings: list[dict[str, Any]],
    review_findings: list[ReviewFinding],
    fuzzy_range: int,
) -> dict[str, Any]:
    adjacency: dict[int, list[int]] = {i: [] for i in range(len(sg_findings))}
    for sg_idx, sg in enumerate(sg_findings):
        for rv_idx, review in enumerate(review_findings):
            if sg["file"] == review.path and abs(int(sg["line"]) - int(review.line)) <= fuzzy_range:
                adjacency[sg_idx].append(rv_idx)

    match_review: dict[int, int] = {}
    sys.setrecursionlimit(max(sys.getrecursionlimit(), len(sg_findings) * 2 + 100))

    def dfs(sg_idx: int, seen: set[int]) -> bool:
        for rv_idx in adjacency[sg_idx]:
            if rv_idx in seen:
                continue
            seen.add(rv_idx)
            if rv_idx not in match_review or dfs(match_review[rv_idx], seen):
                match_review[rv_idx] = sg_idx
                return True
        return False

    for sg_idx in range(len(sg_findings)):
        dfs(sg_idx, set())

    matched_sg = set(match_review.values())
    overlap_details = []
    for rv_idx, sg_idx in match_review.items():
        overlap_details.append(
            {
                "file": sg_findings[sg_idx]["file"],
                "line": int(sg_findings[sg_idx]["line"]),
                "rule_id": sg_findings[sg_idx]["rule_id"],
                "review_summary": review_findings[rv_idx].body,
                "review_source": review_findings[rv_idx].source,
            }
        )

    sg_only = [finding_to_compare_item(sg_findings[i]) for i in range(len(sg_findings)) if i not in matched_sg]
    review_only = [review_findings[i].to_json() for i in range(len(review_findings)) if i not in match_review]
    overlap = len(match_review)
    review_total = len(review_findings)
    return {
        "total": review_total,
        "comparison": {
            "overlap": overlap,
            "slopgate_only": len(sg_only),
            "review_only": len(review_only),
        },
        "coverage_pct": round((overlap / review_total * 100) if review_total else 0, 1),
        "precision_proxy_pct": round((overlap / len(sg_findings) * 100) if sg_findings else 0, 1),
        "overlap_details": overlap_details[:50],
        "review_only_details": review_only[:50],
        "sg_only_details": sg_only[:50],
    }


def combine_streams_by_location(streams: list[ReviewFinding]) -> list[ReviewFinding]:
    merged: dict[tuple[str, int], ReviewFinding] = {}
    for item in streams:
        key = (item.path, item.line)
        existing = merged.get(key)
        if not existing:
            merged[key] = ReviewFinding(
                path=item.path,
                line=item.line,
                body=item.body,
                item_id=item.item_id,
                source=item.source,
                meta={"sources": [item.source]},
            )
            continue
        sources = existing.meta.setdefault("sources", [])
        if item.source not in sources:
            sources.append(item.source)
        if item.body not in existing.body:
            existing.body = f"{existing.body} | {item.body}"[:120]
    return list(merged.values())


def main() -> int:
    args = parse_args()
    if not os.environ.get("GH_TOKEN"):
        auth_proc = run_cmd(["gh", "auth", "status"], check=False)
        if auth_proc.returncode != 0:
            raise BenchmarkError("GH_TOKEN not set and gh is not authenticated")

    requested_base = args.base_opt or args.base_ref
    root = repo_root(args.repo_path)
    slug = owner_repo(root)
    pr_meta = fetch_pr_meta(slug, args.pr_number)

    context = prepare_worktree(root, pr_meta, args.pr_number, requested_base)
    try:
        report, slopgate_stderr = run_slopgate(slug, context)
    finally:
        context.cleanup()

    sg_findings = [finding_to_compare_item(finding) for finding in report.get("findings", [])]
    all_comments = collect_coderabbit_all(slug, args.pr_number)
    actionable_comments = collect_coderabbit_actionable(slug, args.pr_number)
    sentry_findings = collect_sentry_findings(
        args.sentry_helper,
        args.sentry_project,
        args.sentry_stats_period,
        args.sentry_query,
        root,
    )
    combined_actionable = combine_streams_by_location(actionable_comments + sentry_findings)

    all_result = match_stream(sg_findings, all_comments, args.fuzzy_range)
    actionable_result = match_stream(sg_findings, actionable_comments, args.fuzzy_range)
    sentry_result = match_stream(sg_findings, sentry_findings, args.fuzzy_range)
    combined_result = match_stream(sg_findings, combined_actionable, args.fuzzy_range)

    result = {
        "repo": slug,
        "pr": args.pr_number,
        "base": context.compare_base,
        "requested_base": context.requested_base,
        "base_branch": context.base_branch,
        "merged": bool(pr_meta.get("merged")),
        "state": pr_meta.get("state", "unknown"),
        "benchmark_mode": context.mode,
        "checkout_ref": context.target_ref,
        "slopgate": report.get("summary", {}),
        "coderabbit": {"total": len(all_comments)},
        "coderabbit_actionable": {"total": len(actionable_comments)},
        "sentry": {"total": len(sentry_findings)},
        "actionable_plus_sentry": {"total": len(combined_actionable)},
        "comparison": all_result["comparison"],
        "scores": {
            "overlap_all": all_result["comparison"]["overlap"],
            "overlap_actionable": actionable_result["comparison"]["overlap"],
            "overlap_actionable_plus_sentry": combined_result["comparison"]["overlap"],
            "coverage_all_pct": all_result["coverage_pct"],
            "coverage_actionable_pct": actionable_result["coverage_pct"],
            "coverage_actionable_plus_sentry_pct": combined_result["coverage_pct"],
            "precision_proxy_all_pct": all_result["precision_proxy_pct"],
            "precision_proxy_actionable_pct": actionable_result["precision_proxy_pct"],
            "precision_proxy_actionable_plus_sentry_pct": combined_result["precision_proxy_pct"],
        },
        "streams": {
            "coderabbit_all": {"total": len(all_comments)},
            "coderabbit_actionable": {"total": len(actionable_comments)},
            "sentry": {"total": len(sentry_findings)},
            "actionable_plus_sentry": {"total": len(combined_actionable)},
        },
        "comparison_streams": {
            "coderabbit_all": all_result,
            "coderabbit_actionable": actionable_result,
            "sentry": sentry_result,
            "actionable_plus_sentry": combined_result,
        },
        "overlap_details": all_result["overlap_details"],
        "cr_only_details": all_result["review_only_details"],
        "sg_only_details": all_result["sg_only_details"],
        "actionable_overlap_details": actionable_result["overlap_details"],
        "cr_actionable_only_details": actionable_result["review_only_details"],
        "sentry_only_details": sentry_result["review_only_details"],
        "actionable_plus_sentry_only_details": combined_result["review_only_details"],
        "slopgate_stderr": slopgate_stderr,
    }

    print(
        f"Scores: overlap_all={result['scores']['overlap_all']} "
        f"overlap_actionable={result['scores']['overlap_actionable']} "
        f"overlap_actionable_plus_sentry={result['scores']['overlap_actionable_plus_sentry']}",
        file=sys.stderr,
    )

    default_output_path = Path(tempfile.gettempdir()) / f"benchmark-{slug.replace('/', '-')}-{args.pr_number}.json"
    output_path = args.output or str(default_output_path)
    payload = json.dumps(result, indent=2)
    output_file = Path(output_path)
    output_file.parent.mkdir(parents=True, exist_ok=True)
    output_file.write_text(payload + "\n", encoding="utf-8")
    print(f"JSON report: {output_path}", file=sys.stderr)
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except BenchmarkError as exc:
        print(f"Error: {exc}", file=sys.stderr)
        raise SystemExit(1)
