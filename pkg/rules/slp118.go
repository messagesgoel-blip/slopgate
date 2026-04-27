package rules

import (
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

type SLP118 struct{}

func (SLP118) ID() string                { return "SLP118" }
func (SLP118) DefaultSeverity() Severity { return SeverityBlock }
func (SLP118) Description() string {
	return "slice or index access without length guard — may panic on empty collection"
}

func (r SLP118) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) {
			continue
		}
		if !isGoFile(f.Path) && !isJSOrTSFile(f.Path) && !isPythonFile(f.Path) {
			continue
		}

		for _, h := range f.Hunks {
			prevContent := ""
			for _, ln := range h.Lines {
				if ln.Kind != diff.LineAdd {
					prevContent = ""
					continue
				}
				content := stripCommentAndStrings(ln.Content)
				content = strings.TrimSpace(content)
				if content == "" {
					prevContent = ""
					continue
				}

				if strings.HasPrefix(content, "if ") || strings.HasPrefix(content, "for ") ||
					strings.HasPrefix(content, "while ") || strings.HasPrefix(content, "//") ||
					strings.HasPrefix(content, "/*") || strings.HasPrefix(content, "*") {
					prevContent = content
					continue
				}

				if strings.HasPrefix(prevContent, "if len") || strings.HasPrefix(prevContent, "if len(") {
					continue
				}

				if strings.Contains(content, "[0]") || strings.Contains(content, "[0]=") ||
					strings.Contains(content, "[1]") || strings.HasSuffix(content, "[0]") ||
					strings.HasSuffix(content, "[1]") {
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     ln.NewLineNo,
						Message:  "direct index access without length guard — may panic on empty collection",
						Snippet:  ln.Content,
					})
				}
			}
		}
	}
	return out
}
