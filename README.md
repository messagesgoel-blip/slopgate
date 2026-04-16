# slopgate

A pre-commit gate that catches the specific failure modes AI coding agents produce.

Tests that mock everything and assert nothing. Unused imports the AI added "just in case". Catch blocks that swallow errors. TODO bodies committed to main. Debug prints left in production files. Commented-out "safety net" code blocks.

slopgate is not a general-purpose linter. It is aimed at a narrow, specific target: the recurring failure patterns that AI-generated code produces that human reviewers and generic linters miss. It runs on the staged diff, fast (<2s on a 500-line diff), with zero dependencies beyond the Go standard library.

Positioned next to tools like CodeRabbit and Sourcery, not instead of them. slopgate runs locally in milliseconds, before your hosted AI review burns its rate limit on slop a regex could have caught.

## Install

```
go install github.com/messagesgoel-blip/slopgate/cmd/slopgate@latest
```

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
```

Exit codes:
- `0` - no blocking findings
- `1` - blocking findings present
- `2` - configuration or tool error

## Rules (v0.0.2)

| ID | Description | Languages |
|---|---|---|
| SLP001 | Test function with no assertion | Go |
| SLP002 | Tautological assertion (e.g. `assert.Equal(t, x, x)`) | Go |
| SLP003 | Empty error handler (catch/except with no handling) | Go, JS/TS, Python |
| SLP005 | `.only` / `fdescribe` / `fit` committed | TypeScript, JavaScript |
| SLP006 | Panic/throw stub body (`panic("not implemented")`) | Go, JS/TS, Python |
| SLP007 | Import added in diff but never used | Go, JS/TS |
| SLP008 | Error logged but silently returned without recovery | Go, JS/TS, Python |
| SLP009 | Env-var lookup without corresponding setup in diff | Go, JS/TS |
| SLP010 | Added lines in existing test contain no assertion | Go |
| SLP011 | Test body is only assertions, no arrange/act | Go, JS/TS, Python |
| SLP012 | TODO/FIXME/HACK comment added in diff | all |
| SLP013 | Commented-out code block added in diff | all |
| SLP014 | Debug print left in non-test file | Go, TypeScript, JavaScript, Python |

## Configuration

Create a `.slopgate.toml` in your repo root to configure rules:

```toml
# Disable a rule entirely
[rules.SLP014]
ignore = true

# Change severity (block, warn, info)
[rules.SLP012]
severity = "warn"

# Ignore specific paths for a rule
[rules.SLP007]
ignore_paths = ["**/*_test.go"]
```

## License

MIT.
