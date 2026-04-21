package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP039 flags when pagination Total/Len returns page size instead of total matches.
//
// Rationale: When implementing pagination, returning len(page) as Total is a common
// mistake. The total should reflect all matching records, not just the current page.
// AI agents often make this mistake.
//
// Languages: Go.
//
// Scope: only added lines in Go files.
type SLP039 struct{}

func (SLP039) ID() string                { return "SLP039" }
func (SLP039) DefaultSeverity() Severity { return SeverityWarn }
func (SLP039) Description() string {
	return "pagination Total returns page size instead of total matching records"
}

// totalLenRe matches patterns like: Total: len(filtered) or Total: len(items)
var totalLenRe = regexp.MustCompile(`(?i)Total\s*:\s*len\s*\(\s*(filtered|items|results|result)\s*\)`)

// isGo reports whether the file is a Go file.

func (r SLP039) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}
		if !isGoFile(f.Path) {
			continue
		}

		for _, line := range f.AddedLines() {
			if totalLenRe.MatchString(line.Content) {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     line.NewLineNo,
					Message:  r.Description(),
					Snippet:  strings.TrimSpace(line.Content),
				})
			}
		}
	}
	return out
}
