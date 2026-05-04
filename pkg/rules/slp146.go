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
//   - array.forEach(item => asyncOperation(item)) without await
//   - for...of loops with async calls but missing await
//   - forEach on NodeLists in JS/TS
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

// promiseCallInLoop matches a function call that returns a promise but
// isn't preceded by await in a loop context.
var promiseCallInLoop = regexp.MustCompile(`\b(await\s+)?(?:await\s+)?[a-zA-Z_$][a-zA-Z0-9_$]*\s*\(\s*[^)]*\)\s*;?`)

// forEachPattern matches forEach calls
var forEachPattern = regexp.MustCompile(`\.forEach\s*\(`)

// isLoopLine checks if a line contains a loop keyword.
func isLoopLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "for ") ||
		strings.HasPrefix(trimmed, "for(") ||
		strings.HasPrefix(trimmed, "for(") ||
		strings.Contains(trimmed, " for ") ||
		strings.HasPrefix(trimmed, "while ") ||
		strings.HasPrefix(trimmed, "do ") ||
		strings.HasPrefix(trimmed, "for...of")
}

// lineHasAwait checks if the line contains an await keyword at the start level
// (not inside nested parentheses or strings).
func lineHasAwait(line string) bool {
	trimmed := strings.TrimSpace(line)
	// Quick check for standalone await at statement start
	if strings.HasPrefix(trimmed, "await ") {
		return true
	}
	// Check for await in the line (could be in nested expression but that's
	// usually good enough for diff-based detection)
	return strings.Contains(trimmed, " await ")
}

func (r SLP146) Check(d *diff.Diff) []Finding {
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

				// Pattern 2: forEach with function that might return promise
				if forEachPattern.MatchString(line) && strings.Contains(line, "=>") {
					// This is a heuristic - if forEach callback is an arrow fn
					// and doesn't contain 'await' keyword, it's likely unawaited
					if !strings.Contains(line, "await") && !strings.Contains(line, "Promise") {
						out = append(out, Finding{
							RuleID:   r.ID(),
							Severity: r.DefaultSeverity(),
							File:     f.Path,
							Line:     ln.NewLineNo,
							Message:  "forEach callback may contain unawaited promise",
							Snippet:  strings.TrimSpace(line),
						})
					}
				}

				// Pattern 3: Loop line followed by promise call without await
				if isLoopLine(line) {
					// Look at subsequent added lines within the loop body
					j := i + 1
					for j < len(h.Lines) && h.Lines[j].Kind == diff.LineAdd {
						nextLine := h.Lines[j].Content
						// Simple heuristic: line contains function call but no await
						if strings.Contains(nextLine, "(") && !lineHasAwait(nextLine) {
							// Check if this looks like a promise-returning call
							// (simple heuristic: ends with ) and not followed by await or then)
							if !strings.Contains(nextLine, ".then(") && !strings.Contains(nextLine, ".catch(") {
								out = append(out, Finding{
									RuleID:   r.ID(),
									Severity: r.DefaultSeverity(),
									File:     f.Path,
									Line:     h.Lines[j].NewLineNo,
									Message:  "potential unawaited promise in loop body",
									Snippet:  strings.TrimSpace(nextLine),
								})
							}
						}
						j++
					}
				}
			}
		}
	}

	return out
}
