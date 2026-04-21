package rules

import (
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP044 flags errors ignored with _ in Go.
//
// Rationale: Ignoring errors with _ (blank identifier) can hide important error conditions.
// AI agents often use _ to suppress errors they don't want to handle.
//
// Languages: Go.
//
// Scope: only added lines in Go files.
type SLP044 struct{}

func (SLP044) ID() string                { return "SLP044" }
func (SLP044) DefaultSeverity() Severity { return SeverityWarn }
func (SLP044) Description() string {
	return "error ignored with blank identifier - consider handling or logging"
}

func (r SLP044) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}
		if !isGoFile(f.Path) {
			continue
		}

		for _, line := range f.AddedLines() {
			content := line.Content
			trimmed := strings.TrimSpace(content)

			// Skip comments
			if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
				continue
			}

			// Pattern: "something, _ := fn()" or "something, _ = fn()"
			// Flags when the blank identifier _ is used to discard a return value.
			if strings.Contains(content, ", _") && (strings.Contains(content, ":=") || strings.Contains(content, " = ")) {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     line.NewLineNo,
					Message:  r.Description(),
					Snippet:  trimmed,
				})
			}
		}
	}
	return out
}
