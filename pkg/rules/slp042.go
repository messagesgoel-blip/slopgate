package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP042 flags JSON struct fields without json tags.
//
// Rationale: JSON struct fields without explicit json tags rely on Go's default
// field naming, which can cause API contract issues when field names change.
// AI agents often forget json tags.
//
// Languages: Go.
//
// Scope: only added lines in Go files with new struct definitions.
type SLP042 struct{}

func (SLP042) ID() string                { return "SLP042" }
func (SLP042) DefaultSeverity() Severity { return SeverityWarn }
func (SLP042) Description() string {
	return "JSON struct field without json tag may cause API contract issues"
}

// structFieldRe matches a Go struct field definition: Identifier Type with optional
// pointer/slice/map/qualifiers, and optional struct tag or end-of-line.
var structFieldRe = regexp.MustCompile(`^\s*[A-Z]\w*\s+(\*?\[\]|map\[|\*?\w+(\.\w+)*(\[\])?)\s*(\` + "`" + `|//|$)`)

// jsonTagRe matches if a field has a json tag.
var jsonTagRe = regexp.MustCompile(`json:"[^"]*"`)

func (r SLP042) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}
		if !isGoFile(f.Path) {
			continue
		}

		var structLines []diff.Line
		inStruct := false
		for _, line := range f.AddedLines() {
			content := line.Content

			// Detect struct block
			if strings.Contains(content, "struct {") || strings.HasSuffix(strings.TrimSpace(content), "struct {") {
				inStruct = true
			}
			if inStruct && strings.HasPrefix(strings.TrimSpace(content), "}") {
				inStruct = false
			}

			// If inside a struct and line looks like a field definition without a json tag
			if inStruct {
				if structFieldRe.MatchString(content) && !jsonTagRe.MatchString(content) {
					// Skip if it's just whitespace or closing brace context
					trimmed := strings.TrimSpace(content)
					if len(trimmed) > 0 && !strings.HasPrefix(trimmed, "}") {
						structLines = append(structLines, line)
					}
				}
			}
		}

		// Flag if there is at least one field without tags (lowered threshold from 2 to 1)
		if len(structLines) >= 1 {
			for _, line := range structLines {
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