package rules

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP035 flags common code quality and style issues.
//
// Pattern: Unused variables, dead code, inconsistent naming, etc.
//
// Rationale: Code quality issues can lead to maintenance problems
// and potential runtime errors.
type SLP035 struct{}

func (SLP035) ID() string                { return "SLP035" }
func (SLP035) DefaultSeverity() Severity { return SeverityWarn }
func (SLP035) Description() string {
	return "code quality or style issue detected"
}

// slp035UnusedVarPatterns matches patterns of potentially unused variables.
var slp035UnusedVarPatterns = []*regexp.Regexp{
	// Variables declared but not used in the same scope
	regexp.MustCompile(`(?i)const\s+(\w+)\s*=\s*[^;,]*[;,]\s*[^=]*\s*[^;]*[;]`),
	// Catch parameters that are not used
	regexp.MustCompile(`(?i)catch\s*\(\s*(\w+)\s*\)`),
	// Loop variables that are not used
	regexp.MustCompile(`(?i)for\s*\([^;]*;\s*\w+\s*;\s*\)`),
}

// slp035StyleIssues matches common style issues.
var slp035StyleIssues = []*regexp.Regexp{
	// Inconsistent naming patterns (camelCase vs snake_case mix)
	regexp.MustCompile(`(?i)[a-z]+[A-Z][a-zA-Z]*\w*[A-Z]\w*`), // Mixed case that might be inconsistent
	// Multiple consecutive blank lines
	regexp.MustCompile(`\n\s*\n\s*\n`),
	// Trailing whitespace
	regexp.MustCompile(`\s+$`),
	// Tabs mixed with spaces (basic check)
	regexp.MustCompile(`\t.* {2,}|\s{2,}.*\t`),
	// Long lines (typically > 100 chars)
	regexp.MustCompile(`^.{101,}$`),
}

// slp035PotentialDeadCode matches patterns that might indicate dead code.
var slp035PotentialDeadCode = []*regexp.Regexp{
	// Commented-out code blocks
	regexp.MustCompile(`(?s)/\*.*?\*/`),
	// Large commented-out sections
	regexp.MustCompile(`(?s)//\s*[^\n]{50,}`),
	// Console.log statements in production code
	regexp.MustCompile(`(?i)console\.(log|debug|info|warn|error)\s*\(`),
	// Debugger statements
	regexp.MustCompile(`(?i)\bdebugger\b`),
	// TODO/FIXME comments without ticket references
	regexp.MustCompile(`(?i)(TODO|FIXME|HACK|XXX)`),
}

func (r SLP035) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}
		
		// Check all file types
		for _, h := range f.Hunks {
			for _, ln := range h.Lines {
				if ln.Kind != diff.LineAdd {
					continue
				}
				
				content := strings.TrimSpace(ln.Content)
				if content == "" {
					continue
				}
				
				// Check for console.log statements
				if regexp.MustCompile(`(?i)console\.(log|debug|info|warn|error)\s*\(`).MatchString(content) {
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     ln.NewLineNo,
						Message:  "console statement detected in code - remove before production",
						Snippet:  content,
					})
				}
				
				// Check for debugger statements
				if regexp.MustCompile(`(?i)\bdebugger\b`).MatchString(content) {
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     ln.NewLineNo,
						Message:  "debugger statement detected in code - remove before production",
						Snippet:  content,
					})
				}
				
				// Check for TODO/FIXME without ticket references
				if regexp.MustCompile(`(?i)(TODO|FIXME|HACK|XXX)`).MatchString(content) {
					// Check if it has a ticket reference (e.g., CR-123, ISSUE-456)
					hasTicketRef := regexp.MustCompile(`(?i)\b\w+-\d+\b`).MatchString(content)
					if !hasTicketRef {
						out = append(out, Finding{
							RuleID:   r.ID(),
							Severity: r.DefaultSeverity(),
							File:     f.Path,
							Line:     ln.NewLineNo,
							Message:  "TODO/FIXME comment without ticket reference - add ticket number",
							Snippet:  content,
						})
					}
				}
				
				// Check for trailing whitespace
				if regexp.MustCompile(`\s+$`).MatchString(ln.Content) {
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     ln.NewLineNo,
						Message:  "trailing whitespace detected",
						Snippet:  strings.TrimRight(content, " \t"),
					})
				}
				
				// Check for very long lines
				if len(ln.Content) > 100 {
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     ln.NewLineNo,
						Message:  "line is too long (" + strconv.Itoa(len(ln.Content)) + " chars) - consider breaking into multiple lines",
						Snippet:  content[:min(60, len(content))] + "...",
					})
				}
			}
		}
	}
	return out
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}