package rules

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP063 flags struct definitions with more than 15 fields — the God
// Object anti-pattern.
//
// Scope: Go files only.
type SLP063 struct{}

func (SLP063) ID() string                { return "SLP063" }
func (SLP063) DefaultSeverity() Severity { return SeverityWarn }
func (SLP063) Description() string {
	return "struct has too many fields (>15) — consider splitting into smaller types"
}

// slp063StructDef matches `type XXX struct {` and captures the name.
var slp063StructDef = regexp.MustCompile(`^type\s+(\w+)\s+struct\s*\{`)

func (r SLP063) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || !isGoFile(f.Path) {
			continue
		}
		for _, h := range f.Hunks {
			lines := h.Lines
			i := 0
			for i < len(lines) {
				ln := lines[i]
				if ln.Kind != diff.LineAdd {
					i++
					continue
				}
				m := slp063StructDef.FindStringSubmatch(strings.TrimSpace(ln.Content))
				if m == nil {
					i++
					continue
				}
				structName := m[1]
				startLine := ln.NewLineNo
				// Count field lines between { and } on added lines only.
				fieldCount := 0
				depth := strings.Count(ln.Content, "{") - strings.Count(ln.Content, "}")
				j := i + 1
				for j < len(lines) && depth > 0 {
					bl := lines[j]
					if bl.Kind != diff.LineAdd {
						break
					}
					trimmed := strings.TrimSpace(bl.Content)
					if trimmed == "" || strings.HasPrefix(trimmed, "//") {
						// Advance brace counting even for comments.
						depth += strings.Count(bl.Content, "{") - strings.Count(bl.Content, "}")
						j++
						continue
					}
					if strings.Contains(trimmed, "{") {
						depth++
					}
					if strings.Contains(trimmed, "}") {
						depth--
						if depth == 0 {
							break
						}
					}
					// Heuristic field definition: contains a space (at least two
					// tokens: name and type) and is not an embedded struct or
					// closing brace.
					if depth == 1 && strings.Contains(trimmed, " ") {
						fieldCount++
					}
					j++
				}
				if fieldCount > 15 {
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     startLine,
						Message:  "struct " + structName + " has " + strconv.Itoa(fieldCount) + " fields — consider splitting into smaller types",
						Snippet:  strings.TrimSpace(ln.Content),
					})
				}
				if j > i {
					i = j
				} else {
					i++
				}
			}
		}
	}
	return out
}
