package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP024 flags HTTP handlers that return 2xx status codes in catch blocks
// after logging errors. This is a critical bug pattern where catch blocks
// log an error then return res.status(200), preventing webhook retries.
//
// Pattern: catch block containing both error logging AND 2xx return.
//
// Exempt: test files, docs.
type SLP024 struct{}

func (SLP024) ID() string                { return "SLP024" }
func (SLP024) DefaultSeverity() Severity { return SeverityBlock }
func (SLP024) Description() string {
	return "catch block returns 2xx status after logging error — webhook callers will not retry"
}

// slp024CatchWith200 matches catch blocks that return 200/201/202/204.
var slp024CatchWith200 = regexp.MustCompile(`(?i)catch\s*\([^)]*\)\s*\{[^}]*console\.error[^}]*res\.status\s*\(\s*(200|201|202|204)\s*\)|catch\s*\{[^}]*console\.error[^}]*res\.status\s*\(\s*(200|201|202|204)\s*\)`)

// slp024CatchWithJsonSuccess matches catch with json success return.
var slp024CatchWithJsonSuccess = regexp.MustCompile(`(?i)catch[^{]*\{[^}]*console\.error[^}]*res\.json\s*\(\s*\{[^}]*(received|success)\s*:\s*true`)

// slp024CatchWithReturnSuccess matches catch with return success object.
var slp024CatchWithReturnSuccess = regexp.MustCompile(`(?i)catch[^{]*\{[^}]*console\.error[^}]*return\s*\{[^}]*(received|success)\s*:\s*true`)

func (r SLP024) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) {
			continue
		}
		// Only check JS/TS files.
		if !strings.HasSuffix(strings.ToLower(f.Path), ".js") &&
			!strings.HasSuffix(strings.ToLower(f.Path), ".ts") {
			continue
		}
		if strings.Contains(strings.ToLower(f.Path), ".test.") ||
			strings.Contains(strings.ToLower(f.Path), ".spec.") {
			continue
		}

		for _, ln := range f.AddedLines() {
			clean := stripCommentAndStrings(ln.Content)
			if clean == "" {
				continue
			}

			// Check for the dangerous patterns.
			if slp024CatchWith200.MatchString(clean) ||
				slp024CatchWithJsonSuccess.MatchString(clean) ||
				slp024CatchWithReturnSuccess.MatchString(clean) {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "catch block returns 2xx status after logging error — return 5xx so caller retries",
					Snippet:  strings.TrimSpace(ln.Content),
				})
			}
		}
	}
	return out
}
