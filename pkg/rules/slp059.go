package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP059 flags unsanitized exec.Command usage in Go files.
type SLP059 struct{}

func (SLP059) ID() string                { return "SLP059" }
func (SLP059) DefaultSeverity() Severity { return SeverityBlock }
func (SLP059) Description() string {
	return "unsanitized os/exec command with user input"
}

var stringLiteralPattern = regexp.MustCompile(`"[^"]*"`)
var goIdentPattern = regexp.MustCompile(`\b[a-zA-Z_][a-zA-Z0-9_]*\b`)
var execCommandRe = regexp.MustCompile(`\bexec\.Command\s*\(`)

func stripQuotedStrings(s string) string {
	return stringLiteralPattern.ReplaceAllString(s, "")
}

func (r SLP059) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || !strings.HasSuffix(f.Path, ".go") {
			continue
		}
		for _, ln := range f.AddedLines() {
			// Find exec.Command call using word-boundary regex.
			m := execCommandRe.FindStringIndex(ln.Content)
			if m == nil {
				continue
			}
			rest := ln.Content[m[1]:]
			argEnd := strings.Index(rest, ")")
			if argEnd == -1 {
				argEnd = len(rest)
			}
			args := rest[:argEnd]
			// Strip string literals before checking for interpolation.
			unquoted := stripQuotedStrings(args)
			// Any interpolation or concatenation is an immediate red flag.
			if strings.Contains(unquoted, "$") || strings.Contains(unquoted, "+") || strings.Contains(unquoted, "fmt.Sprintf") {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "exec.Command argument may contain user input — sanitize before executing",
					Snippet:  strings.TrimSpace(ln.Content),
				})
				continue
			}
			if goIdentPattern.MatchString(unquoted) {
				// Note: we cannot statically resolve whether a variable is a safe
				// compile-time constant. A local const string is safe, but a
				// variable assigned elsewhere may contain user input. We flag all
				// non-literal variables as potentially unsafe.
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "exec.Command argument may contain user input — sanitize before executing",
					Snippet:  strings.TrimSpace(ln.Content),
				})
			}
		}
	}
	return out
}
