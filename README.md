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

## Rules (v0.0.10)

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

\* SLP020 escalates to **warn** when security-context keywords (password, token, secret, key, session, nonce, salt, credential, auth) appear nearby.

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
