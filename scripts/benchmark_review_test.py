import importlib.util
import subprocess
import sys
import unittest
from pathlib import Path
from unittest.mock import patch


SCRIPT_PATH = Path(__file__).with_name("benchmark_review.py")
SPEC = importlib.util.spec_from_file_location("benchmark_review_under_test", SCRIPT_PATH)
assert SPEC and SPEC.loader
benchmark_review = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = benchmark_review
SPEC.loader.exec_module(benchmark_review)


def completed(stdout: str = "", returncode: int = 0, stderr: str = "") -> subprocess.CompletedProcess[str]:
    return subprocess.CompletedProcess(args=[], returncode=returncode, stdout=stdout, stderr=stderr)


class ParseArgsTest(unittest.TestCase):
    def test_rejects_negative_fuzzy_range(self) -> None:
        argv = ["benchmark_review.py", "/repo", "20", "--fuzzy-range", "-1"]
        with patch.object(sys, "argv", argv):
            with self.assertRaises(SystemExit):
                benchmark_review.parse_args()


class PrepareWorktreeTest(unittest.TestCase):
    def setUp(self) -> None:
        self.repo_root = Path("repo-root")
        self.pr_number = 20
        self.worktree_path = "slopgate-benchmark-test"

    def test_prepare_worktree_uses_local_requested_base_sha(self) -> None:
        pr_meta = {"base": {"ref": "main"}, "merged": False}

        with patch.object(benchmark_review.tempfile, "mkdtemp", return_value=self.worktree_path), \
            patch.object(benchmark_review.os, "getpid", return_value=4321), \
            patch.object(
                benchmark_review,
                "run_cmd",
                side_effect=[
                    completed(),
                    completed(stdout="abc123\n"),
                    completed(),
                ],
            ) as run_cmd:
            context = benchmark_review.prepare_worktree(self.repo_root, pr_meta, self.pr_number, "feature/base")

        self.assertEqual(context.target_ref, "refs/slopgate-benchmark/pr-20-4321")
        self.assertEqual(context.compare_base, "abc123")
        self.assertEqual(context.temp_ref, "refs/slopgate-benchmark/pr-20-4321")
        calls = [call.args[0] for call in run_cmd.call_args_list]
        self.assertEqual(calls[0], ["git", "-C", "repo-root", "fetch", "origin", "refs/pull/20/head:refs/slopgate-benchmark/pr-20-4321"])
        self.assertEqual(calls[1], ["git", "-C", "repo-root", "rev-parse", "--verify", "feature/base"])
        self.assertEqual(calls[2], ["git", "-C", "repo-root", "worktree", "add", "--detach", self.worktree_path, "refs/slopgate-benchmark/pr-20-4321"])

    def test_prepare_worktree_uses_fetch_head_for_unnamed_requested_base(self) -> None:
        pr_meta = {"base": {"ref": "main"}, "merged": False}

        with patch.object(benchmark_review.tempfile, "mkdtemp", return_value=self.worktree_path), \
            patch.object(benchmark_review.os, "getpid", return_value=4321), \
            patch.object(
                benchmark_review,
                "run_cmd",
                side_effect=[
                    completed(),
                    completed(returncode=1, stderr="missing"),
                    completed(),
                    completed(returncode=1, stderr="still missing"),
                    completed(stdout="fedcba\n"),
                    completed(),
                ],
            ):
            context = benchmark_review.prepare_worktree(self.repo_root, pr_meta, self.pr_number, "deadbeef")

        self.assertEqual(context.compare_base, "fedcba")

    def test_prepare_worktree_raises_and_cleans_up_unresolved_requested_base(self) -> None:
        pr_meta = {"base": {"ref": "main"}, "merged": False}

        with patch.object(benchmark_review.tempfile, "mkdtemp", return_value=self.worktree_path), \
            patch.object(benchmark_review.os, "getpid", return_value=4321), \
            patch.object(
                benchmark_review,
                "run_cmd",
                side_effect=[
                    completed(),
                    completed(returncode=1, stderr="missing"),
                    completed(stderr="fetch ok"),
                    completed(returncode=1, stderr="still missing"),
                    completed(returncode=1, stderr="no fetch head"),
                    completed(returncode=1, stderr="no worktree"),
                    completed(),
                ],
            ) as run_cmd, \
            patch.object(benchmark_review.shutil, "rmtree") as rmtree:
            with self.assertRaises(benchmark_review.BenchmarkError):
                benchmark_review.prepare_worktree(self.repo_root, pr_meta, self.pr_number, "unknown-base")

        calls = [call.args[0] for call in run_cmd.call_args_list]
        self.assertIn(["git", "-C", "repo-root", "worktree", "remove", "--force", self.worktree_path], calls)
        self.assertIn(["git", "-C", "repo-root", "update-ref", "-d", "refs/slopgate-benchmark/pr-20-4321"], calls)
        rmtree.assert_called_once_with(Path(self.worktree_path), ignore_errors=True)

    def test_prepare_worktree_merged_path_fetches_and_uses_parent_base(self) -> None:
        pr_meta = {"base": {"ref": "main"}, "merged": True, "merge_commit_sha": "merge-sha"}

        with patch.object(benchmark_review.tempfile, "mkdtemp", return_value=self.worktree_path), \
            patch.object(
                benchmark_review,
                "run_cmd",
                side_effect=[
                    completed(),
                    completed(stdout="merge-sha\n"),
                    completed(stdout="parent-sha\n"),
                    completed(),
                ],
            ) as run_cmd:
            context = benchmark_review.prepare_worktree(self.repo_root, pr_meta, self.pr_number, "")

        self.assertEqual(context.target_ref, "merge-sha")
        self.assertEqual(context.compare_base, "parent-sha")
        self.assertIsNone(context.temp_ref)
        calls = [call.args[0] for call in run_cmd.call_args_list]
        self.assertEqual(calls[0], ["git", "-C", "repo-root", "fetch", "origin", "main"])
        self.assertEqual(calls[1], ["git", "-C", "repo-root", "rev-parse", "--verify", "merge-sha"])
        self.assertEqual(calls[2], ["git", "-C", "repo-root", "rev-parse", "merge-sha^1"])


class MatchStreamTest(unittest.TestCase):
    def test_match_stream_is_one_to_one(self) -> None:
        sg_findings = [
            {"file": "a.js", "line": 10, "rule_id": "SLP001", "severity": "warn", "message": "one"},
            {"file": "a.js", "line": 10, "rule_id": "SLP002", "severity": "warn", "message": "two"},
        ]
        review_findings = [
            benchmark_review.ReviewFinding(path="a.js", line=10, body="review", item_id="1", source="coderabbit", meta={}),
        ]

        result = benchmark_review.match_stream(sg_findings, review_findings, 0)

        self.assertEqual(result["comparison"]["overlap"], 1)
        self.assertEqual(result["comparison"]["slopgate_only"], 1)
        self.assertEqual(result["comparison"]["review_only"], 0)

    def test_match_stream_reports_unmatched_items(self) -> None:
        sg_findings = [
            {"file": "a.js", "line": 10, "rule_id": "SLP001", "severity": "warn", "message": "one"},
            {"file": "b.js", "line": 20, "rule_id": "SLP002", "severity": "warn", "message": "two"},
        ]
        review_findings = [
            benchmark_review.ReviewFinding(path="a.js", line=10, body="review one", item_id="1", source="coderabbit", meta={}),
            benchmark_review.ReviewFinding(path="c.js", line=30, body="review two", item_id="2", source="coderabbit", meta={}),
        ]

        result = benchmark_review.match_stream(sg_findings, review_findings, 0)

        self.assertEqual(result["comparison"]["overlap"], 1)
        self.assertEqual(result["comparison"]["slopgate_only"], 1)
        self.assertEqual(result["comparison"]["review_only"], 1)


if __name__ == "__main__":
    unittest.main()
