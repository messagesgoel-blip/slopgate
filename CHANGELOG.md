# Changelog

## v0.0.14 (2026-04-28)

7 new AI-slop detection rules:

- **SLP121**: Sensitive access mutation may be missing tenant/membership authorization guard
- **SLP122**: Async polling/retry logic added without cancellation or in-flight guard
- **SLP123**: Offset pagination on mutable ordering may drift — prefer cursor/keyset or stable tiebreaker
- **SLP124**: External call uses request/input payload without nearby validation guard
- **SLP125**: Share/role/access mutation without nearby audit logging call
- **SLP126**: Migration adds *_id reference without index — add CREATE INDEX for join/cascade performance
- **SLP127**: slopgate rule implementation changed without corresponding test diff update

Total: 123 rules

## v0.0.13 (2026-04-27)

8 new AI-slop detection rules:

- **SLP113**: Source file changed without test update
- **SLP114**: Error-returning function called as statement — check the error return
- **SLP115**: Narrow extension check — add broader extension coverage (e.g., `.js` without `.mjs`/`.cjs`)
- **SLP116**: Regex nested quantifiers — potential ReDoS vulnerability
- **SLP117**: Unanchored regex — add `^`/`$`/`\b` anchor to prevent substring matches
- **SLP118**: Numeric index access without length guard — panic risk on empty collection
- **SLP119**: TrimSuffix/TrimPrefix result used without checking if suffix/prefix was present
- **SLP120**: Value discarded with `_ = expr` — consider using the value

Total: 116 rules

## v0.0.12 (2026-04-26)

21 new rules:

- **SLP091-SLP093**: Test/mock/fixture quality (hardcoded dates, mock envelope mismatches, mock without assertion)
- **SLP094-SLP096**: Silent failure detection (shell `|| true`, empty catch handlers, missing `set -e`)
- **SLP097-SLP099**: API contract/route testing (destructuring vs envelope, route without test, field change without test update)
- **SLP100-SLP102**: Stub/incomplete implementation (no-op functions, dead feature flags, async without await)
- **SLP103-SLP104**: Magic literal expansion (timeout durations, buffer sizes)
- **SLP106-SLP108**: Resource management (acquire without release, cleanup only in error path, open without defer/timeout)
- **SLP109-SLP110**: Code duplication (similar function bodies, similar files)
- **SLP111-SLP112**: Binary/asset hygiene (binary without .gitignore, generated files without source)

Total: 108 rules

## v0.0.11 (2026-04-25)

- CI integration: slopgate runs in GitHub Actions with `--base origin/main`
- Full git history fetched (`fetch-depth: 0`) for diff comparison

## v0.0.10 (2026-04-24)

55 new rules:

- **SLP036-SLP045**: Semantic rules for CodeRabbit parity (query methods, transaction scoping, webhook patterns)
- **SLP046-SLP070**: 25 AI-slop detection rules (file cohesion, redundant comments, SQL injection, hardcoded secrets, dynamic code execution, concurrent maps, resource leaks)
- **SLP071-SLP080**: 10 AST-aware semantic rules (type assertions, nil pointers, defer close, goroutine races, weak crypto, SQL injection, hardcoded credentials, closed channels, ignored errors, single-impl interfaces)
- **SLP081-SLP090**: 10 CodeRabbit parity rules (React imports, keys, hooks, SQL concat, auth checks, webhook timeouts, hardcoded credentials, missing docstrings, API error handling)

Total: 87 rules

## v0.0.9 (2026-04-23)

- **SLP031-035**: React/TSX issues, missing imports, state anti-patterns, code quality
- **SLP036-SLP045**: Semantic rules for API docs, transaction scoping, webhook patterns, query methods
- **SLP046-SLP070**: 25 AI-slop detection rules

## v0.0.7 (2026-04-21)

- **SLP030**: ORM/query methods without sentinel exclusion

## v0.0.1 (2026-04-19)

Initial release with core rules SLP001-SLP029.
