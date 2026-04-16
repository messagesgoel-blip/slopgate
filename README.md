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
```

Exit codes:
- `0` - no blocking findings
- `1` - blocking findings present
- `2` - configuration or tool error

## Rules (v0.0.1)

| ID | Description | Languages |
|---|---|---|
| SLP001 | Test function with no assertion | Go |
| SLP005 | `.only` / `fdescribe` / `fit` committed | TypeScript, JavaScript |
| SLP012 | TODO/FIXME/HACK comment added in diff | all |
| SLP013 | Commented-out code block added in diff | all |
| SLP014 | Debug print left in non-test file | Go, TypeScript, JavaScript, Python |

More rules in v0.0.2: empty catch blocks, log-and-continue handlers, tautological assertions, env-var drift, unused new imports, panic/throw stub bodies.

## License

MIT.
