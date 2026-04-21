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

		var addedContent strings.Builder
		var readAllLines []diff.Line
		for _, line := range f.AddedLines() {
			addedContent.WriteString(line.Content)
			addedContent.WriteString("\n")
			if readAllRe.MatchString(line.Content) {
				readAllLines = append(readAllLines, line)
			}
		}

		if len(readAllLines) > 0 {
			content := addedContent.String()
			hasEmptyCheck := (strings.Contains(content, "len(") && strings.Contains(content, "== 0")) ||
				strings.Contains(content, "== nil") ||
				strings.Contains(content, "!= nil") ||
				(strings.Contains(content, "len(") && strings.Contains(content, "> 0"))
			hasEmptyBodyCheck := strings.Contains(content, "empty_body") || strings.Contains(content, "Body is required")

			if !hasEmptyCheck && !hasEmptyBodyCheck {
				for _, line := range readAllLines {
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
	}
	return out
}