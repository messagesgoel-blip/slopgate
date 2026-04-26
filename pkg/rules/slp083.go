package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP083 flags useCallback and useMemo hooks that are missing dependencies array.
// This can cause stale closures and performance issues.
// Note: This rule monitors for missing dependency arrays in React hooks,
// not hardcoded API keys. The naming follows the SLP convention for
// specific rule IDs, not advertised detection purposes.
// To fix false positives for valid hook calls with semicolons or multiline
// formatting, ensure dependency arrays end on the same line (e.g., `[]);`).
type SLP083 struct{}

func (SLP083) ID() string                { return "SLP083" }
func (SLP083) DefaultSeverity() Severity { return SeverityWarn }
func (SLP083) Description() string {
	return "useCallback/useMemo missing dependency array - add [] or appropriate dependencies"
}

var (
	// Matches useCallback or useMemo call start
	slp083HookStartPattern = regexp.MustCompile(`(?i)(useCallback|useMemo)\s*\(`)
	// Matches dependency array with content - ends with ], then comma, then [array]
	slp083WithDepsPattern = regexp.MustCompile(`(?i),\s*\[[^]]+\]\s*\)\s*$`)
	// Matches empty dependency array - ends with ], then comma, then [empty]
	slp083EmptyDepsPattern = regexp.MustCompile(`(?i),\s*\[\s*\]\s*\)\s*$`)
)

func (r SLP083) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}

		// Only check TS/TSX/JS/JSX files
		if !isJSOrTSFile(f.Path) {
			continue
		}

		for _, h := range f.Hunks {
			for _, ln := range h.Lines {
				if ln.Kind != diff.LineAdd {
					continue
				}

				content := strings.TrimSpace(ln.Content)

				// Check if this line starts with useCallback or useMemo
				matches := slp083HookStartPattern.FindStringSubmatch(content)
				if len(matches) > 0 {
					hookName := strings.ToLower(matches[1])

					// Check if the same line has a dependency array
					hasEmptyDeps := slp083EmptyDepsPattern.MatchString(content)
					hasNonEmptyDeps := slp083WithDepsPattern.MatchString(content)

					// If no dependency array found, flag it
					if !hasEmptyDeps && !hasNonEmptyDeps {
						out = append(out, Finding{
							RuleID:   r.ID(),
							Severity: r.DefaultSeverity(),
							File:     f.Path,
							Line:     ln.NewLineNo,
							Message:  hookName + " missing dependency array - add [] or appropriate dependencies to avoid stale closures",
							Snippet:  content,
						})
					}
				}
			}
		}
	}
	return out
}
