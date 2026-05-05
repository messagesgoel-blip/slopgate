package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP146 flags unawaited promises in loops or array iteration methods.
// This catches patterns where async operations are started but not properly
// awaited, leading to race conditions and unhandled rejections.
//
// Detected patterns:
//   - array.map(async item => {...}) without Promise.all wrapper
//   - array.forEach with async callback without await
//   - for...of loops with async calls but missing await
//
// Languages: JavaScript, TypeScript
//
// Scope: all source files
type SLP146 struct{}

func (SLP146) ID() string                { return "SLP146" }
func (SLP146) DefaultSeverity() Severity { return SeverityWarn }
func (SLP146) Description() string {
	return "unawaited promise in loop or array iteration"
}

// asyncFunctionInMap matches pattern: array.map(async item => { ... }) or
// array.map(async (item) => { ... })
var asyncFunctionInMap1 = regexp.MustCompile(`\.(map|forEach|filter|reduce)\s*\(\s*async\s*\(`)
var asyncFunctionInMap2 = regexp.MustCompile(`\.(map|forEach|filter|reduce)\s*\(\s*async\s+\w+\s*=>`)

// forEachPattern matches forEach calls
var forEachPattern = regexp.MustCompile(`\.forEach\s*\(`)

// promiseReturningPatterns matches function calls that commonly return promises.
var promiseReturningPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\b(?:fetch|axios|request)\s*[\.(]`),
	regexp.MustCompile(`\b(?:create|find|update|delete|save|remove|query|execute)\s*\(`),
	regexp.MustCompile(`\.\s*(?:then|catch|finally)\s*\(`),
	regexp.MustCompile(`\bnew\s+Promise\s*\(`),
	regexp.MustCompile(`\b(?:db|prisma|knex|sequelize|mongoose)\.\w+\s*\(`),
}

// looksLikePromiseCall checks if a line contains a call that is likely
// to return a promise, rather than a simple synchronous function call.
func looksLikePromiseCall(line string) bool {
	for _, p := range promiseReturningPatterns {
		if p.MatchString(line) {
			return true
		}
	}
	return false
}

// loopLinePattern matches loop keywords as word boundaries.
var loopLinePattern = regexp.MustCompile(`\b(for|while)\b`)

// isLoopLine checks if a line contains a loop keyword, skipping comments.
func isLoopLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	// Skip comment lines
	if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
		return false
	}
	return loopLinePattern.MatchString(trimmed) ||
		strings.HasPrefix(trimmed, "for(") ||
		strings.HasPrefix(trimmed, "while(") ||
		strings.HasPrefix(trimmed, "do ")
}

// awaitPattern matches await as a word boundary (catches "(await foo())" etc).
var awaitPattern = regexp.MustCompile(`\bawait\b`)

// lineHasAwait checks if the line contains an await keyword as a standalone token.
func lineHasAwait(line string) bool {
	return awaitPattern.MatchString(line)
}

func (r SLP146) Check(d *diff.Diff) []Finding {
	if d == nil {
		return nil
	}
	var out []Finding

	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}
		// Only check JS/TS files
		if !strings.HasSuffix(f.Path, ".js") &&
			!strings.HasSuffix(f.Path, ".jsx") &&
			!strings.HasSuffix(f.Path, ".ts") &&
			!strings.HasSuffix(f.Path, ".tsx") {
			continue
		}

		for _, h := range f.Hunks {
			for i, ln := range h.Lines {
				if ln.Kind != diff.LineAdd {
					continue
				}
				line := ln.Content

				// Pattern 1: async callback in map/forEach/filter/reduce
				// Skip if the call is within Promise.all(...) which is safe.
				if (asyncFunctionInMap1.MatchString(line) || asyncFunctionInMap2.MatchString(line)) &&
					!strings.Contains(line, "Promise.all") {
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     ln.NewLineNo,
						Message:  "async function in array iteration without Promise.all wrapper",
						Snippet:  strings.TrimSpace(line),
					})
					continue
				}

				// Pattern 2: forEach with async callback that might return promise
				if forEachPattern.MatchString(line) && strings.Contains(line, "=>") {
					// Only flag if the callback is async or contains a promise-returning call.
					// Synchronous forEach callbacks are not unawaited promise issues.
					isAsync := strings.Contains(line, "async")
					hasAwait := strings.Contains(line, "await") || strings.Contains(line, "Promise")
					if isAsync && !hasAwait {
						out = append(out, Finding{
							RuleID:   r.ID(),
							Severity: r.DefaultSeverity(),
							File:     f.Path,
							Line:     ln.NewLineNo,
							Message:  "async forEach callback without await or Promise.all",
							Snippet:  strings.TrimSpace(line),
						})
					}
				}

				// Pattern 3: Loop line followed by promise-returning call without await.
				// Only flag lines that look like they return promises (contain awaitable
				// patterns like fetch, axios, db calls) rather than any line with ().
				if isLoopLine(line) {
					// Handle single-line loop bodies: for (...) fetch(...)
					if !lineHasAwait(line) && looksLikePromiseCall(line) {
						out = append(out, Finding{
							RuleID:   r.ID(),
							Severity: r.DefaultSeverity(),
							File:     f.Path,
							Line:     ln.NewLineNo,
							Message:  "potential unawaited promise in loop body",
							Snippet:  strings.TrimSpace(line),
						})
					}

					// Track brace depth to constrain scanning to the loop body
					braceDepth := strings.Count(line, "{") - strings.Count(line, "}")

					// Look at subsequent added lines within the loop body
					j := i + 1
					for j < len(h.Lines) && h.Lines[j].Kind == diff.LineAdd {
						nextLine := h.Lines[j].Content

						if !lineHasAwait(nextLine) && looksLikePromiseCall(nextLine) {
							out = append(out, Finding{
								RuleID:   r.ID(),
								Severity: r.DefaultSeverity(),
								File:     f.Path,
								Line:     h.Lines[j].NewLineNo,
								Message:  "potential unawaited promise in loop body",
								Snippet:  strings.TrimSpace(nextLine),
							})
						}

						// Stop when we leave the current braced loop block
						if braceDepth > 0 {
							braceDepth += strings.Count(nextLine, "{") - strings.Count(nextLine, "}")
							if braceDepth <= 0 {
								break
							}
						} else {
							// Non-braced loop body: inspect only next added statement
							break
						}

						j++
					}
				}
			}
		}
	}

	return out
}
