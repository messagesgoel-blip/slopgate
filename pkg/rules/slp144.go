package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP144 flags inconsistent error handling patterns within the same file or
// route handler group. Mixing res.fail(), res.sendError(), next(err), and
// throw err creates confusion and can lead to unhandled errors.
//
// Detected patterns:
//   - Express route handlers mixing res.fail and res.send/next
//   - Mixing throw err with res.* patterns in same file
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
var errorPatterns = map[string]*regexp.Regexp{
	"res-fail":     regexp.MustCompile(`\bres\.fail\s*\(`),
	"res-send":     regexp.MustCompile(`\bres\.(send|json|status)\s*\(`),
	"res-error":    regexp.MustCompile(`\bres\.error\s*\(`),
	"res-next":     regexp.MustCompile(`\bnext\s*\(\s*(?:err)?\s*\)`),
	"throw":        regexp.MustCompile(`\bthrow\s+(?:new )?Error\s*\(`),
	"throw-err":    regexp.MustCompile(`\bthrow\s+err\b`),
	"return-err":   regexp.MustCompile(`\breturn\s+next\s*\(\s*err\s*\)`),
	"callback-err": regexp.MustCompile(`\bcb\s*\(\s*err\s*\)`),
}

// expressRoutePattern matches Express route handler definitions.
var expressRoutePattern = regexp.MustCompile(`(?:app|router)\.(get|post|put|delete|patch|all)\s*\(`)

// isExpressRouteFile determines if the file contains Express-style routing.
func isExpressRouteFile(path string, content string) bool {
	// Check file extension
	if !strings.HasSuffix(path, ".js") && !strings.HasSuffix(path, ".ts") {
		return false
	}
	// Look for route patterns in the file content (only visible in added lines)
	return expressRoutePattern.MatchString(content)
}

// detectMixedPatterns checks if multiple error handling patterns appear in
// the same file, suggesting inconsistency.
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

		// Check if this is an Express route file
		allContent := strings.Join(addedLines, "\n")
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
				Snippet:  "mixing res.fail, res.send, throw, and/or next(err)",
			})
		}
	}

	return out
}
