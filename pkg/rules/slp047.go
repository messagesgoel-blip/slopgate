package rules

import (
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP047 flags comments that just restate the code immediately below them.
//
// Rationale: "What" comments waste space and become stale. Explain *why*,
// not what — comments that describe obvious behaviour add noise and quickly drift.
type SLP047 struct{}

func (SLP047) ID() string                { return "SLP047" }
func (SLP047) DefaultSeverity() Severity { return SeverityInfo }
func (SLP047) Description() string {
	return "comment restates the code below — explain why, not what"
}

func (r SLP047) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}
		// Only look at file types that use // comments.
		if !isGoFile(f.Path) && !strings.HasSuffix(f.Path, ".rs") &&
			!strings.HasSuffix(f.Path, ".cpp") && !strings.HasSuffix(f.Path, ".c") &&
			!strings.HasSuffix(f.Path, ".java") && !strings.HasSuffix(f.Path, ".kt") {
			continue
		}

		for _, h := range f.Hunks {
			lines := h.Lines
			for i := 0; i < len(lines); i++ {
				ln := lines[i]
				if ln.Kind != diff.LineAdd {
					continue
				}
				content := strings.TrimSpace(ln.Content)
				if !strings.HasPrefix(content, "// ") {
					continue
				}
				commentText := strings.TrimSpace(strings.TrimPrefix(content, "// "))
				if commentText == "" {
					continue
				}
				// Find next non-empty added line.
				var nextLine string
				for j := i + 1; j < len(lines); j++ {
					nxt := lines[j]
					if nxt.Kind != diff.LineAdd {
						continue
					}
					nxtContent := strings.TrimSpace(nxt.Content)
					if nxtContent != "" {
						nextLine = nxtContent
						break
					}
				}
				if nextLine == "" {
					continue
				}
				// Normalise: strip trailing '{' and whitespace from code line.
				codeLine := strings.TrimSpace(strings.TrimSuffix(nextLine, "{"))
				commentLine := strings.TrimSpace(strings.TrimSuffix(commentText, "{"))
				if strings.EqualFold(commentLine, codeLine) {
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     ln.NewLineNo,
						Message:  r.Description(),
						Snippet:  strings.TrimSpace(ln.Content),
					})
				}
			}
		}
	}
	return out
}
