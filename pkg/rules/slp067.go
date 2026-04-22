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
	"os.Open(",
	"os.Create(",
	"sql.Open(",
}

func hasResourceAcquisition(line string) bool {
	for _, p := range resourcePatterns {
		if strings.Contains(line, p) {
			return true
		}
	}
	return false
}

// resourceVar extracts a likely variable name from a resource acquisition line.
// For assignments like "resp, err := http.Get(...)" it returns "resp".
func resourceVar(line string) string {
	line = strings.TrimSpace(line)
	if idx := strings.Index(line, ":="); idx > 0 {
		lhs := strings.TrimSpace(line[:idx])
		parts := strings.Split(lhs, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	if idx := strings.Index(line, "="); idx > 0 {
		lhs := strings.TrimSpace(line[:idx])
		parts := strings.Split(lhs, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	return ""
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
			varName := resourceVar(ln.Content)
			foundClose := false
			for j := i + 1; j < len(added); j++ {
				next := added[j].Content
				// Require both the variable name AND the close/defer token.
				if varName != "" {
					if strings.Contains(next, varName+".Close()") ||
						strings.Contains(next, "defer "+varName+".") {
						foundClose = true
						break
					}
				} else {
					// Fallback: any defer or .Close() if we couldn't identify var.
					if strings.Contains(next, ".Close()") || strings.Contains(next, "defer") {
						foundClose = true
						break
					}
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
