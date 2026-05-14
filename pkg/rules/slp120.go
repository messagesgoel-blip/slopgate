package rules

import (
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP120 checks for discarded values using the blank identifier in assignments.
type SLP120 struct{}

func (SLP120) ID() string                { return "SLP120" }
func (SLP120) DefaultSeverity() Severity { return SeverityWarn }
func (SLP120) Description() string {
	return "value discarded with _ = expr — consider using the value or removing the assignment"
}

func (r SLP120) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) || !isGoFile(f.Path) {
			continue
		}

		for _, h := range f.Hunks {
			for _, ln := range h.Lines {
				if ln.Kind != diff.LineAdd {
					continue
				}
				content := stripCommentAndStrings(ln.Content)
				content = strings.TrimSpace(content)
				if content == "" {
					continue
				}

				if strings.HasPrefix(content, "//") || strings.HasPrefix(content, "/*") {
					continue
				}

				if strings.HasPrefix(content, "_ = ") {
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     ln.NewLineNo,
						Message:  "value discarded with '_ =' — the function returns a value that should be used or the assignment removed",
						Snippet:  content,
					})
				}
			}
		}
	}
	return out
}
