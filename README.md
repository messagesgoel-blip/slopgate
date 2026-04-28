# slopgate

A pre-commit gate that catches the specific failure modes AI coding agents produce.

Tests that mock everything and assert nothing. Unused imports the AI added "just in case". Catch blocks that swallow errors. TODO bodies committed to main. Debug prints left in production files. Commented-out "safety net" code blocks.

slopgate is not a general-purpose linter. It is aimed at a narrow, specific target: the recurring failure patterns that AI-generated code produces that human reviewers and generic linters miss. It runs on the staged diff, fast (<2s on a 500-line diff), with one dependency beyond the Go standard library (BurntSushi/toml for config).

Positioned next to tools like CodeRabbit and Sourcery, not instead of them. slopgate runs locally in milliseconds, before your hosted AI review burns its rate limit on slop a regex could have caught.

## Install

```
go install github.com/messagesgoel-blip/slopgate/cmd/slopgate@latest
```

Requires Go 1.22+.

## Use

```
# scan staged changes (pre-commit mode)
slopgate --staged

# scan a branch against main
slopgate --base main

# JSON output for tooling
slopgate --staged --format json

# list all rules
slopgate --list-rules

# use a specific config file
slopgate --staged --config .slopgate.toml
```

### Pre-commit hook

Add to `.git/hooks/pre-commit` or `.githooks/pre-commit`:

```bash
slopgate --staged --no-color
```

Exit codes:
- `0` - no blocking findings
- `1` - blocking findings present
- `2` - configuration or tool error

### GitHub Actions CI

slopgate runs automatically in CI on every push to main and on pull requests. The workflow:

1. Fetches full git history (`fetch-depth: 0`) so base refs are available
2. Builds and vets the code
3. Runs `slopgate --no-color --base origin/main` to compare PR changes against the target branch

For a complete CI setup example, see `.github/workflows/ci.yml`.

## Rules (v0.0.13)

| ID | Description | Default | Languages |
|---|---|---|---|
| SLP001 | Test function with no assertion | warn | Go, JS/TS, Python, Java, Rust |
| SLP002 | Tautological assertion (e.g. `assert.Equal(t, x, x)`) | block | Go, JS/TS, Python, Java, Rust |
| SLP003 | Empty error handler (catch/except with no handling) | warn | Go, JS/TS, Python, Java, Rust |
| SLP005 | `.only` / `fdescribe` / `fit` / `@Disabled` / `@Ignore` committed | block | JS/TS, Java |
| SLP006 | Panic/throw stub body (`panic("not implemented")`) | block | Go, JS/TS, Python, Java, Rust |
| SLP007 | Import added in diff but never used | warn | Go, JS/TS, Python, Java, Rust |
| SLP008 | Error logged but silently returned without recovery | warn | Go, JS/TS, Python, Java, Rust |
| SLP009 | Env-var lookup without corresponding setup in diff | info | Go, JS/TS, Python, Java, Rust |
| SLP010 | Added lines in existing test contain no assertion | warn | Go, JS/TS, Python, Java, Rust |
| SLP011 | Test body is only assertions, no arrange/act | warn | Go |
| SLP012 | TODO/FIXME/HACK comment added in diff | block | all |
| SLP013 | Commented-out code block added in diff | block | all |
| SLP014 | Debug print left in non-test file | block | Go, JS/TS, Python, Java, Rust |
| SLP015 | Linter-suppression comment added instead of fixing the issue | warn | Go, JS/TS, Python, Java, Rust |
| SLP016 | Variable shadows an outer-scope declaration with the same name | warn | Go, JS/TS, Python, Java, Rust |
| SLP017 | Unexplained numeric literal — define a named constant instead | info | Go, JS/TS, Python, Java, Rust |
| SLP018 | Overly broad catch/except catches base exception type | warn | Java, Python |
| SLP019 | Unreachable code after return/throw/panic/break/continue | warn | Go, JS/TS, Python, Java, Rust |
| SLP020 | Insecure random or weak hash — use cryptographic alternative | info* | Go, JS/TS, Python, Java |
| SLP021 | Mixed camelCase and snake_case naming in the same hunk | info | Go, JS/TS, Python, Java, Rust |
| SLP022 | fmt.Errorf uses %v/%s with error arg instead of %w for wrapping | warn | Go |
| SLP023 | Bare type assertion without comma-ok guard panics on mismatch | warn | Go |
| SLP024 | Catch block returns 2xx status after logging error — webhook callers will not retry | block | JS/TS |
| SLP025 | URL concatenation without path validation — could produce malformed URLs | warn | JS/TS |
| SLP026 | SQL NULL check without sentinel exclusion — consider excluding placeholder values | warn | SQL, JS/TS |
| SLP027 | Async function throws synchronously — use return Promise.reject for consistent error handling | warn | JS/TS |
| SLP030 | Query .only/.first/.last without sentinel exclusion — could return placeholder record | warn | JS/TS, Python, Go |
| SLP031 | Documentation indicates external code intake without license validation | warn | all |
| SLP032 | React/TypeScript component may have type or accessibility issues | warn | TSX |
| SLP033 | Missing import statement for referenced type/function | warn | JS/TS |
| SLP034 | Potential state management anti-pattern detected | warn | JS/TS |
| SLP035 | Code quality or style issue detected (console/debugger, TODO without ticket, trailing whitespace, long lines) | warn | all |

| SLP091 | Hardcoded date/time in test fixture (will expire) | block | JS/TS, Go, Python, SQL |
| SLP092 | Mock return shape doesn't match API envelope | warn | JS/TS |
| SLP093 | New mock/setup without corresponding new assertion | warn | all |
| SLP094 | Shell command with \|\| true (silent failure) | block | sh, bash, Makefile, CI YAML |
| SLP095 | Try/catch returns silently without error handling | block | JS/TS, Python, Java |
| SLP096 | Shell script missing set -e (error propagation) | warn | sh, bash |
| SLP097 | Response destructuring vs API envelope mismatch | warn | JS/TS |
| SLP098 | New route/handler without corresponding test | warn | Go, JS/TS, Python, Java |
| SLP099 | Response field changed without test update | warn | Go, JS/TS |
| SLP100 | Function returns zero value with no side effects (no-op) | block | Go, JS/TS, Python, Java, Rust |
| SLP101 | Dead feature flag (both branches identical) | warn | JS/TS, Go, Java |
| SLP102 | Async function with no await (stub) | warn | JS/TS |
| SLP103 | Hardcoded timeout duration | info | Go, JS/TS, Python |
| SLP104 | Hardcoded buffer/size limit | info | Go, JS/TS |
| SLP106 | Resource acquired without release/close in scope | warn | Go, JS/TS, Python, Java, Rust |
| SLP107 | Cleanup/destroy only in error path, missing on success | block | Go, JS/TS, Python |
| SLP108 | Open/connect without defer close or timeout | block | Go, JS/TS |
| SLP109 | Duplicate function body (>80% identical) | warn | Go, JS/TS, Python, Java |
| SLP110 | Duplicate file (>60% identical imports/declarations) | warn | all |
| SLP111 | Binary/executable committed without .gitignore | block | all |
| SLP112 | Generated file committed without corresponding source | warn | Go, JS/TS, protobuf |
| SLP113 | Source file changed without corresponding test update | warn | Go, JS/TS, Python, Java, Kotlin |
| SLP114 | Error-returning function called as statement — check the error return | warn | Go |
| SLP115 | Narrow extension check — add broader extension coverage | info | Go, JS/TS, Python |
| SLP116 | Regex contains nested quantifiers — potential ReDoS vulnerability | warn | Go, JS/TS, Python |
| SLP117 | Unanchored regex — add anchor to prevent unintended substring matches | info | Go, JS/TS |
| SLP118 | Numeric index access without length guard — may panic on empty collection | block | Go, JS/TS, Python |
| SLP119 | TrimSuffix/TrimPrefix result used without checking if suffix/prefix was present | warn | Go, JS/TS, Python |
| SLP120 | Value discarded with `_ = expr` — consider using the value | warn | Go |

\* SLP020 escalates to **warn** when security-context keywords (password, token, secret, key, session, nonce, salt, credential, auth) appear nearby.

### v0.0.13 Changes

- **SLP113-SLP120**: 8 new AI-slop detection rules:
  - SLP113: Source file changed without test update
  - SLP114: Error-returning function called as statement
  - SLP115: Narrow extension check (e.g., `.js` without `.mjs`/`.cjs`)
  - SLP116: Nested quantifier regex (ReDoS vulnerability)
  - SLP117: Unanchored regex (missing `^`/`$`/`\b`)
  - SLP118: Index access without length guard (panic risk)
  - SLP119: TrimSuffix/TrimPrefix without presence check
  - SLP120: Value discarded with `_ = expr`

Total: 116 rules

### v0.0.12 Changes

- **SLP091-SLP093**: Test/mock/fixture quality rules (hardcoded dates, mock envelope mismatches, mock without assertion)
- **SLP094-SLP096**: Silent failure detection (shell \|\| true, empty catch handlers, missing set -e)
- **SLP097-SLP099**: API contract/route testing rules (destructuring vs envelope, route without test, field change without test update)
- **SLP100-SLP102**: Stub/incomplete implementation detection (no-op functions, dead feature flags, async without await)
- **SLP103-SLP104**: Magic literal detection expansion (timeout durations, buffer sizes)
- **SLP106-SLP108**: Resource management rules (acquire without release, cleanup only in error path, open without defer/timeout)
- **SLP109-SLP110**: Code duplication detection (similar function bodies, similar files)
- **SLP111-SLP112**: Binary/asset hygiene (binary without .gitignore, generated files without source)

Total: 108 rules

### v0.0.11 Changes

- **CI Integration**: slopgate now runs in GitHub Actions CI with `--base origin/main` to compare PR changes against the target branch
- Full git history fetched (`fetch-depth: 0`) to ensure base refs are available for diff comparison

### v0.0.10 Changes

- **SLP036-SLP045**: New semantic rules for CodeRabbit parity (query methods, transaction scoping, webhook patterns)
- **SLP046-SLP070**: 25 new rules for AI-slop detection (file cohesion, redundant comments, SQL injection, hardcoded secrets, dynamic code execution, concurrent maps, resource leaks, etc.)
- **SLP071-SLP080**: 10 new AST-aware semantic rules using go/parser + go/types:
  - SLP071: Type assertion without comma-ok idiom
  - SLP072: Potential nil pointer dereference
  - SLP073: Resource acquired without defer close
  - SLP074: Loop variable captured by goroutine (race condition)
  - SLP075: Weak cryptographic functions (md5, sha1, DES, RC4)
  - SLP076: SQL built with string concatenation
  - SLP077: Hardcoded credentials detected in AST
  - SLP078: Select on potentially closed channel
  - SLP079: Ignored error returns from known functions
  - SLP080: Interface with single implementation (possible over-abstraction)
- **SLP081-SLP090**: 10 new rules for CodeRabbit parity:
  - SLP081: React component in TSX/JSX missing React import
  - SLP082: JSX list items missing React key prop
  - SLP083: useCallback/useMemo missing dependency array
  - SLP084: useEffect without cleanup for event listeners/timers
  - SLP085: SQL built with string concatenation or template literals
  - SLP086: Missing authorization check on sensitive endpoint
  - SLP087: Webhook handler without timeout configuration
  - SLP088: Hardcoded credentials in config/settings files
  - SLP089: Exported function/class/module missing docstring
  - SLP090: API route without error handling

Total: 87 rules

### v0.0.9 Changes

- **SLP031-035**: React/TSX component issues, missing imports, state anti-patterns, code quality
- **SLP036-SLP045**: Semantic rules for API docs, transaction scoping, webhook patterns, query methods
- **SLP046-SLP070**: 25 new AI-slop detection rules

### v0.0.7 Changes

- **SLP030**: New rule catching ORM/query methods (.only, .first, .last, .findOne) that select single records without filtering sentinel placeholder values.

## Configuration

Create a `.slopgate.toml` in your repo root to configure rules:

```toml
# Disable a rule entirely
[rules.SLP014]
ignore = true

# Change severity (block, warn, info, off)
[rules.SLP012]
severity = "warn"

# Ignore specific paths for a rule
[rules.SLP007]
ignore_paths = ["**/*_test.go"]
```

### Config discovery

1. `--config PATH` if provided on the command line
2. Walk up from the working directory, stopping at the repo root (`.git` or `go.mod` sentinel), looking for `.slopgate.toml`
3. No config found — all rules use their default severity

### `.slopgateignore`

Add a `.slopgateignore` file (one glob per line) to skip files entirely:

```text
vendor/**
**/migrations/**
```

## Adding a new rule

1. Create `pkg/rules/slpXXX.go` implementing the `Rule` interface:
   - `ID() string` — return the stable rule ID (e.g. `"SLP016"`)
   - `Description() string` — one-line human description
   - `DefaultSeverity() Severity` — `SeverityBlock`, `SeverityWarn`, or `SeverityInfo`
   - `Check(d *diff.Diff) []Finding` — run detection, return findings
2. Create `pkg/rules/slpXXX_test.go` with regression tests using `parseDiff(t, "...")`
3. Register in `pkg/rules/registry.go` `Default()` function
4. Update the rule catalog table in this README

## License

MIT.
