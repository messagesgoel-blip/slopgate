package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP081 flags TSX/JSX files with React components that lack the React import.
// In React 17+, JSX compiles without React import, but older versions or
// certain config may still require it for proper JSX transform.
type SLP081 struct{}

func (SLP081) ID() string                { return "SLP081" }
func (SLP081) DefaultSeverity() Severity { return SeverityWarn }
func (SLP081) Description() string {
	return "React component detected without React import - add import React from 'react' or use JSX pragma"
}

var (
	jsxPattern = regexp.MustCompile(`(?i)<\w+\s|const\s+\w+\s*=\s*\([^)]*\)\s*=>\s*<|const\s+\w+\s*=\s*\([^)]*\)\s*=>\s*\{|function\s+\w+\s*\([^)]*\)\s*\{|export\s+(default\s+)?function\s+\w+|export\s+(default\s+)?const\s+\w+\s*=\s*function|export\s+(default\s+)?const\s+\w+\s*=\s*\([^)]*\)\s*=>`)
)

func (r SLP081) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}

		// Only check TSX files
		if !strings.HasSuffix(strings.ToLower(f.Path), ".tsx") &&
			!strings.HasSuffix(strings.ToLower(f.Path), ".jsx") {
			continue
		}

		// Check if React is imported (match "react" exactly)
		hasReactImport := false
		for _, h := range f.Hunks {
			for _, ln := range h.Lines {
				if ln.Kind == diff.LineAdd {
					content := strings.ToLower(ln.Content)
					// Check for import React from 'react' or import { ... } from 'react'
					if strings.Contains(content, "import") && (strings.Contains(content, `"react"`) || strings.Contains(content, `'react'`)) {
						hasReactImport = true
						break
					}
				}
			}
			if hasReactImport {
				break
			}
		}

		// Check for JSX patterns that require React import
		if !hasReactImport {
			for _, h := range f.Hunks {
				for _, ln := range h.Lines {
					if ln.Kind != diff.LineAdd {
						continue
					}
					if jsxPattern.MatchString(ln.Content) {
						out = append(out, Finding{
							RuleID:   r.ID(),
							Severity: r.DefaultSeverity(),
							File:     f.Path,
							Line:     ln.NewLineNo,
							Message:  "JSX detected without React import - add import React from 'react' or use \"jsxImportSource: react\"",
							Snippet:  strings.TrimSpace(ln.Content),
						})
						break
					}
				}
			}
		}
	}
	return out
}
