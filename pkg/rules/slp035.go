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
var longLinePattern = regexp.MustCompile(`^.{101,}$`)

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
				
				// Process content for checks while preserving original for snippet
				rawContent := ln.Content
				content := strings.TrimSpace(ln.Content)
				
				// Don't skip whitespace-only lines since they're needed for trailing whitespace detection
				if rawContent == "" {
					continue
				}
				
				// Check for console.log statements using direct pattern check
				if consolePattern.MatchString(content) {
					appendFinding(&out, r, f.Path, ln.NewLineNo, "console statement detected in code - remove before production", rawContent)
				}
				
				// Check for debugger statements using direct pattern check
				if debuggerPattern.MatchString(content) {
					appendFinding(&out, r, f.Path, ln.NewLineNo, "debugger statement detected in code - remove before production", rawContent)
				}
				
				// Check for TODO/FIXME without ticket references using direct pattern check
				if todoPattern.MatchString(content) {
					// Check if it has a ticket reference (e.g., CR-123, ISSUE-456)
					hasTicketRef := slp035TicketReferencePattern.MatchString(content)
					if !hasTicketRef {
						appendFinding(&out, r, f.Path, ln.NewLineNo, "TODO/FIXME comment without ticket reference - add ticket number", rawContent)
					}
				}
				
				// Check for trailing whitespace using direct pattern check
				if trailingWhitespacePattern.MatchString(ln.Content) {
					appendFinding(&out, r, f.Path, ln.NewLineNo, "trailing whitespace detected", ln.Content)
				}
				
				// Check for very long lines using direct pattern check
				if longLinePattern.MatchString(ln.Content) {
					snippet := rawContent
					if len(rawContent) > 60 {
						snippet = rawContent[:60] + "..."
					}
					appendFinding(&out, r, f.Path, ln.NewLineNo, "line is too long (" + strconv.Itoa(len(ln.Content)) + " chars) - consider breaking into multiple lines", snippet)
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
