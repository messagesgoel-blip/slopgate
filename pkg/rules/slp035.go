package rules

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP035 flags common code quality and style issues.
//
// Pattern: Console statements, debugger statements, TODOs without ticket references,
// trailing whitespace, and overly long lines.
//
// Rationale: Code quality issues can lead to maintenance problems
// and potential runtime errors.
type SLP035 struct{}

func (SLP035) ID() string                { return "SLP035" }
func (SLP035) DefaultSeverity() Severity { return SeverityWarn }
func (SLP035) Description() string {
	return "code quality or style issue detected"
}

// slp035TicketReferencePattern matches ticket references like SLOP-123 or CODE-456
var slp035TicketReferencePattern = regexp.MustCompile(`(?i)\b\w+-\d+\b`)

// Named regex patterns for efficient lookup
var consolePattern = regexp.MustCompile(`(?i)console\.(log|debug|info|warn|error)\s*\(`)
var debuggerPattern = regexp.MustCompile(`(?i)\bdebugger\b`)
var todoPattern = regexp.MustCompile(`(?i)(TODO|FIXME|HACK|XXX)`)
var trailingWhitespacePattern = regexp.MustCompile(`\s+$`)
var longLinePattern = regexp.MustCompile(`^.{141,}$`)

func (r SLP035) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}
		isDoc := isDocFile(f.Path)
		checkLongLine := !isDoc && isSourceLikeFile(f.Path)

		for _, h := range f.Hunks {
			for _, ln := range h.Lines {
				if ln.Kind != diff.LineAdd {
					continue
				}

				// Skip truly empty lines (no characters at all)
				if ln.Content == "" {
					continue
				}

				content := strings.TrimSpace(ln.Content)

				// Check for trailing whitespace on the raw line first
				// (whitespace-only lines like "   " should still be caught here)
				if trailingWhitespacePattern.MatchString(ln.Content) {
					appendFinding(&out, r, f.Path, ln.NewLineNo, "trailing whitespace detected", ln.Content)
				}

				// Skip whitespace-only lines for remaining checks
				if content == "" {
					continue
				}
				if isDoc {
					continue
				}

				// Check for console.log statements using direct pattern check
				if consolePattern.MatchString(content) {
					appendFinding(&out, r, f.Path, ln.NewLineNo, "console statement detected in code - remove before production", ln.Content)
				}

				// Check for debugger statements using direct pattern check
				if debuggerPattern.MatchString(content) {
					appendFinding(&out, r, f.Path, ln.NewLineNo, "debugger statement detected in code - remove before production", ln.Content)
				}

				// Check for TODO/FIXME without ticket references using direct pattern check
				if todoPattern.MatchString(content) {
					// Check if it has a ticket reference (e.g., CR-123, ISSUE-456)
					hasTicketRef := slp035TicketReferencePattern.MatchString(content)
					if !hasTicketRef {
						appendFinding(&out, r, f.Path, ln.NewLineNo, "TODO/FIXME comment without ticket reference - add ticket number", ln.Content)
					}
				}

				// Check for very long lines using direct pattern check
				if checkLongLine && longLinePattern.MatchString(ln.Content) {
					appendFinding(&out, r, f.Path, ln.NewLineNo, "line is too long ("+strconv.Itoa(len(ln.Content))+" chars) - consider breaking into multiple lines", ln.Content)
				}
			}
		}
	}
	return out
}

// Helper function to append a finding
func appendFinding(out *[]Finding, r SLP035, filePath string, lineNo int, message string, snippet string) {
	*out = append(*out, Finding{
		RuleID:   r.ID(),
		Severity: r.DefaultSeverity(),
		File:     filePath,
		Line:     lineNo,
		Message:  message,
		Snippet:  snippet,
	})
}
