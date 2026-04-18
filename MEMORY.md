# Slopgate Memory

This file tracks key information about the slopgate project for future agent sessions.

## Project Overview

**Repository**: `/srv/storage/repo/slopgate`
**Purpose**: Pre-commit gate for catching AI-generated code failure patterns (slop)
**Positioning**: Runs locally in milliseconds before hosted AI review tools like CodeRabbit
**Language**: Go 1.22+
**Key Dependency**: BurntSushi/toml for config parsing

## Current Version

- **v0.0.6** (branch: `feat/semantic-rules-v0.0.6`)
- 26 rules (SLP001-SLP027, note: SLP004 was never assigned)

## Benchmark Status

- **Current overlap rate**: 7.3% (2 findings matched between slopgate 44 and CodeRabbit 16 on whimsy PR #118)
- **Maturity threshold**: 80% overlap to consider replacing CodeRabbit
- **Gap analysis**: slopgate catches syntactic patterns; CodeRabbit catches semantic/logic bugs
- **Benchmark archive**: `/srv/storage/shared/slopgate-benchmarks/`

## Rule Categories

### Syntactic Slop (v0.0.1-v0.0.4)
- SLP001-SLP023: All syntactic patterns (empty tests, unused imports, debug prints, etc.)
- Fast regex-based detection on staged diff

### Semantic Patterns (v0.0.6 additions)
- SLP024: Webhook ACK on failure (block) - catches `catch { console.error; res.status(200) }`
- SLP025: URL concatenation without validation (warn) - catches `${URL}${Path}` patterns
- SLP026: SQL NULL check without sentinel (warn) - catches `WHERE hash IS NOT NULL` without exclusion
- SLP027: Async function sync throw (warn) - catches `async function { throw }` patterns

### Noise Reduction (SLP017 tuning)
- Exempts HTTP status codes (200, 400, 500, etc.) in `.status()` context
- Exempts common pagination limits (10, 50, 100) in LIMIT/batch context
- Still flags truly magic numbers like `86.9` tax rate

## Key Files

- `cmd/slopgate/main.go` - CLI entrypoint
- `pkg/rules/registry.go` - Rule registration
- `pkg/rules/slp*.go` - Individual rule implementations
- `pkg/diff/parser.go` - Unified diff parsing
- `pkg/config/` - TOML config handling

## Integration Points

1. **Pre-commit**: `slopgate --staged --no-color`
2. **Pre-push**: Check if PR exists, run benchmark
3. **Post-merge**: Run benchmark on merged PRs
4. **codero-finish Phase 6.5**: After CodeRabbit poll, run slopgate benchmark

## Benchmark Scripts

- `/srv/storage/shared/agent-toolkit/bin/run-slopgate-benchmark` - Universal runner
- `/srv/storage/shared/agent-toolkit/bin/benchmark-aggregator` - Collects results

## Development Workflow

1. Create branch: `feat/semantic-rules-vX.Y.Z` or `fix/slpXXX-description`
2. Add rule in `pkg/rules/slpXXX.go` with `Rule` interface
3. Add tests in `pkg/rules/slpXXX_test.go` using `parseDiff(t, "...")`
4. Register in `pkg/rules/registry.go` `Default()` function
5. Update README rule table and version
6. Run `go test ./pkg/rules/...`
7. Update registry_test.go count

## GitHub Token

Available via gitconfig URL rewrite (see ~/.gitconfig for details)

## Next Improvement Areas

1. **Semantic rules expansion**: More patterns from CodeRabbit gap analysis
2. **Cross-language support**: Python async patterns, Go error handling patterns
3. **Config tuning**: Per-project rule severity profiles
4. **CI integration**: GitHub Actions workflow for PR checks

## Memory Updates

- 2026-04-18: Added SLP024-SLP027, tuned SLP017, version bump to v0.0.6