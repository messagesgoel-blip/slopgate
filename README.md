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

## Rules (v0.0.2)

| ID | Description | Default | Languages |
|---|---|---|---|
| SLP001 | Test function with no assertion | warn | Go |
| SLP002 | Tautological assertion (e.g. `assert.Equal(t, x, x)`) | block | Go, JS/TS, Python |
| SLP003 | Empty error handler (catch/except with no handling) | warn | Go, JS/TS, Python |
| SLP005 | `.only` / `fdescribe` / `fit` committed | block | TypeScript, JavaScript |
| SLP006 | Panic/throw stub body (`panic("not implemented")`) | block | Go, JS/TS, Python |
| SLP007 | Import added in diff but never used | warn | Go, JS/TS |
| SLP008 | Error logged but silently returned without recovery | warn | Go, JS/TS, Python |
| SLP009 | Env-var lookup without corresponding setup in diff | info | Go, JS/TS |
| SLP010 | Added lines in existing test contain no assertion | warn | Go |
| SLP011 | Test body is only assertions, no arrange/act | warn | Go, JS/TS, Python |
| SLP012 | TODO/FIXME/HACK comment added in diff | block | all |
| SLP013 | Commented-out code block added in diff | block | all |
| SLP014 | Debug print left in non-test file | block | Go, TypeScript, JavaScript, Python |

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
   - `ID() string` — return the stable rule ID (e.g. `"SLP015"`)
   - `Description() string` — one-line human description
   - `DefaultSeverity() Severity` — `SeverityBlock`, `SeverityWarn`, or `SeverityInfo`
   - `Check(d *diff.Diff) []Finding` — run detection, return findings
2. Create `pkg/rules/slpXXX_test.go` with regression tests using `parseDiff(t, "...")`
3. Register in `pkg/rules/registry.go` `Default()` function
4. Update the rule catalog table in this README

## License

MIT.
