package rules

import (
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP036 flags suspiciously large required lists in OpenAPI/YAML schemas that
// often indicate copy-paste errors or misunderstanding of which fields are
// actually required by the backend handler.
//
// Rationale: AI agents generating or modifying OpenAPI specs sometimes include
// fields like `size`, `saved_at`, or `generated_at` in the `required` list when
// the handler does not actually require them (they may be optional or
// server-generated). This leads to contract mismatches.
//
// Languages: YAML (primarily OpenAPI).
//
// Scope: only added or modified lines that look like a `required:` field in a
// YAML map.
type SLP036 struct{}

func (SLP036) ID() string                { return "SLP036" }
func (SLP036) DefaultSeverity() Severity { return SeverityWarn }
func (SLP036) Description() string {
	return "suspiciously large required list in OpenAPI schema (e.g., size, saved_at, generated_at)"
}

// suspiciousRequiredWords lists words that are rarely actually required in API
// responses (they are often optional or server-generated).
var suspiciousRequiredWords = []string{
	"size",
	"saved_at",
	"generated_at",
}

// isRequiredLine reports whether the line looks like a YAML `required:` field.
func isRequiredLine(line string) bool {
	trimmed := strings.TrimLeft(line, " \t")
	return strings.HasPrefix(trimmed, "required:")
}

// containsSuspiciousWord reports whether the line contains any of the
// suspicious words as a substring (case-insensitive).
func containsSuspiciousWord(line string) bool {
	lower := strings.ToLower(line)
	for _, w := range suspiciousRequiredWords {
		if strings.Contains(lower, w) {
			return true
		}
	}
	return false
}

// countInlineRequiredItems parses the number of items in an inline YAML list
// like "required: [a, b, c]". Returns -1 if not an inline list.
func countInlineRequiredItems(line string) int {
	trimmed := strings.TrimSpace(line)
	if idx := strings.Index(trimmed, "["); idx != -1 {
		listPart := trimmed[idx+1:]
		if end := strings.Index(listPart, "]"); end != -1 {
			listPart = listPart[:end]
		}
		count := 0
		for _, item := range strings.Split(listPart, ",") {
			if strings.TrimSpace(item) != "" {
				count++
			}
		}
		return count
	}
	return -1
}

func (r SLP036) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}
		if !strings.HasSuffix(f.Path, ".yaml") && !strings.HasSuffix(f.Path, ".yml") {
			continue
		}

		lines := f.AddedLines()
		for i, line := range lines {
			if !isRequiredLine(line.Content) {
				continue
			}

			// Check inline list style: "required: [a, b, c]"
			if count := countInlineRequiredItems(line.Content); count >= 0 {
				if containsSuspiciousWord(line.Content) && count > 3 {
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     line.NewLineNo,
						Message:  r.Description(),
						Snippet:  strings.TrimSpace(line.Content),
					})
				}
				continue
			}

			// Flow style: "required:" on its own line, items on subsequent "- item" lines.
			// Scan forward for consecutive list items and check for suspicious words + count.
			suspicious := containsSuspiciousWord(line.Content)
			itemCount := 0
			for j := i + 1; j < len(lines); j++ {
				trimmed := strings.TrimSpace(lines[j].Content)
				if !strings.HasPrefix(trimmed, "- ") {
					break
				}
				itemCount++
				if containsSuspiciousWord(lines[j].Content) {
					suspicious = true
				}
			}
			if suspicious && itemCount > 3 {
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
