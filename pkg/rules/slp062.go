package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP062 flags functions longer than 50 added lines or Go files with
// more than 500 added lines.
//
// Scope: Go files only.
type SLP062 struct{}

func (SLP062) ID() string                { return "SLP062" }
func (SLP062) DefaultSeverity() Severity { return SeverityWarn }
func (SLP062) Description() string {
	return "function or file complexity exceeds thresholds (>50 / >500 lines)"
}

// slp062FuncSignature matches a Go function signature line.
var slp062FuncSignature = regexp.MustCompile(`^func\s*(?:\([^)]+\)\s*)?\w+\s*\(`)

func (r SLP062) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || !isGoFile(f.Path) {
			continue
		}
		added := f.AddedLines()
		// File-level check.
		if len(added) > 500 {
			out = append(out, Finding{
				RuleID:   r.ID(),
				Severity: r.DefaultSeverity(),
				File:     f.Path,
				Line:     0,
				Message:  "file adds " + itoa(len(added)) + " lines — consider refactoring into smaller files",
				Snippet:  "",
			})
		}

		// Function-level check using added lines only.
		for i := 0; i < len(added); i++ {
			ln := added[i]
			trimmed := strings.TrimSpace(ln.Content)
			if !slp062FuncSignature.MatchString(trimmed) {
				continue
			}
			// Extract function name for the message.
			funcName := extractFuncName(trimmed)
			startLine := ln.NewLineNo
			// Brace depth counting on THIS line only.
			depth := strings.Count(ln.Content, "{") - strings.Count(ln.Content, "}")
			if depth <= 0 {
				// Opening brace is on a subsequent added line.
				j := i + 1
				for j < len(added) && depth <= 0 {
					depth += strings.Count(added[j].Content, "{") - strings.Count(added[j].Content, "}")
					j++
				}
				if depth <= 0 {
					continue
				}
				// Start counting from the signature line; body lines
				// start at i+1.
				bodyLines := 1 // signature line counts as part of the function
				for j < len(added) && depth > 0 {
					bodyLines++
					depth += strings.Count(added[j].Content, "{") - strings.Count(added[j].Content, "}")
					j++
				}
				if bodyLines > 50 {
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     startLine,
						Message:  "function " + funcName + " is " + itoa(bodyLines) + " lines — consider breaking it up",
						Snippet:  trimmed,
					})
				}
				if j > i {
					i = j - 1
				}
				continue
			}
			// Opening brace is on the signature line.
			bodyLines := 1
			j := i + 1
			for j < len(added) && depth > 0 {
				bodyLines++
				depth += strings.Count(added[j].Content, "{") - strings.Count(added[j].Content, "}")
				j++
			}
			if bodyLines > 50 {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     startLine,
					Message:  "function " + funcName + " is " + itoa(bodyLines) + " lines — consider breaking it up",
					Snippet:  trimmed,
				})
			}
			if j > i {
				i = j - 1
			}
		}
	}
	return out
}

func extractFuncName(line string) string {
	// Heuristic: after `func`, optionally `(recv) `, then the name before `(`.
	s := strings.TrimPrefix(line, "func")
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "(") {
		idx := strings.Index(s, ")")
		if idx >= 0 {
			s = strings.TrimSpace(s[idx+1:])
		}
	}
	idx := strings.Index(s, "(")
	if idx > 0 {
		return strings.TrimSpace(s[:idx])
	}
	return "<unknown>"
}
