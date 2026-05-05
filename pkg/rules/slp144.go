package rules

import (
	"regexp"
	"sort"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP144 flags inconsistent error handling patterns within the same file or
// route handler group. Mixing res.fail(), next(err), and throw err creates
// confusion and can lead to unhandled errors.
//
// Note: res.status/res.json/res.send are success-path response methods and
// are NOT treated as error handlers. Standard try/catch patterns using
// res.json in try + next(err) in catch are not flagged.
//
// Detected patterns:
//   - Express route handlers mixing res.fail and next(err)
//   - Mixing throw err with res.error patterns in same file
//
// Languages: JavaScript, TypeScript
//
// Scope: files with Express/Koa-style route handlers
type SLP144 struct{}

func (SLP144) ID() string                { return "SLP144" }
func (SLP144) DefaultSeverity() Severity { return SeverityWarn }
func (SLP144) Description() string {
	return "inconsistent error handling patterns in same file or route group"
}

// errorPatterns maps error handling patterns we look for.
// res.status/res.json/res.send are NOT included here because they are
// success-path response methods, not error handlers. Using res.json in a
// try block alongside next(err) in a catch block is standard Express practice,
// not an inconsistency.
var errorPatterns = map[string]*regexp.Regexp{
	"res-fail":     regexp.MustCompile(`\bres\.fail\s*\(`),
	"res-error":    regexp.MustCompile(`\bres\.error\s*\(`),
	"next-err":     regexp.MustCompile(`\b(?:return\s+)?next\s*\(\s*[A-Za-z_$][A-Za-z0-9_$]*\s*\)`),
	"throw":        regexp.MustCompile(`\bthrow\s+(?:new )?Error\s*\(`),
	"throw-err":    regexp.MustCompile(`\bthrow\s+err\b`),
	"callback-err": regexp.MustCompile(`\bcb\s*\(\s*err\s*\)`),
}

// expressRoutePattern matches Express route handler definitions.
var expressRoutePattern = regexp.MustCompile(`(?:app|router)\.(get|post|put|delete|patch|all)\s*\(`)

// isExpressRouteFile determines if the file contains Express-style routing.
func isExpressRouteFile(path string, content string) bool {
	// Check file extension (match the same extensions as Check)
	if !strings.HasSuffix(path, ".js") &&
		!strings.HasSuffix(path, ".jsx") &&
		!strings.HasSuffix(path, ".ts") &&
		!strings.HasSuffix(path, ".tsx") {
		return false
	}
	// Look for route patterns in the file content (only visible in added lines)
	return expressRoutePattern.MatchString(content)
}

// detectMixedPatterns checks if multiple error handling patterns appear in
// the same file, suggesting inconsistency. Results are sorted for determinism.
func detectMixedPatterns(addedLines []string) []string {
	var found []string
	for name, pattern := range errorPatterns {
		for _, line := range addedLines {
			if pattern.MatchString(line) {
				found = append(found, name)
				break
			}
		}
	}
	sort.Strings(found)
	return found
}

func (r SLP144) Check(d *diff.Diff) []Finding {
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

		// Collect all added line content for this file
		var addedLines []string
		for _, ln := range f.AddedLines() {
			addedLines = append(addedLines, ln.Content)
		}

		// Build visible content from context + added lines so route
		// declarations in unchanged code are still detected.
		var visibleLines []string
		for _, h := range f.Hunks {
			for _, ln := range h.Lines {
				if ln.Kind == diff.LineContext || ln.Kind == diff.LineAdd {
					visibleLines = append(visibleLines, ln.Content)
				}
			}
		}
		allContent := strings.Join(visibleLines, "\n")
		if !isExpressRouteFile(f.Path, allContent) {
			continue
		}

		// Detect what error handling patterns are used
		patternsFound := detectMixedPatterns(addedLines)
		if len(patternsFound) > 1 {
			// Find first added line number for reporting
			firstLine := 0
			for _, h := range f.Hunks {
				for _, ln := range h.Lines {
					if ln.Kind == diff.LineAdd {
						firstLine = ln.NewLineNo
						break
					}
				}
				if firstLine > 0 {
					break
				}
			}
			if firstLine == 0 {
				firstLine = 1 // fallback
			}
			// Multiple patterns found - inconsistent
			out = append(out, Finding{
				RuleID:   r.ID(),
				Severity: r.DefaultSeverity(),
				File:     f.Path,
				Line:     firstLine,
				Message:  "inconsistent error handling patterns detected: " + strings.Join(patternsFound, ", "),
				Snippet:  "mixing res.fail, res.error, throw, and/or next(err)",
			})
		}
	}

	return out
}
