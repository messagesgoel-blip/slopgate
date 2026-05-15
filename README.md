# slopgate

**slopgate** is a fast local gate for AI-generated code slop on git diffs.

It catches high-signal failure patterns before hosted review tools run: unfinished stubs, swallowed errors, unsafe SQL construction, missing test updates, weak auth checks, brittle regexes, and other recurring issues that are cheap to detect locally.

---

## Why use it

1. **Fast local feedback**: runs on staged diffs in milliseconds.
2. **Pre-commit quality floor**: blocks known high-risk patterns early.
3. **Complements hosted review**: lets tools like CodeRabbit focus on deeper semantic review.

---

## Install

```bash
go install github.com/messagesgoel-blip/slopgate/cmd/slopgate@latest
```

Requires Go **1.22+**.

### Local binary rebuild

This repo keeps a local `./slopgate` binary for shared hook infrastructure. After rule changes, rebuild it from the repo root:

```bash
go build -buildvcs=false -o slopgate ./cmd/slopgate
```

After merge, refresh the shared install used by workspace hooks:

```bash
install -m 0755 slopgate /srv/storage/shared/tools/bin/slopgate
```

---

## Quick start

```bash
# default mode is staged diff (same as --staged)
slopgate

# explicit staged scan (pre-commit usage)
slopgate --staged

# compare current branch against a base ref
slopgate --base main

# machine-readable output
slopgate --staged --format json

# list all registered rules
slopgate --list-rules
```

---

## CLI reference

| Flag | Description |
|---|---|
| `--staged` | Scan staged changes (`git diff --cached`) |
| `--base <ref>` | Scan `ref...HEAD` |
| `-C <dir>` | Run git from a specific directory |
| `--format text\|json` | Output format (default: `text`) |
| `--no-color` | Disable ANSI colors in text mode |
| `--config <path>` | Use a specific `.slopgate.toml` |
| `--list-rules` | Print rule catalog and exit |

`--staged` and `--base` are mutually exclusive.

---

## Exit codes

| Code | Meaning |
|---|---|
| `0` | No blocking findings (clean, or warn/info only) |
| `1` | One or more blocking findings |
| `2` | Tool/config/git error |

---

## Configuration

Create `.slopgate.toml` in repo root:

```toml
# Disable a rule
[rules.SLP014]
ignore = true

# Override severity: block | warn | info | off
[rules.SLP012]
severity = "warn"

# Ignore paths for a specific rule
[rules.SLP007]
ignore_paths = ["**/*_test.go"]
```

### Config discovery order

1. Path passed via `--config`
2. Auto-discovered `.slopgate.toml` while walking upward from working dir
3. Stop at repo root sentinel (`.git` or `go.mod`)
4. If not found, defaults are used

### `.slopgateignore`

Skip files entirely using glob patterns (one per line):

```text
vendor/**
**/migrations/**
```

---

## Integration

### Pre-commit hook

Add to `.git/hooks/pre-commit` or `.githooks/pre-commit`:

```bash
slopgate --staged --no-color
```

### CI

Typical CI usage:

```bash
slopgate --no-color --base origin/main
```

When using shallow clones, fetch full history (`fetch-depth: 0`) so base refs resolve correctly.

### Benchmarking

Compare Slopgate against review streams on a PR:

```bash
# Legacy wrapper (delegates to benchmark_review.py)
scripts/benchmark-coderabbit.sh /srv/storage/repo/whimsy 174

# Direct usage with full options
python3 scripts/benchmark_review.py /srv/storage/repo/whimsy 174 \
  --sentry-project api --sentry-project app \
  --output /tmp/benchmark-whimsy-174.json
```

Current benchmark behavior:

- uses an isolated temporary worktree, so dirty local branches do not poison results
- benchmarks open PRs against the actual PR head, not the caller's current checkout
- benchmarks merged PRs against the merge commit vs the base branch
- reports legacy `CodeRabbit all comments` overlap plus `actionable unresolved threads`
- optionally ingests Sentry findings with `--sentry-project api --sentry-project app`
- supports `--base-ref` override for comparing against a specific ref instead of PR base

### Comparing benchmarks

Track progress over time with the comparison script:

```bash
# Show trend for a specific repo
python3 scripts/benchmark-compare.py slopgate

# Compare two specific PRs
python3 scripts/benchmark-compare.py slopgate 16 20

# Compare two benchmark files directly
python3 scripts/benchmark-compare.py --file /tmp/bench1.json /tmp/bench2.json

# Show trend across all repos
python3 scripts/benchmark-compare.py --trend
```

Benchmarks are automatically archived to `/srv/storage/shared/slopgate-benchmarks/` by pre-commit, pre-push, and post-merge hooks.

---

## Rule catalog

Current rule set:

- **148 total rules**
- **10 AST-aware Go rules** (`SLP071`–`SLP080`)

Reserved IDs: **SLP004, SLP028, SLP029, SLP105**

### Rule families (high-level)

| Family | ID ranges (primary) | Focus |
|---|---|---|
| Core diff slop checks | `SLP001`–`SLP070` | test quality, code hygiene, safety, API/data smells |
| AST semantic checks | `SLP071`–`SLP080` | Go semantic hazards (nil, SQLi, races, ignored errors) |
| Extended parity checks | `SLP081`–`SLP142` | React/API/auth/audit/pagination/concurrency and newer overlap-driven checks |
| Multi-language semantic rules | `SLP202`–`SLP207` | High-signal bug detection (nil dereference, DB constraints, promise failures, transactions) |


For the complete authoritative list (ID + severity + description), run:

```bash
slopgate --list-rules
```

---

## Phase 2: CodeRabbit Parity Enhancements

Recent improvements to reach 45%+ CodeRabbit overlap:

**SLP001/SLP010 — Assertion expansion** (Jest, pytest, JUnit, Rust matchers)
- Detects `toBe`, `toEqual`, `toThrow`, `mock.assert`, `assertThrows`, `expect()`, `unwrap()`
- Catches untested code in both new and incremental test additions

**SLP098 — Route/handler detection** (Express, Next.js, Flask, Go)
- Express: `app.use()`, `app.param()`, `app.static()`
- Next.js: `export async function GET(...)`, `export const GET = ...`, `export default function handler(...)`
- Matches HTTP methods: GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS

**SLP099 — Response field changes** (additions AND deletions)
- Detects renamed, added, or removed response fields in API contracts
- Flags unmatched test updates on both additions and removals

**SLP100 — Stub detection** (with comment analysis)
- Recognizes TODO, FIXME, WIP, NotImplemented markers in both `//` and `/* */` comments
- Detects `return "placeholder" // TODO` patterns (placeholder strings with stub comments)
- Handles spaced phrases like "not implemented" and "not done"

---

## Adding a new rule

1. Add `pkg/rules/slpXXX.go` implementing `Rule` (or `SemanticRule` for AST rules).
2. Add `pkg/rules/slpXXX_test.go` with regression tests using `parseDiff`.
3. Register it in `pkg/rules/registry.go` (`Default()`).
4. Update `CHANGELOG.md`.
5. Ensure the README rule counts remain accurate.

---

## License

MIT
