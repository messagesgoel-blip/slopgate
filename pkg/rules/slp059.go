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
var goVarPattern = regexp.MustCompile(`\b[a-z][a-zA-Z0-9_]*\b`)

func stripQuotedStrings(s string) string {
	return stringLiteralPattern.ReplaceAllString(s, "")
}

func (SLP059) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || !strings.HasSuffix(f.Path, ".go") {
			continue
		}
		for _, ln := range f.AddedLines() {
			idx := strings.Index(ln.Content, "exec.Command(")
			if idx == -1 {
				idx = strings.Index(ln.Content, "exec.Command (")
			}
			if idx == -1 {
				continue
			}
			rest := ln.Content[idx+len("exec.Command("):]
			argEnd := strings.Index(rest, ")")
			if argEnd == -1 {
				argEnd = len(rest)
			}
			args := rest[:argEnd]
			if strings.Contains(args, "$") || strings.Contains(args, "+") || strings.Contains(args, "fmt.Sprintf") {
				out = append(out, Finding{
					RuleID:   "SLP059",
					Severity: SeverityBlock,
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "exec.Command argument may contain user input — sanitize before executing",
					Snippet:  strings.TrimSpace(ln.Content),
				})
				continue
			}
			unquoted := stripQuotedStrings(args)
			if goVarPattern.MatchString(unquoted) {
				out = append(out, Finding{
					RuleID:   "SLP059",
					Severity: SeverityBlock,
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
