package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP147 flags object destructuring that may access properties of null or
// undefined values without defensive defaults or existence checks.
//
// Detected patterns:
//   - const {prop} = possiblyUndefinedExpr;
//   - let {x, y} = maybeNull without if-check preceding it
//   - var {field} = obj where obj might be undefined
//
// Languages: JavaScript, TypeScript
//
// Scope: all source files
type SLP147 struct{}

func (SLP147) ID() string                { return "SLP147" }
func (SLP147) DefaultSeverity() Severity { return SeverityWarn }
func (SLP147) Description() string {
	return "object destructuring without null/undefined guard"
}

// destructuringPattern matches destructuring assignments.
// Captures: const/let/var, variable names, source expression.
// The trailing semicolon is optional to match semicolon-free code.
var destructuringPattern = regexp.MustCompile(`(const|let|var)\s+{([^}]+)}\s*=\s*([^;]+?);?\s*$`)

// guardPatterns match common defensive checks before destructuring.
var guardPatterns = []*regexp.Regexp{
	regexp.MustCompile(`if\s*\(\s*[^)]+\s*\)\s*{`),           // if (obj) ...
	regexp.MustCompile(`if\s*\(\s*!?\s*[^)]+\s*\)\s*return`), // guard clause
	regexp.MustCompile(`[^=]=\s*[^;]+\s*\|\|`),               // safe fallback before
	regexp.MustCompile(`[^=]=\s*[^;]+\s*\?\?`),               // nullish coalescing
}

// possibleUndefSource matches common sources that can be undefined:
// - function parameters
// - property access
// - function calls
// - import statements
func isPotentiallyUndefinedSource(source string) bool {
	source = strings.TrimSpace(source)
	// Parameters are potentially undefined if caller doesn't pass them
	if strings.Contains(source, "req.") ||
		strings.Contains(source, "res.") ||
		strings.Contains(source, "args.") ||
		strings.Contains(source, "ctx.") ||
		strings.Contains(source, "params.") ||
		strings.Contains(source, "(") { // function call result
		return true
	}
	return false
}

// hasPrecedingGuard checks if any previous line in the hunk has a guard.
func hasPrecedingGuard(h diff.Hunk, idx int) bool {
	// Look at up to 5 lines before to find a guard
	start := idx - 5
	if start < 0 {
		start = 0
	}
	for j := start; j < idx; j++ {
		line := h.Lines[j]
		// Skip deleted lines — they're not live code
		if line.Kind == diff.LineDelete {
			continue
		}
		content := line.Content
		for _, pattern := range guardPatterns {
			if pattern.MatchString(content) {
				return true
			}
		}
	}
	return false
}

func (r SLP147) Check(d *diff.Diff) []Finding {
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

				// Check if this is a destructuring assignment
				if !destructuringPattern.MatchString(line) {
					continue
				}

				// Extract the source expression
				parts := destructuringPattern.FindStringSubmatch(line)
				if len(parts) < 4 {
					continue
				}
				sourceExpr := strings.TrimSpace(parts[3])

				// Skip if source has inline fallback guard (e.g., req.user || {})
				if strings.Contains(sourceExpr, "||") || strings.Contains(sourceExpr, "??") {
					continue
				}

				// Quick heuristic: skip obvious safe sources
				if sourceExpr == "this" ||
					sourceExpr == "window" ||
					sourceExpr == "global" ||
					sourceExpr == "globalThis" ||
					strings.HasPrefix(sourceExpr, "process.env") {
					continue
				}

				// Check if source is potentially undefined
				if !isPotentiallyUndefinedSource(sourceExpr) {
					continue
				}

				// Check for preceding guard
				if hasPrecedingGuard(h, i) {
					continue
				}

				// Flag this destructuring
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "object destructuring from potentially undefined source without guard or defaults",
					Snippet:  strings.TrimSpace(line),
				})
			}
		}
	}

	return out
}
