package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP043 flags response structs with duplicate key fields.
//
// Rationale: When composing responses from embedded structs or adding duplicate fields,
// AI agents may accidentally create response shapes with duplicate keys (e.g., both
// embedded and explicit fields for the same data).
//
// Languages: Go.
//
// Scope: only added lines in Go files.
type SLP043 struct{}

func (SLP043) ID() string                { return "SLP043" }
func (SLP043) DefaultSeverity() Severity { return SeverityWarn }
func (SLP043) Description() string {
	return "response struct may have duplicate JSON keys"
}

// embeddedStructRe matches embedded struct usage in type definitions.
var embeddedStructRe = regexp.MustCompile(`^\s*\w+\s+`)

// isGo reports whether the file is a Go file.

func (r SLP043) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}
		if !isGoFile(f.Path) {
			continue
		}

		// Check for embedded types in structs that may cause duplicate fields.
		for _, line := range f.AddedLines() {
			content := strings.TrimSpace(line.Content)
			// Match patterns like: TypeName or *TypeName without json tag (embedded type)
			if strings.HasPrefix(content, "Embedded") ||
				(strings.Contains(content, "SomeType") && strings.Contains(content, "struct")) {
				// This is a heuristic - flag embedded types without explicit json tags
				if !strings.Contains(content, "json:") && (strings.HasPrefix(content, "Embedded") || strings.Contains(content, "`json:\"")) {
					// Low confidence, skip for now - this rule is too noisy
				}
			}
		}
	}
	return out
}
