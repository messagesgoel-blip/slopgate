package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP123 flags offset pagination on mutable time ordering when no cursor/keyset
// signal is present nearby. This pattern often drifts under concurrent writes.
type SLP123 struct{}

func (SLP123) ID() string                { return "SLP123" }
func (SLP123) DefaultSeverity() Severity { return SeverityWarn }
func (SLP123) Description() string {
	return "offset pagination on mutable ordering may drift — prefer cursor/keyset or explicit stable tiebreaker"
}

var slp123OffsetRe = regexp.MustCompile(`(?i)(\boffset\b|\.offset\s*\()`)
var slp123TimeOrderRe = regexp.MustCompile(`(?is)(order\s+by[^\n]*(created_at|updated_at|timestamp|time)|orderBy\s*\([^\n]*(createdAt|updatedAt|timestamp|time)|sort\s*:\s*["']?(createdAt|updatedAt|timestamp|time))`)
var slp123StableTieBreakerRe = regexp.MustCompile(`(?is)order\s+by[^\n]*\bid\b`)
var slp123CursorSignalRe = regexp.MustCompile(`(?i)(cursor|nextCursor|starting_after|ending_before|\bid\s*[<>]|\bcreated_at\s*[<>]|\bupdated_at\s*[<>])`)

func (r SLP123) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) || isTestFile(f.Path) {
			continue
		}
		if !isGoFile(f.Path) && !isJSOrTSFile(f.Path) && !isPythonFile(f.Path) {
			continue
		}
		for _, h := range f.Hunks {
			for i, ln := range h.Lines {
				if ln.Kind != diff.LineAdd {
					continue
				}
				content := strings.TrimSpace(slp115StripCommentsPreservingStrings(ln.Content))
				if content == "" || !slp123OffsetRe.MatchString(content) {
					continue
				}

				start := i - 12
				if start < 0 {
					start = 0
				}
				end := i + 12
				if end >= len(h.Lines) {
					end = len(h.Lines) - 1
				}

				var window strings.Builder
				for j := start; j <= end; j++ {
					if h.Lines[j].Kind == diff.LineDelete {
						continue
					}
					stripped := strings.TrimSpace(slp115StripCommentsPreservingStrings(h.Lines[j].Content))
					if stripped == "" {
						continue
					}
					window.WriteString(stripped)
					window.WriteByte('\n')
				}

				windowText := window.String()
				if !slp123TimeOrderRe.MatchString(windowText) {
					continue
				}
				if slp123CursorSignalRe.MatchString(windowText) {
					continue
				}
				if slp123StableTieBreakerRe.MatchString(windowText) {
					continue
				}

				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "offset pagination on time-ordered feed without cursor/tiebreaker — results can drift under concurrent writes",
					Snippet:  strings.TrimSpace(ln.Content),
				})
			}
		}
	}
	return out
}
