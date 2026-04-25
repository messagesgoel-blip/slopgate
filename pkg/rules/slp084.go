package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP084 flags useEffect hooks that need cleanup but don't have one.
// This can cause memory leaks from event listeners, timers, etc.
type SLP084 struct{}

func (SLP084) ID() string                { return "SLP084" }
func (SLP084) DefaultSeverity() Severity { return SeverityWarn }
func (SLP084) Description() string {
	return "useEffect may need cleanup - add return statement to clean up event listeners, timers, etc."
}

var (
	// Matches useEffect that adds event listeners without cleanup
	slp084EventListenerPattern = regexp.MustCompile(`(?i)(addEventListener|addEventListener\s*\([^)]*['"](?:click|scroll|resize|beforeunload|hashchange|popstate)['"])`)
	// Matches useEffect that sets timer without cleanup
	slp084TimerPattern = regexp.MustCompile(`(?i)(setTimeout|setInterval)\s*\(`)
	// Matches useEffect that sets state after async without cleanup guard
	slp084AsyncStatePattern = regexp.MustCompile(`(?i)await\s+\w+.*[\s\S]*?\.then\s*\(\s*\w+\s*=>\s*\w+\s*=\s*`)
	// Matches useEffect with async callback
	slp084AsyncEffectPattern = regexp.MustCompile(`(?i)useEffect\s*\(\s*\(\s*\)\s*=>\s*async`)
)

func (r SLP084) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}

		// Only check TS/TSX/JS/JSX files
		if !strings.HasSuffix(strings.ToLower(f.Path), ".ts") &&
			!strings.HasSuffix(strings.ToLower(f.Path), ".tsx") &&
			!strings.HasSuffix(strings.ToLower(f.Path), ".js") &&
			!strings.HasSuffix(strings.ToLower(f.Path), ".jsx") {
			continue
		}

		for _, h := range f.Hunks {
			for _, ln := range h.Lines {
				if ln.Kind != diff.LineAdd {
					continue
				}

				content := strings.TrimSpace(ln.Content)

				// Check if this is an useEffect with async or event listeners
				hasEffect := strings.Contains(content, "useEffect")
				hasCleanup := strings.Contains(content, "return") && strings.Contains(content, "cleanup")

				if hasEffect && !hasCleanup {
					// Check for event listener patterns
					if slp084EventListenerPattern.MatchString(content) {
						out = append(out, Finding{
							RuleID:   r.ID(),
							Severity: r.DefaultSeverity(),
							File:     f.Path,
							Line:     ln.NewLineNo,
							Message:  "useEffect adds event listener without cleanup - add return () => { cleanup }",
							Snippet:  content,
						})
						continue
					}

					// Check for timer patterns
					if slp084TimerPattern.MatchString(content) {
						out = append(out, Finding{
							RuleID:   r.ID(),
							Severity: r.DefaultSeverity(),
							File:     f.Path,
							Line:     ln.NewLineNo,
							Message:  "useEffect sets timer without cleanup - add return () => { clearTimeout(id) }",
							Snippet:  content,
						})
						continue
					}

					// Check for async pattern
					if slp084AsyncEffectPattern.MatchString(content) {
						// Check if there's a cleanup pattern in the same hunk
						hasCleanupInHunk := false
						for _, l := range h.Lines {
							if strings.Contains(l.Content, "return") && (strings.Contains(l.Content, "cleanup") || strings.Contains(l.Content, "abortController") || strings.Contains(l.Content, "isMounted")) {
								hasCleanupInHunk = true
								break
							}
						}
						if !hasCleanupInHunk {
							out = append(out, Finding{
								RuleID:   r.ID(),
								Severity: r.DefaultSeverity(),
								File:     f.Path,
								Line:     ln.NewLineNo,
								Message:  "useEffect with async operation may need cleanup - add cleanup function or abort controller",
								Snippet:  content,
							})
						}
					}
				}
			}
		}
	}
	return out
}
