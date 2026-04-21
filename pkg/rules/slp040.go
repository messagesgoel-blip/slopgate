package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP040 flags HTTP handlers that read request bodies without validating for empty content.
//
// Rationale: Reading request bodies without checking for empty content can lead to
// issues when clients send empty bodies. AI agents often forget to check for empty
// body after reading it.
//
// Languages: Go.
//
// Scope: only added lines in Go files.
type SLP040 struct{}

func (SLP040) ID() string                { return "SLP040" }
func (SLP040) DefaultSeverity() Severity { return SeverityWarn }
func (SLP040) Description() string {
	return "HTTP handler reads body without validating for empty content"
}

// readAllRe matches io.ReadAll or io.ReadAll http request body.
var readAllRe = regexp.MustCompile(`io\.ReadAll\s*\(\s*(http\.MaxBytesReader|r\.Body)`)

func (r SLP040) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}
		if !isGoFile(f.Path) {
			continue
		}

		lines := f.AddedLines()
		for i, line := range lines {
			if !readAllRe.MatchString(line.Content) {
				continue
			}
			// Check a window around this line for empty-body validation.
			found := false
			start := i - 5
			if start < 0 {
				start = 0
			}
			end := i + 10
			if end > len(lines) {
				end = len(lines)
			}
			for j := start; j < end; j++ {
				c := lines[j].Content
				if (strings.Contains(c, "len(") && strings.Contains(c, "== 0")) ||
					strings.Contains(c, "== nil") ||
					strings.Contains(c, "!= nil") ||
					(strings.Contains(c, "len(") && strings.Contains(c, "> 0")) ||
					strings.Contains(c, "empty_body") ||
					strings.Contains(c, "Body is required") {
					found = true
					break
				}
			}
			if !found {
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
