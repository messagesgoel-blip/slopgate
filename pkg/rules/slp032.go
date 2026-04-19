package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP032 flags React/TypeScript component issues that relate to
// missing type imports, accessibility concerns, or improper patterns.
//
// Pattern: TSX files with React components that lack proper type imports
// or have common React anti-patterns.
//
// Rationale: React components without proper typing or with accessibility
// issues can cause runtime errors and poor user experience.
type SLP032 struct{}

func (SLP032) ID() string                { return "SLP032" }
func (SLP032) DefaultSeverity() Severity { return SeverityWarn }
func (SLP032) Description() string {
	return "React/TypeScript component may have type or accessibility issues"
}

// slp032MissingReactImport matches TSX files without React import.
var slp032MissingReactImport = regexp.MustCompile(`(?i)(?:^|\s)import\s+(?:\w+,?\s*)?(?:\{[^}]*\})?\s*from\s+["']react["']`)

// slp032ComponentPatterns matches React component patterns that might have issues.
var slp032ComponentPatterns = []*regexp.Regexp{
	// JSX element without proper React import
	regexp.MustCompile(`(?s)<\w+\s+.*>\s*</\w+>|<\w+\s*/>`),
	// Function component without React import
	regexp.MustCompile(`(?i)export\s+function\s+\w+\s*\([^)]*\)\s*{`),
	// Arrow function component without React import
	regexp.MustCompile(`(?i)const\s+\w+\s*=\s*\([^)]*\)\s*=>\s*{`),
	// useState without React import
	regexp.MustCompile(`(?i)useState\s*\(|React\.useState\s*\(`),
	// useEffect without React import
	regexp.MustCompile(`(?i)useEffect\s*\(|React\.useEffect\s*\(`),
	// Missing accessibility attributes
	regexp.MustCompile(`(?i)<button[^>]*>([^<]*)</button>`), // button without aria-label or accessible content
}

func (r SLP032) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}
		
		// Only check TSX files
		if !strings.HasSuffix(strings.ToLower(f.Path), ".tsx") {
			continue
		}

		// Check if React is imported
		hasReactImport := false
		for _, h := range f.Hunks {
			for _, ln := range h.Lines {
				if ln.Kind == diff.LineAdd {
					content := strings.ToLower(ln.Content)
					if strings.Contains(content, "import") && strings.Contains(content, "react") {
						hasReactImport = true
						break
					}
				}
			}
			if hasReactImport {
				break
			}
		}

		for _, h := range f.Hunks {
			for _, ln := range h.Lines {
				if ln.Kind != diff.LineAdd {
					continue
				}
				
				content := ln.Content
				
				// Check for React component patterns if React isn't imported
				if !hasReactImport {
					for _, pattern := range slp032ComponentPatterns {
						if pattern.MatchString(content) {
							// Avoid flagging import statements themselves
							if !strings.Contains(strings.ToLower(content), "import") {
								out = append(out, Finding{
									RuleID:   r.ID(),
									Severity: r.DefaultSeverity(),
									File:     f.Path,
									Line:     ln.NewLineNo,
									Message:  "React component detected without React import - add import React from 'react'",
									Snippet:  strings.TrimSpace(ln.Content),
								})
								break
							}
						}
					}
				}
				
				// Check for accessibility issues
				if strings.Contains(content, "<button") && !strings.Contains(content, "aria-") {
					if !strings.Contains(content, "title=") && !strings.Contains(content, "children") {
						out = append(out, Finding{
							RuleID:   r.ID(),
							Severity: r.DefaultSeverity(),
							File:     f.Path,
							Line:     ln.NewLineNo,
							Message:  "button element without accessibility attributes - add aria-label or ensure accessible content",
							Snippet:  strings.TrimSpace(ln.Content),
						})
					}
				}
			}
		}
	}
	return out
}