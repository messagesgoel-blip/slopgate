package rules

import (
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP067 flags resource acquisitions without deferred or explicit close.
type SLP067 struct{}

func (SLP067) ID() string                { return "SLP067" }
func (SLP067) DefaultSeverity() Severity { return SeverityWarn }
func (SLP067) Description() string {
	return "resource acquired without deferred close"
}

var resourcePatterns = []string{
	"http.Get(",
	"http.Post(",
	"http.Do(",
	"db.Query",
	"db.QueryContext",
	"db.QueryRow",
	"rows.Next",
}

func hasResourceAcquisition(line string) bool {
	for _, p := range resourcePatterns {
		if strings.Contains(line, p) {
			return true
		}
	}
	return false
}

func (r SLP067) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || !strings.HasSuffix(f.Path, ".go") {
			continue
		}
		added := f.AddedLines()
		for i, ln := range added {
			if !hasResourceAcquisition(ln.Content) {
				continue
			}
			foundClose := false
			for j := i + 1; j < len(added); j++ {
				if strings.Contains(added[j].Content, ".Close()") ||
					strings.Contains(added[j].Content, "defer") {
					foundClose = true
					break
				}
			}
			if !foundClose {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "resource acquired without deferred close — add defer resp.Body.Close() or similar",
					Snippet:  strings.TrimSpace(ln.Content),
				})
			}
		}
	}
	return out
}
