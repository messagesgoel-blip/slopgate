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
// It must start with optional whitespace, then "required:", optionally followed
// by a space and then the list (which we don't parse here).
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

// requiredItemsCount parses the number of items in a YAML required list.
// Handles both inline list syntax (e.g., "required: [a, b, c]") and
// multi-line syntax where items appear on subsequent lines counted separately.
func requiredItemsCount(line string) int {
	trimmed := strings.TrimSpace(line)
	// Inline list: "required: [a, b, c]"
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
	// Flow syntax: "required:" with no inline list — items follow on next lines.
	// We can't count from a single line, so return a large number so the
	// threshold check doesn't suppress it.
	return 999
}

func (r SLP036) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}
		// Only check YAML files.
		if !strings.HasSuffix(f.Path, ".yaml") && !strings.HasSuffix(f.Path, ".yml") {
			continue
		}
		for _, line := range f.AddedLines() {
			if isRequiredLine(line.Content) && containsSuspiciousWord(line.Content) && requiredItemsCount(line.Content) > 3 {
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
